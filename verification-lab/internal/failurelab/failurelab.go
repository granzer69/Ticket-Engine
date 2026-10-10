package failurelab

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"ticketengine/internal/reconcile"
	"ticketengine/verification-lab/internal/claims"
	"ticketengine/verification-lab/internal/metrics"
)

// DefaultScenarioIDs is the full safe failure-lab sequence.
var DefaultScenarioIDs = []string{
	"worker-unavailable",
	"restart-inventory",
	"dlq-growth",
	"reconcile-after-stress",
}

// Registry maps scenario IDs to documented tests (claim registry alignment).
var Registry = map[string][]string{
	"worker-unavailable": {
		"integration:TestPersistRecovery",
		"health:GET /readyz stream pending",
	},
	"restart-inventory": {
		"integration:TestRestartDoesNotMintInventory",
		"internal/inventory:TestSeedDecision",
	},
	"dlq-growth": {
		"internal/persist:TestMovesFailedMessageToDLQ",
		"integration:TestPersistDLQ",
		"gates:P3",
	},
	"reconcile-after-stress": {
		"internal/reconcile:TestReconcileInventoryDrift",
		"integration:TestReconcileCLIReadOnly",
		"gates:P6",
	},
}

// ScenarioMeta describes a failure-lab scenario for operators and the UI catalog.
type ScenarioMeta struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Warning     string   `json:"warning"`
	ManualSteps []string `json:"manual_steps,omitempty"`
	SafeAuto    bool     `json:"safe_automated"`
	Registry    []string `json:"registry_tests"`
}

// Catalog returns static scenario metadata (no side effects).
func Catalog() []ScenarioMeta {
	return []ScenarioMeta{
		{
			ID:      "worker-unavailable",
			Title:   "Worker stalled — stream pending grows",
			Warning: "Stop the persist worker before running; lab only probes pending via Redis (no arbitrary writes).",
			ManualSteps: []string{
				"Stop `go run . worker` (or docker worker service) while keeping API + Redis up.",
				"Run this scenario: one bounded POST /book, then compare stream pending before/after.",
				"Restart the worker after inspection to drain the stream.",
			},
			SafeAuto: true,
			Registry: Registry["worker-unavailable"],
		},
		{
			ID:      "restart-inventory",
			Title:   "Serve restart does not mint inventory (INV-4)",
			Warning: "Automated path runs integration TestRestartDoesNotMintInventory when MySQL is reachable; full process restart remains manual.",
			ManualSteps: []string{
				"Note MySQL ticket row count (or lab snapshot mysql.total).",
				"Restart API (`go run .`) without running `seed` again.",
				"Confirm row count unchanged; compare with integration test output in evidence.",
			},
			SafeAuto: false,
			Registry: Registry["restart-inventory"],
		},
		{
			ID:          "dlq-growth",
			Title:       "DLQ length (read-only)",
			Warning:     "Read-only Redis XLEN on bookings.dlq; inducing DLQ growth is covered by P3 go tests, not browser automation.",
			ManualSteps: []string{},
			SafeAuto:    true,
			Registry:    Registry["dlq-growth"],
		},
		{
			ID:      "reconcile-after-stress",
			Title:   "Reconcile CLI after load (read-only)",
			Warning: "Runs `go run . reconcile` read-only against current Redis/MySQL; drift reports INCONCLUSIVE, clean reports PASS.",
			ManualSteps: []string{
				"Optional: run a bounded load preset first to stress the stack.",
				"Run this scenario to execute reconcile with no Redis mutations.",
			},
			SafeAuto: true,
			Registry: Registry["reconcile-after-stress"],
		},
	}
}

// Opts configures failure-lab execution.
type Opts struct {
	RepoRoot   string
	TargetBase string
	APIKey     string
	RedisAddr  string
	MySQLDSN   string
}

// Run executes selected failure-lab scenarios.
func Run(ctx context.Context, opts Opts, col metrics.Collector, ids []string) []claims.Result {
	if len(ids) == 0 {
		ids = DefaultScenarioIDs
	}
	out := make([]claims.Result, 0, len(ids))
	for _, id := range ids {
		out = append(out, runOne(ctx, opts, col, id))
	}
	return out
}

func runOne(ctx context.Context, opts Opts, col metrics.Collector, id string) claims.Result {
	now := time.Now().UTC().Format(time.RFC3339)
	r := claims.Result{
		ID:            id,
		RegistryTests: Registry[id],
		Evidence:      map[string]interface{}{"ran_at": now},
	}
	switch id {
	case "worker-unavailable":
		return scenarioWorkerUnavailable(ctx, opts, col, r)
	case "restart-inventory":
		return scenarioRestartInventory(ctx, opts, r)
	case "dlq-growth":
		return scenarioDLQGrowth(ctx, opts, col, r)
	case "reconcile-after-stress":
		return scenarioReconcileAfterStress(ctx, opts, r)
	default:
		r.Status = claims.StatusInconclusive
		r.Message = "unknown failure-lab scenario id"
		r.FailureReason = "unknown_scenario"
		return r
	}
}

