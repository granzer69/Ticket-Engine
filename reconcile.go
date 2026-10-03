package main

import (
	"context"
	"fmt"
	"log"

	"ticketengine/internal/booking"
)

// RunReconciliation compares Redis queue length with MySQL available tickets and logs drift.
func RunReconciliation(ctx context.Context) error {
	var available int64
	if err := db.Model(&Ticket{}).Where("state = ?", "available").Count(&available).Error; err != nil {
		return err
	}
	redisLen, err := redisClient.LLen(ctx, booking.KeyQueue).Result()
	if err != nil {
		return err
	}
	if redisLen != available {
		log.Printf("WARN: reconcile drift redis=%d mysql_available=%d", redisLen, available)
		if getEnv("TICKET_ENGINE_REDIS_SYNC", "") == "1" {
			return syncRedisQueue(true)
		}
		return fmt.Errorf("inventory drift detected (set TICKET_ENGINE_REDIS_SYNC=1 to rebuild queue)")
	}
	log.Printf("reconcile ok: redis=%d mysql_available=%d", redisLen, available)
	return nil
}
