package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"ticketengine/internal/persist"
)

// runPersistConsumer is the only entry that should invoke consumer.Run (overridden in tests).
var runPersistConsumer = func(ctx context.Context, c *persist.Consumer) {
	c.Run(ctx)
}

func startPersistWorkerLoop(ctx context.Context) {
	consumer := persist.NewConsumer(redisClient, db, persist.ConsumerName)
	if err := consumer.EnsureGroup(ctx); err != nil {
		log.Fatalf("persist consumer group: %v", err)
	}
	log.Printf("Persist consumer running (name=%s)", persist.ConsumerName)
	runPersistConsumer(ctx, consumer)
}

func runWorkerCommand() {
	log.Println("Starting persist worker...")
	initMySQL()
	initRedis()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("worker shutdown signal received")
		cancel()
	}()

	startPersistWorkerLoop(ctx)
	log.Println("persist worker stopped")
}
