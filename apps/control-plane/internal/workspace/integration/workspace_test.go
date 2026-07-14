//go:build integration

package workspace_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Nene7ko/NeKiro/apps/control-plane/internal/catalog"
	catalogpostgres "github.com/Nene7ko/NeKiro/apps/control-plane/internal/catalog/postgres"
	"github.com/Nene7ko/NeKiro/apps/control-plane/internal/workspace"
	workspacepostgres "github.com/Nene7ko/NeKiro/apps/control-plane/internal/workspace/postgres"
	"github.com/Nene7ko/NeKiro/contracts"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestWorkspaceInstallationPersistenceLifecycleAndResolution(t *testing.T) {
	ctx := context.Background()
	catalogService, workspaceService := integrationServices(t, ctx)
	owner := workspace.AuthenticatedCaller{ID: "owner-a"}
	if _, err := workspaceService.CreateWorkspace(ctx, owner, contracts.CreateWorkspaceRequest{WorkspaceID: "workspace-integration"}); err != nil {
		t.Fatalf("create Workspace: %v", err)
	}
	installation, err := workspaceService.Install(ctx, owner, "workspace-integration", contracts.InstallAgentRequest{
		AgentID: "runtime-a", VersionConstraint: "^1.0.0", AcceptedPermissions: []string{"document.read"},
	})
	if err != nil {
		t.Fatalf("install Agent: %v", err)
	}
	if installation.InstalledVersion != "1.0.0" || installation.Status != "enabled" {
		t.Fatalf("installation = %#v", installation)
	}
	resolved, err := workspaceService.Resolve(ctx, contracts.ResolveAgentRequest{
		InvocationID: "invocation-integration", RootTaskID: "task-integration", TraceID: "trace-integration",
		WorkspaceID: "workspace-integration", AgentID: "runtime-a", Version: "1.0.0", Capability: "document.read",
	})
	if err != nil || resolved.Card.Version != "1.0.0" || resolved.Installation.InstallationID != installation.InstallationID {
		t.Fatalf("exact resolution = %#v, %v", resolved, err)
	}
	if _, err := workspaceService.UpdateInstallation(ctx, owner, "workspace-integration", installation.InstallationID, "disabled"); err != nil {
		t.Fatalf("disable Installation: %v", err)
	}
	if _, err := workspaceService.Uninstall(ctx, owner, "workspace-integration", installation.InstallationID); err != nil {
		t.Fatalf("uninstall Installation: %v", err)
	}
	listed, err := workspaceService.ListInstallations(ctx, owner, "workspace-integration", 1, nil)
	if err != nil || len(listed.Items) != 1 || listed.Items[0].Status != "uninstalled" {
		t.Fatalf("historical Installation list = %#v, %v", listed, err)
	}
	second, err := workspaceService.Install(ctx, owner, "workspace-integration", contracts.InstallAgentRequest{AgentID: "runtime-a", VersionConstraint: "^1.0.0", AcceptedPermissions: []string{"document.read"}})
	if err != nil {
		t.Fatalf("reinstall Agent: %v", err)
	}
	if _, err := catalogService.Disable(ctx, catalog.AuthenticatedCaller{ID: "owner-a"}, "runtime-a", "1.0.0"); err != nil {
		t.Fatalf("disable pinned Catalog version: %v", err)
	}
	if _, err := workspaceService.Resolve(ctx, contracts.ResolveAgentRequest{
		InvocationID: "invocation-after-disable", RootTaskID: "task-after-disable", TraceID: "trace-after-disable",
		WorkspaceID: "workspace-integration", AgentID: "runtime-a", Version: second.InstalledVersion, Capability: "document.read",
	}); !errors.Is(err, workspace.ErrAgentDisabled) {
		t.Fatalf("resolution after Catalog disable = %v", err)
	}
	current, err := workspaceService.GetInstallation(ctx, owner, "workspace-integration", installation.InstallationID)
	if err != nil || current.Status != "uninstalled" {
		t.Fatalf("historical Installation changed = %#v, %v", current, err)
	}

}

