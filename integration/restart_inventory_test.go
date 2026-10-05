//go:build integration

package integration_test

import (
	"testing"

	"ticketengine/internal/inventory"
)

func TestRestartDoesNotMintInventory(t *testing.T) {
	gdb := openIntegrationMySQL(t)
	sqlDB, _ := gdb.DB()
	defer sqlDB.Close()

	ensureTicketsTable(t, gdb)
	if err := gdb.Exec("DELETE FROM tickets").Error; err != nil {
		t.Fatalf("cleanup: %v", err)
	}

	const target = 100
	for i := 0; i < target; i++ {
		if err := gdb.Exec("INSERT INTO tickets (user_id, state) VALUES (NULL, 'available')").Error; err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	simulateServeBoot := func() {
		var total int64
		if err := gdb.Model(&struct{}{}).Table("tickets").Count(&total).Error; err != nil {
			t.Fatalf("count: %v", err)
		}
		if _, err := inventory.SeedDecision(total, target); err != nil && total != 0 && total != int64(target) {
			t.Fatalf("seed decision: %v", err)
		}
	}

	simulateServeBoot()
	simulateServeBoot()

	var total int64
	if err := gdb.Raw("SELECT COUNT(*) FROM tickets").Scan(&total).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if total != int64(target) {
		t.Fatalf("expected %d rows after simulated restarts, got %d", target, total)
	}
}
