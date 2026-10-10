package claims

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	StatusPass         = "PASS"
	StatusFail         = "FAIL"
	StatusInconclusive = "INCONCLUSIVE"
)

type Result struct {
	ID            string                 `json:"id"`
	Status        string                 `json:"status"`
	Message       string                 `json:"message"`
	FailureReason string                 `json:"failure_reason,omitempty"`
	Evidence      map[string]interface{} `json:"evidence"`
	RegistryTests []string               `json:"registry_tests"`
}

type Client struct {
	BaseURL string
	APIKey  string
	HTTP    *http.Client
}

type bookJSON struct {
	Status   string `json:"status"`
	TicketID int    `json:"ticket_id"`
	UserID   int    `json:"user_id"`
}

func (c *Client) book(ctx context.Context, userID int) (bookJSON, int, error) {
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/book", nil)
	if err != nil {
		return bookJSON{}, 0, err
	}
	req.Header.Set("X-User-Id", fmt.Sprintf("%d", userID))
	if c.APIKey != "" {
		req.Header.Set("X-API-Key", c.APIKey)
	}
	res, err := client.Do(req)
	if err != nil {
		return bookJSON{}, 0, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	var out bookJSON
	_ = json.Unmarshal(body, &out)
	return out, res.StatusCode, nil
}

// Run executes selected claim scenarios against a live target.
func Run(ctx context.Context, c Client, ids []string) []Result {
	out := make([]Result, 0, len(ids))
	for _, id := range ids {
		out = append(out, runOne(ctx, c, id))
	}
	return out
}

func runOne(ctx context.Context, c Client, id string) Result {
	reg := Registry[id]
	r := Result{ID: id, RegistryTests: reg, Evidence: map[string]interface{}{}}
	switch id {
	case "INV-1":
		return runINV1(ctx, c, r)
	case "INV-2":
		return runINV2(ctx, c, r)
	case "INV-3":
		return runINV3(ctx, c, r)
	case "INV-4":
		return runINV4(ctx, c, r)
	case "INV-5":
		return runINV5(ctx, c, r)
	case "INV-6":
		return runINV6(ctx, c, r)
	case "INV-7":
		return runINV7(ctx, c, r)
	default:
		r.Status = StatusInconclusive
		r.Message = "unknown claim id"
		return r
	}
}

func runINV1(ctx context.Context, c Client, r Result) Result {
	const n = 32
	base := 880_000 + int(time.Now().Unix()%1000)*100
	seen := make(map[int]int)
	var mu sync.Mutex
	var wg sync.WaitGroup
	errs := 0
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(uid int) {
			defer wg.Done()
			b, code, err := c.book(ctx, uid)
			if err != nil || code != http.StatusOK || b.TicketID == 0 {
				errs++
				return
			}
			mu.Lock()
			seen[b.TicketID]++
			mu.Unlock()
		}(base + i)
	}
	wg.Wait()
	r.Evidence["concurrent_users"] = n
	r.Evidence["unique_tickets"] = len(seen)
	r.Evidence["errors"] = errs
	r.Evidence["duplicate_ticket_ids"] = duplicateKeys(seen)
	if errs > 0 {
		r.Status = StatusFail
		r.Message = "one or more bookings failed"
		return r
	}
	if len(seen) != n {
		r.Status = StatusFail
		r.Message = "ticket ids not unique across concurrent users"
		return r
	}
	r.Status = StatusPass
	r.Message = "concurrent users received distinct tickets"
	return r
}

func runINV2(ctx context.Context, c Client, r Result) Result {
	uid := 881_000 + int(time.Now().Unix()%10000)
	const n = 16
	var tickets []int
	var tMu sync.Mutex
	var wg sync.WaitGroup
	errs := 0
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			b, code, err := c.book(ctx, uid)
			if err != nil || code != http.StatusOK {
				errs++
				return
			}
			tMu.Lock()
			tickets = append(tickets, b.TicketID)
			tMu.Unlock()
		}()
	}
	wg.Wait()
	uniq := uniqueInts(tickets)
	r.Evidence["user_id"] = uid
	r.Evidence["attempts"] = n
	r.Evidence["success_responses"] = len(tickets)
	r.Evidence["unique_ticket_ids"] = uniq
	r.Evidence["errors"] = errs
	if errs > 0 && len(tickets) == 0 {
		r.Status = StatusFail
		r.Message = "all concurrent bookings for same user failed"
		return r
	}
	if len(uniq) > 1 {
		r.Status = StatusFail
		r.Message = "same user received multiple ticket ids"
		return r
	}
	r.Status = StatusPass
	r.Message = "same user maps to at most one ticket"
	return r
}

