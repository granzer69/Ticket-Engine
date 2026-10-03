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
