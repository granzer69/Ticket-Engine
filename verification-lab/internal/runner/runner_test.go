package runner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSmokePresetMockTarget(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/metrics":
			_ = json.NewEncoder(w).Encode(map[string]int64{"total_requests": 1, "booking_success": 1})
		case "/book":
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	eng := New(Config{TargetBase: srv.URL})
	id, err := eng.StartSmoke(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		rep, ok := eng.Get(id)
		if ok && rep.Status != "running" {
			if rep.Status != "completed" || rep.HTTP200 != 100 {
				t.Fatalf("report: %+v", rep)
			}
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("timeout waiting for smoke run")
}
