//go:build integration

package postgres

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Nene7ko/NeKiro/apps/control-plane/internal/workspace"
	"github.com/Nene7ko/NeKiro/contracts"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestInstallationInspectionStoreReturnsCompleteCurrentHistoryAndKeysetPages(t *testing.T) {
	ctx := context.Background()
	store, pool := inspectionStoreForTest(t, ctx)
	workspaceValue := contracts.Workspace{WorkspaceID: "workspace-inspection", OwnerID: "owner-a", CreatedAt: time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC)}
	if _, err := store.CreateWorkspace(ctx, workspaceValue); err != nil {
		t.Fatalf("create Workspace: %v", err)
	}
	installedAt := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)
	historicalAt := installedAt.Add(2 * time.Minute)
	rows := []contracts.Installation{
		{InstallationID: "installation-c", WorkspaceID: workspaceValue.WorkspaceID, AgentID: "runtime-c", VersionConstraint: "^1.0.0", InstalledVersion: "1.0.2", AcceptedPermissions: []string{"document.read", "document.write"}, Status: "enabled", InstalledAt: installedAt, UpdatedAt: installedAt},
		{InstallationID: "installation-a", WorkspaceID: workspaceValue.WorkspaceID, AgentID: "runtime-a", VersionConstraint: "~2.0.0", InstalledVersion: "2.0.4", AcceptedPermissions: []string{"document.read"}, Status: "uninstalled", InstalledAt: installedAt, UpdatedAt: historicalAt, UninstalledAt: &historicalAt},
		{InstallationID: "installation-b", WorkspaceID: workspaceValue.WorkspaceID, AgentID: "runtime-b", VersionConstraint: "^3.0.0", InstalledVersion: "3.0.1", AcceptedPermissions: []string{}, Status: "disabled", InstalledAt: installedAt, UpdatedAt: installedAt.Add(time.Minute)},
		{InstallationID: "installation-d", WorkspaceID: workspaceValue.WorkspaceID, AgentID: "runtime-d", VersionConstraint: "^4.0.0", InstalledVersion: "4.0.0", AcceptedPermissions: []string{"document.read"}, Status: "enabled", InstalledAt: installedAt.Add(time.Hour), UpdatedAt: installedAt.Add(time.Hour)},
	}
	for _, row := range rows {
		insertInspectionRow(t, ctx, pool, row)
	}

	actual, err := store.GetInstallation(ctx, workspaceValue.WorkspaceID, "installation-a")
	if err != nil {
		t.Fatalf("read historical Installation: %v", err)
	}
	if !reflect.DeepEqual(actual, rows[1]) {
		t.Fatalf("historical Installation = %#v, want %#v", actual, rows[1])
	}

	first, hasMore, err := store.ListInstallations(ctx, workspaceValue.WorkspaceID, 2, nil)
	if err != nil || !hasMore {
		t.Fatalf("first page = %#v, hasMore=%v, err=%v", first, hasMore, err)
	}
	assertInspectionIDs(t, first, []string{"installation-a", "installation-b"})
	second, hasMore, err := store.ListInstallations(ctx, workspaceValue.WorkspaceID, 2, &workspace.InstallationPosition{InstalledAt: first[1].InstalledAt, InstallationID: first[1].InstallationID})
	if err != nil || hasMore {
		t.Fatalf("second page = %#v, hasMore=%v, err=%v", second, hasMore, err)
	}
	assertInspectionIDs(t, second, []string{"installation-c", "installation-d"})
}

func TestInstallationInspectionStoreReturnsNonNilEmptyHistory(t *testing.T) {
	ctx := context.Background()
	store, _ := inspectionStoreForTest(t, ctx)
	workspaceValue := contracts.Workspace{WorkspaceID: "workspace-empty-inspection", OwnerID: "owner-a", CreatedAt: time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC)}
	if _, err := store.CreateWorkspace(ctx, workspaceValue); err != nil {
		t.Fatalf("create Workspace: %v", err)
	}
	items, hasMore, err := store.ListInstallations(ctx, workspaceValue.WorkspaceID, 25, nil)
	if err != nil {
		t.Fatalf("list empty history: %v", err)
	}
	if items == nil || len(items) != 0 || hasMore {
		t.Fatalf("empty history = %#v, hasMore=%v", items, hasMore)
	}
}

