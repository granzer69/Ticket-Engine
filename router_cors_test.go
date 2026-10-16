package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORS_reflectsConfiguredOrigin(t *testing.T) {
	t.Setenv("TICKET_CORS_ORIGIN", "https://lab.example.com")
	req := httptest.NewRequest(http.MethodOptions, "/book", nil)
	rec := httptest.NewRecorder()
	enableCORS(func(w http.ResponseWriter, _ *http.Request) {})(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "https://lab.example.com" {
		t.Fatalf("origin header %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORS_devWildcardAllowedWhenNotProduction(t *testing.T) {
	t.Setenv("TICKET_ENV", "")
	t.Setenv("TICKET_CORS_ORIGIN", "*")
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	enableCORS(func(w http.ResponseWriter, _ *http.Request) {})(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatal("expected wildcard in dev")
	}
}
