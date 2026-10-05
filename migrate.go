package main

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func migrationColumnExists(sqlDB *sql.DB, table, column string) bool {
	var count int
	err := sqlDB.QueryRow(
		`SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ? AND COLUMN_NAME = ?`,
		table, column,
	).Scan(&count)
	return err == nil && count > 0
}

func applyMigrations() {
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("migrations: db handle: %v", err)
	}

	path := getEnv("TICKET_ENGINE_MIGRATIONS_DIR", "migrations")
	entries, err := os.ReadDir(path)
	if err != nil {
		log.Fatalf("migrations: read dir %s: %v", path, err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		full := filepath.Join(path, e.Name())
		body, err := os.ReadFile(full)
		if err != nil {
			log.Fatalf("migrations: read %s: %v", full, err)
		}
		if e.Name() == "002_ticket_state.sql" && migrationColumnExists(sqlDB, "tickets", "state") {
			if err := apply002IndexUpgrade(sqlDB); err != nil {
				log.Fatalf("migrations: exec %s index upgrade: %v", e.Name(), err)
			}
			log.Printf("skip migration %s (state column exists)", e.Name())
			continue
		}
		stmts := splitSQLStatements(string(body))
		for _, stmt := range stmts {
			if _, err := sqlDB.Exec(stmt); err != nil {
				log.Fatalf("migrations: exec %s: %v", e.Name(), err)
			}
		}
		log.Printf("applied migration %s", e.Name())
	}
}

func apply002IndexUpgrade(sqlDB *sql.DB) error {
	// Idempotent fix when 002 partially applied (state added but unique index step failed).
	if _, err := sqlDB.Exec(`ALTER TABLE tickets MODIFY user_id BIGINT NULL`); err != nil {
		return err
	}
	if _, err := sqlDB.Exec(`DROP INDEX idx_tickets_user_id ON tickets`); err != nil {
		if !strings.Contains(err.Error(), "1091") && !strings.Contains(strings.ToLower(err.Error()), "check that it exists") {
			return err
		}
	}
	_, err := sqlDB.Exec(`CREATE UNIQUE INDEX idx_tickets_user_id ON tickets (user_id)`)
	if err != nil && strings.Contains(err.Error(), "1061") {
		return nil
	}
	return err
}

func splitSQLStatements(sql string) []string {
	var b strings.Builder
	for _, line := range strings.Split(sql, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	var out []string
	for _, part := range strings.Split(b.String(), ";") {
		s := strings.TrimSpace(part)
		if s == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}

