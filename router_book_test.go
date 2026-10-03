package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"ticketengine/internal/booking"
	"ticketengine/internal/persist"
)

func TestBookHandlerIdempotentHTTP(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open("file:booktest?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("sqlite: %v", err)
	}
	if err := testDB.AutoMigrate(&Ticket{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := testDB.Create(&Ticket{ID: 42, State: "available"}).Error; err != nil {
		t.Fatalf("seed row: %v", err)
	}
	db = testDB

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	redisClient = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()
	if err := redisClient.LPush(ctx, booking.KeyQueue, "42").Err(); err != nil {
		t.Fatalf("lpush: %v", err)
	}
	alloc, err := booking.NewAllocator(ctx, redisClient)
	if err != nil {
		t.Fatalf("allocator: %v", err)
	}
	ticketAllocator = alloc
	if err := persist.NewConsumer(redisClient, db, "test").EnsureGroup(ctx); err != nil {
		t.Fatalf("group: %v", err)
	}

	handler := apiKeyMiddleware(ticketHandler)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/book", bytes.NewReader(nil))
		req.Header.Set(httpUserHeader, "99")
		res := httptest.NewRecorder()
		handler(res, req)
		if res.Code != http.StatusOK {
			t.Fatalf("request %d status %d body %s", i, res.Code, res.Body.String())
		}
	}
}
