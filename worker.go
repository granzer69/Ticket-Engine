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
	name := resolvePersistConsumerName()
	consumer := persist.NewConsumer(redisClient, db, name)
	if err := consumer.EnsureGroup(ctx); err != nil {
		log.Fatalf("persist consumer group: %v", err)
	}
	log.Printf("Persist consumer running (name=%s)", name)
	runPersistConsumer(ctx, consumer)
}

func resolvePersistConsumerName() string {
	if v := os.Getenv("TICKET_PERSIST_CONSUMER_NAME"); v != "" {
		return v
	}
	if h := os.Getenv("HOSTNAME"); h != "" {
		return "worker-" + h
	}
	return persist.ConsumerName
}

func runWorkerCommand() {
	log.Println("Starting persist worker...")
	os.Setenv("TICKET_ENGINE_ROLE", "worker")
	initMySQL()
	initRedis()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		drainCtx, drainCancel := context.WithTimeout(context.Background(), persist.ShutdownDrainTimeout)
		defer drainCancel()

		consumer := persist.NewConsumer(redisClient, db, resolvePersistConsumerName())
		pending, err := consumer.GroupPendingCount(drainCtx)
		if err != nil {
			log.Printf("worker shutdown: group pending count: %v", err)
		} else {
			log.Printf("worker shutdown signal received (group_pending=%d), draining up to %s", pending, persist.ShutdownDrainTimeout)
		}

		cancel()
	}()

	startPersistWorkerLoop(ctx)
	log.Println("persist worker stopped")
}