func scenarioWorkerUnavailable(ctx context.Context, opts Opts, col metrics.Collector, r claims.Result) claims.Result {
	r.Evidence["manual_checklist"] = Catalog()[0].ManualSteps
	before, _ := col.Collect(ctx)
	pendingBefore := before.StreamPending
	r.Evidence["stream_pending_before"] = pendingBefore
	r.Evidence["worker_pending_before"] = before.Worker.Pending
	r.Evidence["field_provenance"] = before.FieldProvenance

	client := claims.Client{BaseURL: opts.TargetBase, APIKey: opts.APIKey}
	uid := 890_000 + int(time.Now().Unix()%10000)
	bookStart := time.Now()
	b, code, err := client.Book(ctx, uid)
	r.Evidence["probe_user_id"] = uid
	r.Evidence["book_http_status"] = code
	r.Evidence["book_ticket_id"] = b.TicketID
	if err != nil {
		r.Status = claims.StatusInconclusive
		r.Message = "booking probe failed (target unreachable?)"
		r.FailureReason = "book_probe_error"
		r.Evidence["book_error"] = err.Error()
		return r
	}
	if code != http.StatusOK || b.TicketID == 0 {
		r.Status = claims.StatusInconclusive
		r.Message = "booking did not succeed; cannot observe stream backlog"
		r.FailureReason = "book_not_ok"
		return r
	}

	time.Sleep(400 * time.Millisecond)
	after, _ := col.Collect(ctx)
	pendingAfter := after.StreamPending
	r.Evidence["stream_pending_after"] = pendingAfter
	r.Evidence["worker_pending_after"] = after.Worker.Pending
	r.Evidence["book_latency_ms"] = time.Since(bookStart).Milliseconds()

	delta := pendingAfter - pendingBefore
	r.Evidence["stream_pending_delta"] = delta

	workerLive := after.FieldProvenance["worker"] == metrics.Live
	if after.FieldProvenance["stream_pending"] != metrics.Live {
		r.Status = claims.StatusInconclusive
		r.Message = "stream pending telemetry unavailable (Redis not wired to verilab?)"
		r.FailureReason = "telemetry_unavailable"
		return r
	}

	if delta > 0 {
		r.Status = claims.StatusPass
		r.Message = "stream pending increased after booking (worker likely stalled)"
		return r
	}

	if workerLive && after.Worker.Pending == 0 && pendingAfter == 0 {
		r.Status = claims.StatusInconclusive
		r.Message = "pending did not grow — worker may still be draining; stop worker and re-run"
		r.FailureReason = "worker_still_active"
		return r
	}

	r.Status = claims.StatusInconclusive
	r.Message = "pending unchanged; inconclusive without stopped worker"
	r.FailureReason = "no_pending_growth"
	return r
}

func scenarioRestartInventory(ctx context.Context, opts Opts, r claims.Result) claims.Result {
	meta := Catalog()[1]
	r.Evidence["manual_checklist"] = meta.ManualSteps

	seed := claims.RunGoTest(ctx, opts.RepoRoot, "./internal/inventory", "TestSeedDecision", nil)
	r.Evidence["unit_seed_decision"] = seed

	if opts.RepoRoot == "" {
		r.Status = claims.StatusInconclusive
		r.Message = "repo root unavailable for restart inventory tests"
		r.FailureReason = "repo_root_missing"
		return r
	}

	if !claims.MySQLReachable(opts.MySQLDSN) {
		if seed.Passed {
			r.Status = claims.StatusInconclusive
			r.Message = "seed decision unit PASS; integration restart test skipped (MySQL unavailable)"
			r.Evidence["integration"] = "skipped_no_mysql"
			return r
		}
		r.Status = claims.StatusFail
		r.Message = "inventory unit tests failed"
		r.FailureReason = "go_test_failed"
		return r
	}

	integ := claims.RunGoTest(ctx, opts.RepoRoot, "./integration", "TestRestartDoesNotMintInventory", []string{"-tags=integration"})
	r.Evidence["integration_go_test"] = integ
	if integ.Passed {
		r.Status = claims.StatusPass
		r.Message = "simulated serve restarts do not mint rows (integration TestRestartDoesNotMintInventory)"
		return r
	}
	if integ.Skipped {
		r.Status = claims.StatusInconclusive
		r.Message = "integration restart test skipped in this environment"
		r.FailureReason = "integration_skipped"
		return r
	}
	r.Status = claims.StatusInconclusive
	r.Message = "integration restart test did not pass; use manual serve restart checklist"
	r.FailureReason = "integration_failed"
	return r
}