func TestConcurrentInstallLeavesOneCurrentInstallation(t *testing.T) {
	ctx := context.Background()
	_, workspaceService := integrationServices(t, ctx)
	owner := workspace.AuthenticatedCaller{ID: "owner-a"}
	if _, err := workspaceService.CreateWorkspace(ctx, owner, contracts.CreateWorkspaceRequest{WorkspaceID: "workspace-race"}); err != nil {
		t.Fatalf("create Workspace: %v", err)
	}
	var wait sync.WaitGroup
	results := make(chan error, 100)
	for index := 0; index < 100; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			_, err := workspaceService.Install(ctx, owner, "workspace-race", contracts.InstallAgentRequest{AgentID: "runtime-a", VersionConstraint: "^1.0.0", AcceptedPermissions: []string{"document.read"}})
			results <- err
		}()
	}
	wait.Wait()
	close(results)
	successes := 0
	conflicts := 0
	for err := range results {
		if err == nil {
			successes++
		} else if errors.Is(err, workspace.ErrConflict) {
			conflicts++
		} else {
			t.Fatalf("concurrent install error: %v", err)
		}
	}
	if successes != 1 || conflicts != 99 {
		t.Fatalf("concurrent installs successes=%d conflicts=%d", successes, conflicts)
	}
	listed, err := workspaceService.ListInstallations(ctx, owner, "workspace-race", 100, nil)
	if err != nil || len(listed.Items) != 1 {
		t.Fatalf("current Installation count = %#v, %v", listed, err)
	}
}

func TestWorkspaceCreateReadSurvivesStoreReconstruction(t *testing.T) {
	ctx := context.Background()
	catalogService, workspaceService := integrationServices(t, ctx)
	owner := workspace.AuthenticatedCaller{ID: "owner-a"}
	created, err := workspaceService.CreateWorkspace(ctx, owner, contracts.CreateWorkspaceRequest{WorkspaceID: "workspace-root"})
	if err != nil {
		t.Fatalf("create Workspace: %v", err)
	}
	if _, err := workspaceService.CreateWorkspace(ctx, workspace.AuthenticatedCaller{ID: "owner-b"}, contracts.CreateWorkspaceRequest{WorkspaceID: "workspace-root"}); !errors.Is(err, workspace.ErrConflict) {
		t.Fatalf("duplicate Workspace = %v, want conflict", err)
	}

	initial, err := workspaceService.GetWorkspace(ctx, owner, "workspace-root")
	if err != nil || !sameWorkspace(initial, created) {
		t.Fatalf("initial Workspace read = %#v, %v", initial, err)
	}
	if _, err := workspaceService.GetWorkspace(ctx, workspace.AuthenticatedCaller{ID: "owner-b"}, "workspace-root"); !errors.Is(err, workspace.ErrForbidden) {
		t.Fatalf("non-owner Workspace read = %v, want forbidden", err)
	}
	if _, err := workspaceService.GetWorkspace(ctx, owner, "missing-root"); !errors.Is(err, workspace.ErrNotFound) {
		t.Fatalf("unknown Workspace read = %v, want not found", err)
	}

	databaseURL := integrationDatabaseURL(t)
	reconstructedPool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("reopen Workspace database pool: %v", err)
	}
	t.Cleanup(reconstructedPool.Close)
	reconstructedStore, err := workspacepostgres.NewStore(reconstructedPool)
	if err != nil {
		t.Fatal(err)
	}
	validator, err := contracts.NewValidator()
	if err != nil {
		t.Fatal(err)
	}
	reconstructedService, err := workspace.NewService(reconstructedStore, catalogService, workspace.OwnerPolicy{}, validator, time.Now, workspace.NewRandomID)
	if err != nil {
		t.Fatal(err)
	}
	restarted, err := reconstructedService.GetWorkspace(ctx, owner, "workspace-root")
	if err != nil {
		t.Fatalf("reconstructed Workspace read: %v", err)
	}
	if !sameWorkspace(restarted, created) {
		t.Fatalf("reconstructed Workspace = %#v, want %#v", restarted, created)
	}

	failedContext, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := reconstructedService.GetWorkspace(failedContext, owner, "workspace-root"); !errors.Is(err, workspace.ErrDependency) {
		t.Fatalf("canceled Workspace read = %v, want dependency", err)
	}
}

