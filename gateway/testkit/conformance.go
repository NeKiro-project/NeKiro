package testkit

import (
	"context"
	"errors"
	"testing"

	"github.com/NeKiro-project/NeKiro/gateway"
)

// ProviderConformanceDriver supplies a provider and a deterministic control
// hook for injecting provider-observed status. The hook represents the
// provider-specific test fixture; it does not belong to the production
// Provider interface and never performs network I/O.
type ProviderConformanceDriver interface {
	Provider() gateway.Provider
	SetObservedStatus(gateway.RouteStatus) error
}

// ProviderConformanceFixture contains the exact desired route, one route with
// a capability the provider does not support, and the two provider observations
// needed to exercise asynchronous lifecycle behavior. Unsupported must use a
// distinct desired revision and require at least one unadvertised capability.
// Observed must be a programmed status for Spec's desired revision, and Deleted
// must be the same route/revision with state deleted.
type ProviderConformanceFixture struct {
	Spec          gateway.RouteSpec
	Unsupported   gateway.RouteSpec
	StaleRevision gateway.RouteRevision
	UnknownKey    gateway.RouteKey
	Observed      gateway.RouteStatus
	Deleted       gateway.RouteStatus
}

// RunProviderConformance verifies the backend-neutral provider contract. It
// does not assume a proxy, data-plane readiness, polling, retry, recovery, or
// provider switching. A provider without drain capability is tested for an
// explicit unsupported outcome; all other route lifecycle operations remain
// available.
func RunProviderConformance(t testing.TB, driver ProviderConformanceDriver, fixture ProviderConformanceFixture) {
	t.Helper()
	if driver == nil {
		t.Fatal("provider conformance driver is nil")
	}
	provider := driver.Provider()
	if provider == nil {
		t.Fatal("provider conformance driver returned nil provider")
	}
	if err := fixture.Spec.Validate(); err != nil {
		t.Fatalf("fixture spec: %v", err)
	}
	if err := fixture.Unsupported.Validate(); err != nil {
		t.Fatalf("fixture unsupported spec: %v", err)
	}
	if fixture.Unsupported.Revision().Equal(fixture.Spec.Revision()) {
		t.Fatal("fixture unsupported spec must use a distinct desired revision")
	}
	if err := fixture.StaleRevision.Validate(); err != nil {
		t.Fatalf("fixture stale revision: %v", err)
	}
	if fixture.StaleRevision.Equal(fixture.Spec.Revision()) {
		t.Fatal("fixture stale revision equals desired revision")
	}
	if err := fixture.UnknownKey.Validate(); err != nil {
		t.Fatalf("fixture unknown key: %v", err)
	}
	if fixture.UnknownKey.Equal(fixture.Spec.Key()) {
		t.Fatal("fixture unknown key equals desired route key")
	}
	if err := fixture.Observed.Validate(); err != nil {
		t.Fatalf("fixture observed status: %v", err)
	}
	if fixture.Observed.Key() != fixture.Spec.Key() ||
		!fixture.Observed.DesiredRevision().Equal(fixture.Spec.Revision()) ||
		fixture.Observed.State() != gateway.RouteStateProgrammed {
		t.Fatal("fixture observed status must be programmed for the exact desired route")
	}
	if err := fixture.Deleted.Validate(); err != nil {
		t.Fatalf("fixture deleted status: %v", err)
	}
	if fixture.Deleted.Key() != fixture.Spec.Key() ||
		!fixture.Deleted.DesiredRevision().Equal(fixture.Spec.Revision()) ||
		fixture.Deleted.State() != gateway.RouteStateDeleted {
		t.Fatal("fixture deleted status must be deleted for the exact desired route")
	}

	if err := provider.Name().Validate(); err != nil {
		t.Fatalf("provider name: %v", err)
	}
	capabilities := provider.Capabilities()
	if err := capabilities.Validate(); err != nil {
		t.Fatalf("provider capabilities: %v", err)
	}
	if missing := capabilities.Missing(fixture.Spec.RequiredCapabilities()); len(missing) != 0 {
		t.Fatalf("provider is missing fixture requirements: %v", missing)
	}
	if missing := capabilities.Missing(fixture.Unsupported.RequiredCapabilities()); len(missing) == 0 {
		t.Fatal("provider supports every fixture unsupported requirement")
	}

	drainRequest, err := gateway.NewDrainRequest(fixture.Spec.Revision())
	if err != nil {
		t.Fatalf("drain request: %v", err)
	}
	deleteRequest, err := gateway.NewDeleteRequest(fixture.Spec.Revision())
	if err != nil {
		t.Fatalf("delete request: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := provider.Reconcile(ctx, fixture.Spec); !errors.Is(err, gateway.ErrCanceled) || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled reconcile error = %v, want canceled/context.Canceled", err)
	}
	if _, err := provider.Status(ctx, fixture.Spec.Key()); !errors.Is(err, gateway.ErrCanceled) || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled status error = %v, want canceled/context.Canceled", err)
	}
	if _, err := provider.BeginDrain(ctx, fixture.Spec.Key(), drainRequest); !errors.Is(err, gateway.ErrCanceled) || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled drain error = %v, want canceled/context.Canceled", err)
	}
	if _, err := provider.Delete(ctx, fixture.Spec.Key(), deleteRequest); !errors.Is(err, gateway.ErrCanceled) || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled delete error = %v, want canceled/context.Canceled", err)
	}

	if _, err := provider.Status(context.Background(), fixture.UnknownKey); !errors.Is(err, gateway.ErrNotFound) {
		t.Fatalf("unknown status error = %v, want not_found", err)
	}
	if _, err := provider.Reconcile(context.Background(), fixture.Unsupported); !errors.Is(err, gateway.ErrUnsupported) {
		t.Fatalf("unsupported reconcile error = %v, want unsupported", err)
	}
	if _, err := provider.Status(context.Background(), fixture.Unsupported.Key()); !errors.Is(err, gateway.ErrNotFound) {
		t.Fatalf("status after unsupported reconcile = %v, want not_found", err)
	}

	first, err := provider.Reconcile(context.Background(), fixture.Spec)
	if err != nil {
		t.Fatalf("first reconcile: %v", err)
	}
	assertResultMatchesSpec(t, first, fixture.Spec)

	repeated, err := provider.Reconcile(context.Background(), fixture.Spec)
	if err != nil {
		t.Fatalf("repeated reconcile: %v", err)
	}
	assertResultMatchesSpec(t, repeated, fixture.Spec)

	if err := driver.SetObservedStatus(fixture.Observed); err != nil {
		t.Fatalf("set programmed observation: %v", err)
	}
	observed, err := provider.Status(context.Background(), fixture.Spec.Key())
	if err != nil {
		t.Fatalf("programmed status: %v", err)
	}
	if !observed.Equal(fixture.Observed) {
		t.Fatalf("programmed status = %#v, want %#v", observed, fixture.Observed)
	}

	staleDeleteRequest, err := gateway.NewDeleteRequest(fixture.StaleRevision)
	if err != nil {
		t.Fatalf("stale delete request: %v", err)
	}
	if _, err := provider.Delete(context.Background(), fixture.Spec.Key(), staleDeleteRequest); !errors.Is(err, gateway.ErrStale) {
		t.Fatalf("stale delete error = %v, want stale", err)
	}

	if capabilities.Supports(gateway.CapabilityDrain) {
		drained, drainErr := provider.BeginDrain(context.Background(), fixture.Spec.Key(), drainRequest)
		if drainErr != nil {
			t.Fatalf("drain: %v", drainErr)
		}
		assertResultMatchesSpec(t, drained, fixture.Spec)
		if drained.State() != gateway.RouteStateDraining {
			t.Fatalf("drain state = %q, want draining", drained.State())
		}
	} else {
		if _, drainErr := provider.BeginDrain(context.Background(), fixture.Spec.Key(), drainRequest); !errors.Is(drainErr, gateway.ErrUnsupported) {
			t.Fatalf("unsupported drain error = %v, want unsupported", drainErr)
		}
	}

	deleting, err := provider.Delete(context.Background(), fixture.Spec.Key(), deleteRequest)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	assertResultMatchesSpec(t, deleting, fixture.Spec)
	if deleting.State() != gateway.RouteStateDeleting {
		t.Fatalf("delete state = %q, want deleting", deleting.State())
	}

	if err := driver.SetObservedStatus(fixture.Deleted); err != nil {
		t.Fatalf("set deleted observation: %v", err)
	}
	deleted, err := provider.Status(context.Background(), fixture.Spec.Key())
	if err != nil {
		t.Fatalf("deleted status: %v", err)
	}
	if !deleted.Equal(fixture.Deleted) {
		t.Fatalf("deleted status = %#v, want %#v", deleted, fixture.Deleted)
	}

	if err := provider.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := provider.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
	if _, err := provider.Status(context.Background(), fixture.Spec.Key()); !errors.Is(err, gateway.ErrClosed) {
		t.Fatalf("status after close = %v, want closed", err)
	}
	if _, err := provider.Reconcile(context.Background(), fixture.Spec); !errors.Is(err, gateway.ErrClosed) {
		t.Fatalf("reconcile after close = %v, want closed", err)
	}
	if _, err := provider.BeginDrain(context.Background(), fixture.Spec.Key(), drainRequest); !errors.Is(err, gateway.ErrClosed) {
		t.Fatalf("drain after close = %v, want closed", err)
	}
	if _, err := provider.Delete(context.Background(), fixture.Spec.Key(), deleteRequest); !errors.Is(err, gateway.ErrClosed) {
		t.Fatalf("delete after close = %v, want closed", err)
	}
}

func assertResultMatchesSpec(t testing.TB, result gateway.ReconcileResult, spec gateway.RouteSpec) {
	t.Helper()
	if err := result.Validate(); err != nil {
		t.Fatalf("invalid reconcile result: %v", err)
	}
	if !result.Key().Equal(spec.Key()) || !result.DesiredRevision().Equal(spec.Revision()) {
		t.Fatalf("reconcile result identity = key %q revision %q, want key %q revision %q", result.Key(), result.DesiredRevision(), spec.Key(), spec.Revision())
	}
}
