package runner

import (
	"math"

	"ticketengine/verification-lab/internal/loadgen"
	"ticketengine/verification-lab/internal/metrics"
)

// BenchmarkSummary is exported under report.extra["benchmark"] for benchmark presets.
type BenchmarkSummary struct {
	Profile                string         `json:"profile"`
	Description            string         `json:"description,omitempty"`
	AssumedInventorySeed   int64          `json:"assumed_inventory_seed"`
	WorkloadMode           string         `json:"workload_mode"`
	RequestsPerSec         float64        `json:"requests_per_sec,omitempty"`
	LatencyMs              map[string]int64 `json:"latency_ms,omitempty"`
	QueueConsumed          *int64         `json:"queue_consumed,omitempty"`
	StreamPendingEnd       *int64         `json:"stream_pending_end,omitempty"`
	MySQLSoldDelta         *int64         `json:"mysql_sold_delta,omitempty"`
	TelemetryOverheadNote  string         `json:"telemetry_overhead_note"`
}

const benchmarkTelemetryNote = "RPS and client latency percentiles come from the lab load generator only; they include HTTP round-trip and exclude snapshot collection (collect_ms on SSE frames)."

func attachBenchmarkSummary(rep *Report, prof *BenchmarkProfile, st loadgen.Stats, startSnap, endSnap *metrics.Snapshot) {
	if rep == nil || !IsBenchmarkPreset(rep.Preset) {
		return
	}
	if rep.Extra == nil {
		rep.Extra = map[string]interface{}{}
	}
	sum := BenchmarkSummary{
		Profile:               rep.Preset,
		WorkloadMode:          workloadModeFromExtra(rep.Extra),
		TelemetryOverheadNote: benchmarkTelemetryNote,
	}
	if prof != nil {
		sum.Description = prof.Description
		sum.AssumedInventorySeed = prof.AssumedInventorySeed
		sum.WorkloadMode = prof.Mode
	} else if v, ok := rep.Extra["assumed_inventory_seed"]; ok {
		switch n := v.(type) {
		case int64:
			sum.AssumedInventorySeed = n
		case int:
			sum.AssumedInventorySeed = int64(n)
		case float64:
			sum.AssumedInventorySeed = int64(n)
		}
	}
	if st.DurationMs > 0 {
		sum.RequestsPerSec = round2(float64(st.LogicalRequests) / (float64(st.DurationMs) / 1000))
	}
	lat := map[string]int64{}
	if st.LatencyP50Ms > 0 {
		lat["p50"] = st.LatencyP50Ms
	}
	if st.LatencyP95Ms > 0 {
		lat["p95"] = st.LatencyP95Ms
	}
	if st.LatencyP99Ms > 0 {
		lat["p99"] = st.LatencyP99Ms
	}
	if len(lat) > 0 {
		sum.LatencyMs = lat
	}
	if consumed := queueConsumedFromSnapshots(startSnap, endSnap); consumed != nil {
		sum.QueueConsumed = consumed
	}
	if endSnap != nil && endSnap.FieldProvenance["stream_pending"] == metrics.Live {
		v := endSnap.StreamPending
		sum.StreamPendingEnd = &v
	}
	if startSnap != nil && endSnap != nil &&
		startSnap.FieldProvenance["mysql"] == metrics.Live &&
		endSnap.FieldProvenance["mysql"] == metrics.Live {
		delta := endSnap.MySQL.Sold - startSnap.MySQL.Sold
		sum.MySQLSoldDelta = &delta
	}
	rep.Extra["benchmark"] = sum
	if sum.AssumedInventorySeed > 0 {
		rep.Extra["assumed_inventory_seed"] = sum.AssumedInventorySeed
		rep.Extra["inventory_assumption_note"] = "Pass/fail for exhaustion profiles uses LIVE queue depth when available; assumed_inventory_seed documents the standard 15k lab seed, not live inventory."
	}
}

func workloadModeFromExtra(extra map[string]interface{}) string {
	if extra == nil {
		return ""
	}
	if v, ok := extra["workload_mode"].(string); ok {
		return v
	}
	return ""
}

func round2(f float64) float64 {
	return math.Round(f*100) / 100
}
