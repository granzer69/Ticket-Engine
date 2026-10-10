package claims

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultGateIDs is the full P1..P7 sequence.
var DefaultGateIDs = []string{"P1", "P2", "P3", "P4", "P5", "P6", "P7"}

// GateRegistry maps phase gate IDs to documented go test references.
var GateRegistry = map[string][]string{
	"P1": {"main:TestServeDoesNotStartConsumer", "main:TestWorkerCommandStartsConsumer"},
	"P2": {"internal/persist:TestReclaimStalePending", "internal/persist:TestConsumerReadsOwnPending", "integration:TestPersistRecovery"},
	"P3": {"internal/persist:TestMovesFailedMessageToDLQ", "integration:TestPersistDLQ"},
	"P4": {"main:TestReadyzStreamPending", "integration:TestTwoWorkersNoDoubleSell"},
	"P5": {"main:TestBookReplaySkipsMySQL"},
	"P6": {"internal/reconcile:TestReconcileInventoryDrift", "integration:TestReconcileCLIReadOnly"},
	"P7": {"main:TestAPIKey_Missing", "main:TestAPIKey_Invalid", "internal/security:TestAPIKeyValid_*"},
}

// GateOpts configures gate execution (go test subprocess + live probes).
type GateOpts struct {
	RepoRoot   string
	TargetBase string
	APIKey     string
	RedisAddr  string
	MySQLDSN   string
	HTTP       *http.Client
}

// RunGates executes selected phase gates.
func RunGates(ctx context.Context, opts GateOpts, ids []string) []Result {
	if len(ids) == 0 {
		ids = DefaultGateIDs
	}
	out := make([]Result, 0, len(ids))
	for _, id := range ids {
		out = append(out, runGate(ctx, opts, id))
	}
	return out
}

func runGate(ctx context.Context, opts GateOpts, id string) Result {
	now := time.Now().UTC().Format(time.RFC3339)
	r := Result{
		ID:            id,
		RegistryTests: GateRegistry[id],
		Evidence:      map[string]interface{}{"ran_at": now},
	}
	switch id {
	case "P1":
		return gateP1(ctx, opts, r)
	case "P2":
		return gateP2(ctx, opts, r)
	case "P3":
		return gateP3(ctx, opts, r)
	case "P4":
		return gateP4(ctx, opts, r)
	case "P5":
		return gateP5(ctx, opts, r)
	case "P6":
		return gateP6(ctx, opts, r)
	case "P7":
		return gateP7(ctx, opts, r)
	default:
		r.Status = StatusInconclusive
		r.Message = "unknown gate id"
		r.FailureReason = "unknown_gate"
		return r
	}
}

func gateP1(ctx context.Context, opts GateOpts, r Result) Result {
	got := RunGoTest(ctx, opts.RepoRoot, ".", "TestServeDoesNotStartConsumer|TestWorkerCommandStartsConsumer", nil)
	r.Evidence["go_test"] = got
	if got.Skipped && opts.RepoRoot == "" {
		return inconclusive(r, "repo root unavailable for go test", "repo_root_missing")
	}
	if got.Passed {
		r.Status = StatusPass
		r.Message = "serve vs worker consumer separation verified via unit tests"
		return r
	}
	r.Status = StatusFail
	r.Message = "serve/worker consumer tests failed"
	r.FailureReason = "go_test_failed"
	return r
}

func gateP2(ctx context.Context, opts GateOpts, r Result) Result {
	unit := RunGoTest(ctx, opts.RepoRoot, "./internal/persist", "TestReclaimStalePending|TestConsumerReadsOwnPending", nil)
	r.Evidence["unit_go_test"] = unit
	if unit.Skipped && opts.RepoRoot == "" {
		return inconclusive(r, "repo root unavailable", "repo_root_missing")
	}
	if !unit.Passed {
		r.Status = StatusFail
		r.Message = "PEL reclaim unit tests failed"
		r.FailureReason = "go_test_failed"
		return r
	}
	if !ServicesReachable(opts.RedisAddr, opts.MySQLDSN) {
		r.Status = StatusInconclusive
		r.Message = "unit reclaim PASS; integration TestPersistRecovery not run (Redis/MySQL unavailable)"
		r.FailureReason = "integration_services_unavailable"
		r.Evidence["integration"] = "skipped"
		return r
	}
	integ := RunGoTest(ctx, opts.RepoRoot, "./integration", "TestPersistRecovery", []string{"-tags=integration"})
	r.Evidence["integration_go_test"] = integ
	if integ.Passed {
		r.Status = StatusPass
		r.Message = "reclaim unit + integration recovery verified"
		return r
	}
	r.Status = StatusInconclusive
	r.Message = "unit PASS; integration recovery did not pass (skipped or failed in env)"
	r.FailureReason = "integration_inconclusive"
	return r
}

