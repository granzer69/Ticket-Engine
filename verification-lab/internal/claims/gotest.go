package claims

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// GoTestOutcome is the result of a single go test invocation.
type GoTestOutcome struct {
	Command  string `json:"go_test_command"`
	ExitCode int    `json:"exit_code"`
	Passed   bool   `json:"passed"`
	Skipped  bool   `json:"skipped"`
	Output   string `json:"output_tail"`
}

// RunGoTest runs `go test` in repoRoot for pkg with -run pattern.
func RunGoTest(ctx context.Context, repoRoot, pkg, runPattern string, extraArgs []string) GoTestOutcome {
	out := GoTestOutcome{Command: "", ExitCode: -1}
	if repoRoot == "" {
		out.Output = "VERILAB_REPO_ROOT not set and go.mod not found from cwd"
		out.Skipped = true
		return out
	}
	if ctx.Err() != nil {
		out.Output = ctx.Err().Error()
		return out
	}

	args := []string{"test", "-count=1", "-timeout", "3m"}
	for _, a := range extraArgs {
		args = append(args, a)
	}
	if runPattern != "" {
		args = append(args, "-run", runPattern)
	}
	args = append(args, pkg)

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = repoRoot
	cmd.Env = os.Environ()

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	out.Command = fmt.Sprintf("cd %s && go %s", repoRoot, strings.Join(args, " "))
	err := cmd.Run()
	if err == nil {
		out.ExitCode = 0
	} else if ee, ok := err.(*exec.ExitError); ok {
		out.ExitCode = ee.ExitCode()
	} else {
		out.Output = tailLines(buf.String(), 40)
		out.Output += "\n" + err.Error()
		return out
	}
	text := buf.String()
	out.Output = tailLines(text, 40)
	out.Passed = out.ExitCode == 0 && !goTestOutputFailed(text)
	out.Skipped = strings.Contains(text, "[no tests to run]") || strings.Contains(text, "build constraints exclude all Go files")
	return out
}

func goTestOutputFailed(text string) bool {
	if strings.Contains(text, "--- FAIL:") {
		return true
	}
	lines := strings.Split(text, "\n")
	for _, ln := range lines {
		if strings.HasPrefix(ln, "FAIL\t") || strings.HasPrefix(ln, "FAIL ") {
			return true
		}
	}
	return false
}

func tailLines(s string, max int) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) <= max {
		return s
	}
	return strings.Join(lines[len(lines)-max:], "\n")
}

// DiscoverRepoRoot returns VERILAB_REPO_ROOT or the nearest directory containing go.mod.
func DiscoverRepoRoot() string {
	if v := os.Getenv("VERILAB_REPO_ROOT"); v != "" {
		return v
	}
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	for dir := cwd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
	}
	return ""
}

// ServicesReachable reports whether Redis and MySQL DSN are likely usable for integration tests.
func ServicesReachable(redisAddr, mysqlDSN string) bool {
	if redisAddr == "" || mysqlDSN == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if !pingRedis(ctx, redisAddr) {
		return false
	}
	return pingMySQL(ctx, mysqlDSN)
}

func pingRedis(ctx context.Context, addr string) bool {
	d := net.Dialer{Timeout: 2 * time.Second}
	c, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return false
	}
	c.Close()
	return true
}

func pingMySQL(ctx context.Context, dsn string) bool {
	gdb, err := gorm.Open(mysql.Open(dsn), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		return false
	}
	sqlDB, err := gdb.DB()
	if err != nil {
		return false
	}
	defer sqlDB.Close()
	return sqlDB.PingContext(ctx) == nil
}
