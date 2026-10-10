package reconcile

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"

	"ticketengine/internal/booking"
	"ticketengine/internal/persist"
)

// Exit codes for the reconcile CLI.
const (
	ExitClean       = 0
	ExitDrift       = 1
	ExitOperational = 2
)

// PendingDisposition classifies PEL entries for operators.
type PendingDisposition string

const (
	PendingExpected   PendingDisposition = "EXPECTED"
	PendingStale      PendingDisposition = "STALE"
	PendingSuspicious PendingDisposition = "SUSPICIOUS"
	PendingUnknown    PendingDisposition = "UNKNOWN"
)

// Issue is a single drift finding.
type Issue struct {
	Section string
	Detail  string
}

// PendingEntry describes one PEL message.
type PendingEntry struct {
	ID          string
	Idle        time.Duration
	Retries     int64
	Disposition PendingDisposition
	TicketID    int
	UserID      int
}

// Report is the full read-only reconciliation outcome.
type Report struct {
	Issues []Issue

	RedisQueueLen    int64
	MySQLAvailable   int64
	MySQLSold        int64
	MySQLTotal       int64
	HashUserEntries  int64
	StreamLen        int64
	StreamLag        int64
	PendingTotal     int64
	DLQLen           int64
	PendingBreakdown []PendingEntry
}

func (r *Report) HasDrift() bool {
	return len(r.Issues) > 0
}

func (r *Report) ExitCode() int {
	if r.HasDrift() {
		return ExitDrift
	}
	return ExitClean
}

// Run performs read-only checks across Redis and MySQL.
func Run(ctx context.Context, db *gorm.DB, rdb *redis.Client) (*Report, error) {
	rep := &Report{}

	if err := loadInventoryCounts(ctx, db, rdb, rep); err != nil {
		return rep, err
	}
	if err := checkDuplicateOwnership(ctx, db, rdb, rep); err != nil {
		return rep, err
	}

	streamState, err := loadStreamState(ctx, rdb, rep)
	if err != nil {
		return rep, err
	}
	if err := checkInventoryAlignment(ctx, db, rdb, rep, streamState); err != nil {
		return rep, err
	}
	if err := checkOwnershipOrphans(ctx, db, rdb, rep, streamState); err != nil {
		return rep, err
	}

	return rep, nil
}

func loadInventoryCounts(ctx context.Context, db *gorm.DB, rdb *redis.Client, rep *Report) error {
	if err := db.Model(&struct{}{}).Table("tickets").Count(&rep.MySQLTotal).Error; err != nil {
		return fmt.Errorf("mysql count tickets: %w", err)
	}
	if err := db.Model(&struct{}{}).Table("tickets").Where("state = ?", "available").Count(&rep.MySQLAvailable).Error; err != nil {
		return fmt.Errorf("mysql count available: %w", err)
	}
	if err := db.Model(&struct{}{}).Table("tickets").Where("state = ?", "sold").Count(&rep.MySQLSold).Error; err != nil {
		return fmt.Errorf("mysql count sold: %w", err)
	}

	redisLen, err := rdb.LLen(ctx, booking.KeyQueue).Result()
	if err != nil {
		return fmt.Errorf("redis llen queue: %w", err)
	}
	rep.RedisQueueLen = redisLen

	hlen, err := rdb.HLen(ctx, booking.KeyUserBooking).Result()
	if err != nil {
		return fmt.Errorf("redis hlen user hash: %w", err)
	}
	rep.HashUserEntries = hlen

	return nil
}

func checkInventoryAlignment(ctx context.Context, db *gorm.DB, rdb *redis.Client, rep *Report, stream streamState) error {
	inFlight, err := explainedInFlightCount(ctx, db, rdb, stream)
	if err != nil {
		return err
	}
	expectedQueue := rep.MySQLAvailable - inFlight
	if rep.RedisQueueLen != expectedQueue {
		rep.Issues = append(rep.Issues, Issue{
			Section: "inventory",
			Detail: fmt.Sprintf(
				"redis queue length %d != mysql available %d minus in-flight %d (expected queue %d)",
				rep.RedisQueueLen, rep.MySQLAvailable, inFlight, expectedQueue,
			),
		})
	}
	return nil
}

