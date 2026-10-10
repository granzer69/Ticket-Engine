package metrics

import (
	"sync"
	"time"
)

const defaultRingCap = 200

// TelemetryEvent is a single timeline entry (snapshot deltas or run lifecycle).
type TelemetryEvent struct {
	Seq           uint64    `json:"seq"`
	At            time.Time `json:"at"`
	Kind          string    `json:"kind"`
	Message       string    `json:"message"`
	RunID         string    `json:"run_id,omitempty"`
	CorrelationID string    `json:"correlation_id,omitempty"`
	Provenance    string    `json:"provenance"`
}

// RingBuffer stores recent events in insertion order with a fixed capacity.
type RingBuffer struct {
	mu    sync.Mutex
	cap   int
	buf   []TelemetryEvent
	next  uint64
	start int
	len   int
}

func NewRingBuffer(cap int) *RingBuffer {
	if cap <= 0 {
		cap = defaultRingCap
	}
	return &RingBuffer{cap: cap, buf: make([]TelemetryEvent, cap)}
}

func (r *RingBuffer) Add(ev TelemetryEvent) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.next++
	ev.Seq = r.next
	if r.len < r.cap {
		idx := (r.start + r.len) % r.cap
		r.buf[idx] = ev
		r.len++
		return
	}
	r.buf[r.start] = ev
	r.start = (r.start + 1) % r.cap
}

func (r *RingBuffer) SnapshotTail(n int) []TelemetryEvent {
	r.mu.Lock()
	defer r.mu.Unlock()
	if n <= 0 || r.len == 0 {
		return nil
	}
	if n > r.len {
		n = r.len
	}
	out := make([]TelemetryEvent, n)
	for i := 0; i < n; i++ {
		idx := (r.start + r.len - n + i) % r.cap
		out[i] = r.buf[idx]
	}
	return out
}

func (r *RingBuffer) LastSeq() uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.next
}
