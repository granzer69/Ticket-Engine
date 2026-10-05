package persist

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestConsumerAppliesSoldState(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()

	db, err := gorm.Open(sqlite.Open("file:persist?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE tickets (id INTEGER PRIMARY KEY, user_id INTEGER, state TEXT, sold_at DATETIME)`).Error; err != nil {
		t.Fatalf("schema: %v", err)
	}
	if err := db.Exec(`INSERT INTO tickets (id, state) VALUES (7, 'available')`).Error; err != nil {
		t.Fatalf("insert: %v", err)
	}

	c := NewConsumer(rdb, db, "t1")
	if err := c.EnsureGroup(ctx); err != nil {
		t.Fatalf("group: %v", err)
	}
	if err := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: StreamKey,
		Values: map[string]interface{}{"ticket_id": "7", "user_id": "3"},
	}).Err(); err != nil {
		t.Fatalf("xadd: %v", err)
	}

	msg, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group: ConsumerGroup, Consumer: "t1", Streams: []string{StreamKey, ">"}, Count: 1,
	}).Result()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if err := c.handleMessage(ctx, msg[0].Messages[0]); err != nil {
		t.Fatalf("handle: %v", err)
	}
	var state string
	if err := db.Raw(`SELECT state FROM tickets WHERE id = 7`).Scan(&state).Error; err != nil {
		t.Fatalf("scan: %v", err)
	}
	if state != "sold" {
		t.Fatalf("expected sold, got %s", state)
	}
}

func TestConsumerDuplicateHandleIdempotent(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()

	db, err := gorm.Open(sqlite.Open("file:dup?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE tickets (id INTEGER PRIMARY KEY, user_id INTEGER, state TEXT, sold_at DATETIME)`).Error; err != nil {
		t.Fatalf("schema: %v", err)
	}
	if err := db.Exec(`INSERT INTO tickets (id, state) VALUES (9, 'available')`).Error; err != nil {
		t.Fatalf("insert: %v", err)
	}

	c := NewConsumer(rdb, db, "t1")
	msg := redis.XMessage{
		ID:     "1-0",
		Values: map[string]interface{}{"ticket_id": "9", "user_id": "2"},
	}
	if err := c.handleMessage(ctx, msg); err != nil {
		t.Fatalf("first handle: %v", err)
	}
	if err := c.handleMessage(ctx, msg); err != nil {
		t.Fatalf("second handle: %v", err)
	}
	var state string
	if err := db.Raw(`SELECT state FROM tickets WHERE id = 9`).Scan(&state).Error; err != nil {
		t.Fatalf("scan: %v", err)
	}
	if state != "sold" {
		t.Fatalf("expected sold, got %s", state)
	}
}

func TestReclaimStalePending(t *testing.T) {
	oldIdle := ReclaimMinIdle
	ReclaimMinIdle = 50 * time.Millisecond
	t.Cleanup(func() { ReclaimMinIdle = oldIdle })

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	start := time.Unix(1_700_000_000, 0)
	mr.SetTime(start)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()

	db, err := gorm.Open(sqlite.Open("file:reclaim?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE tickets (id INTEGER PRIMARY KEY, user_id INTEGER, state TEXT, sold_at DATETIME)`).Error; err != nil {
		t.Fatalf("schema: %v", err)
	}
	if err := db.Exec(`INSERT INTO tickets (id, state) VALUES (11, 'available')`).Error; err != nil {
		t.Fatalf("insert: %v", err)
	}

	c := NewConsumer(rdb, db, "reclaimer")
	if err := c.EnsureGroup(ctx); err != nil {
		t.Fatalf("group: %v", err)
	}
	if err := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: StreamKey,
		Values: map[string]interface{}{"ticket_id": "11", "user_id": "5"},
	}).Err(); err != nil {
		t.Fatalf("xadd: %v", err)
	}

	_, err = rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group: ConsumerGroup, Consumer: "dead-worker", Streams: []string{StreamKey, ">"}, Count: 1,
	}).Result()
	if err != nil {
		t.Fatalf("dead read: %v", err)
	}

	mr.SetTime(start.Add(200 * time.Millisecond))
	c.reclaimStale(ctx)
	if !c.WaitIdle(2 * time.Second) {
		t.Fatal("reclaim handlers did not finish")
	}

	var state string
	if err := db.Raw(`SELECT state FROM tickets WHERE id = 11`).Scan(&state).Error; err != nil {
		t.Fatalf("scan: %v", err)
	}
	if state != "sold" {
		t.Fatalf("expected sold after reclaim, got %s", state)
	}

	pending, err := c.GroupPendingCount(ctx)
	if err != nil {
		t.Fatalf("pending: %v", err)
	}
	if pending != 0 {
		t.Fatalf("expected pending 0 after reclaim, got %d", pending)
	}
}

