package claims

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestINV3IdempotentReplay(t *testing.T) {
	var ticket int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ticket == 0 {
			ticket = 42
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "success",
			"ticket_id": ticket,
			"user_id":   1,
		})
	}))
	defer srv.Close()

	res := Run(context.Background(), Client{BaseURL: srv.URL}, []string{"INV-3"})
	if len(res) != 1 || res[0].Status != StatusPass {
		t.Fatalf("got %+v", res[0])
	}
}

func TestINV1UniqueTickets(t *testing.T) {
	var mu sync.Mutex
	issued := map[int]int{}
	next := 100
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		next++
		tid := next
		issued[tid]++
		mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "ticket_id": tid})
	}))
	defer srv.Close()

	res := Run(context.Background(), Client{BaseURL: srv.URL}, []string{"INV-1"})
	if res[0].Status != StatusPass {
		t.Fatalf("INV-1: %+v", res[0])
	}
}
