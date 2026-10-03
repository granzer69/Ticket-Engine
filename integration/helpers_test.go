//go:build integration

package integration_test

import (
	"fmt"
	"os"
	"testing"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func integrationDSN() string {
	host := envDefault("MYSQL_HOST", "127.0.0.1:3306")
	user := envDefault("MYSQL_USER", "ticket")
	pass := envDefault("MYSQL_PASSWORD", "ticket")
	name := integrationDatabase()
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, pass, host, name)
}

func integrationDatabase() string {
	if v := os.Getenv("TICKET_INTEGRATION_DATABASE"); v != "" {
		return v
	}
	return envDefault("MYSQL_DATABASE", "ticketdb")
}

func envDefault(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func openIntegrationMySQL(t *testing.T) *gorm.DB {
	rootHost := envDefault("MYSQL_HOST", "127.0.0.1:3306")
	user := envDefault("MYSQL_USER", "ticket")
	pass := envDefault("MYSQL_PASSWORD", "ticket")
	dbName := integrationDatabase()

	rootDSN := fmt.Sprintf("%s:%s@tcp(%s)/?charset=utf8mb4&parseTime=True&loc=Local", user, pass, rootHost)
	root, err := gorm.Open(mysql.Open(rootDSN), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Skipf("mysql not available: %v", err)
	}
	if err := root.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", dbName)).Error; err != nil {
		// Existing volumes may not grant CREATE DATABASE to ticket user; fall back to primary DB.
		dbName = envDefault("MYSQL_DATABASE", "ticketdb")
	}
	sqlRoot, _ := root.DB()
	if sqlRoot != nil {
		sqlRoot.Close()
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		user, pass, rootHost, dbName)
	gdb, err := gorm.Open(mysql.Open(dsn), &gorm.Config{SkipDefaultTransaction: true})
	if err != nil {
		t.Fatalf("open integration db: %v", err)
	}
	return gdb
}

func ensureTicketsTable(t *testing.T, gdb *gorm.DB) {
	if err := gdb.Exec(`CREATE TABLE IF NOT EXISTS tickets (
		id BIGINT AUTO_INCREMENT PRIMARY KEY,
		user_id BIGINT NOT NULL DEFAULT 0,
		created_at BIGINT NULL,
		sold_at DATETIME(3) NULL,
		state VARCHAR(16) NOT NULL DEFAULT 'available',
		INDEX idx_tickets_user_id (user_id),
		INDEX idx_tickets_state (state)
	)`).Error; err != nil {
		t.Fatalf("ensure table: %v", err)
	}
}
