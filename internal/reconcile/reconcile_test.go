package reconcile

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"ticketengine/internal/booking"
	"ticketengine/internal/persist"
)

type ticketRow struct {
	ID     int `gorm:"primaryKey"`
	UserID *int
	State  string `gorm:"size:16;default:available"`
}

func (ticketRow) TableName() string { return "tickets" }

var reclaimMinIdleMu sync.Mutex

func withReclaimMinIdle(t *testing.T, d time.Duration) {
	reclaimMinIdleMu.Lock()
	old := persist.ReclaimMinIdle
	persist.ReclaimMinIdle = d
	t.Cleanup(func() {
		persist.ReclaimMinIdle = old
		reclaimMinIdleMu.Unlock()
	})
}

func setupReconcileTest(t *testing.T) (*gorm.DB, *redis.Client, *miniredis.Miniredis) {
	dsn := fmt.Sprintf("file:reconcile_%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("sqlite: %v", err)
	}
	if err := db.AutoMigrate(&ticketRow{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return db, rdb, mr
}

func seedAlignedInventory(ctx context.Context, db *gorm.DB, rdb *redis.Client, ids ...int) {
	for _, id := range ids {
		if err := db.Create(&ticketRow{ID: id, State: "available"}).Error; err != nil {
			panic(err)
		}
		if err := rdb.RPush(ctx, booking.KeyQueue, id).Err(); err != nil {
			panic(err)
		}
	}
}

func TestReconcile(t *testing.T) {
	ctx := context.Background()
	db, rdb, _ := setupReconcileTest(t)
	seedAlignedInventory(ctx, db, rdb, 1, 2, 3)

	rep, err := Run(ctx, db, rdb)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if rep.HasDrift() {
		t.Fatalf("expected clean: %s", rep.Format())
	}
	if rep.ExitCode() != ExitClean {
		t.Fatalf("exit code %d", rep.ExitCode())
	}
	if !strings.Contains(rep.Format(), "RECONCILE RESULT: CLEAN") {
		t.Fatalf("output: %s", rep.Format())
	}
}

func TestReconcileInventoryDrift(t *testing.T) {
	ctx := context.Background()
	db, rdb, _ := setupReconcileTest(t)
	seedAlignedInventory(ctx, db, rdb, 1, 2)
	_ = rdb.RPush(ctx, booking.KeyQueue, 99)

	rep, err := Run(ctx, db, rdb)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !rep.HasDrift() {
		t.Fatal("expected inventory drift")
	}
	if !strings.Contains(rep.Format(), "inventory") {
		t.Fatalf("output: %s", rep.Format())
	}
}

func TestReconcileDuplicateOwnership(t *testing.T) {
	ctx := context.Background()
	db, rdb, _ := setupReconcileTest(t)
	uid := 42
	if err := db.Create(&ticketRow{ID: 1, State: "sold", UserID: &uid}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&ticketRow{ID: 2, State: "sold", UserID: &uid}).Error; err != nil {
		t.Fatal(err)
	}
	_ = rdb.HSet(ctx, booking.KeyUserBooking, "10", "1", "11", "1")

	rep, err := Run(ctx, db, rdb)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !rep.HasDrift() {
		t.Fatal("expected duplicate drift")
	}
}

func TestReconcileOrphanAllocation(t *testing.T) {
	ctx := context.Background()
	db, rdb, _ := setupReconcileTest(t)
	if err := db.Create(&ticketRow{ID: 5, State: "available"}).Error; err != nil {
		t.Fatal(err)
	}
	_ = rdb.HSet(ctx, booking.KeyUserBooking, "7", "5")

	rep, err := Run(ctx, db, rdb)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if !rep.HasDrift() {
		t.Fatalf("expected orphan drift: %s", rep.Format())
	}
}

func TestReconcileLegitimatePendingNotFalsePositive(t *testing.T) {
	withReclaimMinIdle(t, time.Hour)
	ctx := context.Background()
	db, rdb, _ := setupReconcileTest(t)
	if err := db.Create(&ticketRow{ID: 8, State: "available"}).Error; err != nil {
		t.Fatal(err)
	}
	_ = rdb.HSet(ctx, booking.KeyUserBooking, "99", "8")
	_ = rdb.XGroupCreateMkStream(ctx, persist.StreamKey, persist.ConsumerGroup, "0")
	_, err := rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: persist.StreamKey,
		Values: map[string]interface{}{"ticket_id": 8, "user_id": 99},
	}).Result()
	if err != nil {
		t.Fatal(err)
	}
	_, err = rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    persist.ConsumerGroup,
		Consumer: "reconcile-test",
		Streams:  []string{persist.StreamKey, ">"},
		Count:    1,
	}).Result()
	if err != nil {
		t.Fatal(err)
	}

	rep, err := Run(ctx, db, rdb)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if rep.HasDrift() {
		t.Fatalf("expected clean during expected pending: %s", rep.Format())
	}
}

func TestReconcileRedisUnavailable(t *testing.T) {
	ctx := context.Background()
	db, rdb, mr := setupReconcileTest(t)
	seedAlignedInventory(ctx, db, rdb, 1)
	mr.Close()

	_, err := Run(ctx, db, rdb)
	if err == nil {
		t.Fatal("expected operational error")
	}
}

func TestReconcileMySQLUnavailable(t *testing.T) {
	ctx := context.Background()
	db, rdb, _ := setupReconcileTest(t)
	sqlDB, _ := db.DB()
	sqlDB.Close()

	_, err := Run(ctx, db, rdb)
	if err == nil {
		t.Fatal("expected operational error")
	}
}

func TestReconcileReadOnlyGuarantee(t *testing.T) {
	ctx := context.Background()
	db, rdb, _ := setupReconcileTest(t)
	seedAlignedInventory(ctx, db, rdb, 1, 2)
	before := snapshotState(ctx, t, db, rdb)

	_, err := Run(ctx, db, rdb)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	after := snapshotState(ctx, t, db, rdb)
	if before != after {
		t.Fatalf("state changed:\nbefore=%s\nafter=%s", before, after)
	}
}

func TestClassifyPendingDisposition(t *testing.T) {
	withReclaimMinIdle(t, 2*time.Minute)
	if classifyPending(0, persist.MaxDeliveryAttempts) != PendingSuspicious {
		t.Fatal("expected suspicious on high retries")
	}
	if classifyPending(persist.ReclaimMinIdle, 0) != PendingStale {
		t.Fatal("expected stale on long idle")
	}
	if classifyPending(0, 0) != PendingExpected {
		t.Fatal("expected expected on fresh pending")
	}
}

func snapshotState(ctx context.Context, t *testing.T, db *gorm.DB, rdb *redis.Client) string {
	var count int64
	if err := db.Model(&ticketRow{}).Count(&count).Error; err != nil {
		t.Fatalf("count mysql: %v", err)
	}
	q, _ := rdb.LLen(ctx, booking.KeyQueue).Result()
	h, _ := rdb.HLen(ctx, booking.KeyUserBooking).Result()
	s, _ := rdb.XLen(ctx, persist.StreamKey).Result()
	d, _ := rdb.XLen(ctx, persist.DLQStreamKey).Result()
	return strings.Join([]string{
		"mysql:" + fmt.Sprint(count),
		"q:" + fmt.Sprint(q),
		"h:" + fmt.Sprint(h),
		"s:" + fmt.Sprint(s),
		"d:" + fmt.Sprint(d),
	}, "|")
}
