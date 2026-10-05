package persist

import "time"

const (
	StreamKey     = "bookings.stream"
	ConsumerGroup = "bookings-mysql"
	ConsumerName  = "worker-1"
)

// ReclaimMinIdle must exceed worst-case handleMessage duration so live workers are not reclaimed.
var ReclaimMinIdle = 2 * time.Minute

// ReclaimInterval is how often the consumer scans for stale pending entries.
var ReclaimInterval = 10 * time.Second

// ShutdownDrainTimeout bounds graceful shutdown while in-flight handlers finish.
var ShutdownDrainTimeout = 30 * time.Second