func TestInstallationInspectionStorePropagatesQueryAndScanFailures(t *testing.T) {
	ctx := context.Background()
	store, pool := inspectionStoreForTest(t, ctx)
	workspaceValue := contracts.Workspace{WorkspaceID: "workspace-failure-inspection", OwnerID: "owner-a", CreatedAt: time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC)}
	if _, err := store.CreateWorkspace(ctx, workspaceValue); err != nil {
		t.Fatalf("create Workspace: %v", err)
	}
	row := contracts.Installation{InstallationID: "installation-failure", WorkspaceID: workspaceValue.WorkspaceID, AgentID: "runtime-failure", VersionConstraint: "^1.0.0", InstalledVersion: "1.0.0", AcceptedPermissions: []string{"document.read"}, Status: "enabled", InstalledAt: time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)}
	insertInspectionRow(t, ctx, pool, row)

	canceled, cancel := context.WithCancel(ctx)
	cancel()
	items, _, err := store.ListInstallations(canceled, workspaceValue.WorkspaceID, 25, nil)
	if !errors.Is(err, workspace.ErrDependency) || items != nil {
		t.Fatalf("canceled list = %#v, %v; want dependency and no items", items, err)
	}
	if _, err := store.GetInstallation(canceled, workspaceValue.WorkspaceID, row.InstallationID); !errors.Is(err, workspace.ErrDependency) {
		t.Fatalf("canceled read = %v, want dependency", err)
	}

	if _, err := pool.Exec(ctx, `ALTER TABLE workspace.installations ALTER COLUMN accepted_permissions TYPE integer USING 1`); err != nil {
		t.Fatalf("inject scan failure: %v", err)
	}
	items, _, err = store.ListInstallations(ctx, workspaceValue.WorkspaceID, 25, nil)
	if !errors.Is(err, workspace.ErrDependency) || items != nil {
		t.Fatalf("scan failure list = %#v, %v; want dependency and no items", items, err)
	}
}

func inspectionStoreForTest(t *testing.T, ctx context.Context) (*Store, *pgxpool.Pool) {
	t.Helper()
	databaseURL := inspectionDatabaseURL(t)
	connection, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect dedicated test database: %v", err)
	}
	if _, err := connection.Exec(ctx, `DROP SCHEMA IF EXISTS workspace CASCADE`); err != nil {
		_ = connection.Close(ctx)
		t.Fatalf("reset Workspace schema: %v", err)
	}
	if err := Migrate(ctx, connection, "up"); err != nil {
		_ = connection.Close(ctx)
		t.Fatalf("migrate Workspace schema: %v", err)
	}
	if err := connection.Close(ctx); err != nil {
		t.Fatalf("close migration connection: %v", err)
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("open Workspace pool: %v", err)
	}
	t.Cleanup(pool.Close)
	store, err := NewStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	return store, pool
}

func insertInspectionRow(t *testing.T, ctx context.Context, pool *pgxpool.Pool, value contracts.Installation) {
	t.Helper()
	_, err := pool.Exec(ctx, `
INSERT INTO workspace.installations (
    installation_id, workspace_id, agent_id, version_constraint, installed_version,
    accepted_permissions, status, installed_at, updated_at, uninstalled_at
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		value.InstallationID, value.WorkspaceID, value.AgentID, value.VersionConstraint, value.InstalledVersion,
		value.AcceptedPermissions, value.Status, value.InstalledAt, value.UpdatedAt, value.UninstalledAt,
	)
	if err != nil {
		t.Fatalf("insert Installation %s: %v", value.InstallationID, err)
	}
}

func assertInspectionIDs(t *testing.T, items []contracts.Installation, want []string) {
	t.Helper()
	if len(items) != len(want) {
		t.Fatalf("Installation count = %d, want %d: %#v", len(items), len(want), items)
	}
	for index, item := range items {
		if item.InstallationID != want[index] {
			t.Fatalf("Installation[%d] = %s, want %s", index, item.InstallationID, want[index])
		}
	}
}

func inspectionDatabaseURL(t *testing.T) string {
	t.Helper()
	databaseURL := os.Getenv("NEKIRO_TEST_DATABASE_URL")
	if strings.TrimSpace(databaseURL) == "" {
		t.Fatal("NEKIRO_TEST_DATABASE_URL is required for inspection integration tests")
	}
	configuration, err := pgx.ParseConfig(databaseURL)
	if err != nil || !strings.HasSuffix(configuration.Database, "_test") {
		t.Fatal("inspection integration database must be a valid database ending in _test")
	}
	return databaseURL
}
