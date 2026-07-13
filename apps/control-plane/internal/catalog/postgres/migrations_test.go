package postgres

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmbeddedMigrationMatchesOwnedSQLFile(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "migrations", "001_catalog.sql"))
	if err != nil {
		t.Fatalf("read owned migration: %v", err)
	}
	want := strings.ReplaceAll(string(data), "\r\n", "\n")
	got := strings.ReplaceAll(migration001, "\r\n", "\n")
	if got != want {
		t.Fatal("embedded migration differs from apps/control-plane/migrations/001_catalog.sql")
	}
}
