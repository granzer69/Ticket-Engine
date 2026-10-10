package runner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"

	"ticketengine/verification-lab/internal/claims"
	"ticketengine/verification-lab/internal/loadgen"
	"ticketengine/verification-lab/internal/metrics"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Config struct {
	TargetBase string
	APIKey     string
	RedisAddr  string
	MySQLDSN   string
	RepoRoot   string
}

type Report struct {
	RunID            string                 `json:"run_id"`
	Preset           string                 `json:"preset"`
	Status           string                 `json:"status"`
	TargetBase       string                 `json:"target_base"`
	LogicalRequests  int                    `json:"logical_requests,omitempty"`
	HTTP200          int                    `json:"http_200,omitempty"`
	HTTP404          int                    `json:"http_404,omitempty"`
	InventoryAtStart *int64                 `json:"inventory_at_start,omitempty"`
	Errors           int                    `json:"errors,omitempty"`
	DurationMs       int64                  `json:"duration_ms,omitempty"`
	LatencyP95Ms     int64                  `json:"latency_p95_ms,omitempty"`
	LatencyP99Ms     int64                  `json:"latency_p99_ms,omitempty"`
	Claims           []claims.Result        `json:"claims,omitempty"`
	SnapshotStart    *metrics.Snapshot      `json:"snapshot_start,omitempty"`
	SnapshotEnd      *metrics.Snapshot      `json:"snapshot_end,omitempty"`
	FieldProvenance  map[string]string      `json:"field_provenance,omitempty"`
	Extra            map[string]interface{} `json:"extra,omitempty"`
}

type Engine struct {
	cfg   Config
	mu    sync.RWMutex
	runs  map[string]*Report
	col   metrics.Collector
	Hub   *metrics.Hub

	activeRunID   string
	activeCorrID  string
}

// ActiveRun implements metrics.RunContextProvider.
func (e *Engine) ActiveRun() (runID, correlationID string) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.activeRunID, e.activeCorrID
}

func (e *Engine) setActiveRun(runID, corr string) {
	e.mu.Lock()
	e.activeRunID = runID
	e.activeCorrID = corr
	e.mu.Unlock()
}

func (e *Engine) clearActiveRun() {
	e.mu.Lock()
	e.activeRunID = ""
	e.activeCorrID = ""
	e.mu.Unlock()
}

func New(cfg Config) *Engine {
	return &Engine{
		cfg: cfg,
		runs: map[string]*Report{},
		col: metrics.Collector{
			TargetBase: cfg.TargetBase,
			RedisAddr:  cfg.RedisAddr,
			MySQLDSN:   cfg.MySQLDSN,
		},
	}
}

func (e *Engine) Get(id string) (*Report, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	r, ok := e.runs[id]
	if !ok {
		return nil, false
	}
	cp := *r
	return &cp, true
}

func (e *Engine) StartSmoke(ctx context.Context) (string, error) {
	id := newRunID()
	rep := &Report{
		RunID:      id,
		Preset:     "smoke",
		Status:     "running",
		TargetBase: e.cfg.TargetBase,
	}
	e.mu.Lock()
	e.runs[id] = rep
	e.mu.Unlock()

	go func() {
		defer e.clearActiveRun()
		e.setActiveRun(id, id)
		if e.Hub != nil {
			e.Hub.EmitRunEvent("run", "smoke started", id, id)
		}
		startSnap, _ := e.col.Collect(ctx)
		st, err := loadgen.Run(ctx, loadgen.Options{
			BaseURL:     e.cfg.TargetBase,
			Workers:     8,
			Total:       100,
			UserIDStart: 700_000,
			APIKey:      e.cfg.APIKey,
		})
		endSnap, _ := e.col.Collect(ctx)
		e.mu.Lock()
		defer e.mu.Unlock()
		rep.SnapshotEnd = &endSnap
		rep.FieldProvenance = map[string]string{
			"load_stats": metrics.Derived,
		}
		e.finalizeLoadReport(rep, startSnap, endSnap, st, err, WorkloadAllSuccess, nil, nil)
		if e.Hub != nil {
			e.Hub.EmitRunEvent("run", "smoke finished: "+rep.Status, id, id)
		}
	}()
	return id, nil
}

type loadPresetSpec struct {
	Preset      string
	Total       int
	UserIDStart int
	Workers     int
	Timeout     time.Duration
}

func (e *Engine) StartHeavy(ctx context.Context) (string, error) {
	return e.startLoad(ctx, loadPresetSpec{
		Preset:      "heavy-100k",
		Total:       100_000,
		UserIDStart: freshUserIDBase(100_000),
		Workers:     32,
		Timeout:     15 * time.Second,
	}, WorkloadExhaustion, nil)
}

func (e *Engine) StartBoundedLoad(ctx context.Context, preset string, total int) (string, error) {
	workers := boundedWorkers(total)
	return e.startLoad(ctx, loadPresetSpec{
		Preset:      preset,
		Total:       total,
		UserIDStart: 800_000 + total,
		Workers:     workers,
		Timeout:     15 * time.Second,
	}, WorkloadExhaustion, nil)
}

