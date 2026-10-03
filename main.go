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

	"ticketengine/internal/persist"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "seed" {
		runSeedCommand()
		return
	}
	if len(os.Args) > 1 && os.Args[1] == "reconcile" {
		runReconcileCommand()
		return
	}

	log.Println("Starting system initialization...")
	initMySQL()
	initRedis()
	initBookingAllocator()

	consumer := persist.NewConsumer(redisClient, db, persist.ConsumerName)
	if err := consumer.EnsureGroup(context.Background()); err != nil {
		log.Fatalf("persist consumer group: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go consumer.Run(ctx)

	prepareRuntimeData()

	if httpPort == "" {
		httpPort = "8080"
	}

	http.HandleFunc("/book", enableCORS(apiKeyMiddleware(ticketHandler)))
	http.HandleFunc("/tickets/count", enableCORS(ticketCountHandler))
	http.HandleFunc("/metrics", enableCORS(metricsHandler))
	http.HandleFunc("/healthz", healthzHandler)
	http.HandleFunc("/readyz", readyzHandler(consumer))

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
		log.Println("shutdown signal received, draining...")
		cancel()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("Server listening on port %s", httpPort)
	log.Printf("Endpoints: POST /book | GET /tickets/count | GET /metrics | GET /healthz | GET /readyz")
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
	log.Println("server stopped")
}

func runSeedCommand() {
	log.Println("Running inventory seed (explicit command)...")
	initMySQL()
	initRedis()
	if err := SeedInventory(); err != nil {
		log.Fatalf("seed failed: %v", err)
	}
	log.Println("Seed complete.")
}

func runReconcileCommand() {
	initMySQL()
	initRedis()
	if err := RunReconciliation(context.Background()); err != nil {
		log.Fatalf("reconcile failed: %v", err)
	}
	log.Println("reconcile complete")
}