func scenarioDLQGrowth(ctx context.Context, opts Opts, col metrics.Collector, r claims.Result) claims.Result {
	snap, _ := col.Collect(ctx)
	r.Evidence["dlq_len"] = snap.Redis.DLQLen
	r.Evidence["dlq_provenance"] = snap.FieldProvenance["redis"]
	r.Evidence["p3_registry"] = Registry["dlq-growth"]

	unit := claims.RunGoTest(ctx, opts.RepoRoot, "./internal/persist", "TestMovesFailedMessageToDLQ", nil)
	r.Evidence["unit_go_test"] = unit

	if snap.FieldProvenance["redis"] != metrics.Live && snap.Redis.DLQLen == 0 {
		r.Status = claims.StatusInconclusive
		r.Message = "DLQ length unavailable (Redis not reachable from verilab)"
		r.FailureReason = "redis_unavailable"
		return r
	}

	if opts.RepoRoot != "" && !unit.Passed && !unit.Skipped {
		r.Status = claims.StatusFail
		r.Message = "DLQ unit test failed"
		r.FailureReason = "go_test_failed"
		return r
	}

	r.Status = claims.StatusPass
	r.Message = fmt.Sprintf("read-only DLQ len=%d; P3 covered by TestMovesFailedMessageToDLQ", snap.Redis.DLQLen)
	if unit.Passed {
		r.Evidence["p3_note"] = "unit DLQ move PASS"
	} else if unit.Skipped {
		r.Evidence["p3_note"] = "unit test skipped (no repo root)"
	}
	return r
}

func scenarioReconcileAfterStress(ctx context.Context, opts Opts, r claims.Result) claims.Result {
	if opts.RepoRoot == "" {
		r.Status = claims.StatusInconclusive
		r.Message = "repo root unavailable for reconcile CLI"
		r.FailureReason = "repo_root_missing"
		return r
	}
	if !claims.ServicesReachable(opts.RedisAddr, opts.MySQLDSN) {
		r.Status = claims.StatusInconclusive
		r.Message = "Redis/MySQL unreachable; cannot run reconcile CLI"
		r.FailureReason = "services_unavailable"
		return r
	}

	out := runReconcileCLI(ctx, opts.RepoRoot)
	r.Evidence["reconcile_cli"] = out
	if out.ExitCode == reconcile.ExitOperational {
		r.Status = claims.StatusFail
		r.Message = "reconcile CLI operational error"
		r.FailureReason = "reconcile_error"
		return r
	}
	if out.ExitCode == reconcile.ExitDrift {
		r.Status = claims.StatusInconclusive
		r.Message = "reconcile reported drift (read-only); inspect output"
		r.FailureReason = "reconcile_drift"
		return r
	}
	if out.ExitCode == reconcile.ExitClean {
		r.Status = claims.StatusPass
		r.Message = "reconcile clean (no drift detected)"
		return r
	}
	r.Status = claims.StatusInconclusive
	r.Message = fmt.Sprintf("reconcile exited with code %d", out.ExitCode)
	r.FailureReason = "reconcile_unexpected_exit"
	return r
}

type reconcileCLIOutcome struct {
	Command  string `json:"command"`
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output_tail"`
}

func runReconcileCLI(ctx context.Context, repoRoot string) reconcileCLIOutcome {
	out := reconcileCLIOutcome{Command: fmt.Sprintf("cd %s && go run . reconcile", repoRoot), ExitCode: -1}
	if ctx.Err() != nil {
		out.Output = ctx.Err().Error()
		return out
	}
	cmd := exec.CommandContext(ctx, "go", "run", ".", "reconcile")
	cmd.Dir = repoRoot
	cmd.Env = os.Environ()
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	text := buf.String()
	out.Output = tailLines(text, 48)
	if err == nil {
		out.ExitCode = reconcile.ExitClean
		return out
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		out.ExitCode = exitErr.ExitCode()
		return out
	}
	out.Output = tailLines(text+"\n"+err.Error(), 48)
	return out
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

// SummarizeFailureRun returns overall run status for the failure-lab preset.
func SummarizeFailureRun(results []claims.Result) (status string, failureReason string) {
	hasFail := false
	hasInconclusive := false
	for _, r := range results {
		switch r.Status {
		case claims.StatusFail:
			hasFail = true
		case claims.StatusInconclusive:
			hasInconclusive = true
		}
	}
	if hasFail {
		return "failed", "failure_lab_failed"
	}
	if hasInconclusive {
		return "completed", "failure_lab_inconclusive"
	}
	return "completed", ""
}
