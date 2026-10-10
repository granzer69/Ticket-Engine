package metrics

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"ticketengine/internal/booking"
	"ticketengine/internal/persist"
)

func TestCollectorRedisAndWorkerLive(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer mr.Close()

	ctx := context.Background()
	rdbAddr := mr.Addr()
	rdb := redis.NewClient(&redis.Options{Addr: rdbAddr})
	defer rdb.Close()
	_ = rdb.LPush(ctx, booking.KeyQueue, "1", "2").Err()
	_ = rdb.HSet(ctx, booking.KeyUserBooking, "7", "1").Err()
	_ = rdb.XGroupCreateMkStream(ctx, persist.StreamKey, persist.ConsumerGroup, "0").Err()
	_ = rdb.XAdd(ctx, &redis.XAddArgs{Stream: persist.StreamKey, Values: map[string]interface{}{"ticket_id": "1"}}).Err()

	metricsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(TargetMetrics{TotalRequests: 10, BookingSuccess: 8, BookingFailures: 2})
	}))
	defer metricsSrv.Close()

	col := Collector{TargetBase: metricsSrv.URL, RedisAddr: rdbAddr}
	snap, err := col.Collect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if snap.FieldProvenance["target_metrics"] != Live {
		t.Fatalf("metrics prov %s", snap.FieldProvenance["target_metrics"])
	}
	if snap.Redis.QueueLen != 2 {
		t.Fatalf("queue %d", snap.Redis.QueueLen)
	}
	if snap.Redis.UserHashLen != 1 {
		t.Fatalf("hash %d", snap.Redis.UserHashLen)
	}
	if snap.FieldProvenance["worker"] != Live {
		t.Fatalf("worker prov %s", snap.FieldProvenance["worker"])
	}
	if snap.Worker.Pending < 0 {
		t.Fatalf("worker pending %d", snap.Worker.Pending)
	}
}

func TestHubSubscribeCancels(t *testing.T) {
	metricsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(TargetMetrics{})
	}))
	defer metricsSrv.Close()

	h := NewHub(Collector{TargetBase: metricsSrv.URL}, nil)
	h.Interval = 50 * time.Millisecond
	h.Start()

	ctx, cancel := context.WithCancel(context.Background())
	ch := h.Subscribe(ctx)
	cancel()
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		select {
		case _, ok := <-ch:
			if !ok {
				h.Stop()
				return
			}
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}
	h.Stop()
	t.Fatal("subscribe channel did not close after cancel")
}
