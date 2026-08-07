package testkit

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/NeKiro-project/NeKiro/gateway"
)

func TestFakeProviderRejectsUnsupportedRequirementsWithoutRecordingRoute(t *testing.T) {
	provider, err := NewFakeProvider(FakeConfig{Name: mustProviderName(t, "fake")})
	if err != nil {
		t.Fatalf("NewFakeProvider: %v", err)
	}
	spec := mustSpec(t, "rev-1", strings.Repeat("a", 64), gateway.CapabilityForwarding)
	_, err = provider.Reconcile(context.Background(), spec)
	if !errors.Is(err, gateway.ErrUnsupported) {
		t.Fatalf("reconcile error = %v, want unsupported", err)
	}
	var outcome *gateway.OutcomeError
	if !errors.As(err, &outcome) || outcome.Cause() != gateway.CauseRequiredCapability {
		t.Fatalf("unsupported cause = %v, want required_capability", err)
	}
	if _, err := provider.Status(context.Background(), spec.Key()); !errors.Is(err, gateway.ErrNotFound) {
		t.Fatalf("status after rejected reconcile = %v, want not_found", err)
	}
}

func TestFakeProviderDoesNotEmulateRouterOwnedDiscovery(t *testing.T) {
	provider, err := NewFakeProvider(FakeConfig{Name: mustProviderName(t, "fake")})
	if err != nil {
		t.Fatalf("NewFakeProvider: %v", err)
	}
	spec, err := gateway.NewRouteSpec(gateway.RouteSpecInput{
		Key:            mustKey(t, "route-router"),
		Revision:       mustRevision(t, "rev-1"),
		ReleaseID:      "release-a",
		CardDigest:     strings.Repeat("a", 64),
		AgentID:        "agent-a",
		AgentVersion:   "1.0.0",
		EndpointOrigin: "https://agent.example",
		EndpointPath:   "/a2a",
		Audience:       "https://agent.example",
		DiscoveryOwner: gateway.DiscoveryOwnerRouter,
		BackendRef:     mustBackend(t, "backend-a"),
	})
	if err != nil {
		t.Fatalf("NewRouteSpec: %v", err)
	}
	if _, err := provider.Reconcile(context.Background(), spec); !errors.Is(err, gateway.ErrUnsupported) {
		t.Fatalf("router-owned reconcile = %v, want unsupported", err)
	}
	if _, err := provider.Status(context.Background(), spec.Key()); !errors.Is(err, gateway.ErrNotFound) {
		t.Fatalf("router-owned status = %v, want not_found", err)
	}
}

func TestFakeProviderEnforcesExactRevisionAndLifecycle(t *testing.T) {
	provider, err := NewFakeProvider(FakeConfig{Name: mustProviderName(t, "fake")})
	if err != nil {
		t.Fatalf("NewFakeProvider: %v", err)
	}
	spec := mustSpec(t, "rev-1", strings.Repeat("a", 64))
	if result, err := provider.Reconcile(context.Background(), spec); err != nil || result.State() != gateway.RouteStateAccepted {
		t.Fatalf("reconcile = %#v, %v; want accepted", result, err)
	}

	wrongRevision := mustRevision(t, "rev-2")
	wrongDrain, err := gateway.NewDrainRequest(wrongRevision)
	if err != nil {
		t.Fatalf("wrong drain request: %v", err)
	}
	if _, err := provider.BeginDrain(context.Background(), spec.Key(), wrongDrain); !errors.Is(err, gateway.ErrStale) {
		t.Fatalf("wrong drain error = %v, want stale", err)
	}

	drain, err := gateway.NewDrainRequest(spec.Revision())
	if err != nil {
		t.Fatalf("drain request: %v", err)
	}
	if _, err := provider.BeginDrain(context.Background(), spec.Key(), drain); !errors.Is(err, gateway.ErrUnsupported) {
		t.Fatalf("drain = %v, want unsupported", err)
	}

	wrongDelete, err := gateway.NewDeleteRequest(wrongRevision)
	if err != nil {
		t.Fatalf("wrong delete request: %v", err)
	}
	if _, err := provider.Delete(context.Background(), spec.Key(), wrongDelete); !errors.Is(err, gateway.ErrStale) {
		t.Fatalf("wrong delete error = %v, want stale", err)
	}
	deleteRequest, err := gateway.NewDeleteRequest(spec.Revision())
	if err != nil {
		t.Fatalf("delete request: %v", err)
	}
	if result, err := provider.Delete(context.Background(), spec.Key(), deleteRequest); err != nil || result.State() != gateway.RouteStateDeleting {
		t.Fatalf("delete = %#v, %v; want deleting", result, err)
	}
	if result, err := provider.Delete(context.Background(), spec.Key(), deleteRequest); err != nil || result.State() != gateway.RouteStateDeleting {
		t.Fatalf("repeated delete = %#v, %v; want idempotent deleting", result, err)
	}

	deleted := mustStatus(t, spec, gateway.RouteStateDeleted, "provider-rev-2")
	if err := provider.SetObservedStatus(deleted); err != nil {
		t.Fatalf("set deleted status: %v", err)
	}
	if _, err := provider.BeginDrain(context.Background(), spec.Key(), drain); !errors.Is(err, gateway.ErrUnsupported) {
		t.Fatalf("drain deleted route = %v, want unsupported", err)
	}
	if result, err := provider.Delete(context.Background(), spec.Key(), deleteRequest); err != nil || result.State() != gateway.RouteStateDeleted {
		t.Fatalf("delete after observed deletion = %#v, %v; want deleted", result, err)
	}
}

func TestFakeProviderReportsNoDataPlaneCapabilities(t *testing.T) {
	provider, err := NewFakeProvider(FakeConfig{Name: mustProviderName(t, "fake")})
	if err != nil {
		t.Fatalf("NewFakeProvider: %v", err)
	}
	if values := provider.Capabilities().Values(); len(values) != 0 {
		t.Fatalf("fake capabilities = %v, want empty", values)
	}
}

func TestFakeProviderRejectsRevisionReuseAndHonorsContextAndClose(t *testing.T) {
	provider, err := NewFakeProvider(FakeConfig{Name: mustProviderName(t, "fake")})
	if err != nil {
		t.Fatalf("NewFakeProvider: %v", err)
	}
	first := mustSpec(t, "rev-1", strings.Repeat("a", 64))
	if _, err := provider.Reconcile(context.Background(), first); err != nil {
		t.Fatalf("first reconcile: %v", err)
	}
	second := mustSpec(t, "rev-1", strings.Repeat("b", 64))
	if _, err := provider.Reconcile(context.Background(), second); !errors.Is(err, gateway.ErrInvalid) {
		t.Fatalf("revision reuse error = %v, want invalid", err)
	}
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := provider.Status(canceled, first.Key()); !errors.Is(err, gateway.ErrCanceled) || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled status = %v, want canceled/context.Canceled", err)
	}
	if _, err := provider.Reconcile(nil, first); !errors.Is(err, gateway.ErrInvalid) {
		t.Fatalf("nil context reconcile = %v, want invalid", err)
	}
	if err := provider.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := provider.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
	if _, err := provider.Status(context.Background(), first.Key()); !errors.Is(err, gateway.ErrClosed) {
		t.Fatalf("status after close = %v, want closed", err)
	}
}
