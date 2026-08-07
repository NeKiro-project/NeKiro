package testkit

import (
	"strings"
	"testing"

	"github.com/NeKiro-project/NeKiro/gateway"
)

func TestFakeProviderSatisfiesProviderConformance(t *testing.T) {
	provider, err := NewFakeProvider(FakeConfig{Name: mustProviderName(t, "fake")})
	if err != nil {
		t.Fatalf("NewFakeProvider: %v", err)
	}
	spec := mustSpec(t, "rev-1", strings.Repeat("a", 64))
	fixture := ProviderConformanceFixture{
		Spec:          spec,
		StaleRevision: mustRevision(t, "rev-stale"),
		UnknownKey:    mustKey(t, "unknown-route"),
		Observed:      mustStatus(t, spec, gateway.RouteStateProgrammed, "provider-rev-1"),
		Deleted:       mustStatus(t, spec, gateway.RouteStateDeleted, "provider-rev-2"),
	}
	RunProviderConformance(t, provider, fixture)
}

func TestFakeProviderConformanceReportsUnsupportedOptionalDrain(t *testing.T) {
	provider, err := NewFakeProvider(FakeConfig{Name: mustProviderName(t, "fake-no-drain")})
	if err != nil {
		t.Fatalf("NewFakeProvider: %v", err)
	}
	spec := mustSpec(t, "rev-1", strings.Repeat("a", 64))
	fixture := ProviderConformanceFixture{
		Spec:          spec,
		StaleRevision: mustRevision(t, "rev-stale"),
		UnknownKey:    mustKey(t, "unknown-route"),
		Observed:      mustStatus(t, spec, gateway.RouteStateProgrammed, "provider-rev-1"),
		Deleted:       mustStatus(t, spec, gateway.RouteStateDeleted, "provider-rev-2"),
	}
	RunProviderConformance(t, provider, fixture)
}

func mustProviderName(t testing.TB, value string) gateway.ProviderName {
	t.Helper()
	name, err := gateway.NewProviderName(value)
	if err != nil {
		t.Fatalf("NewProviderName: %v", err)
	}
	return name
}

func mustKey(t testing.TB, value string) gateway.RouteKey {
	t.Helper()
	key, err := gateway.NewRouteKey(value)
	if err != nil {
		t.Fatalf("NewRouteKey: %v", err)
	}
	return key
}

func mustRevision(t testing.TB, value string) gateway.RouteRevision {
	t.Helper()
	revision, err := gateway.NewRouteRevision(value)
	if err != nil {
		t.Fatalf("NewRouteRevision: %v", err)
	}
	return revision
}

func mustBackend(t testing.TB, value string) gateway.BackendRef {
	t.Helper()
	ref, err := gateway.NewBackendRef(value)
	if err != nil {
		t.Fatalf("NewBackendRef: %v", err)
	}
	return ref
}

func mustSpec(t testing.TB, revision, digest string, required ...gateway.Capability) gateway.RouteSpec {
	t.Helper()
	spec, err := gateway.NewRouteSpec(gateway.RouteSpecInput{
		Key:                  mustKey(t, "route-a"),
		Revision:             mustRevision(t, revision),
		ReleaseID:            "release-a",
		CardDigest:           digest,
		AgentID:              "agent-a",
		AgentVersion:         "1.0.0",
		EndpointOrigin:       "https://agent.example",
		EndpointPath:         "/a2a",
		Audience:             "https://agent.example",
		DiscoveryOwner:       gateway.DiscoveryOwnerGateway,
		BackendRef:           mustBackend(t, "backend-a"),
		RequiredCapabilities: required,
	})
	if err != nil {
		t.Fatalf("NewRouteSpec: %v", err)
	}
	return spec
}

func mustStatus(t testing.TB, spec gateway.RouteSpec, state gateway.RouteState, observed string) gateway.RouteStatus {
	t.Helper()
	observedRevision := mustRevision(t, observed)
	status, err := gateway.NewRouteStatus(gateway.RouteStatusInput{
		Key:              spec.Key(),
		State:            state,
		DesiredRevision:  spec.Revision(),
		ObservedRevision: &observedRevision,
	})
	if err != nil {
		t.Fatalf("NewRouteStatus: %v", err)
	}
	return status
}
