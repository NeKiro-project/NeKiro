package testkit

import (
	"errors"
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
		Unsupported:   mustSpec(t, "rev-unsupported", strings.Repeat("b", 64), gateway.CapabilityForwarding),
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
		Unsupported:   mustSpec(t, "rev-unsupported", strings.Repeat("b", 64), gateway.CapabilityForwarding),
		StaleRevision: mustRevision(t, "rev-stale"),
		UnknownKey:    mustKey(t, "unknown-route"),
		Observed:      mustStatus(t, spec, gateway.RouteStateProgrammed, "provider-rev-1"),
		Deleted:       mustStatus(t, spec, gateway.RouteStateDeleted, "provider-rev-2"),
	}
	RunProviderConformance(t, provider, fixture)
}

func TestProviderConformanceFixtureValidationRejectsInvalidContracts(t *testing.T) {
	name := mustProviderName(t, "fake")
	emptyCapabilities, err := gateway.NewCapabilities()
	if err != nil {
		t.Fatalf("empty capabilities: %v", err)
	}
	forwardingCapabilities, err := gateway.NewCapabilities(gateway.CapabilityForwarding)
	if err != nil {
		t.Fatalf("forwarding capabilities: %v", err)
	}

	tests := map[string]struct {
		name         gateway.ProviderName
		capabilities gateway.Capabilities
		mutate       func(*ProviderConformanceFixture)
		wantInvalid  bool
	}{
		"provider name": {capabilities: emptyCapabilities, wantInvalid: true},
		"spec": {name: name, capabilities: emptyCapabilities, wantInvalid: true, mutate: func(f *ProviderConformanceFixture) {
			f.Spec = gateway.RouteSpec{}
		}},
		"unsupported spec": {name: name, capabilities: emptyCapabilities, wantInvalid: true, mutate: func(f *ProviderConformanceFixture) {
			f.Unsupported = gateway.RouteSpec{}
		}},
		"unsupported revision": {name: name, capabilities: emptyCapabilities, mutate: func(f *ProviderConformanceFixture) {
			f.Unsupported = f.Spec
		}},
		"stale revision": {name: name, capabilities: emptyCapabilities, wantInvalid: true, mutate: func(f *ProviderConformanceFixture) {
			f.StaleRevision = gateway.RouteRevision{}
		}},
		"same stale revision": {name: name, capabilities: emptyCapabilities, mutate: func(f *ProviderConformanceFixture) {
			f.StaleRevision = f.Spec.Revision()
		}},
		"unknown key": {name: name, capabilities: emptyCapabilities, wantInvalid: true, mutate: func(f *ProviderConformanceFixture) {
			f.UnknownKey = gateway.RouteKey{}
		}},
		"same unknown key": {name: name, capabilities: emptyCapabilities, mutate: func(f *ProviderConformanceFixture) {
			f.UnknownKey = f.Spec.Key()
		}},
		"observed status": {name: name, capabilities: emptyCapabilities, wantInvalid: true, mutate: func(f *ProviderConformanceFixture) {
			f.Observed = gateway.RouteStatus{}
		}},
		"wrong observed state": {name: name, capabilities: emptyCapabilities, mutate: func(f *ProviderConformanceFixture) {
			f.Observed = f.Deleted
		}},
		"deleted status": {name: name, capabilities: emptyCapabilities, wantInvalid: true, mutate: func(f *ProviderConformanceFixture) {
			f.Deleted = gateway.RouteStatus{}
		}},
		"wrong deleted state": {name: name, capabilities: emptyCapabilities, mutate: func(f *ProviderConformanceFixture) {
			f.Deleted = f.Observed
		}},
		"missing required capability": {name: name, capabilities: emptyCapabilities, mutate: func(f *ProviderConformanceFixture) {
			f.Spec = mustSpec(t, "rev-1", strings.Repeat("a", 64), gateway.CapabilityForwarding)
			f.Observed = mustStatus(t, f.Spec, gateway.RouteStateProgrammed, "provider-rev-1")
			f.Deleted = mustStatus(t, f.Spec, gateway.RouteStateDeleted, "provider-rev-2")
		}},
		"supported unsupported fixture": {name: name, capabilities: forwardingCapabilities},
	}

	for testName, testCase := range tests {
		t.Run(testName, func(t *testing.T) {
			fixture := validConformanceFixture(t)
			if testCase.mutate != nil {
				testCase.mutate(&fixture)
			}
			if err := validateProviderConformanceFixture(testCase.name, testCase.capabilities, fixture); err == nil {
				t.Fatal("validation unexpectedly succeeded")
			} else if testCase.wantInvalid && !errors.Is(err, gateway.ErrInvalid) {
				t.Fatalf("validation error = %v, want wrapped invalid", err)
			}
		})
	}
}

func validConformanceFixture(t testing.TB) ProviderConformanceFixture {
	t.Helper()
	spec := mustSpec(t, "rev-1", strings.Repeat("a", 64))
	return ProviderConformanceFixture{
		Spec:          spec,
		Unsupported:   mustSpec(t, "rev-unsupported", strings.Repeat("b", 64), gateway.CapabilityForwarding),
		StaleRevision: mustRevision(t, "rev-stale"),
		UnknownKey:    mustKey(t, "unknown-route"),
		Observed:      mustStatus(t, spec, gateway.RouteStateProgrammed, "provider-rev-1"),
		Deleted:       mustStatus(t, spec, gateway.RouteStateDeleted, "provider-rev-2"),
	}
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
