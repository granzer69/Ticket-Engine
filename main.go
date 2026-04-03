package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"
)

func main() {
	log.Println("Starting system initialization...")
	initMySQL()
	initRedis()
	initPubSub()
	prepareData()

	log.Println("Initialization complete. Starting pubsub router...")
	go func() {
		if err := pubsubRouter.Run(context.Background()); err != nil {
			log.Fatalf("pubsubRouter error: %v", err)
		}
	}()

	if httpPort == "" {
		httpPort = "8080"
	}

	http.HandleFunc("/book", enableCORS(ticketHandler))
	http.HandleFunc("/tickets/count", enableCORS(ticketCountHandler))
	http.HandleFunc("/metrics", enableCORS(metricsHandler))

	// Metrics logger — prints stats every 5 seconds during load
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

	log.Printf("Server listening on port %s", httpPort)
	log.Printf("Endpoints: POST /book | GET /tickets/count | GET /metrics")
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
