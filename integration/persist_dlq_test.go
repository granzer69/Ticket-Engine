//go:build integration

package integration_test

import (
	"context"
	"testing"

	"ticketengine/internal/persist"

	"github.com/go-redis/redis/v8"
)

func TestPersistDLQ(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: envDefault("REDIS_HOST", "127.0.0.1:6379")})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not available: %v", err)
	}
	gdb := openIntegrationMySQL(t)
	ensureTicketsTable(t, gdb)

	const ticketID = 88002
	const ownerID = 44002
	const otherID = 44003
	if err := gdb.Exec(`DELETE FROM tickets WHERE id = ?`, ticketID).Error; err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := gdb.Exec(`INSERT INTO tickets (id, user_id, state) VALUES (?, ?, 'sold')`, ticketID, ownerID).Error; err != nil {
		t.Fatalf("insert: %v", err)
	}

	oldMax := persist.MaxDeliveryAttempts
	persist.MaxDeliveryAttempts = 1
	t.Cleanup(func() { persist.MaxDeliveryAttempts = oldMax })

	_ = rdb.Del(ctx, persist.StreamKey, persist.DLQStreamKey)
	if err := persist.EnqueueReplay(ctx, rdb, ticketID, otherID); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	const consumerName = "integration-dlq"
	consumer := persist.NewConsumer(rdb, gdb, consumerName)
	if err := consumer.EnsureGroup(ctx); err != nil {
		t.Fatalf("ensure group: %v", err)
	}

	read, err := rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    persist.ConsumerGroup,
		Consumer: consumerName,
		Streams:  []string{persist.StreamKey, ">"},
		Count:    1,
	}).Result()
	if err != nil {
		t.Fatalf("xreadgroup: %v", err)
	}
	if len(read) == 0 || len(read[0].Messages) != 1 {
		t.Fatalf("expected one stream message, got %v", read)
	}
	consumer.ProcessMessageSync(ctx, read[0].Messages[0])

	dlqLen, err := rdb.XLen(ctx, persist.DLQStreamKey).Result()
	if err != nil {
		t.Fatalf("dlq len: %v", err)
	}
	if dlqLen != 1 {
		t.Fatalf("expected 1 dlq message, got %d", dlqLen)
	}

	pending, err := consumer.GroupPendingCount(ctx)
	if err != nil {
		t.Fatalf("pending: %v", err)
	}
	if pending != 0 {
		t.Fatalf("expected original stream pending 0 after dlq ack, got %d", pending)
	}

	var state string
	var uid int64
	if err := gdb.Raw(`SELECT state, user_id FROM tickets WHERE id = ?`, ticketID).Row().Scan(&state, &uid); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if state != "sold" || uid != ownerID {
		t.Fatalf("ticket must stay sold to original owner, got state=%s user=%d", state, uid)
	}
}
