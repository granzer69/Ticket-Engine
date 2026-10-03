package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

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
		stmts := splitSQLStatements(string(body))
		for _, stmt := range stmts {
			if _, err := sqlDB.Exec(stmt); err != nil {
				log.Fatalf("migrations: exec %s: %v", e.Name(), err)
			}
		}
		log.Printf("applied migration %s", e.Name())
	}
}

func splitSQLStatements(sql string) []string {
	var out []string
	for _, part := range strings.Split(sql, ";") {
		s := strings.TrimSpace(part)
		if s == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}

