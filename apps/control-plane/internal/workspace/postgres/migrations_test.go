package postgres

import (
	"bytes"
	"context"
	"io/fs"
	"testing"
)

func TestMigrateRejectsUnsupportedDirectionBeforeUsingConnection(t *testing.T) {
	for _, direction := range []string{"down", "sideways", ""} {
		if err := Migrate(context.Background(), nil, direction); err == nil {
			t.Fatalf("Migrate direction %q succeeded", direction)
		}
	}
}

func TestEmbeddedMigrationsAreCanonicalOrderedFiles(t *testing.T) {
	migrationFiles, err := loadMigrationFiles()
	if err != nil {
		t.Fatal(err)
	}
	entries, err := fs.ReadDir(migrationFiles, ".")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"001_workspace.sql", "002_workspace_installation_release.sql"}
	if len(entries) != len(want) {
		t.Fatalf("embedded migration count = %d, want %d", len(entries), len(want))
	}
	for index, entry := range entries {
		if entry.IsDir() || entry.Name() != want[index] {
			t.Fatalf("embedded migration %d = %q, want %q", index, entry.Name(), want[index])
		}
		data, err := fs.ReadFile(migrationFiles, entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(data, []byte("---- create above / drop below ----")) {
			t.Fatalf("embedded migration %s lacks the forward/backward boundary", entry.Name())
		}
	}
}
