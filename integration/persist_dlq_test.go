//go:build integration

package integration_test

import (
	"context"
	"testing"
	"time"

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
	sqlDB, _ := gdb.DB()
	defer sqlDB.Close()
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

	runCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	consumer := persist.NewConsumer(rdb, gdb, persist.ConsumerName)
	if err := consumer.EnsureGroup(ctx); err != nil {
		t.Fatalf("ensure group: %v", err)
	}
	go consumer.Run(runCtx)

	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		dlqLen, err := rdb.XLen(ctx, persist.DLQStreamKey).Result()
		if err != nil {
			t.Fatalf("dlq len: %v", err)
		}
		if dlqLen >= 1 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

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
