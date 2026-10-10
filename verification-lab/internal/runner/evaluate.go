package runner

import (
	"fmt"

	"ticketengine/verification-lab/internal/loadgen"
	"ticketengine/verification-lab/internal/metrics"
)

// LoadWorkloadMode describes how HTTP outcomes are interpreted for pass/fail.
const (
	WorkloadAllSuccess  = "all_success"
	WorkloadExhaustion  = "exhaustion"
)

// inventoryAtStartFromSnapshot returns queue depth when LIVE, else nil.
func inventoryAtStartFromSnapshot(snap *metrics.Snapshot) *int64 {
	if snap == nil {
		return nil
	}
	if snap.FieldProvenance["queue_remaining"] != metrics.Live {
		return nil
	}
	v := snap.QueueRemaining
	return &v
}

// queueConsumedFromSnapshots is new tickets drawn from the Redis queue (HTTP 200 includes idempotent replays).
func queueConsumedFromSnapshots(start, end *metrics.Snapshot) *int64 {
	if start == nil || end == nil {
		return nil
	}
	if start.FieldProvenance["queue_remaining"] != metrics.Live || end.FieldProvenance["queue_remaining"] != metrics.Live {
		return nil
	}
	consumed := start.QueueRemaining - end.QueueRemaining
	if consumed < 0 {
		consumed = 0
	}
	return &consumed
}

// EvaluateLoadPass decides whether a bounded load run met its workload intent.
func EvaluateLoadPass(st loadgen.Stats, mode string, inventoryAtStart *int64, startSnap, endSnap *metrics.Snapshot) (ok bool, failureReason string) {
	transportAndOther := st.Errors + st.OtherStatus
	definitive := st.HTTP200 + st.HTTP404

	if transportAndOther != 0 {
		return false, fmt.Sprintf("transport_or_unexpected_status: errors=%d other_status=%d", st.Errors, st.OtherStatus)
	}
	if definitive != st.LogicalRequests {
		return false, fmt.Sprintf("incomplete_definitive_responses: http_200=%d http_404=%d logical_requests=%d",
			st.HTTP200, st.HTTP404, st.LogicalRequests)
	}

	switch mode {
	case WorkloadAllSuccess:
		if st.HTTP200 != st.LogicalRequests {
			return false, fmt.Sprintf("expected_all_http_200: got %d of %d", st.HTTP200, st.LogicalRequests)
		}
		return true, ""
	case WorkloadExhaustion:
		queueConsumed := queueConsumedFromSnapshots(startSnap, endSnap)
		if inventoryAtStart != nil && queueConsumed != nil {
			if *queueConsumed > *inventoryAtStart {
				return false, fmt.Sprintf("queue_consumed_exceeds_inventory: consumed=%d inventory_at_start=%d",
					*queueConsumed, *inventoryAtStart)
			}
			return true, ""
		}
		// Without LIVE queue snapshots, do not infer from http_200 (replays also return 200).
		return true, ""
	default:
		return false, fmt.Sprintf("unknown_workload_mode: %s", mode)
	}
}
