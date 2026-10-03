package persist

import "time"

const (
	StreamKey         = "bookings.stream"
	DLQStreamKey      = "bookings.dlq"
	ConsumerGroup     = "bookings-mysql"
	ConsumerName      = "worker-1"
	MaxHandleAttempts = 5
	PendingMinIdle    = 2 * time.Second
)
