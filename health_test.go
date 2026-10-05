package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"ticketengine/internal/persist"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestReadyzStreamPending(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	oldRedis := redisClient
	oldDB := db
	oldMax := persist.MaxReadyPendingCount
	t.Cleanup(func() {
		redisClient = oldRedis
		db = oldDB
		persist.MaxReadyPendingCount = oldMax
	})

	gdb, err := gorm.Open(sqlite.Open("file:readyz?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("sqlite: %v", err)
	}
	db = gdb

	redisClient = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()
	if err := redisClient.XGroupCreateMkStream(ctx, persist.StreamKey, persist.ConsumerGroup, "0").Err(); err != nil {
		t.Fatalf("group: %v", err)
	}
	if err := redisClient.XAdd(ctx, &redis.XAddArgs{
		Stream: persist.StreamKey,
		Values: map[string]interface{}{"ticket_id": "1", "user_id": "1"},
	}).Err(); err != nil {
		t.Fatalf("xadd: %v", err)
	}
	if _, err := redisClient.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group: persist.ConsumerGroup, Consumer: "probe", Streams: []string{persist.StreamKey, ">"}, Count: 1,
	}).Result(); err != nil {
		t.Fatalf("readgroup: %v", err)
	}

	persist.MaxReadyPendingCount = 0
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()
	readyzHandler(nil)(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when pending > max, got %d body=%s", rec.Code, rec.Body.String())
	}

	persist.MaxReadyPendingCount = 10
	rec2 := httptest.NewRecorder()
	readyzHandler(nil)(rec2, req)
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 when pending within limit, got %d body=%s", rec2.Code, rec2.Body.String())
	}
}
