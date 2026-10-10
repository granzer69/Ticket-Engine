package metrics

import (
	"context"
	"sync"
	"time"
)

// RunContextProvider supplies active lab run metadata for telemetry frames.
type RunContextProvider interface {
	ActiveRun() (runID, correlationID string)
}

// Hub polls the collector on a fixed interval and fans out enriched frames to SSE subscribers.
type Hub struct {
	Collector Collector
	Interval  time.Duration
	Runs      RunContextProvider

	mu       sync.RWMutex
	latest   Snapshot
	sequence uint64
	events   *RingBuffer

	prevTarget TargetMetrics
	prevAt     time.Time

	stop     chan struct{}
	stopped  chan struct{}
	startOnce sync.Once
}

func NewHub(col Collector, runs RunContextProvider) *Hub {
	return &Hub{
		Collector: col,
		Interval:  200 * time.Millisecond,
		Runs:      runs,
		events:    NewRingBuffer(defaultRingCap),
		stop:      make(chan struct{}),
		stopped:   make(chan struct{}),
	}
}

func (h *Hub) Start() {
	h.startOnce.Do(func() {
		go h.loop()
	})
}

func (h *Hub) Stop() {
	close(h.stop)
	<-h.stopped
}

func (h *Hub) Latest() Snapshot {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.latest
}

func (h *Hub) Subscribe(ctx context.Context) <-chan Snapshot {
	ch := make(chan Snapshot, 2)
	go func() {
		defer close(ch)
		tick := time.NewTicker(h.Interval)
		defer tick.Stop()
		var lastSeq uint64
		for {
			select {
			case <-ctx.Done():
				return
			case <-h.stop:
				return
			case <-tick.C:
				snap := h.Latest()
				if snap.Sequence == 0 {
					continue
				}
				if snap.Sequence == lastSeq {
					continue
				}
				lastSeq = snap.Sequence
				snap.StaleMs = time.Since(snap.At).Milliseconds()
				select {
				case ch <- snap:
				default:
				}
			}
		}
	}()
	return ch
}

func (h *Hub) loop() {
	tick := time.NewTicker(h.Interval)
	defer tick.Stop()
	defer close(h.stopped)

	for {
		select {
		case <-h.stop:
			return
		case <-tick.C:
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			raw, err := h.Collector.Collect(ctx)
			cancel()
			if err != nil {
				continue
			}
			frame := h.enrich(raw)
			h.mu.Lock()
			h.latest = frame
			h.mu.Unlock()
		}
	}
}

func (h *Hub) enrich(raw Snapshot) Snapshot {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.sequence++
	raw.Sequence = h.sequence
	raw.Events = h.events.SnapshotTail(32)

	if h.Runs != nil {
		runID, corr := h.Runs.ActiveRun()
		raw.ActiveRunID = runID
		raw.CorrelationID = corr
	}

	// API rates from target metric deltas.
	if raw.FieldProvenance["target_metrics"] == Live && !h.prevAt.IsZero() {
		dt := raw.At.Sub(h.prevAt).Seconds()
		if dt > 0 {
			raw.APIRates = APIRates{
				RequestsPerSec:  float64(raw.TargetMetrics.TotalRequests-h.prevTarget.TotalRequests) / dt,
				SuccessPerSec:   float64(raw.TargetMetrics.BookingSuccess-h.prevTarget.BookingSuccess) / dt,
				FailuresPerSec:  float64(raw.TargetMetrics.BookingFailures-h.prevTarget.BookingFailures) / dt,
			}
			raw.FieldProvenance["api_rates"] = Derived
		}
	}
	if raw.FieldProvenance["api_rates"] == "" {
		raw.FieldProvenance["api_rates"] = Unavailable
	}

	// Persistence lag heuristic.
	if raw.FieldProvenance["target_metrics"] == Live && raw.FieldProvenance["mysql"] == Live {
		raw.PersistenceLag.SuccessGap = raw.TargetMetrics.BookingSuccess - raw.MySQL.Sold
		raw.FieldProvenance["persistence_lag"] = Derived
		if raw.APIRates.SuccessPerSec > 0.01 {
			raw.PersistenceLag.EstimatedMs = float64(raw.Redis.StreamPending) / raw.APIRates.SuccessPerSec * 1000
		}
	} else {
		raw.FieldProvenance["persistence_lag"] = Unavailable
	}

	h.emitDeltas(raw)
	h.prevTarget = raw.TargetMetrics
	h.prevAt = raw.At

	raw.FieldProvenance["sequence"] = Derived
	raw.FieldProvenance["events"] = Derived
	return raw
}

func (h *Hub) emitDeltas(s Snapshot) {
	prev := h.latest
	if prev.Sequence == 0 {
		return
	}
	runID := s.ActiveRunID
	corr := s.CorrelationID
	emit := func(kind, msg string) {
		h.events.Add(TelemetryEvent{
			At:            s.At,
			Kind:          kind,
			Message:       msg,
			RunID:         runID,
			CorrelationID: corr,
			Provenance:    Derived,
		})
	}
	if s.Redis.StreamPending > prev.Redis.StreamPending+5 {
		emit("redis", "stream pending increased")
	}
	if s.Redis.DLQLen > prev.Redis.DLQLen {
		emit("redis", "DLQ length increased")
	}
	if s.PersistenceLag.SuccessGap > prev.PersistenceLag.SuccessGap+10 {
		emit("persist", "success vs sold gap widened")
	}
}

// EmitRunEvent records a LIVE lifecycle event from the runner.
func (h *Hub) EmitRunEvent(kind, message, runID, correlationID string) {
	if h == nil || h.events == nil {
		return
	}
	h.events.Add(TelemetryEvent{
		At:            time.Now().UTC(),
		Kind:          kind,
		Message:       message,
		RunID:         runID,
		CorrelationID: correlationID,
		Provenance:    Live,
	})
}
