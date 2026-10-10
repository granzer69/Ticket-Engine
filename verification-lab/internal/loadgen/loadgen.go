package loadgen

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Options bounds concurrent workers and total logical booking attempts.
type Options struct {
	BaseURL     string
	Workers     int
	Total       int
	UserIDStart int
	APIKey      string
	Timeout     time.Duration
}

// Stats from a bounded load run.
type Stats struct {
	LogicalRequests int
	HTTP200         int
	HTTP404         int
	OtherStatus     int
	Errors          int
	DurationMs      int64
	LatencyP50Ms    int64
	LatencyP95Ms    int64
	LatencyP99Ms    int64
}

// Run issues POST /book with distinct X-User-Id per logical request (unless UserIDStart reused by caller).
func Run(ctx context.Context, opt Options) (Stats, error) {
	if opt.Workers <= 0 {
		opt.Workers = 4
	}
	if opt.Total <= 0 {
		return Stats{}, fmt.Errorf("loadgen: total must be positive")
	}
	if opt.Timeout <= 0 {
		opt.Timeout = 10 * time.Second
	}
	client := &http.Client{
		Timeout: opt.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:        opt.Workers * 2,
			MaxIdleConnsPerHost: opt.Workers * 2,
			MaxConnsPerHost:     opt.Workers * 2,
		},
	}

	var (
		http200   int64
		http404   int64
		other     int64
		errors    int64
		latencies []int64
		latMu     sync.Mutex
	)

	start := time.Now()
	jobs := make(chan int, opt.Workers*2)
	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()
		for uid := range jobs {
			if ctx.Err() != nil {
				return
			}
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, opt.BaseURL+"/book", nil)
			if err != nil {
				atomic.AddInt64(&errors, 1)
				continue
			}
			req.Header.Set("X-User-Id", fmt.Sprintf("%d", uid))
			if opt.APIKey != "" {
				req.Header.Set("X-API-Key", opt.APIKey)
			}
			t0 := time.Now()
			res, err := client.Do(req)
			ms := time.Since(t0).Milliseconds()
			latMu.Lock()
			latencies = append(latencies, ms)
			latMu.Unlock()
			if err != nil {
				atomic.AddInt64(&errors, 1)
				continue
			}
			io.Copy(io.Discard, res.Body)
			res.Body.Close()
			switch res.StatusCode {
			case http.StatusOK:
				atomic.AddInt64(&http200, 1)
			case http.StatusNotFound:
				atomic.AddInt64(&http404, 1)
			default:
				atomic.AddInt64(&other, 1)
			}
		}
	}

	for w := 0; w < opt.Workers; w++ {
		wg.Add(1)
		go worker()
	}

	go func() {
		for i := 0; i < opt.Total; i++ {
			if ctx.Err() != nil {
				break
			}
			jobs <- opt.UserIDStart + i
		}
		close(jobs)
	}()

	wg.Wait()

	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
	p50, p95, p99 := percentile(latencies, 50), percentile(latencies, 95), percentile(latencies, 99)

	return Stats{
		LogicalRequests: opt.Total,
		HTTP200:         int(atomic.LoadInt64(&http200)),
		HTTP404:         int(atomic.LoadInt64(&http404)),
		OtherStatus:     int(atomic.LoadInt64(&other)),
		Errors:          int(atomic.LoadInt64(&errors)),
		DurationMs:      time.Since(start).Milliseconds(),
		LatencyP50Ms:    p50,
		LatencyP95Ms:    p95,
		LatencyP99Ms:    p99,
	}, nil
}

func percentile(sorted []int64, p int) int64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}
	idx := (len(sorted)*p + 99) / 100
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
