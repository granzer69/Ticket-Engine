package claims

import (
	"context"
)

// DefaultSecurityClaimIDs are Slice 7 lab probes (automated tests only).
var DefaultSecurityClaimIDs = []string{
	"S7-CORS-1",
	"S7-REDIS-1",
	"S7-LOG-1",
	"S7-PROD-1",
}

type SecurityOpts struct {
	RepoRoot string
}

// RunSecurityClaims runs go test probes for documented S7 items.
func RunSecurityClaims(ctx context.Context, opts SecurityOpts) []Result {
	root := opts.RepoRoot
	if root == "" {
		root = DiscoverRepoRoot()
	}
	out := make([]Result, 0, len(DefaultSecurityClaimIDs))
	for _, id := range DefaultSecurityClaimIDs {
		out = append(out, runSecurityOne(ctx, root, id))
	}
	return out
}

func runSecurityOne(ctx context.Context, repoRoot, id string) Result {
	r := Result{ID: id, Evidence: map[string]interface{}{}}
	switch id {
	case "S7-CORS-1":
		r.RegistryTests = []string{"main:TestCORS_*"}
		got := RunGoTest(ctx, repoRoot, ".", "TestCORS_", nil, nil)
		r.Evidence["go_test"] = got
		return finalizeGoTestResult(r, got, "CORS tests", repoRoot)
	case "S7-REDIS-1":
		r.RegistryTests = []string{"integration:TestRedisAuth"}
		got := RunGoTest(ctx, repoRoot, "./integration", "TestRedisAuth", []string{"-tags=integration"}, IntegrationEnvFromOS())
		r.Evidence["go_test"] = got
		return finalizeGoTestResult(r, got, "Redis AUTH client wiring", repoRoot)
	case "S7-LOG-1":
		r.RegistryTests = []string{"internal/audit:TestLogRedaction_*"}
		got := RunGoTest(ctx, repoRoot, "./internal/audit", "TestLogRedaction_", nil, nil)
		r.Evidence["go_test"] = got
		return finalizeGoTestResult(r, got, "log redaction", repoRoot)
	case "S7-PROD-1":
		r.RegistryTests = []string{"internal/prodconfig:TestValidate_*"}
		got := RunGoTest(ctx, repoRoot, "./internal/prodconfig", "TestValidate_", nil, nil)
		r.Evidence["go_test"] = got
		return finalizeGoTestResult(r, got, "production profile validation", repoRoot)
	default:
		r.Status = StatusInconclusive
		r.Message = "unknown security claim"
		r.FailureReason = "unknown_claim"
		return r
	}
}

func finalizeGoTestResult(r Result, got GoTestOutcome, label string, repoRoot string) Result {
	if repoRoot == "" {
		r.Status = StatusInconclusive
		r.Message = "repo root unavailable for " + label
		r.FailureReason = "repo_root_missing"
		return r
	}
	if got.Passed {
		r.Status = StatusPass
		r.Message = label + " PASS"
		return r
	}
	if got.Skipped {
		r.Status = StatusInconclusive
		r.Message = label + " skipped in this environment"
		r.FailureReason = "go_test_skipped"
		return r
	}
	r.Status = StatusFail
	r.Message = label + " failed"
	r.FailureReason = "go_test_failed"
	return r
}