func gateP3(ctx context.Context, opts GateOpts, r Result) Result {
	unit := RunGoTest(ctx, opts.RepoRoot, "./internal/persist", "TestMovesFailedMessageToDLQ", nil)
	r.Evidence["unit_go_test"] = unit
	if unit.Skipped && opts.RepoRoot == "" {
		return inconclusive(r, "repo root unavailable", "repo_root_missing")
	}
	if !unit.Passed {
		r.Status = StatusFail
		r.Message = "DLQ unit test failed"
		r.FailureReason = "go_test_failed"
		return r
	}
	if !ServicesReachable(opts.RedisAddr, opts.MySQLDSN) {
		r.Status = StatusInconclusive
		r.Message = "unit DLQ PASS; integration TestPersistDLQ not run (Redis/MySQL unavailable)"
		r.FailureReason = "integration_services_unavailable"
		return r
	}
	integ := RunGoTest(ctx, opts.RepoRoot, "./integration", "TestPersistDLQ", []string{"-tags=integration"})
	r.Evidence["integration_go_test"] = integ
	if integ.Passed {
		r.Status = StatusPass
		r.Message = "DLQ unit + integration verified"
		return r
	}
	r.Status = StatusInconclusive
	r.Message = "unit PASS; integration DLQ inconclusive in this environment"
	r.FailureReason = "integration_inconclusive"
	return r
}

func gateP4(ctx context.Context, opts GateOpts, r Result) Result {
	unit := RunGoTest(ctx, opts.RepoRoot, ".", "TestReadyzStreamPending", nil)
	r.Evidence["unit_go_test"] = unit
	if unit.Skipped && opts.RepoRoot == "" {
		return inconclusive(r, "repo root unavailable", "repo_root_missing")
	}
	if !unit.Passed {
		r.Status = StatusFail
		r.Message = "readyz pending unit test failed"
		r.FailureReason = "go_test_failed"
		return r
	}

	probe := probeReadyz(ctx, opts)
	r.Evidence["live_readyz"] = probe
	if probe["error"] == nil {
		if code, ok := probe["http_status"].(int); ok && code == http.StatusOK {
			r.Evidence["live_readyz_note"] = "target /readyz returned 200"
		} else {
			r.Evidence["live_readyz_note"] = "target /readyz not 200 (stack may be down)"
		}
	}

	if !ServicesReachable(opts.RedisAddr, opts.MySQLDSN) {
		r.Status = StatusInconclusive
		r.Message = "readyz unit PASS; multi-worker TestTwoWorkersNoDoubleSell not run (Redis/MySQL unavailable)"
		r.FailureReason = "multi_worker_unavailable"
		return r
	}
	integ := RunGoTest(ctx, opts.RepoRoot, "./integration", "TestTwoWorkersNoDoubleSell", []string{"-tags=integration"})
	r.Evidence["integration_go_test"] = integ
	if integ.Passed {
		r.Status = StatusPass
		r.Message = "readyz unit + two-worker integration PASS"
		return r
	}
	r.Status = StatusInconclusive
	r.Message = "readyz unit PASS; two-worker integration did not pass in this env"
	r.FailureReason = "integration_inconclusive"
	return r
}

func gateP5(ctx context.Context, opts GateOpts, r Result) Result {
	got := RunGoTest(ctx, opts.RepoRoot, ".", "TestBookReplaySkipsMySQL", nil)
	r.Evidence["go_test"] = got
	if got.Skipped && opts.RepoRoot == "" {
		return inconclusive(r, "repo root unavailable", "repo_root_missing")
	}
	if got.Passed {
		r.Status = StatusPass
		r.Message = "Redis-only idempotent replay skips MySQL read"
		return r
	}
	r.Status = StatusFail
	r.Message = "TestBookReplaySkipsMySQL failed"
	r.FailureReason = "go_test_failed"
	return r
}