func TestConsumerReadsOwnPending(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()

	db, err := gorm.Open(sqlite.Open("file:pending0?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE tickets (id INTEGER PRIMARY KEY, user_id INTEGER, state TEXT, sold_at DATETIME)`).Error; err != nil {
		t.Fatalf("schema: %v", err)
	}
	if err := db.Exec(`INSERT INTO tickets (id, state) VALUES (12, 'available')`).Error; err != nil {
		t.Fatalf("insert: %v", err)
	}

	c := NewConsumer(rdb, db, "worker-1")
	if err := c.EnsureGroup(ctx); err != nil {
		t.Fatalf("group: %v", err)
	}
	if err := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: StreamKey,
		Values: map[string]interface{}{"ticket_id": "12", "user_id": "6"},
	}).Err(); err != nil {
		t.Fatalf("xadd: %v", err)
	}

	_, err = rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group: ConsumerGroup, Consumer: "worker-1", Streams: []string{StreamKey, ">"}, Count: 1,
	}).Result()
	if err != nil {
		t.Fatalf("first read: %v", err)
	}

	c.readAndProcess(ctx, "0", 0)
	if !c.WaitIdle(2 * time.Second) {
		t.Fatal("pending handlers did not finish")
	}

	var state string
	if err := db.Raw(`SELECT state FROM tickets WHERE id = 12`).Scan(&state).Error; err != nil {
		t.Fatalf("scan: %v", err)
	}
	if state != "sold" {
		t.Fatalf("expected sold from pending replay, got %s", state)
	}
}

func TestMovesFailedMessageToDLQ(t *testing.T) {
	oldMax := MaxDeliveryAttempts
	MaxDeliveryAttempts = 1
	t.Cleanup(func() { MaxDeliveryAttempts = oldMax })

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()

	db, err := gorm.Open(sqlite.Open("file:dlq?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE tickets (id INTEGER PRIMARY KEY, user_id INTEGER, state TEXT, sold_at DATETIME)`).Error; err != nil {
		t.Fatalf("schema: %v", err)
	}

	c := NewConsumer(rdb, db, "dlq-worker")
	if err := c.EnsureGroup(ctx); err != nil {
		t.Fatalf("group: %v", err)
	}
	if err := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: StreamKey,
		Values: map[string]interface{}{"ticket_id": "404", "user_id": "9"},
	}).Err(); err != nil {
		t.Fatalf("xadd: %v", err)
	}

	read, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group: ConsumerGroup, Consumer: "dlq-worker", Streams: []string{StreamKey, ">"}, Count: 1,
	}).Result()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	c.processMessageSync(ctx, read[0].Messages[0])

	pending, err := c.GroupPendingCount(ctx)
	if err != nil {
		t.Fatalf("pending: %v", err)
	}
	if pending != 0 {
		t.Fatalf("expected no pending after dlq, got %d", pending)
	}

	dlqLen, err := rdb.XLen(ctx, DLQStreamKey).Result()
	if err != nil {
		t.Fatalf("xlen dlq: %v", err)
	}
	if dlqLen != 1 {
		t.Fatalf("expected 1 dlq entry, got %d", dlqLen)
	}
}
