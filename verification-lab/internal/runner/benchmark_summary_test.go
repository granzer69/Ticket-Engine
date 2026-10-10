package runner

import (
	"testing"

	"ticketengine/verification-lab/internal/loadgen"
	"ticketengine/verification-lab/internal/metrics"
)

func TestAttachBenchmarkSummary_shapesExport(t *testing.T) {
	start := &metrics.Snapshot{
		QueueRemaining:  15_000,
		StreamPending:   2,
		MySQL:           metrics.MySQLCounts{Sold: 100},
		FieldProvenance: map[string]string{
			"queue_remaining": metrics.Live,
			"stream_pending":  metrics.Live,
			"mysql":           metrics.Live,
		},
	}
	end := &metrics.Snapshot{
		QueueRemaining:  14_900,
		StreamPending:   0,
		MySQL:           metrics.MySQLCounts{Sold: 200},
		FieldProvenance: map[string]string{
			"queue_remaining": metrics.Live,
			"stream_pending":  metrics.Live,
			"mysql":           metrics.Live,
		},
	}
	prof := benchmarkProfiles()["benchmark-smoke"]
	rep := &Report{
		Preset: "benchmark-smoke",
		Extra:  map[string]interface{}{"workload_mode": WorkloadAllSuccess},
	}
	st := loadgen.Stats{
		LogicalRequests: 100,
		HTTP200:         100,
		DurationMs:      1000,
		LatencyP50Ms:    5,
		LatencyP95Ms:    12,
		LatencyP99Ms:    20,
	}
	attachBenchmarkSummary(rep, &prof, st, start, end)

	bm, ok := rep.Extra["benchmark"].(BenchmarkSummary)
	if !ok {
		t.Fatalf("benchmark extra type %T", rep.Extra["benchmark"])
	}
	if bm.RequestsPerSec != 100 {
		t.Fatalf("rps: %v", bm.RequestsPerSec)
	}
	if bm.LatencyMs["p99"] != 20 {
		t.Fatalf("p99: %v", bm.LatencyMs)
	}
	if bm.QueueConsumed == nil || *bm.QueueConsumed != 100 {
		t.Fatalf("queue_consumed: %v", bm.QueueConsumed)
	}
	if bm.StreamPendingEnd == nil || *bm.StreamPendingEnd != 0 {
		t.Fatalf("stream_pending_end: %v", bm.StreamPendingEnd)
	}
	if bm.MySQLSoldDelta == nil || *bm.MySQLSoldDelta != 100 {
		t.Fatalf("mysql_sold_delta: %v", bm.MySQLSoldDelta)
	}
	if bm.AssumedInventorySeed != AssumedInventorySeed {
		t.Fatalf("seed: %d", bm.AssumedInventorySeed)
	}
	if bm.TelemetryOverheadNote == "" {
		t.Fatal("expected telemetry note")
	}
}