func TestInstallPinsCommittedFieldsAndIgnoresNewPublication(t *testing.T) {
	ctx := context.Background()
	catalogService, workspaceService := integrationServices(t, ctx)
	owner := workspace.AuthenticatedCaller{ID: "owner-a"}
	if _, err := workspaceService.CreateWorkspace(ctx, owner, contracts.CreateWorkspaceRequest{WorkspaceID: "workspace-pin"}); err != nil {
		t.Fatal(err)
	}
	installation, err := workspaceService.Install(ctx, owner, "workspace-pin", contracts.InstallAgentRequest{
		AgentID: "runtime-a", VersionConstraint: "^1.0.0", AcceptedPermissions: []string{"document.read"},
	})
	if err != nil {
		t.Fatalf("install pinned Agent: %v", err)
	}
	if installation.InstalledVersion != "1.0.0" || installation.Status != "enabled" || installation.AcceptedPermissions[0] != "document.read" {
		t.Fatalf("installation = %#v", installation)
	}
	stored, err := workspaceService.GetInstallation(ctx, owner, "workspace-pin", installation.InstallationID)
	if err != nil || !sameInstallation(stored, installation) {
		t.Fatalf("stored installation = %#v, %v; created = %#v", stored, err, installation)
	}

	newCard := integrationCard()
	newCard.Version = "1.1.0"
	if err := registerPublishedCard(ctx, catalogService, newCard); err != nil {
		t.Fatalf("publish newer matching Card: %v", err)
	}
	unchanged, err := workspaceService.GetInstallation(ctx, owner, "workspace-pin", installation.InstallationID)
	if err != nil || !sameInstallation(unchanged, installation) {
		t.Fatalf("new publication mutated installation = %#v, %v; original = %#v", unchanged, err, installation)
	}

	emptyCard := integrationCard()
	emptyCard.AgentID = "runtime-empty"
	emptyCard.Name = "Runtime Empty Permission"
	emptyCard.Skills[0].RequiredPermissions = []string{}
	emptyCard.Permissions = []contracts.PermissionDeclaration{}
	if err := registerPublishedCard(ctx, catalogService, emptyCard); err != nil {
		t.Fatalf("publish empty-permission Card: %v", err)
	}
	if _, err := workspaceService.CreateWorkspace(ctx, owner, contracts.CreateWorkspaceRequest{WorkspaceID: "workspace-empty-install"}); err != nil {
		t.Fatal(err)
	}
	emptyInstallation, err := workspaceService.Install(ctx, owner, "workspace-empty-install", contracts.InstallAgentRequest{
		AgentID: "runtime-empty", VersionConstraint: "^1.0.0", AcceptedPermissions: []string{},
	})
	if err != nil {
		t.Fatalf("install empty-permission Agent: %v", err)
	}
	if emptyInstallation.AcceptedPermissions == nil || len(emptyInstallation.AcceptedPermissions) != 0 {
		t.Fatalf("empty accepted permissions = %#v", emptyInstallation.AcceptedPermissions)
	}
	emptyStored, err := workspaceService.GetInstallation(ctx, owner, "workspace-empty-install", emptyInstallation.InstallationID)
	if err != nil || emptyStored.AcceptedPermissions == nil || len(emptyStored.AcceptedPermissions) != 0 {
		t.Fatalf("stored empty accepted permissions = %#v, %v", emptyStored.AcceptedPermissions, err)
	}

	failedContext, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := workspaceService.Install(failedContext, owner, "workspace-pin", contracts.InstallAgentRequest{
		AgentID: "runtime-a", VersionConstraint: "^1.0.0", AcceptedPermissions: []string{"document.read"},
	}); !errors.Is(err, workspace.ErrDependency) {
		t.Fatalf("canceled install = %v, want dependency", err)
	}
}

func integrationServices(t *testing.T, ctx context.Context) (*catalog.Service, *workspace.Service) {
	t.Helper()
	databaseURL := integrationDatabaseURL(t)
	if _, err := pgx.ParseConfig(databaseURL); err != nil {
		t.Fatal("integration database URL was rejected")
	}
	connection, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect integration database: %v", err)
	}
	if _, err := connection.Exec(ctx, `DROP SCHEMA IF EXISTS workspace CASCADE`); err != nil {
		t.Fatal(err)
	}
	if _, err := connection.Exec(ctx, `DROP SCHEMA IF EXISTS catalog CASCADE`); err != nil {
		t.Fatal(err)
	}
	if err := catalogpostgres.Migrate(ctx, connection, "up"); err != nil {
		t.Fatal(err)
	}
	if err := workspacepostgres.Migrate(ctx, connection, "up"); err != nil {
		t.Fatal(err)
	}
	_ = connection.Close(ctx)

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	catalogStore, err := catalogpostgres.NewStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	workspaceStore, err := workspacepostgres.NewStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	validator, err := contracts.NewValidator()
	if err != nil {
		t.Fatal(err)
	}
	catalogService, err := catalog.NewService(catalogStore, validator, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if err := registerPublishedCard(ctx, catalogService, integrationCard()); err != nil {
		t.Fatalf("publish fixture Card: %v", err)
	}
	workspaceService, err := workspace.NewService(workspaceStore, catalogService, workspace.OwnerPolicy{}, validator, time.Now, workspace.NewRandomID)
	if err != nil {
		t.Fatal(err)
	}
	return catalogService, workspaceService
}

