package main

import (
	"context"
	"net/http"
	"time"

	"ticketengine/internal/persist"
)

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func readyzHandler(consumer *persist.Consumer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := redisClient.Ping(ctx).Err(); err != nil {
			http.Error(w, "redis unavailable", http.StatusServiceUnavailable)
			return
		}
		sqlDB, err := db.DB()
		if err != nil || sqlDB.PingContext(ctx) != nil {
			http.Error(w, "mysql unavailable", http.StatusServiceUnavailable)
			return
		}
		pending, err := persist.GroupPendingCountRDB(ctx, redisClient)
		if err != nil {
			http.Error(w, "stream pending check failed", http.StatusServiceUnavailable)
			return
		}
		if pending > persist.MaxReadyPendingCount {
			http.Error(w, "stream pending too high", http.StatusServiceUnavailable)
			return
		}
		if consumer == nil {
			// Serve mode: stream backlog check above is sufficient for API readiness.
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("ready"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ready"))
	}
}
