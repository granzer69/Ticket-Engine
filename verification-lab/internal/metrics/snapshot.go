package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"ticketengine/internal/booking"
	"ticketengine/internal/persist"
)

// Provenance labels for dashboard fields.
const (
	Live        = "LIVE"
	Derived     = "DERIVED"
	Unavailable = "UNAVAILABLE"
)

type TargetMetrics struct {
	TotalRequests   int64 `json:"total_requests"`
	BookingSuccess  int64 `json:"booking_success"`
	BookingFailures int64 `json:"booking_failures"`
}

type APIRates struct {
	RequestsPerSec float64 `json:"requests_per_sec"`
	SuccessPerSec  float64 `json:"success_per_sec"`
	FailuresPerSec float64 `json:"failures_per_sec"`
}

type RedisState struct {
	QueueLen      int64 `json:"queue_len"`
	UserHashLen   int64 `json:"user_hash_len"`
	StreamLen     int64 `json:"stream_len"`
	StreamPending int64 `json:"stream_pending"`
	DLQLen        int64 `json:"dlq_len"`
}

type WorkerStats struct {
	Consumers       int64  `json:"consumers"`
	Pending         int64  `json:"pending"`
	LastDeliveredID string `json:"last_delivered_id,omitempty"`
}

type MySQLCounts struct {
	Available int64 `json:"available"`
	Sold      int64 `json:"sold"`
	Total     int64 `json:"total"`
}

type PersistenceLag struct {
	SuccessGap  int64   `json:"success_gap"`
	EstimatedMs float64 `json:"estimated_ms"`
}

// Snapshot is a single telemetry frame for SSE / REST.
type Snapshot struct {
	At              time.Time         `json:"at"`
	Sequence        uint64            `json:"sequence"`
	CollectMs       int64             `json:"collect_ms"`
	StaleMs         int64             `json:"stale_ms,omitempty"`
	TargetBase      string            `json:"target_base"`
	ActiveRunID     string            `json:"active_run_id,omitempty"`
	CorrelationID   string            `json:"correlation_id,omitempty"`
	TargetMetrics   TargetMetrics     `json:"target_metrics"`
	APIRates        APIRates          `json:"api_rates"`
	Redis           RedisState        `json:"redis"`
	Worker          WorkerStats       `json:"worker"`
	MySQL           MySQLCounts       `json:"mysql"`
	PersistenceLag  PersistenceLag    `json:"persistence_lag"`
	Events          []TelemetryEvent  `json:"events"`
	FieldProvenance map[string]string `json:"field_provenance"`

	// Legacy flat fields (UI compat).
	QueueRemaining int64 `json:"queue_remaining"`
	StreamPending  int64 `json:"stream_pending"`
}

type Collector struct {
	TargetBase string
	HTTPClient *http.Client
	RedisAddr  string
	MySQLDSN   string
}

func (c Collector) Collect(ctx context.Context) (Snapshot, error) {
	t0 := time.Now()
	snap := Snapshot{
		At:              time.Now().UTC(),
		TargetBase:      c.TargetBase,
		FieldProvenance: map[string]string{},
	}
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{Timeout: 2 * time.Second}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.TargetBase+"/metrics", nil)
	if err == nil {
		res, err := c.HTTPClient.Do(req)
		if err == nil && res.StatusCode == http.StatusOK {
			var m TargetMetrics
			if json.NewDecoder(res.Body).Decode(&m) == nil {
				snap.TargetMetrics = m
				snap.FieldProvenance["target_metrics"] = Live
			}
			res.Body.Close()
		}
	}
	if snap.FieldProvenance["target_metrics"] == "" {
		snap.FieldProvenance["target_metrics"] = Unavailable
	}

	if c.RedisAddr != "" {
		rdb := redis.NewClient(&redis.Options{Addr: c.RedisAddr})
		defer rdb.Close()
		if err := rdb.Ping(ctx).Err(); err == nil {
			if n, err := rdb.LLen(ctx, booking.KeyQueue).Result(); err == nil {
				snap.Redis.QueueLen = n
				snap.QueueRemaining = n
				snap.FieldProvenance["queue_remaining"] = Live
				snap.FieldProvenance["redis"] = Live
			}
			if n, err := rdb.HLen(ctx, booking.KeyUserBooking).Result(); err == nil {
				snap.Redis.UserHashLen = n
			}
			if n, err := rdb.XLen(ctx, persist.StreamKey).Result(); err == nil {
				snap.Redis.StreamLen = n
			}
			if pending, err := persist.GroupPendingCountRDB(ctx, rdb); err == nil {
				snap.Redis.StreamPending = pending
				snap.StreamPending = pending
				snap.FieldProvenance["stream_pending"] = Live
			}
			if n, err := rdb.XLen(ctx, persist.DLQStreamKey).Result(); err == nil {
				snap.Redis.DLQLen = n
			}
			groups, err := rdb.XInfoGroups(ctx, persist.StreamKey).Result()
			if err == nil {
				for _, g := range groups {
					if g.Name == persist.ConsumerGroup {
						snap.Worker.Consumers = g.Consumers
						snap.Worker.Pending = g.Pending
						snap.Worker.LastDeliveredID = g.LastDeliveredID
						snap.FieldProvenance["worker"] = Live
						break
					}
				}
			}
			if snap.FieldProvenance["worker"] == "" && snap.FieldProvenance["stream_pending"] == Live {
				snap.Worker.Pending = snap.Redis.StreamPending
				snap.FieldProvenance["worker"] = Live
			}
		}
	}
	if snap.FieldProvenance["queue_remaining"] == "" {
		snap.FieldProvenance["queue_remaining"] = Unavailable
	}
	if snap.FieldProvenance["stream_pending"] == "" {
		snap.FieldProvenance["stream_pending"] = Unavailable
	}
	if snap.FieldProvenance["redis"] == "" {
		snap.FieldProvenance["redis"] = Unavailable
	}
	if snap.FieldProvenance["worker"] == "" {
		snap.FieldProvenance["worker"] = Unavailable
	}

	if c.MySQLDSN != "" {
		gdb, err := gorm.Open(mysql.Open(c.MySQLDSN), &gorm.Config{SkipDefaultTransaction: true})
		if err == nil {
			sqlDB, _ := gdb.DB()
			if sqlDB != nil {
				defer sqlDB.Close()
			}
			var available, sold, total int64
			if err := gdb.Raw("SELECT COUNT(*) FROM tickets WHERE state = 'available'").Scan(&available).Error; err == nil {
				if err := gdb.Raw("SELECT COUNT(*) FROM tickets WHERE state = 'sold'").Scan(&sold).Error; err == nil {
					if err := gdb.Raw("SELECT COUNT(*) FROM tickets").Scan(&total).Error; err == nil {
						snap.MySQL = MySQLCounts{Available: available, Sold: sold, Total: total}
						snap.FieldProvenance["mysql"] = Live
					}
				}
			}
		}
	}
	if snap.FieldProvenance["mysql"] == "" {
		snap.FieldProvenance["mysql"] = Unavailable
	}

	snap.CollectMs = time.Since(t0).Milliseconds()
	snap.FieldProvenance["collect_ms"] = Derived
	return snap, nil
}

// DSN builds a MySQL DSN from standard ticket env pieces.
func DSN(host, user, pass, database string) string {
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, pass, host, database)
}