func explainedInFlightCount(ctx context.Context, db *gorm.DB, rdb *redis.Client, stream streamState) (int64, error) {
	fields, err := rdb.HGetAll(ctx, booking.KeyUserBooking).Result()
	if err != nil {
		return 0, fmt.Errorf("redis hgetall user hash: %w", err)
	}
	var count int64
	for uidStr, tidStr := range fields {
		userID, err := strconv.Atoi(uidStr)
		if err != nil {
			continue
		}
		ticketID, err := strconv.Atoi(tidStr)
		if err != nil {
			continue
		}
		var state string
		if err := db.Raw(`SELECT state FROM tickets WHERE id = ?`, ticketID).Scan(&state).Error; err != nil {
			return 0, err
		}
		if state != "available" {
			continue
		}
		if ref, ok := stream.pending[ticketID]; ok && ref.userID == userID {
			count++
			continue
		}
		if u, ok := stream.undelivered[ticketID]; ok && u == userID {
			count++
		}
	}
	return count, nil
}

type ticketOwner struct {
	TicketID int
	UserID   *int
	State    string
}

func checkDuplicateOwnership(ctx context.Context, db *gorm.DB, rdb *redis.Client, rep *Report) error {
	type dupUser struct {
		UserID int
		Count  int64
	}
	var dups []dupUser
	err := db.Raw(`
		SELECT user_id AS user_id, COUNT(*) AS count
		FROM tickets
		WHERE state = 'sold' AND user_id IS NOT NULL
		GROUP BY user_id
		HAVING COUNT(*) > 1`).Scan(&dups).Error
	if err != nil {
		return fmt.Errorf("mysql duplicate users: %w", err)
	}
	for _, d := range dups {
		rep.Issues = append(rep.Issues, Issue{
			Section: "duplicate_ownership",
			Detail:  fmt.Sprintf("mysql user_id %d owns %d sold tickets", d.UserID, d.Count),
		})
	}

	fields, err := rdb.HGetAll(ctx, booking.KeyUserBooking).Result()
	if err != nil {
		return fmt.Errorf("redis hgetall user hash: %w", err)
	}
	ticketToUser := map[string]string{}
	for uid, tid := range fields {
		if prev, ok := ticketToUser[tid]; ok {
			rep.Issues = append(rep.Issues, Issue{
				Section: "duplicate_ownership",
				Detail:  fmt.Sprintf("redis ticket %s claimed by users %s and %s", tid, prev, uid),
			})
			continue
		}
		ticketToUser[tid] = uid
	}
	return nil
}

type streamState struct {
	pending     map[int]pendingRef
	undelivered map[int]int
}

func loadStreamState(ctx context.Context, rdb *redis.Client, rep *Report) (streamState, error) {
	st := streamState{
		pending:     map[int]pendingRef{},
		undelivered: map[int]int{},
	}

	streamLen, err := rdb.XLen(ctx, persist.StreamKey).Result()
	if err != nil && !keyMissing(err) {
		return st, fmt.Errorf("redis xlen stream: %w", err)
	}
	rep.StreamLen = streamLen

	lastDelivered, lagErr := consumerGroupLastDelivered(ctx, rdb)
	if lagErr != nil && !keyMissing(lagErr) && !noGroup(lagErr) {
		return st, fmt.Errorf("redis xinfo groups: %w", lagErr)
	}
	st.undelivered, rep.StreamLag = undeliveredAllocations(ctx, rdb, lastDelivered)

	pending, err := persist.GroupPendingCountRDB(ctx, rdb)
	if err != nil && !keyMissing(err) && !noGroup(err) {
		return st, fmt.Errorf("redis pending count: %w", err)
	}
	rep.PendingTotal = pending

	dlqLen, err := rdb.XLen(ctx, persist.DLQStreamKey).Result()
	if err != nil && !keyMissing(err) {
		return st, fmt.Errorf("redis xlen dlq: %w", err)
	}
	rep.DLQLen = dlqLen
	if dlqLen > 0 {
		rep.Issues = append(rep.Issues, Issue{
			Section: "dlq",
			Detail:  fmt.Sprintf("bookings.dlq has %d entries (failed persist)", dlqLen),
		})
	}

	if pending > 0 {
		ext, err := rdb.XPendingExt(ctx, &redis.XPendingExtArgs{
			Stream: persist.StreamKey,
			Group:  persist.ConsumerGroup,
			Start:  "-",
			End:    "+",
			Count:  1000,
		}).Result()
		if err != nil {
			rep.PendingBreakdown = append(rep.PendingBreakdown, PendingEntry{
				ID:          "-",
				Disposition: PendingUnknown,
			})
			rep.Issues = append(rep.Issues, Issue{
				Section: "stream_pending",
				Detail:  fmt.Sprintf("could not inspect pending entries: %v", err),
			})
			return st, nil
		}
		staleCount := 0
		suspiciousCount := 0
		for _, e := range ext {
			entry := PendingEntry{
				ID:      e.ID,
				Idle:    time.Duration(e.Idle) * time.Millisecond,
				Retries: e.RetryCount,
			}
			entry.Disposition = classifyPending(entry.Idle, entry.Retries)
			switch entry.Disposition {
			case PendingStale:
				staleCount++
			case PendingSuspicious:
				suspiciousCount++
			}
			tid, uid, err := streamMessageTicketUser(ctx, rdb, e.ID)
			if err != nil {
				entry.Disposition = PendingUnknown
			} else {
				entry.TicketID = tid
				entry.UserID = uid
				st.pending[tid] = pendingRef{userID: uid, disposition: entry.Disposition}
			}
			rep.PendingBreakdown = append(rep.PendingBreakdown, entry)
		}
		if staleCount > 0 {
			rep.Issues = append(rep.Issues, Issue{
				Section: "stream_pending",
				Detail:  fmt.Sprintf("%d STALE pending entries (idle >= %s)", staleCount, persist.ReclaimMinIdle),
			})
		}
		if suspiciousCount > 0 {
			rep.Issues = append(rep.Issues, Issue{
				Section: "stream_pending",
				Detail:  fmt.Sprintf("%d SUSPICIOUS pending entries (high delivery count)", suspiciousCount),
			})
		}
	}

	return st, nil
}

