package runner

import (
	"context"
	"time"
)

// Standard V2 lab seed inventory (see docs/BENCHMARKS.md); used for report assumptions only.
const AssumedInventorySeed = 15_000

// BenchmarkProfile is a named verification-lab load profile.
type BenchmarkProfile struct {
	Preset               string
	Total                int
	Workers              int
	Mode                 string
	AssumedInventorySeed int64
	Timeout              time.Duration
	Description          string
}

func benchmarkProfiles() map[string]BenchmarkProfile {
	return map[string]BenchmarkProfile{
		"benchmark-smoke": {
			Preset:               "benchmark-smoke",
			Total:                100,
			Workers:              8,
			Mode:                 WorkloadAllSuccess,
			AssumedInventorySeed: AssumedInventorySeed,
			Description:          "100 logical bookings; expects all HTTP 200 on a healthy target.",
		},
		"benchmark-exhaustion-15k": {
			Preset:               "benchmark-exhaustion-15k",
			Total:                100_000,
			Workers:              32,
			Mode:                 WorkloadExhaustion,
			AssumedInventorySeed: AssumedInventorySeed,
			Timeout:              15 * time.Second,
			Description:          "100k logical attempts against a 15k-ticket seed; pass uses queue/inventory rules (not http_200 count).",
		},
		"benchmark-exhaustion-mini": {
			Preset:               "benchmark-exhaustion-mini",
			Total:                5_000,
			Workers:              16,
			Mode:                 WorkloadExhaustion,
			AssumedInventorySeed: AssumedInventorySeed,
			Timeout:              15 * time.Second,
			Description:          "5k logical attempts for quick exhaustion sanity on a 15k seed.",
		},
		"benchmark-saturated-15k": {
			Preset:               "benchmark-saturated-15k",
			Total:                15_000,
			Workers:              24,
			Mode:                 WorkloadAllSuccess,
			AssumedInventorySeed: AssumedInventorySeed,
			Timeout:              20 * time.Second,
			Description:          "Exactly 15k distinct users; expects all HTTP 200 when inventory is full at start.",
		},
	}
}

func IsBenchmarkPreset(preset string) bool {
	_, ok := benchmarkProfiles()[preset]
	return ok
}

// StartBenchmark runs a named benchmark profile.
func (e *Engine) StartBenchmark(ctx context.Context, preset string) (string, error) {
	prof, ok := benchmarkProfiles()[preset]
	if !ok {
		return "", errUnknownBenchmarkPreset(preset)
	}
	workers := prof.Workers
	if workers <= 0 {
		workers = boundedWorkers(prof.Total)
	}
	timeout := prof.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	userBase := freshUserIDBase(prof.Total)
	if prof.Mode == WorkloadAllSuccess && preset == "benchmark-smoke" {
		userBase = 700_000
	}
	return e.startLoad(ctx, loadPresetSpec{
		Preset:      prof.Preset,
		Total:       prof.Total,
		UserIDStart: userBase,
		Workers:     workers,
		Timeout:     timeout,
	}, prof.Mode, &prof)
}

type unknownBenchmarkPreset string

func (e unknownBenchmarkPreset) Error() string {
	return "unknown benchmark preset: " + string(e)
}

func errUnknownBenchmarkPreset(preset string) error {
	return unknownBenchmarkPreset(preset)
}
