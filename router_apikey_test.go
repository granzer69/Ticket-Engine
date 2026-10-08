package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"ticketengine/internal/booking"
)

func setupAPIKeyBookHandler(t *testing.T) http.HandlerFunc {
	dsn := "file:apikey_" + strings.ReplaceAll(t.Name(), "/", "_") + "?mode=memory"
	testDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("sqlite: %v", err)
	}
	if err := testDB.AutoMigrate(&Ticket{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := testDB.Create(&Ticket{ID: 1, State: "available"}).Error; err != nil {
		t.Fatalf("seed: %v", err)
	}
	db = testDB

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	redisClient = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()
	if err := redisClient.LPush(ctx, booking.KeyQueue, "1").Err(); err != nil {
		t.Fatalf("lpush: %v", err)
	}
	alloc, err := booking.NewAllocator(ctx, redisClient)
	if err != nil {
		t.Fatalf("allocator: %v", err)
	}
	ticketAllocator = alloc
	return apiKeyMiddleware(ticketHandler)
}

func TestAPIKey_Valid(t *testing.T) {
	const key = "slice7-test-api-key"
	t.Setenv("TICKET_API_KEY", key)
	handler := setupAPIKeyBookHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/book", bytes.NewReader(nil))
	req.Header.Set(httpUserHeader, "42")
	req.Header.Set("X-API-Key", key)
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), key) {
		t.Fatal("response must not contain api key")
	}
}

func TestAPIKey_Missing(t *testing.T) {
	const key = "slice7-test-api-key"
	t.Setenv("TICKET_API_KEY", key)
	handler := setupAPIKeyBookHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/book", bytes.NewReader(nil))
	req.Header.Set(httpUserHeader, "42")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d want 401", rec.Code)
	}
	if strings.Contains(rec.Body.String(), key) {
		t.Fatal("response must not contain api key")
	}
}

func TestAPIKey_Invalid(t *testing.T) {
	const key = "slice7-test-api-key"
	t.Setenv("TICKET_API_KEY", key)
	handler := setupAPIKeyBookHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/book", bytes.NewReader(nil))
	req.Header.Set(httpUserHeader, "42")
	req.Header.Set("X-API-Key", "not-the-right-key")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status %d want 401", rec.Code)
	}
	body := rec.Body.String()
	if strings.Contains(body, key) {
		t.Fatal("response must not echo required api key")
	}
}

func TestAPIKey_NotRequiredOnPublicRoutes(t *testing.T) {
	const key = "slice7-test-api-key"
	t.Setenv("TICKET_API_KEY", key)

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	redisClient = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()
	if err := redisClient.LPush(ctx, booking.KeyQueue, "9", "8").Err(); err != nil {
		t.Fatalf("lpush: %v", err)
	}

	metricsReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	metricsRec := httptest.NewRecorder()
	metricsHandler(metricsRec, metricsReq)
	if metricsRec.Code != http.StatusOK {
		t.Fatalf("metrics status %d body %s", metricsRec.Code, metricsRec.Body.String())
	}

	countReq := httptest.NewRequest(http.MethodGet, "/tickets/count", nil)
	countRec := httptest.NewRecorder()
	ticketCountHandler(countRec, countReq)
	if countRec.Code != http.StatusOK {
		t.Fatalf("tickets/count status %d body %s", countRec.Code, countRec.Body.String())
	}
	if !strings.Contains(countRec.Body.String(), `"remaining":2`) {
		t.Fatalf("unexpected count body %s", countRec.Body.String())
	}

	bookHandler := setupAPIKeyBookHandler(t)
	bookReq := httptest.NewRequest(http.MethodPost, "/book", bytes.NewReader(nil))
	bookReq.Header.Set(httpUserHeader, "1")
	bookRec := httptest.NewRecorder()
	bookHandler(bookRec, bookReq)
	if bookRec.Code != http.StatusUnauthorized {
		t.Fatalf("book without key status %d want 401", bookRec.Code)
	}
}

func TestBookHandlerIdempotentHTTP_WithAPIKey(t *testing.T) {
	const key = "slice7-idempotent-key"
	t.Setenv("TICKET_API_KEY", key)
	handler := setupAPIKeyBookHandler(t)

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/book", bytes.NewReader(nil))
		req.Header.Set(httpUserHeader, "77")
		req.Header.Set("X-API-Key", key)
		rec := httptest.NewRecorder()
		handler(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("request %d status %d body %s", i, rec.Code, rec.Body.String())
		}
	}
}

func TestAPIKey_OpenWhenEnvUnset(t *testing.T) {
	t.Setenv("TICKET_API_KEY", "")
	handler := setupAPIKeyBookHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/book", bytes.NewReader(nil))
	req.Header.Set(httpUserHeader, "43")
	rec := httptest.NewRecorder()
	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
}
