//go:build integration

package integration_test

import (
	"testing"

	"ticketengine/internal/inventory"
)

func TestPartialInventoryRefused(t *testing.T) {
	gdb := openIntegrationMySQL(t)
	sqlDB, _ := gdb.DB()
	defer sqlDB.Close()

	ensureTicketsTable(t, gdb)
	if err := gdb.Exec("DELETE FROM tickets").Error; err != nil {
		t.Fatalf("cleanup: %v", err)
	}
	if err := gdb.Exec("INSERT INTO tickets (user_id) VALUES (0)").Error; err != nil {
		t.Fatalf("insert: %v", err)
	}
	var total int64
	if err := gdb.Raw("SELECT COUNT(*) FROM tickets").Scan(&total).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	_, seedErr := inventory.SeedDecision(total, 15000)
	if seedErr == nil {
		t.Fatal("expected refusal to top up partial inventory")
	}
}
