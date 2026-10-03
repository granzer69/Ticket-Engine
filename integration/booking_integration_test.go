//go:build integration

package integration_test

import (
	"context"
	"testing"

	"ticketengine/internal/booking"

	"github.com/go-redis/redis/v8"
)

func TestIntegrationBookingIdempotency(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{Addr: envDefault("REDIS_HOST", "127.0.0.1:6379")})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not available: %v", err)
	}

	_ = rdb.Del(ctx, booking.KeyQueue, booking.KeyUserBooking)
	if err := rdb.LPush(ctx, booking.KeyQueue, "5001", "5002").Err(); err != nil {
		t.Fatalf("lpush: %v", err)
	}

	alloc, err := booking.NewAllocator(ctx, rdb)
	if err != nil {
		t.Fatalf("allocator: %v", err)
	}

	first, err := alloc.Allocate(ctx, 9001)
	if err != nil || first.Replay || first.TicketID == 0 {
		t.Fatalf("first: %+v err=%v", first, err)
	}
	replay, err := alloc.Allocate(ctx, 9001)
	if err != nil || !replay.Replay || replay.TicketID != first.TicketID {
		t.Fatalf("replay: %+v err=%v want ticket %d", replay, err, first.TicketID)
	}
}