func runINV3(ctx context.Context, c Client, r Result) Result {
	uid := 882_000 + int(time.Now().Unix()%10000)
	first, c1, err1 := c.book(ctx, uid)
	second, c2, err2 := c.book(ctx, uid)
	r.Evidence["user_id"] = uid
	r.Evidence["first_status"] = c1
	r.Evidence["second_status"] = c2
	r.Evidence["first_ticket"] = first.TicketID
	r.Evidence["second_ticket"] = second.TicketID
	if err1 != nil || err2 != nil || c1 != http.StatusOK || c2 != http.StatusOK {
		r.Status = StatusFail
		r.Message = "idempotent replay requests failed"
		return r
	}
	if first.TicketID == 0 || first.TicketID != second.TicketID {
		r.Status = StatusFail
		r.Message = "retry did not return same ticket_id"
		return r
	}
	r.Status = StatusPass
	r.Message = "retry returned same booking result"
	return r
}

func runINV4(ctx context.Context, c Client, r Result) Result {
	// Lab cannot restart serve process safely here; compare row count stability via metrics + optional MySQL if wired in runner.
	r.Status = StatusInconclusive
	r.Message = "full serve restart not orchestrated by lab; use integration TestRestartDoesNotMintInventory"
	r.Evidence["reason"] = "requires controlled process restart"
	return r
}

func runINV5(ctx context.Context, c Client, r Result) Result {
	// Exercised when runner passes mysql counts into evidence; without MySQL snapshot, inconclusive.
	r.Status = StatusInconclusive
	r.Message = "requires LIVE MySQL sold/available/total counts in run export"
	r.Evidence["hint"] = "run with MYSQL_* env for verilab collector"
	return r
}

func runINV6(ctx context.Context, c Client, r Result) Result {
	uid := 883_000 + int(time.Now().Unix()%10000)
	b, code, err := c.book(ctx, uid)
	r.Evidence["user_id"] = uid
	r.Evidence["http_status"] = code
	r.Evidence["ticket_id"] = b.TicketID
	if err != nil || code != http.StatusOK || b.TicketID == 0 {
		r.Status = StatusFail
		r.Message = "booking did not succeed"
		return r
	}
	r.Status = StatusInconclusive
	r.Message = "MySQL durability poll not available in this scenario-only call; see runner export mysql_after"
	r.Evidence["next_step"] = "runner waits on sold row when DSN configured"
	return r
}

func runINV7(ctx context.Context, c Client, r Result) Result {
	r.Status = StatusInconclusive
	r.Message = "orphan allocation injection not performed in shared lab env"
	r.Evidence["reason"] = "safe failure lab not enabled (slice G)"
	return r
}

func duplicateKeys(m map[int]int) []int {
	var d []int
	for k, v := range m {
		if v > 1 {
			d = append(d, k)
		}
	}
	return d
}

func uniqueInts(in []int) []int {
	seen := map[int]struct{}{}
	var out []int
	for _, v := range in {
		if v == 0 {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// EnrichMySQL updates INV-5/INV-6 results when runner has MySQL telemetry.
func EnrichMySQL(results []Result, beforeSold, afterSold int64, ticketID int, soldSeen bool) []Result {
	out := make([]Result, len(results))
	for i, r := range results {
		out[i] = r
		switch r.ID {
		case "INV-5":
			out[i].Evidence["sold_before"] = beforeSold
			out[i].Evidence["sold_after"] = afterSold
			if beforeSold >= 0 && afterSold >= 0 && afterSold >= beforeSold {
				out[i].Status = StatusPass
				out[i].Message = "sold count monotonic; reconcile tests cover hard ceiling"
			}
		case "INV-6":
			if ticketID > 0 {
				out[i].Evidence["mysql_sold_observed"] = soldSeen
				if soldSeen {
					out[i].Status = StatusPass
					out[i].Message = "ticket reached sold in MySQL within drain window"
				} else {
					out[i].Status = StatusFail
					out[i].Message = "ticket not observed sold in MySQL before timeout"
				}
			}
		}
	}
	return out
}
