package main

import (
	"context"
	"fmt"
	"log"

	"ticketengine/internal/inventory"
)

// SeedInventory creates the initial ticket rows when the database is empty.
// It never tops up partial inventory (restart-safe).
func SeedInventory() error {
	var total int64
	if err := db.Model(&Ticket{}).Count(&total).Error; err != nil {
		return fmt.Errorf("seed: count tickets: %w", err)
	}

	action, err := inventory.SeedDecision(total, ticketCount)
	if err != nil {
		return err
	}
	if action == "noop" {
		log.Printf("seed: %d tickets already present, skipping insert", total)
		return syncRedisQueue(true)
	}

	log.Printf("seed: inserting %d tickets into MySQL...", ticketCount)
	var batch []Ticket
	for i := 0; i < ticketCount; i++ {
		batch = append(batch, Ticket{UserID: dummyUser})
	}
	if err := db.CreateInBatches(batch, 1000).Error; err != nil {
		return fmt.Errorf("seed: insert: %w", err)
	}
	return syncRedisQueue(true)
}

// syncRedisQueue aligns the Redis list with MySQL-available tickets.
// When forceRebuild is true, the list key is replaced (used after seed).
// When forceRebuild is false, an empty list is loaded from MySQL; mismatches only warn unless TICKET_ENGINE_REDIS_SYNC=1.
func syncRedisQueue(forceRebuild bool) error {
	ctx := context.Background()

	var tickets []Ticket
	if err := db.Where("user_id = ?", dummyUser).Find(&tickets).Error; err != nil {
		return fmt.Errorf("redis sync: load available: %w", err)
	}

	redisLen, err := redisClient.LLen(ctx, queueTicket).Result()
	if err != nil {
		return fmt.Errorf("redis sync: llen: %w", err)
	}

	available := int64(len(tickets))
	syncEnv := getEnv("TICKET_ENGINE_REDIS_SYNC", "") == "1"

	switch {
	case redisLen == 0 && available > 0:
		log.Printf("redis sync: queue empty, loading %d available ticket ids", available)
		return pushTicketIDs(ctx, tickets)
	case forceRebuild:
		log.Printf("redis sync: rebuilding queue with %d available ticket ids", available)
		if err := redisClient.Del(ctx, queueTicket).Err(); err != nil {
			return fmt.Errorf("redis sync: del queue: %w", err)
		}
		return pushTicketIDs(ctx, tickets)
	case redisLen != available:
		msg := fmt.Sprintf("redis sync: queue length %d != MySQL available %d", redisLen, available)
		if syncEnv {
			log.Printf("%s; TICKET_ENGINE_REDIS_SYNC=1 rebuilding queue (user hash preserved)", msg)
			if err := redisClient.Del(ctx, queueTicket).Err(); err != nil {
				return fmt.Errorf("redis sync: del queue: %w", err)
			}
			return pushTicketIDs(ctx, tickets)
		}
		log.Printf("WARN: %s; set TICKET_ENGINE_REDIS_SYNC=1 to rebuild from MySQL", msg)
	default:
		log.Printf("redis sync: queue length %d matches MySQL available tickets", redisLen)
	}
	return nil
}

func pushTicketIDs(ctx context.Context, tickets []Ticket) error {
	if len(tickets) == 0 {
		return nil
	}
	var ids []interface{}
	for _, t := range tickets {
		ids = append(ids, t.ID)
	}
	for i := 0; i < len(ids); i += 1000 {
		end := i + 1000
		if end > len(ids) {
			end = len(ids)
		}
		if err := redisClient.LPush(ctx, queueTicket, ids[i:end]...).Err(); err != nil {
			return fmt.Errorf("redis sync: lpush: %w", err)
		}
	}
	return nil
}

// prepareRuntimeData runs on normal server start (no inventory insert).
func prepareRuntimeData() {
	if err := syncRedisQueue(false); err != nil {
		log.Fatalf("prepare runtime data: %v", err)
	}
}
