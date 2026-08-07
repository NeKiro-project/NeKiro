package gateway

import (
	"errors"
	"strings"
	"testing"
)

func TestRouteSpecPreservesExactFactsAndCopiesRequirements(t *testing.T) {
	required := []Capability{CapabilityForwarding, CapabilityRetryPolicyControl}
	spec := mustRouteSpec(t, RouteSpecInput{
		Key:                  mustRouteKey(t, "route-a"),
		Revision:             mustRouteRevision(t, "rev-1"),
		ReleaseID:            "release-a",
		CardDigest:           strings.Repeat("a", 64),
		AgentID:              "agent-a",
		AgentVersion:         "1.0.0",
		EndpointOrigin:       "https://agent.example",
		EndpointPath:         "/a2a",
		Audience:             "https://agent.example",
		DiscoveryOwner:       DiscoveryOwnerGateway,
		BackendRef:           mustBackendRef(t, "backend/a"),
		RequiredCapabilities: required,
	})

	required[0] = CapabilityDrain
	returned := spec.RequiredCapabilities().Values()
	if len(returned) != 2 || returned[0] != CapabilityForwarding || returned[1] != CapabilityRetryPolicyControl {
		t.Fatalf("requirements changed through input alias: %v", returned)
	}
	returned[0] = CapabilityDrain
	if got := spec.RequiredCapabilities().Values()[0]; got != CapabilityForwarding {
		t.Fatalf("requirements changed through returned slice: %q", got)
	}
	if spec.CanonicalEndpoint() != "https://agent.example/a2a" || spec.Audience() != "https://agent.example" {
		t.Fatalf("canonical route facts = %q / %q", spec.CanonicalEndpoint(), spec.Audience())
	}
}

func TestRouteSpecRejectsNoncanonicalOrMismatchedEndpointFacts(t *testing.T) {
	cases := map[string]func(*RouteSpecInput){
		"mismatched audience": func(input *RouteSpecInput) { input.Audience = "https://other.example" },
		"uppercase origin":    func(input *RouteSpecInput) { input.EndpointOrigin = "https://AGENT.example" },
		"default origin port": func(input *RouteSpecInput) { input.EndpointOrigin = "https://agent.example:443" },
		"path query":          func(input *RouteSpecInput) { input.EndpointPath = "/a2a?x=1" },
		"escaped path":        func(input *RouteSpecInput) { input.EndpointPath = "/a%32a" },
		"oversized endpoint":  func(input *RouteSpecInput) { input.EndpointPath = "/" + strings.Repeat("a", 2048) },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			input := validRouteInput(t)
			mutate(&input)
			if _, err := NewRouteSpec(input); !errors.Is(err, ErrInvalid) {
				t.Fatalf("error = %v, want invalid", err)
			}
		})
	}
}

func TestRouteSpecPreservesExplicitRouterOwnedDiscovery(t *testing.T) {
	input := validRouteInput(t)
	input.DiscoveryOwner = DiscoveryOwnerRouter
	spec, err := NewRouteSpec(input)
	if err != nil {
		t.Fatalf("NewRouteSpec: %v", err)
	}
	if spec.DiscoveryOwner() != DiscoveryOwnerRouter {
		t.Fatalf("discovery owner = %q, want router", spec.DiscoveryOwner())
	}
}

func TestRouteStatusRequiresDistinctObservedRevisionForStale(t *testing.T) {
	key := mustRouteKey(t, "route-a")
	desired := mustRouteRevision(t, "desired-1")
	same := desired
	if _, err := NewRouteStatus(RouteStatusInput{Key: key, State: RouteStateStaleRevision, DesiredRevision: desired, ObservedRevision: &same}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("same revision stale status = %v, want invalid", err)
	}
	observed := mustRouteRevision(t, "observed-1")
	status, err := NewRouteStatus(RouteStatusInput{Key: key, State: RouteStateStaleRevision, DesiredRevision: desired, ObservedRevision: &observed})
	if err != nil {
		t.Fatalf("stale status: %v", err)
	}
	got, ok := status.ObservedRevision()
	if !ok || !got.Equal(observed) {
		t.Fatalf("observed revision = %q, %v", got, ok)
	}
}

func validRouteInput(t testing.TB) RouteSpecInput {
	t.Helper()
	return RouteSpecInput{
		Key:            mustRouteKey(t, "route-a"),
		Revision:       mustRouteRevision(t, "rev-1"),
		ReleaseID:      "release-a",
		CardDigest:     strings.Repeat("a", 64),
		AgentID:        "agent-a",
		AgentVersion:   "1.0.0",
		EndpointOrigin: "https://agent.example",
		EndpointPath:   "/a2a",
		Audience:       "https://agent.example",
		DiscoveryOwner: DiscoveryOwnerGateway,
		BackendRef:     mustBackendRef(t, "backend-a"),
	}
}

func mustRouteSpec(t testing.TB, input RouteSpecInput) RouteSpec {
	t.Helper()
	spec, err := NewRouteSpec(input)
	if err != nil {
		t.Fatalf("NewRouteSpec: %v", err)
	}
	return spec
}

func mustRouteKey(t testing.TB, value string) RouteKey {
	t.Helper()
	key, err := NewRouteKey(value)
	if err != nil {
		t.Fatalf("NewRouteKey: %v", err)
	}
	return key
}

func mustRouteRevision(t testing.TB, value string) RouteRevision {
	t.Helper()
	revision, err := NewRouteRevision(value)
	if err != nil {
		t.Fatalf("NewRouteRevision: %v", err)
	}
	return revision
}

func mustBackendRef(t testing.TB, value string) BackendRef {
	t.Helper()
	ref, err := NewBackendRef(value)
	if err != nil {
		t.Fatalf("NewBackendRef: %v", err)
	}
	return ref
}