func boundedWorkers(total int) int {
	switch {
	case total <= 1_000:
		return 8
	case total <= 10_000:
		return 16
	case total <= 50_000:
		return 24
	default:
		return 32
	}
}

func (e *Engine) startLoad(ctx context.Context, spec loadPresetSpec, mode string, benchProf *BenchmarkProfile) (string, error) {
	id := newRunID()
	extra := map[string]interface{}{
		"workload_mode":   mode,
		"requested_total": spec.Total,
	}
	if benchProf != nil {
		extra["assumed_inventory_seed"] = benchProf.AssumedInventorySeed
		extra["benchmark_profile"] = benchProf.Preset
	}
	rep := &Report{
		RunID:           id,
		Preset:          spec.Preset,
		Status:          "running",
		TargetBase:      e.cfg.TargetBase,
		LogicalRequests: spec.Total,
		Extra:           extra,
	}
	e.mu.Lock()
	e.runs[id] = rep
	e.mu.Unlock()

	go func() {
		defer e.clearActiveRun()
		e.setActiveRun(id, id)
		if e.Hub != nil {
			e.Hub.EmitRunEvent("run", spec.Preset+" started", id, id)
		}
		startSnap, _ := e.col.Collect(ctx)
		inv := inventoryAtStartFromSnapshot(&startSnap)
		e.mu.Lock()
		rep.SnapshotStart = &startSnap
		rep.InventoryAtStart = inv
		if inv != nil {
			rep.Extra["inventory_at_start"] = *inv
		}
		e.mu.Unlock()

		st, err := loadgen.Run(ctx, loadgen.Options{
			BaseURL:     e.cfg.TargetBase,
			Workers:     spec.Workers,
			Total:       spec.Total,
			UserIDStart: spec.UserIDStart,
			APIKey:      e.cfg.APIKey,
			Timeout:     spec.Timeout,
		})
		endSnap, _ := e.col.Collect(ctx)
		e.mu.Lock()
		defer e.mu.Unlock()
		rep.SnapshotEnd = &endSnap
		rep.FieldProvenance = map[string]string{"load_stats": metrics.Derived}
		e.finalizeLoadReport(rep, startSnap, endSnap, st, err, mode, inv, benchProf)
		if e.Hub != nil {
			e.Hub.EmitRunEvent("run", spec.Preset+" finished: "+rep.Status, id, id)
		}
	}()
	return id, nil
}

func freshUserIDBase(span int) int {
	// Avoid overlapping prior lab runs (idempotent replays) while staying in safe int range.
	var b [4]byte
	_, _ = rand.Read(b[:])
	base := 1_000_000 + int(b[0])<<16 + int(b[1])<<8 + int(b[2])
	if base > 2_000_000_000-span {
		base = 2_000_000_000 - span
	}
	return base
}

func (e *Engine) finalizeLoadReport(rep *Report, startSnap, endSnap metrics.Snapshot, st loadgen.Stats, runErr error, mode string, inventoryAtStart *int64, benchProf *BenchmarkProfile) {
	if runErr != nil {
		rep.Status = "failed"
		if rep.Extra == nil {
			rep.Extra = map[string]interface{}{}
		}
		rep.Extra["error"] = runErr.Error()
		rep.Extra["failure_reason"] = "loadgen_error"
		return
	}
	rep.LogicalRequests = st.LogicalRequests
	rep.HTTP200 = st.HTTP200
	rep.HTTP404 = st.HTTP404
	rep.Errors = st.Errors + st.OtherStatus
	rep.DurationMs = st.DurationMs
	rep.LatencyP95Ms = st.LatencyP95Ms
	rep.LatencyP99Ms = st.LatencyP99Ms
	if rep.SnapshotStart == nil {
		rep.SnapshotStart = &startSnap
	}
	ok, reason := EvaluateLoadPass(st, mode, inventoryAtStart, &startSnap, &endSnap)
	if ok {
		rep.Status = "completed"
	} else {
		rep.Status = "failed"
		if rep.Extra == nil {
			rep.Extra = map[string]interface{}{}
		}
		rep.Extra["failure_reason"] = reason
	}
	if consumed := queueConsumedFromSnapshots(&startSnap, &endSnap); consumed != nil && rep.Extra != nil {
		rep.Extra["queue_consumed"] = *consumed
	}
	if IsBenchmarkPreset(rep.Preset) {
		prof := benchProf
		if prof == nil {
			if p, ok := benchmarkProfiles()[rep.Preset]; ok {
				prof = &p
			}
		}
		attachBenchmarkSummary(rep, prof, st, &startSnap, &endSnap)
	}
}

