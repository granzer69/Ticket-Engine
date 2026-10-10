package claims

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestGateSessionAll runs P1..P7 when VERILAB_GATE_SMOKE=1 (local/CI with repo + optional services).
func TestGateSessionAll(t *testing.T) {
	if os.Getenv("VERILAB_GATE_SMOKE") == "" {
		t.Skip("set VERILAB_GATE_SMOKE=1 to run full gate session")
	}
	root := os.Getenv("VERILAB_REPO_ROOT")
	if root == "" {
		root = DiscoverRepoRoot()
	}
	dsn := os.Getenv("VERILAB_MYSQL_DSN")
	if dsn == "" {
		dsn = "ticket:ticket@tcp(127.0.0.1:3306)/ticketdb?charset=utf8mb4&parseTime=True&loc=Local"
	}
	target := os.Getenv("VERILAB_TARGET")
	if target == "" {
		target = "http://127.0.0.1:8080"
	}
	redis := os.Getenv("REDIS_HOST")
	if redis == "" {
		redis = "127.0.0.1:6379"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	results := RunGates(ctx, GateOpts{
		RepoRoot:   root,
		TargetBase: target,
		RedisAddr:  redis,
		MySQLDSN:   dsn,
	}, DefaultGateIDs)
	for _, r := range results {
		t.Logf("%s %s %s", r.ID, r.Status, r.Message)
	}
	for _, r := range results {
		if r.Status == StatusFail {
			t.Fatalf("gate %s failed: %s", r.ID, r.Message)
		}
	}
}