func integrationDatabaseURL(t *testing.T) string {
	t.Helper()
	databaseURL := os.Getenv("NEKIRO_TEST_DATABASE_URL")
	if strings.TrimSpace(databaseURL) == "" {
		t.Fatal("NEKIRO_TEST_DATABASE_URL is required")
	}
	configuration, err := pgx.ParseConfig(databaseURL)
	if err != nil || !strings.HasSuffix(configuration.Database, "_test") {
		t.Fatal("integration database must end in _test")
	}
	return databaseURL
}

func sameWorkspace(left, right contracts.Workspace) bool {
	return left.WorkspaceID == right.WorkspaceID && left.OwnerID == right.OwnerID &&
		left.CreatedAt.Equal(right.CreatedAt) && left.UpdatedAt.Equal(right.UpdatedAt)
}

func sameInstallation(left, right contracts.Installation) bool {
	if left.InstallationID != right.InstallationID || left.WorkspaceID != right.WorkspaceID ||
		left.AgentID != right.AgentID || left.VersionConstraint != right.VersionConstraint ||
		left.InstalledVersion != right.InstalledVersion || left.Status != right.Status ||
		!left.InstalledAt.Equal(right.InstalledAt) || !left.UpdatedAt.Equal(right.UpdatedAt) ||
		(left.UninstalledAt == nil) != (right.UninstalledAt == nil) ||
		left.UninstalledAt != nil && !left.UninstalledAt.Equal(*right.UninstalledAt) ||
		len(left.AcceptedPermissions) != len(right.AcceptedPermissions) {
		return false
	}
	for index := range left.AcceptedPermissions {
		if left.AcceptedPermissions[index] != right.AcceptedPermissions[index] {
			return false
		}
	}
	return true
}

func registerPublishedCard(ctx context.Context, service *catalog.Service, card contracts.AgentCard) error {
	body, err := json.Marshal(contracts.RegisterAgentRequest{Card: card})
	if err != nil {
		return err
	}
	if _, err := service.Register(ctx, catalog.AuthenticatedCaller{ID: "owner-a"}, body); err != nil && !errors.Is(err, catalog.ErrConflict) {
		return fmt.Errorf("register Card: %w", err)
	}
	if _, err := service.Publish(ctx, catalog.AuthenticatedCaller{ID: "owner-a"}, card.AgentID, card.Version); err != nil && !errors.Is(err, catalog.ErrConflict) {
		return fmt.Errorf("publish Card: %w", err)
	}
	return nil
}

func integrationCard() contracts.AgentCard {
	return contracts.AgentCard{SchemaVersion: "0.2", AgentID: "runtime-a", Name: "Runtime A", Description: "Integration fixture", Owner: contracts.AgentOwner{ID: "owner-a", DisplayName: "Owner"}, Version: "1.0.0", Protocol: contracts.AgentProtocol{Type: "a2a", Version: "0.3.0", Transport: "JSONRPC", Endpoint: "https://agent.example.test/a2a"}, Skills: []contracts.AgentSkill{{ID: "document.read", Name: "Read", Description: "Read", InputSchema: contracts.JSONSchema{"type": "object"}, OutputSchema: contracts.JSONSchema{"type": "object"}, RequiredPermissions: []string{"document.read"}}}, Authentication: contracts.AgentAuthentication{Type: "none"}, Permissions: []contracts.PermissionDeclaration{{ID: "document.read", Description: "Read"}}, Limits: contracts.AgentLimits{TimeoutMS: 1000, MaxInputBytes: json.Number("1000"), MaxOutputBytes: json.Number("1000"), Streaming: false}}
}