func consumerGroupLastDelivered(ctx context.Context, rdb *redis.Client) (string, error) {
	groups, err := rdb.XInfoGroups(ctx, persist.StreamKey).Result()
	if err == nil {
		for _, g := range groups {
			if g.Name == persist.ConsumerGroup {
				if g.LastDeliveredID == "" {
					return "0-0", nil
				}
				return g.LastDeliveredID, nil
			}
		}
		return "0-0", nil
	}
	raw, err2 := rdb.Do(ctx, "XINFO", "GROUPS", persist.StreamKey).Result()
	if err2 != nil {
		return "", err
	}
	entries, ok := raw.([]interface{})
	if !ok {
		return "", err
	}
	for _, entry := range entries {
		pairs, ok := entry.([]interface{})
		if !ok {
			continue
		}
		var name, lastID string
		for i := 0; i+1 < len(pairs); i += 2 {
			key, _ := pairs[i].(string)
			switch key {
			case "name":
				name, _ = pairs[i+1].(string)
			case "last-delivered-id":
				lastID, _ = pairs[i+1].(string)
			}
		}
		if name == persist.ConsumerGroup {
			if lastID == "" {
				return "0-0", nil
			}
			return lastID, nil
		}
	}
	return "0-0", nil
}

func undeliveredAllocations(ctx context.Context, rdb *redis.Client, lastDelivered string) (map[int]int, int64) {
	out := map[int]int{}
	start := "-"
	if lastDelivered != "" && lastDelivered != "0-0" {
		start = "(" + lastDelivered
	}
	msgs, err := rdb.XRange(ctx, persist.StreamKey, start, "+").Result()
	if err != nil {
		return out, 0
	}
	for _, msg := range msgs {
		tid, err := strconv.Atoi(fmt.Sprint(msg.Values["ticket_id"]))
		if err != nil {
			continue
		}
		uid, err := strconv.Atoi(fmt.Sprint(msg.Values["user_id"]))
		if err != nil {
			continue
		}
		out[tid] = uid
	}
	return out, int64(len(msgs))
}

type pendingRef struct {
	userID      int
	disposition PendingDisposition
}

func classifyPending(idle time.Duration, retries int64) PendingDisposition {
	if retries >= persist.MaxDeliveryAttempts-1 {
		return PendingSuspicious
	}
	if idle >= persist.ReclaimMinIdle {
		return PendingStale
	}
	return PendingExpected
}

func streamMessageTicketUser(ctx context.Context, rdb *redis.Client, id string) (int, int, error) {
	msgs, err := rdb.XRange(ctx, persist.StreamKey, id, id).Result()
	if err != nil {
		return 0, 0, err
	}
	if len(msgs) == 0 {
		return 0, 0, fmt.Errorf("message %s not found in stream", id)
	}
	tid, err := strconv.Atoi(fmt.Sprint(msgs[0].Values["ticket_id"]))
	if err != nil {
		return 0, 0, err
	}
	uid, err := strconv.Atoi(fmt.Sprint(msgs[0].Values["user_id"]))
	if err != nil {
		return 0, 0, err
	}
	return tid, uid, nil
}

