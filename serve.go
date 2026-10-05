package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

// bootstrapAPIServer initializes dependencies for HTTP serve mode. It does not start the persist consumer.
func bootstrapAPIServer() {
	log.Println("Starting system initialization...")
	initMySQL()
	initRedis()
	initBookingAllocator()
	prepareRuntimeData()
}

func runServeCommand() {
	bootstrapAPIServer()

	if httpPort == "" {
		httpPort = "8080"
	}

	http.HandleFunc("/book", enableCORS(apiKeyMiddleware(ticketHandler)))
	http.HandleFunc("/tickets/count", enableCORS(ticketCountHandler))
	http.HandleFunc("/metrics", enableCORS(metricsHandler))
	http.HandleFunc("/healthz", healthzHandler)
	http.HandleFunc("/readyz", readyzHandler(nil))

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			reqs := atomic.LoadInt64(&totalRequests)
			if reqs > 0 {
				log.Printf("[metrics] requests=%d success=%d failures=%d",
					reqs,
					atomic.LoadInt64(&bookingSuccess),
					atomic.LoadInt64(&bookingFailures),
				)
			}
		}
	}()

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", httpPort),
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutdown signal received, draining HTTP...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("Server listening on port %s", httpPort)
	log.Printf("Endpoints: POST /book | GET /tickets/count | GET /metrics | GET /healthz | GET /readyz")
	log.Println("NOTE: serve mode does not run the persist consumer; run `go run . worker` to persist stream bookings to MySQL")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
	log.Println("server stopped")
}
