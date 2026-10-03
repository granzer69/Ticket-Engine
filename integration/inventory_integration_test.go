//go:build integration

package integration_test

import (
	"fmt"
	"os"
	"testing"

	"ticketengine/internal/inventory"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func integrationDSN() string {
	host := envDefault("MYSQL_HOST", "127.0.0.1:3306")
	user := envDefault("MYSQL_USER", "ticket")
	pass := envDefault("MYSQL_PASSWORD", "ticket")
	name := envDefault("MYSQL_DATABASE", "ticketdb")
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, pass, host, name)
}

func envDefault(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func TestPartialInventoryRefused(t *testing.T) {
	gdb, err := gorm.Open(mysql.Open(integrationDSN()), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Skipf("mysql not available: %v", err)
	}
	sqlDB, _ := gdb.DB()
	defer sqlDB.Close()

	if err := gdb.Exec(`CREATE TABLE IF NOT EXISTS tickets (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		user_id BIGINT NOT NULL DEFAULT 0,
		created_at BIGINT NULL,
		sold_at DATETIME(3) NULL
	)`).Error; err != nil {
		t.Fatalf("ensure table: %v", err)
	}
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
	_, err = inventory.SeedDecision(total, 15000)
	if err == nil {
		t.Fatal("expected refusal to top up partial inventory")
	}
}
