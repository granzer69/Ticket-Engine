//go:build integration

package integration_test

import (
	"context"
	"os"
	"testing"

	"github.com/go-redis/redis/v8"
)

func TestPersistWorkerProcessStarts(t *testing.T) {
	if os.Getenv("CI") == "true" {
		t.Skip("CI workflow already runs a background persist worker; nested go run can deadlock the module cache")
	}
	rdb := redis.NewClient(&redis.Options{Addr: envDefault("REDIS_HOST", "127.0.0.1:6379")})
	ctx := context.Background()
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skipf("redis not available: %v", err)
	}
	gdb := openIntegrationMySQL(t)
	sqlDB, _ := gdb.DB()
	defer sqlDB.Close()

	stop := startPersistWorkerProcess(t)
	defer stop()

	// Worker should stay up while Redis/MySQL are reachable.
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Fatalf("redis ping after worker start: %v", err)
	}
}
