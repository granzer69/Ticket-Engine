package persist

import (
	"context"
	"testing"

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
	c.processMessage(ctx, msg[0].Messages[0])

	var state string
	if err := db.Raw(`SELECT state FROM tickets WHERE id = 7`).Scan(&state).Error; err != nil {
		t.Fatalf("scan: %v", err)
	}
	if state != "sold" {
		t.Fatalf("expected sold, got %s", state)
	}
}

func TestPoisonMessageMovesToDLQ(t *testing.T) {
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

	c := NewConsumer(rdb, db, "t1")
	if err := c.EnsureGroup(ctx); err != nil {
		t.Fatalf("group: %v", err)
	}
	id, err := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: StreamKey,
		Values: map[string]interface{}{"ticket_id": "not-a-number", "user_id": "1"},
	}).Result()
	if err != nil {
		t.Fatalf("xadd: %v", err)
	}

	read, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group: ConsumerGroup, Consumer: "t1", Streams: []string{StreamKey, ">"}, Count: 1,
	}).Result()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	c.processMessage(ctx, read[0].Messages[0])

	dlqLen, err := rdb.XLen(ctx, DLQStreamKey).Result()
	if err != nil {
		t.Fatalf("xlen dlq: %v", err)
	}
	if dlqLen != 1 {
		t.Fatalf("expected 1 dlq entry, got %d", dlqLen)
	}

	pending, err := rdb.XPending(ctx, StreamKey, ConsumerGroup).Result()
	if err != nil {
		t.Fatalf("xpending: %v", err)
	}
	if pending.Count != 0 {
		t.Fatalf("expected message acked, pending=%d", pending.Count)
	}
	_ = id
}

func TestTransientFailureRetriesBeforeDLQ(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()

	db, err := gorm.Open(sqlite.Open("file:retry?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("sqlite: %v", err)
	}
	if err := db.Exec(`CREATE TABLE tickets (id INTEGER PRIMARY KEY, user_id INTEGER, state TEXT, sold_at DATETIME)`).Error; err != nil {
		t.Fatalf("schema: %v", err)
	}

	c := NewConsumer(rdb, db, "t1")
	if err := c.EnsureGroup(ctx); err != nil {
		t.Fatalf("group: %v", err)
	}
	if err := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: StreamKey,
		Values: map[string]interface{}{"ticket_id": "99", "user_id": "1"},
	}).Err(); err != nil {
		t.Fatalf("xadd: %v", err)
	}
	read, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group: ConsumerGroup, Consumer: "t1", Streams: []string{StreamKey, ">"}, Count: 1,
	}).Result()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	msg := read[0].Messages[0]
	for i := 0; i < MaxHandleAttempts-1; i++ {
		c.processMessage(ctx, msg)
	}
	pending, err := rdb.XPending(ctx, StreamKey, ConsumerGroup).Result()
	if err != nil {
		t.Fatalf("xpending: %v", err)
	}
	if pending.Count == 0 {
		t.Fatal("expected pending message before success")
	}

	if err := db.Exec(`INSERT INTO tickets (id, state) VALUES (99, 'available')`).Error; err != nil {
		t.Fatalf("insert ticket: %v", err)
	}
	c.processMessage(ctx, msg)

	var state string
	if err := db.Raw(`SELECT state FROM tickets WHERE id = 99`).Scan(&state).Error; err != nil {
		t.Fatalf("scan: %v", err)
	}
	if state != "sold" {
		t.Fatalf("expected sold after retry, got %s", state)
	}
}
