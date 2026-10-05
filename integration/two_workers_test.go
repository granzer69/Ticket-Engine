//go:build integration

package integration_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"ticketengine/internal/persist"

	"github.com/go-redis/redis/v8"
)

func TestTwoWorkersNoDoubleSell(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: envDefault("REDIS_HOST", "127.0.0.1:6379")})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not available: %v", err)
	}
	gdb := openIntegrationMySQL(t)
	ensureTicketsTable(t, gdb)

	const ticketA = 88101
	const ticketB = 88102
	for _, id := range []int{ticketA, ticketB} {
		if err := gdb.Exec(`DELETE FROM tickets WHERE id = ?`, id).Error; err != nil {
			t.Fatalf("delete: %v", err)
		}
		if err := gdb.Exec(`INSERT INTO tickets (id, user_id, state) VALUES (?, NULL, 'available')`, id).Error; err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	_ = rdb.Del(ctx, persist.StreamKey)
	if err := persist.EnqueueReplay(ctx, rdb, ticketA, 501); err != nil {
		t.Fatalf("enqueue a: %v", err)
	}
	if err := persist.EnqueueReplay(ctx, rdb, ticketB, 502); err != nil {
		t.Fatalf("enqueue b: %v", err)
	}

	runCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	w1 := persist.NewConsumer(rdb, gdb, "worker-a")
	w2 := persist.NewConsumer(rdb, gdb, "worker-b")
	if err := w1.EnsureGroup(ctx); err != nil {
		t.Fatalf("group: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); w1.Run(runCtx) }()
	go func() { defer wg.Done(); w2.Run(runCtx) }()

	deadline := time.Now().Add(12 * time.Second)
	for time.Now().Before(deadline) {
		soldA, _ := persist.IsSold(gdb, ticketA)
		soldB, _ := persist.IsSold(gdb, ticketB)
		if soldA && soldB {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	for _, pair := range []struct {
		id   int
		user int64
	}{
		{ticketA, 501},
		{ticketB, 502},
	} {
		var state string
		var uid int64
		if err := gdb.Raw(`SELECT state, user_id FROM tickets WHERE id = ?`, pair.id).Row().Scan(&state, &uid); err != nil {
			t.Fatalf("scan: %v", err)
		}
		if state != "sold" || uid != pair.user {
			t.Fatalf("ticket %d expected sold to %d, got state=%s user=%d", pair.id, pair.user, state, uid)
		}
	}

	cancel()
	wg.Wait()
}
