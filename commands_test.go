package main

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"ticketengine/internal/persist"
)

func setupCommandTestDeps(t *testing.T) {
	testDB, err := gorm.Open(sqlite.Open("file:cmdtest?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("sqlite: %v", err)
	}
	if err := testDB.AutoMigrate(&Ticket{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	db = testDB

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	redisClient = redis.NewClient(&redis.Options{Addr: mr.Addr()})
}

func TestServeDoesNotStartConsumer(t *testing.T) {
	setupCommandTestDeps(t)

	var started atomic.Bool
	old := runPersistConsumer
	runPersistConsumer = func(ctx context.Context, c *persist.Consumer) {
		started.Store(true)
	}
	t.Cleanup(func() { runPersistConsumer = old })

	bootstrapAPIServer()

	if started.Load() {
		t.Fatal("serve bootstrap must not start persist consumer")
	}
}

func TestWorkerCommandStartsConsumer(t *testing.T) {
	setupCommandTestDeps(t)

	started := make(chan struct{}, 1)
	old := runPersistConsumer
	runPersistConsumer = func(ctx context.Context, c *persist.Consumer) {
		started <- struct{}{}
		<-ctx.Done()
	}
	t.Cleanup(func() { runPersistConsumer = old })

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		startPersistWorkerLoop(ctx)
	}()
	t.Cleanup(cancel)

	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("worker did not start persist consumer")
	}
	cancel()
	wg.Wait()
}
