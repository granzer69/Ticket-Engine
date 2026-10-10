package claims

import (
	"testing"
)

func TestSummarizeGateRun(t *testing.T) {
	passOnly := []Result{{ID: "P1", Status: StatusPass}}
	st, reason := SummarizeGateRun(passOnly)
	if st != "completed" || reason != "" {
		t.Fatalf("pass: %s %s", st, reason)
	}

	mixed := []Result{
		{ID: "P1", Status: StatusPass},
		{ID: "P2", Status: StatusInconclusive},
	}
	st, reason = SummarizeGateRun(mixed)
	if st != "completed" || reason != "gates_inconclusive" {
		t.Fatalf("inconclusive: %s %s", st, reason)
	}

	fail := []Result{{ID: "P5", Status: StatusFail}}
	st, reason = SummarizeGateRun(fail)
	if st != "failed" || reason != "gate_failed" {
		t.Fatalf("fail: %s %s", st, reason)
	}
}

func TestGoTestOutputFailed(t *testing.T) {
	if goTestOutputFailed("ok  \tpackage\n") {
		t.Fatal("expected pass output")
	}
	if !goTestOutputFailed("--- FAIL: TestFoo (0.00s)\nFAIL\n") {
		t.Fatal("expected fail detection")
	}
}

func TestGateRegistryKeys(t *testing.T) {
	for _, id := range DefaultGateIDs {
		if len(GateRegistry[id]) == 0 {
			t.Fatalf("missing registry for %s", id)
		}
	}
}
