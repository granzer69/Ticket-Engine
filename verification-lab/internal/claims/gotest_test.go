package claims

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunGoTestP5InRepo(t *testing.T) {
	root := DiscoverRepoRoot()
	if root == "" {
		t.Skip("repo root not found")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	out := RunGoTest(ctx, root, ".", "TestBookReplaySkipsMySQL", nil)
	if !out.Passed {
		t.Fatalf("expected pass: %+v\n%s", out, out.Output)
	}
	if out.Command == "" {
		t.Fatal("expected command string")
	}
}

func TestDiscoverRepoRootFromModule(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	// When tests run from verification-lab/internal/claims, discovery should find module root.
	root := DiscoverRepoRoot()
	if root == "" {
		t.Skip("no go.mod ancestor from " + wd)
	}
	if _, err := os.Stat(filepath.Join(root, "go.mod")); err != nil {
		t.Fatalf("invalid root %s: %v", root, err)
	}
}
