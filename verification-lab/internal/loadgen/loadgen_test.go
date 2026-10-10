package loadgen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestRunBounded(t *testing.T) {
	var hits int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&hits, 1)
		if r.Header.Get("X-User-Id") == "" {
			http.Error(w, "missing user", 400)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	st, err := Run(context.Background(), Options{
		BaseURL:     srv.URL,
		Workers:     8,
		Total:       50,
		UserIDStart: 1000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if st.LogicalRequests != 50 || st.HTTP200 != 50 || st.Errors != 0 {
		t.Fatalf("stats: %+v hits=%d", st, hits)
	}
	if hits != 50 {
		t.Fatalf("expected 50 hits, got %d", hits)
	}
}