func checkOwnershipOrphans(ctx context.Context, db *gorm.DB, rdb *redis.Client, rep *Report, stream streamState) error {
	fields, err := rdb.HGetAll(ctx, booking.KeyUserBooking).Result()
	if err != nil {
		return fmt.Errorf("redis hgetall user hash: %w", err)
	}

	for uidStr, tidStr := range fields {
		userID, err := strconv.Atoi(uidStr)
		if err != nil {
			rep.Issues = append(rep.Issues, Issue{
				Section: "ownership",
				Detail:  fmt.Sprintf("invalid redis user id field %q", uidStr),
			})
			continue
		}
		ticketID, err := strconv.Atoi(tidStr)
		if err != nil {
			rep.Issues = append(rep.Issues, Issue{
				Section: "ownership",
				Detail:  fmt.Sprintf("invalid redis ticket id for user %d: %q", userID, tidStr),
			})
			continue
		}

		var row ticketOwner
		err = db.Raw(`SELECT id AS ticket_id, user_id, state FROM tickets WHERE id = ?`, ticketID).
			Scan(&row).Error
		if err != nil {
			return fmt.Errorf("mysql load ticket %d: %w", ticketID, err)
		}
		if row.TicketID == 0 {
			rep.Issues = append(rep.Issues, Issue{
				Section: "ownership",
				Detail:  fmt.Sprintf("redis allocation user=%d ticket=%d missing mysql row", userID, ticketID),
			})
			continue
		}

		if row.State == "sold" {
			if row.UserID == nil || *row.UserID != userID {
				mysqlUID := 0
				if row.UserID != nil {
					mysqlUID = *row.UserID
				}
				rep.Issues = append(rep.Issues, Issue{
					Section: "ownership",
					Detail:  fmt.Sprintf("ticket %d sold to mysql user %d but redis hash:user has user %d", ticketID, mysqlUID, userID),
				})
			}
			continue
		}

		if row.State == "available" {
			if ref, ok := stream.pending[ticketID]; ok && ref.userID == userID {
				continue
			}
			if uid, ok := stream.undelivered[ticketID]; ok && uid == userID {
				continue
			}
			rep.Issues = append(rep.Issues, Issue{
				Section: "orphan_allocation",
				Detail:  fmt.Sprintf("redis owns ticket %d for user %d but mysql state=%s (not explained by expected pending)", ticketID, userID, row.State),
			})
		}
	}
	return nil
}

func keyMissing(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "no such key") || err == redis.Nil
}

func noGroup(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "NOGROUP")
}

// Format renders human-readable output for operators.
func (r *Report) Format() string {
	var b strings.Builder
	if r.HasDrift() {
		b.WriteString("RECONCILE RESULT: DRIFT DETECTED\n")
	} else {
		b.WriteString("RECONCILE RESULT: CLEAN\n")
	}
	b.WriteString("\n[inventory]\n")
	b.WriteString(fmt.Sprintf("  redis_queue_len: %d\n", r.RedisQueueLen))
	b.WriteString(fmt.Sprintf("  mysql_available: %d\n", r.MySQLAvailable))
	b.WriteString(fmt.Sprintf("  mysql_sold: %d\n", r.MySQLSold))
	b.WriteString(fmt.Sprintf("  mysql_total: %d\n", r.MySQLTotal))
	b.WriteString(fmt.Sprintf("  hash_user_entries: %d\n", r.HashUserEntries))

	b.WriteString("\n[stream]\n")
	b.WriteString(fmt.Sprintf("  stream_len: %d\n", r.StreamLen))
	b.WriteString(fmt.Sprintf("  group_lag: %d\n", r.StreamLag))
	b.WriteString(fmt.Sprintf("  pending_total: %d\n", r.PendingTotal))
	b.WriteString(fmt.Sprintf("  dlq_len: %d\n", r.DLQLen))

	if len(r.PendingBreakdown) > 0 {
		b.WriteString("\n[pending_detail]\n")
		for _, p := range r.PendingBreakdown {
			b.WriteString(fmt.Sprintf("  id=%s ticket=%d user=%d idle=%s retries=%d %s\n",
				p.ID, p.TicketID, p.UserID, p.Idle.Round(time.Millisecond), p.Retries, p.Disposition))
		}
	}

	if len(r.Issues) > 0 {
		b.WriteString("\n[issues]\n")
		bySection := map[string][]string{}
		for _, iss := range r.Issues {
			bySection[iss.Section] = append(bySection[iss.Section], iss.Detail)
		}
		for section, details := range bySection {
			b.WriteString(fmt.Sprintf("  [%s]\n", section))
			for _, d := range details {
				b.WriteString(fmt.Sprintf("    - %s\n", d))
			}
		}
	}
	return b.String()
}
