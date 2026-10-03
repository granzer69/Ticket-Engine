package main

import (
	"strings"
	"testing"
)

func TestSplitSQLStatementsIgnoresCommentSemicolons(t *testing.T) {
	sql := `-- header; with semicolon in comment
CREATE TABLE IF NOT EXISTS t (id INT);
`
	stmts := splitSQLStatements(sql)
	if len(stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d: %v", len(stmts), stmts)
	}
	if !strings.Contains(stmts[0], "CREATE TABLE") {
		t.Fatalf("unexpected stmt: %q", stmts[0])
	}
}
