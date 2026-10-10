package metrics

import (
	"testing"
)

func TestRingBufferCapAndTail(t *testing.T) {
	r := NewRingBuffer(3)
	for i := 0; i < 5; i++ {
		r.Add(TelemetryEvent{Kind: "k", Message: string(rune('a' + i))})
	}
	tail := r.SnapshotTail(2)
	if len(tail) != 2 {
		t.Fatalf("tail len %d", len(tail))
	}
	if tail[0].Message != "d" || tail[1].Message != "e" {
		t.Fatalf("tail order: %+v", tail)
	}
	if r.LastSeq() != 5 {
		t.Fatalf("seq %d", r.LastSeq())
	}
}