func (e *Engine) StartClaims(ctx context.Context, claimIDs []string) (string, error) {
	if len(claimIDs) == 0 {
		claimIDs = []string{"INV-1", "INV-2", "INV-3", "INV-4", "INV-5", "INV-6", "INV-7"}
	}
	id := newRunID()
	rep := &Report{
		RunID:      id,
		Preset:     "claims",
		Status:     "running",
		TargetBase: e.cfg.TargetBase,
	}
	e.mu.Lock()
	e.runs[id] = rep
	e.mu.Unlock()

	go func() {
		defer e.clearActiveRun()
		e.setActiveRun(id, id)
		if e.Hub != nil {
			e.Hub.EmitRunEvent("run", "claims started", id, id)
		}
		startSnap, _ := e.col.Collect(ctx)
		var soldBefore int64
		if startSnap.FieldProvenance["mysql"] == metrics.Live {
			soldBefore = startSnap.MySQL.Sold
		}

		client := claims.Client{BaseURL: e.cfg.TargetBase, APIKey: e.cfg.APIKey}
		results := claims.Run(ctx, client, claimIDs)

		// INV-6 durability poll when MySQL configured
		var inv6Ticket int
		for _, cr := range results {
			if cr.ID == "INV-6" {
				if tid, ok := cr.Evidence["ticket_id"].(int); ok {
					inv6Ticket = tid
				}
			}
		}
		soldSeen := false
		if inv6Ticket > 0 && e.cfg.MySQLDSN != "" {
			soldSeen = waitTicketSold(ctx, e.cfg.MySQLDSN, inv6Ticket, 15*time.Second)
		}

		endSnap, _ := e.col.Collect(ctx)
		var soldAfter int64
		if endSnap.FieldProvenance["mysql"] == metrics.Live {
			soldAfter = endSnap.MySQL.Sold
		}
		results = claims.EnrichMySQL(results, soldBefore, soldAfter, inv6Ticket, soldSeen)

		e.mu.Lock()
		defer e.mu.Unlock()
		rep.SnapshotStart = &startSnap
		rep.SnapshotEnd = &endSnap
		rep.Claims = results
		rep.FieldProvenance = map[string]string{"claims": metrics.Derived}
		rep.Status = "completed"
		if e.Hub != nil {
			e.Hub.EmitRunEvent("run", "claims finished", id, id)
		}
	}()

	return id, nil
}

func (e *Engine) StartGates(ctx context.Context, gateIDs []string) (string, error) {
	if len(gateIDs) == 0 {
		gateIDs = claims.DefaultGateIDs
	}
	id := newRunID()
	rep := &Report{
		RunID:      id,
		Preset:     "gates",
		Status:     "running",
		TargetBase: e.cfg.TargetBase,
	}
	e.mu.Lock()
	e.runs[id] = rep
	e.mu.Unlock()

	go func() {
		defer e.clearActiveRun()
		e.setActiveRun(id, id)
		if e.Hub != nil {
			e.Hub.EmitRunEvent("run", "gates started", id, id)
		}
		startSnap, _ := e.col.Collect(ctx)
		repoRoot := e.cfg.RepoRoot
		if repoRoot == "" {
			repoRoot = claims.DiscoverRepoRoot()
		}
		results := claims.RunGates(ctx, claims.GateOpts{
			RepoRoot:   repoRoot,
			TargetBase: e.cfg.TargetBase,
			APIKey:     e.cfg.APIKey,
			RedisAddr:  e.cfg.RedisAddr,
			MySQLDSN:   e.cfg.MySQLDSN,
		}, gateIDs)
		endSnap, _ := e.col.Collect(ctx)
		status, failureReason := claims.SummarizeGateRun(results)

		e.mu.Lock()
		defer e.mu.Unlock()
		rep.SnapshotStart = &startSnap
		rep.SnapshotEnd = &endSnap
		rep.Claims = results
		rep.FieldProvenance = map[string]string{"gates": metrics.Derived}
		rep.Status = status
		if failureReason != "" {
			if rep.Extra == nil {
				rep.Extra = map[string]interface{}{}
			}
			rep.Extra["failure_reason"] = failureReason
		}
		if e.Hub != nil {
			e.Hub.EmitRunEvent("run", "gates finished: "+rep.Status, id, id)
		}
	}()

	return id, nil
}

func (e *Engine) Collector() metrics.Collector {
	return e.col
}

func newRunID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func waitTicketSold(ctx context.Context, dsn string, ticketID int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return false
		}
		gdb, err := gorm.Open(mysql.Open(dsn), &gorm.Config{SkipDefaultTransaction: true})
		if err != nil {
			time.Sleep(200 * time.Millisecond)
			continue
		}
		sqlDB, _ := gdb.DB()
		var state string
		err = gdb.Raw("SELECT state FROM tickets WHERE id = ? LIMIT 1", ticketID).Scan(&state).Error
		if sqlDB != nil {
			sqlDB.Close()
		}
		if err == nil && state == "sold" {
			return true
		}
		time.Sleep(250 * time.Millisecond)
	}
	return false
}
