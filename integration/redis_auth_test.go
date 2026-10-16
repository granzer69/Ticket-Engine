//go:build integration

package integration_test

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

func TestRedisAuth(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	mr.RequireAuth("integration-secret")
	ctx := context.Background()

	bad := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	if err := bad.Ping(ctx).Err(); err == nil {
		t.Fatal("expected ping failure without password")
	}
	bad.Close()

	good := redis.NewClient(&redis.Options{Addr: mr.Addr(), Password: "integration-secret"})
	if err := good.Ping(ctx).Err(); err != nil {
		t.Fatalf("ping with password: %v", err)
	}
	good.Close()
}
