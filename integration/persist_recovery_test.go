//go:build integration

package integration_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"ticketengine/internal/persist"

	"github.com/go-redis/redis/v8"
)

func TestPersistRecovery(t *testing.T) {
	oldIdle := persist.ReclaimMinIdle
	oldInterval := persist.ReclaimInterval
	persist.ReclaimMinIdle = 50 * time.Millisecond
	persist.ReclaimInterval = 100 * time.Millisecond
	t.Cleanup(func() {
		persist.ReclaimMinIdle = oldIdle
		persist.ReclaimInterval = oldInterval
	})

	rdb := redis.NewClient(&redis.Options{Addr: envDefault("REDIS_HOST", "127.0.0.1:6379")})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not available: %v", err)
	}
	gdb := openIntegrationMySQL(t)
	ensureTicketsTable(t, gdb)

	const ticketID = 88001
	const userID = 44001
	if err := gdb.Exec(`DELETE FROM tickets WHERE id = ?`, ticketID).Error; err != nil {
		t.Fatalf("delete ticket: %v", err)
	}
	if err := gdb.Exec(`INSERT INTO tickets (id, user_id, state) VALUES (?, NULL, 'available')`, ticketID).Error; err != nil {
		t.Fatalf("insert ticket: %v", err)
	}

	_ = rdb.Del(ctx, persist.StreamKey)
	if err := persist.EnqueueReplay(ctx, rdb, ticketID, userID); err != nil {
		t.Fatalf("enqueue: %v", err)
	}

	pauseStarted := make(chan struct{})
	var pauseOnce sync.Once
	persist.MessagePause = func(pauseCtx context.Context, tid int) error {
		if tid != ticketID {
			return nil
		}
		pauseOnce.Do(func() { close(pauseStarted) })
		select {
		case <-pauseCtx.Done():
			return pauseCtx.Err()
		case <-time.After(30 * time.Second):
			return errors.New("pause timeout")
		}
	}
	t.Cleanup(func() { persist.MessagePause = nil })

	runCtx, runCancel := context.WithCancel(ctx)
	consumer := persist.NewConsumer(rdb, gdb, "integration-pause")
	if err := consumer.EnsureGroup(ctx); err != nil {
		t.Fatalf("ensure group: %v", err)
	}

	go consumer.Run(runCtx)

	select {
	case <-pauseStarted:
	case <-time.After(10 * time.Second):
		t.Fatal("consumer did not reach handle pause")
	}

	runCancel()
	time.Sleep(300 * time.Millisecond)

	recoverCtx, recoverCancel := context.WithTimeout(ctx, 15*time.Second)
	defer recoverCancel()
	recoverConsumer := persist.NewConsumer(rdb, gdb, "integration-recover")
	go recoverConsumer.Run(recoverCtx)

	deadline := time.Now().Add(12 * time.Second)
	recovered := false
	for time.Now().Before(deadline) {
		sold, err := persist.IsSold(gdb, ticketID)
		if err != nil {
			t.Fatalf("is sold: %v", err)
		}
		if sold {
			recovered = true
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if !recovered {
		t.Fatal("ticket not sold after recovery window")
	}
	recoverCancel()

	var state string
	var uid int64
	if err := gdb.Raw(`SELECT state, user_id FROM tickets WHERE id = ?`, ticketID).Row().Scan(&state, &uid); err != nil {
		t.Fatalf("scan ticket: %v", err)
	}
	if state != "sold" {
		t.Fatalf("expected sold, got %s", state)
	}
	if uid != userID {
		t.Fatalf("expected user_id %d, got %d", userID, uid)
	}

	var soldCount int64
	if err := gdb.Raw(`SELECT COUNT(*) FROM tickets WHERE id = ? AND state = 'sold'`, ticketID).Scan(&soldCount).Error; err != nil {
		t.Fatalf("count sold: %v", err)
	}
	if soldCount != 1 {
		t.Fatalf("expected one sold row, got %d", soldCount)
	}

	pending, err := consumer.GroupPendingCount(ctx)
	if err != nil {
		t.Fatalf("pending: %v", err)
	}
	if pending != 0 {
		t.Fatalf("expected no group pending after recovery, got %d", pending)
	}
}
