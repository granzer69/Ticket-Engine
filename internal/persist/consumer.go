package persist

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
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

// Consumer reads the bookings stream and applies MySQL updates.
type Consumer struct {
	redis *redis.Client
	db    *gorm.DB
	name  string
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

func (c *Consumer) Run(ctx context.Context) {
	reclaimTicker := time.NewTicker(5 * time.Second)
	defer reclaimTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-reclaimTicker.C:
			c.reclaimPending(ctx)
		default:
			c.readAndProcess(ctx, ">")
		}
	}
}

func (c *Consumer) reclaimPending(ctx context.Context) {
	ext, err := c.redis.XPendingExt(ctx, &redis.XPendingExtArgs{
		Stream: StreamKey,
		Group:  ConsumerGroup,
		Start:  "-",
		End:    "+",
		Count:  10,
	}).Result()
	if err != nil && err != redis.Nil {
		log.Printf("[persist] xpendingext: %v", err)
		return
	}
	var ids []string
	for _, p := range ext {
		if p.Idle >= PendingMinIdle {
			ids = append(ids, p.ID)
		}
	}
	if len(ids) == 0 {
		return
	}
	msgs, err := c.redis.XClaim(ctx, &redis.XClaimArgs{
		Stream:   StreamKey,
		Group:    ConsumerGroup,
		Consumer: c.name,
		MinIdle:  PendingMinIdle,
		Messages: ids,
	}).Result()
	if err != nil && err != redis.Nil {
		log.Printf("[persist] xclaim: %v", err)
		return
	}
	for _, msg := range msgs {
		c.processMessage(ctx, msg)
	}
}

func (c *Consumer) readAndProcess(ctx context.Context, streamID string) {
	streams, err := c.redis.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    ConsumerGroup,
		Consumer: c.name,
		Streams:  []string{StreamKey, streamID},
		Count:    10,
		Block:    time.Second * 2,
	}).Result()
	if err == redis.Nil {
		return
	}
	if err != nil {
		log.Printf("[persist] xreadgroup: %v", err)
		time.Sleep(time.Second)
		return
	}
	for _, s := range streams {
		for _, msg := range s.Messages {
			c.processMessage(ctx, msg)
		}
	}
}

func (c *Consumer) processMessage(ctx context.Context, msg redis.XMessage) {
	if err := c.handleMessage(ctx, msg); err != nil {
		if isPermanentError(err) {
			c.moveToDLQ(ctx, msg, err)
			_ = c.redis.XAck(ctx, StreamKey, ConsumerGroup, msg.ID).Err()
			return
		}
		attempts, incErr := c.redis.Incr(ctx, retryKey(msg.ID)).Result()
		if incErr == nil {
			c.redis.Expire(ctx, retryKey(msg.ID), 24*time.Hour)
		}
		if attempts >= int64(MaxHandleAttempts) {
			log.Printf("[persist] max attempts for %s: %v", msg.ID, err)
			c.moveToDLQ(ctx, msg, err)
			_ = c.redis.XAck(ctx, StreamKey, ConsumerGroup, msg.ID).Err()
			_ = c.redis.Del(ctx, retryKey(msg.ID)).Err()
			return
		}
		log.Printf("[persist] handle %s (attempt %d): %v", msg.ID, attempts, err)
		return
	}
	if err := c.redis.XAck(ctx, StreamKey, ConsumerGroup, msg.ID).Err(); err != nil {
		log.Printf("[persist] xack %s: %v", msg.ID, err)
	}
	_ = c.redis.Del(ctx, retryKey(msg.ID)).Err()
}

func retryKey(msgID string) string {
	return "persist:retry:" + msgID
}

func isPermanentError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "ticket_id:") || strings.Contains(s, "user_id:")
}

func (c *Consumer) moveToDLQ(ctx context.Context, msg redis.XMessage, cause error) {
	values := map[string]interface{}{
		"source_id": msg.ID,
		"error":     cause.Error(),
	}
	for k, v := range msg.Values {
		values[k] = v
	}
	if err := c.redis.XAdd(ctx, &redis.XAddArgs{Stream: DLQStreamKey, Values: values}).Err(); err != nil {
		log.Printf("[persist] dlq add: %v", err)
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
