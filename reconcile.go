package main

import (
	"context"

	"ticketengine/internal/reconcile"
)

// RunReconciliation compares Redis and MySQL booking state (read-only).
func RunReconciliation(ctx context.Context) (*reconcile.Report, error) {
	return reconcile.Run(ctx, db, redisClient)
}
