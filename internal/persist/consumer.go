package persist

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"gorm.io/gorm"
)

// TicketRow is the persistence view of a ticket.
type TicketRow struct {
	ID     int
	UserID *int
	State  string
}

// MessagePause, when set, runs at the start of handleMessage (integration tests).
var MessagePause func(ctx context.Context, ticketID int) error

// Consumer reads the bookings stream and applies MySQL updates.
type Consumer struct {
	redis    *redis.Client
	db       *gorm.DB
	name     string
	inflight sync.WaitGroup
}

func NewConsumer(rdb *redis.Client, db *gorm.DB, consumerName string) *Consumer {
	if consumerName == "" {
		consumerName = ConsumerName
	}
	return &Consumer{redis: rdb, db: db, name: consumerName}
}

func (c *Consumer) EnsureGroup(ctx context.Context) error {
	err := c.redis.XGroupCreateMkStream(ctx, StreamKey, ConsumerGroup, "0").Err()
	if err != nil && !strings.Contains(err.Error(), "BUSYGROUP") {
		return err
	}
	return nil
}

// GroupPendingCount returns total pending messages for the consumer group.
func (c *Consumer) GroupPendingCount(ctx context.Context) (int64, error) {
	pending, err := c.redis.XPending(ctx, StreamKey, ConsumerGroup).Result()
	if err != nil {
		return 0, err
	}
	return pending.Count, nil
}

func (c *Consumer) Run(ctx context.Context) {
	reclaimTicker := time.NewTicker(ReclaimInterval)
	defer reclaimTicker.Stop()

	for ctx.Err() == nil {
		c.reclaimStale(ctx)
		c.readAndProcess(ctx, "0", 0)

		select {
		case <-ctx.Done():
		case <-reclaimTicker.C:
			c.reclaimStale(ctx)
		default:
		}

		if ctx.Err() != nil {
			break
		}
		c.readAndProcess(ctx, ">", 2*time.Second)
	}

	pending, err := c.GroupPendingCount(context.Background())
	if err != nil {
		log.Printf("[persist] shutdown: pending count: %v", err)
	} else {
		log.Printf("[persist] shutdown drain started (group_pending=%d)", pending)
	}
	if !c.waitInflight(ShutdownDrainTimeout) {
		log.Printf("[persist] shutdown drain timed out after %s", ShutdownDrainTimeout)
	}
}

func (c *Consumer) waitInflight(timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		c.inflight.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

func (c *Consumer) readAndProcess(ctx context.Context, streamID string, block time.Duration) {
	streams, err := c.redis.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    ConsumerGroup,
		Consumer: c.name,
		Streams:  []string{StreamKey, streamID},
		Count:    10,
		Block:    block,
	}).Result()
	if err == redis.Nil {
		return
	}
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		log.Printf("[persist] xreadgroup (%s): %v", streamID, err)
		time.Sleep(time.Second)
		return
	}
	for _, s := range streams {
		for _, msg := range s.Messages {
			c.processMessage(ctx, msg)
		}
	}
}

func (c *Consumer) reclaimStale(ctx context.Context) {
	if ctx.Err() != nil {
		return
	}
	minIdleMs := ReclaimMinIdle.Milliseconds()
	if minIdleMs < 1 {
		minIdleMs = 1
	}
	start := "0-0"
	for {
		if ctx.Err() != nil {
			return
		}
		next, msgs, err := c.autoClaim(ctx, start, minIdleMs, 10)
		if err != nil {
			if ctx.Err() == nil {
				log.Printf("[persist] xautoclaim: %v", err)
			}
			return
		}
		for _, msg := range msgs {
			c.processMessage(ctx, msg)
		}
		if next == "0-0" || len(msgs) == 0 {
			return
		}
		start = next
	}
}

