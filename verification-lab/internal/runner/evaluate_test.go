package runner

import (
	"strings"
	"testing"
	"time"

	"ticketengine/verification-lab/internal/loadgen"
	"ticketengine/verification-lab/internal/metrics"
)

func TestEvaluateLoadPass_exhaustion(t *testing.T) {
	cases := []struct {
		name      string
		st        loadgen.Stats
		inventory *int64
		wantOK    bool
		wantSub   string
	}{
		{
			name: "100k attempts 15k ok 85k sold out",
			st: loadgen.Stats{LogicalRequests: 100_000, HTTP200: 15_000, HTTP404: 85_000},
			inventory: int64Ptr(15_000),
			wantOK:    true,
		},
		{
			name: "all 200 when under inventory",
			st:   loadgen.Stats{LogicalRequests: 1_000, HTTP200: 1_000},
			inventory: int64Ptr(15_000),
			wantOK:    true,
		},
		{
			name: "transport error fails",
			st:   loadgen.Stats{LogicalRequests: 100, HTTP200: 99, Errors: 1},
			wantOK:  false,
			wantSub: "transport_or_unexpected_status",
		},
		{
			name: "unexpected status fails",
			st:   loadgen.Stats{LogicalRequests: 100, HTTP200: 99, OtherStatus: 1},
			wantOK:  false,
			wantSub: "transport_or_unexpected_status",
		},
		{
			name: "missing responses fails",
			st:   loadgen.Stats{LogicalRequests: 100, HTTP200: 50, HTTP404: 40},
			wantOK:  false,
			wantSub: "incomplete_definitive_responses",
		},
		{
			name: "queue consumed exceeds inventory",
			st:   loadgen.Stats{LogicalRequests: 100, HTTP200: 20, HTTP404: 80},
			inventory: int64Ptr(15),
			wantOK:  false,
			wantSub: "queue_consumed_exceeds_inventory",
		},
		{
			name: "inventory unknown still passes on definitive split",
			st:   loadgen.Stats{LogicalRequests: 50_000, HTTP200: 15_000, HTTP404: 35_000},
			wantOK: true,
		},
		{
			name: "depleted queue idempotent 200s do not fail",
			st:   loadgen.Stats{LogicalRequests: 100_000, HTTP200: 14_760, HTTP404: 85_240},
			inventory: int64Ptr(0),
			wantOK: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var start, end *metrics.Snapshot
			switch tc.name {
			case "queue consumed exceeds inventory":
				start = &metrics.Snapshot{
					QueueRemaining:  20,
					FieldProvenance: map[string]string{"queue_remaining": metrics.Live},
				}
				end = &metrics.Snapshot{
					QueueRemaining:  0,
					FieldProvenance: map[string]string{"queue_remaining": metrics.Live},
				}
			case "depleted queue idempotent 200s do not fail":
				start = &metrics.Snapshot{
					QueueRemaining:  0,
					FieldProvenance: map[string]string{"queue_remaining": metrics.Live},
				}
				end = &metrics.Snapshot{
					QueueRemaining:  0,
					FieldProvenance: map[string]string{"queue_remaining": metrics.Live},
				}
			}
			ok, reason := EvaluateLoadPass(tc.st, WorkloadExhaustion, tc.inventory, start, end)
			if ok != tc.wantOK {
				t.Fatalf("ok=%v want %v reason=%q", ok, tc.wantOK, reason)
			}
			if !tc.wantOK && tc.wantSub != "" && reason != "" && !strings.Contains(reason, tc.wantSub) {
				t.Fatalf("reason %q want substring %q", reason, tc.wantSub)
			}
		})
	}
}

func TestEvaluateLoadPass_allSuccess(t *testing.T) {
	ok, _ := EvaluateLoadPass(loadgen.Stats{LogicalRequests: 100, HTTP200: 100}, WorkloadAllSuccess, nil, nil, nil)
	if !ok {
		t.Fatal("expected pass")
	}
	ok, reason := EvaluateLoadPass(loadgen.Stats{LogicalRequests: 100, HTTP200: 99, HTTP404: 1}, WorkloadAllSuccess, nil, nil, nil)
	if ok {
		t.Fatal("expected fail")
	}
	if !strings.Contains(reason, "expected_all_http_200") {
		t.Fatalf("reason: %s", reason)
	}
}

func TestInventoryAtStartFromSnapshot(t *testing.T) {
	snap := &metrics.Snapshot{
		QueueRemaining:  15_000,
		FieldProvenance: map[string]string{"queue_remaining": metrics.Live},
		At:              time.Now(),
	}
	v := inventoryAtStartFromSnapshot(snap)
	if v == nil || *v != 15_000 {
		t.Fatalf("got %v", v)
	}
	snap.FieldProvenance["queue_remaining"] = metrics.Unavailable
	if inventoryAtStartFromSnapshot(snap) != nil {
		t.Fatal("expected nil when not LIVE")
	}
}

func int64Ptr(n int64) *int64 { return &n }
