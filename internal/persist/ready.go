package persist

import (
	"context"

	"github.com/go-redis/redis/v8"
)

// GroupPendingCountRDB returns pending entries for the bookings consumer group.
func GroupPendingCountRDB(ctx context.Context, rdb *redis.Client) (int64, error) {
	pending, err := rdb.XPending(ctx, StreamKey, ConsumerGroup).Result()
	if err != nil {
		return 0, err
	}
	return pending.Count, nil
}
