package ledger

import (
	"bytes"
	"context"
	"errors"
	"io/fs"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestCheckSchemaRequiresDatabaseAndPropagatesQueryFailure(t *testing.T) {
	if err := CheckSchema(context.Background(), nil); err == nil {
		t.Fatal("nil readiness database accepted")
	}
	queryErr := errors.New("database unavailable")
	if err := CheckSchema(context.Background(), rowQuerierStub{row: scanRow{err: queryErr}}); !errors.Is(err, queryErr) {
		t.Fatalf("query error = %v, want %v", err, queryErr)
	}
}

func TestCheckSchemaAcceptsExactShapeAndRejectsMismatch(t *testing.T) {
	ready := []any{int32(ExpectedSchemaVersion), 17, 21, 17, 21, true, true, true, true}
	if err := CheckSchema(context.Background(), rowQuerierStub{row: scanRow{values: ready}}); err != nil {
		t.Fatalf("exact schema rejected: %v", err)
	}
	weakened := append([]any(nil), ready...)
	weakened[7] = false
	if err := CheckSchema(context.Background(), rowQuerierStub{row: scanRow{values: weakened}}); !errors.Is(err, ErrSchemaVersionMismatch) {
		t.Fatalf("weakened schema error = %v", err)
	}
}

func TestMigrateRequiresConnection(t *testing.T) {
	if err := Migrate(context.Background(), nil, "up"); err == nil {
		t.Fatal("nil migration connection accepted")
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
	want := []string{"001_ledger.sql", "002_release_provenance.sql"}
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

type rowQuerierStub struct{ row pgx.Row }

func (stub rowQuerierStub) QueryRow(context.Context, string, ...any) pgx.Row { return stub.row }

type scanRow struct {
	values []any
	err    error
}

func (row scanRow) Scan(dest ...any) error {
	if row.err != nil {
		return row.err
	}
	return (valueScanner{values: row.values}).Scan(dest...)
}