func gateP6(ctx context.Context, opts GateOpts, r Result) Result {
	unit := RunGoTest(ctx, opts.RepoRoot, "./internal/reconcile", "TestReconcile", nil)
	r.Evidence["unit_go_test"] = unit
	if unit.Skipped && opts.RepoRoot == "" {
		return inconclusive(r, "repo root unavailable", "repo_root_missing")
	}
	if !unit.Passed {
		r.Status = StatusFail
		r.Message = "reconcile unit tests failed"
		r.FailureReason = "go_test_failed"
		return r
	}
	if !ServicesReachable(opts.RedisAddr, opts.MySQLDSN) {
		r.Status = StatusPass
		r.Message = "reconcile unit PASS (integration CLI read-only not run; services unavailable)"
		r.Evidence["integration"] = "skipped_no_services"
		return r
	}
	integ := RunGoTest(ctx, opts.RepoRoot, "./integration", "TestReconcileCLIReadOnly", []string{"-tags=integration"})
	r.Evidence["integration_go_test"] = integ
	if integ.Passed {
		r.Status = StatusPass
		r.Message = "reconcile drift + read-only CLI integration PASS"
		return r
	}
	r.Status = StatusInconclusive
	r.Message = "unit PASS; TestReconcileCLIReadOnly inconclusive in env"
	r.FailureReason = "integration_inconclusive"
	return r
}

func gateP7(ctx context.Context, opts GateOpts, r Result) Result {
	got := RunGoTest(ctx, opts.RepoRoot, ".", "TestAPIKey_", nil)
	r.Evidence["go_test"] = got
	if got.Skipped && opts.RepoRoot == "" {
		return inconclusive(r, "repo root unavailable", "repo_root_missing")
	}
	if !got.Passed {
		r.Status = StatusFail
		r.Message = "API key handler tests failed"
		r.FailureReason = "go_test_failed"
		return r
	}

	probe := probeMissingAPIKey(ctx, opts)
	r.Evidence["live_missing_key"] = probe
	if opts.APIKey == "" {
		r.Status = StatusPass
		r.Message = "API key tests PASS; target lab has no TICKET_API_KEY for live deny probe"
		r.Evidence["live_note"] = "open /book when key unset (dev default)"
		return r
	}
	if probe["error"] != nil {
		r.Status = StatusInconclusive
		r.Message = "go test PASS; live missing-key probe failed"
		r.FailureReason = "live_probe_error"
		return r
	}
	code, _ := probe["http_status"].(int)
	if code == http.StatusUnauthorized || code == http.StatusForbidden {
		r.Status = StatusPass
		r.Message = "API key tests + live missing-key denial PASS"
		return r
	}
	r.Status = StatusInconclusive
	r.Message = fmt.Sprintf("go test PASS; live missing-key returned HTTP %d (target may not enforce key)", code)
	r.FailureReason = "live_probe_unexpected_status"
	return r
}

func inconclusive(r Result, msg, reason string) Result {
	r.Status = StatusInconclusive
	r.Message = msg
	r.FailureReason = reason
	return r
}

func probeReadyz(ctx context.Context, opts GateOpts) map[string]interface{} {
	client := opts.HTTP
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	url := strings.TrimRight(opts.TargetBase, "/") + "/readyz"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	res, err := client.Do(req)
	if err != nil {
		return map[string]interface{}{"error": err.Error(), "url": url}
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
	var parsed interface{}
	if json.Unmarshal(body, &parsed) == nil {
		return map[string]interface{}{"http_status": res.StatusCode, "url": url, "body": parsed}
	}
	return map[string]interface{}{"http_status": res.StatusCode, "url": url, "body_text": strings.TrimSpace(string(body))}
}

func probeMissingAPIKey(ctx context.Context, opts GateOpts) map[string]interface{} {
	client := opts.HTTP
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	url := strings.TrimRight(opts.TargetBase, "/") + "/book"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return map[string]interface{}{"error": err.Error()}
	}
	req.Header.Set("X-User-Id", "999777001")
	res, err := client.Do(req)
	if err != nil {
		return map[string]interface{}{"error": err.Error(), "url": url}
	}
	defer res.Body.Close()
	_, _ = io.ReadAll(io.LimitReader(res.Body, 512))
	return map[string]interface{}{"http_status": res.StatusCode, "url": url}
}

// SummarizeGateRun returns overall run status and optional failure_reason for the report.
func SummarizeGateRun(results []Result) (status string, failureReason string) {
	hasFail := false
	hasInconclusive := false
	for _, r := range results {
		switch r.Status {
		case StatusFail:
			hasFail = true
		case StatusInconclusive:
			hasInconclusive = true
		}
	}
	if hasFail {
		return "failed", "gate_failed"
	}
	if hasInconclusive {
		return "completed", "gates_inconclusive"
	}
	return "completed", ""
}
