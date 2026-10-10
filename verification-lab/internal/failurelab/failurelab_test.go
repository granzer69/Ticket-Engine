package failurelab

import (
	"testing"

	"ticketengine/verification-lab/internal/claims"
)

func TestCatalogCoversDefaults(t *testing.T) {
	cat := Catalog()
	ids := map[string]bool{}
	for _, m := range cat {
		ids[m.ID] = true
	}
	for _, id := range DefaultScenarioIDs {
		if !ids[id] {
			t.Fatalf("catalog missing %s", id)
		}
	}
}

func TestSummarizeFailureRun(t *testing.T) {
	st, reason := SummarizeFailureRun([]claims.Result{
		{ID: "a", Status: claims.StatusPass},
	})
	if st != "completed" || reason != "" {
		t.Fatalf("pass: %s %s", st, reason)
	}
	st, reason = SummarizeFailureRun([]claims.Result{
		{ID: "a", Status: claims.StatusFail},
	})
	if st != "failed" || reason != "failure_lab_failed" {
		t.Fatalf("fail: %s %s", st, reason)
	}
	st, reason = SummarizeFailureRun([]claims.Result{
		{ID: "a", Status: claims.StatusInconclusive},
	})
	if st != "completed" || reason != "failure_lab_inconclusive" {
		t.Fatalf("inconclusive: %s %s", st, reason)
	}
}