func (c *Consumer) autoClaim(ctx context.Context, start string, minIdleMs int64, count int) (string, []redis.XMessage, error) {
	raw, err := c.redis.Do(ctx, "XAUTOCLAIM", StreamKey, ConsumerGroup, c.name, minIdleMs, start, "COUNT", count).Result()
	if err != nil {
		return start, nil, err
	}
	return parseXAutoClaimResult(raw)
}

func parseXAutoClaimResult(raw interface{}) (string, []redis.XMessage, error) {
	parts, ok := raw.([]interface{})
	if !ok || len(parts) < 2 {
		return "0-0", nil, fmt.Errorf("unexpected xautoclaim reply type")
	}
	next, _ := parts[0].(string)
	if next == "" {
		next = "0-0"
	}
	msgs, err := parseXStreamMessages(parts[1])
	return next, msgs, err
}

func parseXStreamMessages(raw interface{}) ([]redis.XMessage, error) {
	entries, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected stream message list")
	}
	out := make([]redis.XMessage, 0, len(entries))
	for _, entry := range entries {
		pair, ok := entry.([]interface{})
		if !ok || len(pair) < 2 {
			continue
		}
		id, _ := pair[0].(string)
		fieldsRaw, ok := pair[1].([]interface{})
		if !ok {
			continue
		}
		values := map[string]interface{}{}
		for i := 0; i+1 < len(fieldsRaw); i += 2 {
			k, _ := fieldsRaw[i].(string)
			values[k] = fieldsRaw[i+1]
		}
		out = append(out, redis.XMessage{ID: id, Values: values})
	}
	return out, nil
}

func (c *Consumer) processMessage(ctx context.Context, msg redis.XMessage) {
	c.inflight.Add(1)
	defer c.inflight.Done()

	if err := c.handleMessage(ctx, msg); err != nil {
		log.Printf("[persist] handle %s: %v", msg.ID, err)
		return
	}
	if err := c.redis.XAck(ctx, StreamKey, ConsumerGroup, msg.ID).Err(); err != nil {
		log.Printf("[persist] xack %s: %v", msg.ID, err)
	}
}

func (c *Consumer) handleMessage(ctx context.Context, msg redis.XMessage) error {
	tidStr := msg.Values["ticket_id"]
	uidStr := msg.Values["user_id"]
	ticketID, err := strconv.Atoi(fmt.Sprint(tidStr))
	if err != nil {
		return fmt.Errorf("ticket_id: %w", err)
	}
	userID, err := strconv.Atoi(fmt.Sprint(uidStr))
	if err != nil {
		return fmt.Errorf("user_id: %w", err)
	}
	if MessagePause != nil {
		if err := MessagePause(ctx, ticketID); err != nil {
			return err
		}
	}
	soldAt := time.Now()
	res := c.db.Exec(
		`UPDATE tickets SET user_id = ?, sold_at = ?, state = 'sold' WHERE id = ? AND state = 'available'`,
		userID, soldAt, ticketID,
	)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		var row TicketRow
		if err := c.db.Raw(`SELECT id, user_id, state FROM tickets WHERE id = ?`, ticketID).Scan(&row).Error; err != nil {
			return err
		}
		if row.State == "sold" && row.UserID != nil && *row.UserID == userID {
			return nil
		}
		return fmt.Errorf("ticket %d not available (state=%s)", ticketID, row.State)
	}
	return nil
}

// EnqueueReplay adds a persist job when idempotent HTTP replay needs MySQL catch-up.
func EnqueueReplay(ctx context.Context, rdb *redis.Client, ticketID, userID int) error {
	return rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: StreamKey,
		Values: map[string]interface{}{
			"ticket_id": ticketID,
			"user_id":   userID,
		},
	}).Err()
}

// IsSold returns whether the ticket is already persisted as sold.
func IsSold(db *gorm.DB, ticketID int) (bool, error) {
	var state string
	err := db.Raw(`SELECT state FROM tickets WHERE id = ?`, ticketID).Scan(&state).Error
	if err != nil {
		return false, err
	}
	if state == "" {
		return false, nil
	}
	return state == "sold", nil
}
