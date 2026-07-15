//go:build integration

package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
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
	if !sameInstallation(actual, rows[1]) {
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

func TestInstallationInspectionStoreTraverses101RowsWithoutDuplicates(t *testing.T) {
	ctx := context.Background()
	store, pool := inspectionStoreForTest(t, ctx)
	installedAt := time.Date(2026, 7, 15, 11, 0, 0, 0, time.UTC)
	workspaceValue := contracts.Workspace{WorkspaceID: "workspace-inspection-scale", OwnerID: "owner-a", CreatedAt: installedAt, UpdatedAt: installedAt}
	if _, err := store.CreateWorkspace(ctx, workspaceValue); err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 101; index++ {
		insertInspectionRow(t, ctx, pool, contracts.Installation{
			InstallationID:      fmt.Sprintf("installation-%03d", index),
			WorkspaceID:         workspaceValue.WorkspaceID,
			AgentID:             fmt.Sprintf("runtime-%03d", index),
			VersionConstraint:   "^1.0.0",
			InstalledVersion:    "1.0.0",
			AcceptedPermissions: []string{},
			Status:              "enabled",
			InstalledAt:         installedAt,
			UpdatedAt:           installedAt,
		})
	}

	var all []contracts.Installation
	var after *workspace.InstallationPosition
	for {
		page, hasMore, err := store.ListInstallations(ctx, workspaceValue.WorkspaceID, 25, after)
		if err != nil {
			t.Fatal(err)
		}
		all = append(all, page...)
		if !hasMore {
			break
		}
		last := page[len(page)-1]
		after = &workspace.InstallationPosition{InstalledAt: last.InstalledAt, InstallationID: last.InstallationID}
	}
	if len(all) != 101 {
		t.Fatalf("traversed %d installations, want 101", len(all))
	}
	for index, value := range all {
		wantID := fmt.Sprintf("installation-%03d", index)
		if value.InstallationID != wantID {
			t.Fatalf("installation[%d] = %q, want %q", index, value.InstallationID, wantID)
		}
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

func TestLifecycleStoreAdvancesStaleCandidateAfterLockedRow(t *testing.T) {
	ctx := context.Background()
	store, pool := inspectionStoreForTest(t, ctx)
	installedAt := time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC)
	row := contracts.Installation{
		InstallationID:      "installation-monotonic",
		WorkspaceID:         "workspace-monotonic",
		AgentID:             "runtime-monotonic",
		VersionConstraint:   "^1.0.0",
		InstalledVersion:    "1.0.0",
		AcceptedPermissions: []string{"document.read"},
		Status:              "enabled",
		InstalledAt:         installedAt,
		UpdatedAt:           installedAt,
	}
	workspaceValue := contracts.Workspace{WorkspaceID: row.WorkspaceID, OwnerID: "owner-a", CreatedAt: installedAt, UpdatedAt: installedAt}
	if _, err := store.CreateWorkspace(ctx, workspaceValue); err != nil {
		t.Fatal(err)
	}
	insertInspectionRow(t, ctx, pool, row)

	stale := installedAt.Add(-time.Hour)
	disabled, err := store.ChangeInstallationStatus(ctx, row.WorkspaceID, row.InstallationID, "disabled", stale)
	if err != nil {
		t.Fatalf("stale disable: %v", err)
	}
	if !disabled.UpdatedAt.After(row.UpdatedAt) {
		t.Fatalf("stale disable timestamp = %s, previous = %s", disabled.UpdatedAt, row.UpdatedAt)
	}

	terminal, err := store.UninstallInstallation(ctx, row.WorkspaceID, row.InstallationID, stale)
	if err != nil {
		t.Fatalf("stale uninstall: %v", err)
	}
	if !terminal.UpdatedAt.After(disabled.UpdatedAt) || terminal.UninstalledAt == nil || !terminal.UninstalledAt.Equal(terminal.UpdatedAt) {
		t.Fatalf("stale uninstall timestamps = %#v", terminal)
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

func sameInstallation(left, right contracts.Installation) bool {
	if left.InstallationID != right.InstallationID || left.WorkspaceID != right.WorkspaceID || left.AgentID != right.AgentID || left.VersionConstraint != right.VersionConstraint || left.InstalledVersion != right.InstalledVersion || left.Status != right.Status || !left.InstalledAt.Equal(right.InstalledAt) || !left.UpdatedAt.Equal(right.UpdatedAt) || (left.UninstalledAt == nil) != (right.UninstalledAt == nil) || len(left.AcceptedPermissions) != len(right.AcceptedPermissions) {
		return false
	}
	if left.UninstalledAt != nil && !left.UninstalledAt.Equal(*right.UninstalledAt) {
		return false
	}
	for index := range left.AcceptedPermissions {
		if left.AcceptedPermissions[index] != right.AcceptedPermissions[index] {
			return false
		}
	}
	return true
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
