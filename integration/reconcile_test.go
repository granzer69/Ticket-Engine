//go:build integration

package integration_test

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/go-redis/redis/v8"

	"ticketengine/internal/booking"
	"ticketengine/internal/reconcile"
)

func TestReconcileDrift(t *testing.T) {
	gdb := openIntegrationMySQL(t)
	rdb := openIntegrationRedis(t)
	ctx := context.Background()

	if err := gdb.Exec("DELETE FROM tickets").Error; err != nil {
		t.Fatal(err)
	}
	if err := gdb.Exec("INSERT INTO tickets (id, user_id, state) VALUES (1, NULL, 'available')").Error; err != nil {
		t.Fatal(err)
	}
	_ = rdb.Del(ctx, booking.KeyQueue, booking.KeyUserBooking, "bookings.stream", "bookings.dlq")
	_ = rdb.RPush(ctx, booking.KeyQueue, 1, 2)

	rep, err := reconcile.Run(ctx, gdb, rdb)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !rep.HasDrift() {
		t.Fatalf("expected drift: %s", rep.Format())
	}
	if !strings.Contains(rep.Format(), "DRIFT DETECTED") {
		t.Fatal(rep.Format())
	}
}

func TestReconcileCLIReadOnly(t *testing.T) {
	gdb := openIntegrationMySQL(t)
	rdb := openIntegrationRedis(t)
	ctx := context.Background()

	if err := gdb.Exec("DELETE FROM tickets").Error; err != nil {
		t.Fatal(err)
	}
	if err := gdb.Exec("INSERT INTO tickets (id, user_id, state) VALUES (10, NULL, 'available')").Error; err != nil {
		t.Fatal(err)
	}
	_ = rdb.Del(ctx, booking.KeyQueue)
	_ = rdb.RPush(ctx, booking.KeyQueue, 10)

	beforeQ, _ := rdb.LLen(ctx, booking.KeyQueue).Result()
	beforeH, _ := rdb.HLen(ctx, booking.KeyUserBooking).Result()

	cmd := exec.Command("go", "run", ".", "reconcile")
	cmd.Dir = moduleRootDir()
	cmd.Env = persistWorkerEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) || exitErr.ExitCode() == reconcile.ExitOperational {
			t.Fatalf("reconcile command failed: %v\n%s", err, out)
		}
	}
	if !strings.Contains(string(out), "RECONCILE RESULT:") {
		t.Fatalf("missing reconcile header in output:\n%s", out)
	}
	afterQ, _ := rdb.LLen(ctx, booking.KeyQueue).Result()
	afterH, _ := rdb.HLen(ctx, booking.KeyUserBooking).Result()
	if beforeQ != afterQ || beforeH != afterH {
		t.Fatalf("redis mutated: q %d->%d h %d->%d", beforeQ, afterQ, beforeH, afterH)
	}
}

func openIntegrationRedis(t *testing.T) *redis.Client {
	host := envDefault("REDIS_HOST", "127.0.0.1:6379")
	rdb := redis.NewClient(&redis.Options{Addr: host})
	if err := rdb.Ping(context.Background()).Err(); err != nil {
		t.Skipf("redis not available: %v", err)
	}
	return rdb
}
