package routing

import (
	"strings"
	"testing"

	"github.com/NeKiro-project/NeKiro/apps/a2a-router/internal/transport/a2a"
	"github.com/NeKiro-project/NeKiro/registry"
	"github.com/NeKiro-project/NeKiro/registry/testkit"
)

func TestSnapshotSelectorPinsOneReadyEndpointAndPreservesIdentity(t *testing.T) {
	target, snapshot := selectionFixture(t, []string{"runtime-b-directory"})
	capabilities, _ := registry.NewCapabilities(registry.CapabilitySnapshot, registry.CapabilityObserve)
	directory, err := testkit.NewFakeDirectory(testkit.FakeConfig{Capabilities: capabilities, QueueCapacity: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := directory.Bind(target, snapshot); err != nil {
		t.Fatal(err)
	}
	selector, err := NewSnapshotSelector(directory, "a2a")
	if err != nil {
		t.Fatal(err)
	}
	input := a2a.Target{AgentID: target.AgentID(), Version: target.AgentCardVersion(), ReleaseID: target.ReleaseID(), CardDigest: target.CardDigest(), Endpoint: "http://runtime-b:8092", Audience: target.Audience()}
	selected, err := selector.Select(t.Context(), input, a2a.ContextHeaders{InvocationID: "inv-1"})
	if err != nil {
		t.Fatalf("Select failed: %v", err)
	}
	if selected.Endpoint != "http://runtime-b-directory:8092" || selected.Audience != input.Audience || selected.ReleaseID != input.ReleaseID {
		t.Fatalf("selected=%#v", selected)
	}
}

func TestSnapshotSelectorRejectsAmbiguousReadySet(t *testing.T) {
	target, snapshot := selectionFixture(t, []string{"runtime-b-a", "runtime-b-b"})
	capabilities, _ := registry.NewCapabilities(registry.CapabilitySnapshot, registry.CapabilityObserve)
	directory, _ := testkit.NewFakeDirectory(testkit.FakeConfig{Capabilities: capabilities, QueueCapacity: 1})
	_ = directory.Bind(target, snapshot)
	selector, _ := NewSnapshotSelector(directory, "a2a")
	_, err := selector.Select(t.Context(), a2a.Target{AgentID: target.AgentID(), Version: target.AgentCardVersion(), ReleaseID: target.ReleaseID(), CardDigest: target.CardDigest(), Endpoint: "http://runtime-b:8092", Audience: target.Audience()}, a2a.ContextHeaders{})
	if err == nil {
		t.Fatal("ambiguous ready set accepted")
	}
}

func selectionFixture(t *testing.T, addresses []string) (registry.ReleaseTarget, registry.InstanceSnapshot) {
	t.Helper()
	target, err := registry.NewReleaseTarget(registry.ReleaseTargetInput{AgentID: "runtime-b", AgentCardVersion: "1.0.0", ReleaseID: "release-b", CardDigest: strings.Repeat("a", 64), CanonicalEndpoint: "http://runtime-b:8092/", Audience: "http://runtime-b:8092"})
	if err != nil {
		t.Fatal(err)
	}
	instances := make([]registry.Instance, 0, len(addresses))
	for _, address := range addresses {
		endpoint, _ := registry.NewNetworkEndpoint(registry.NetworkEndpointInput{AddressType: registry.AddressTypeDNS, Address: address, PortName: "a2a", Port: 8092, Protocol: registry.TransportProtocolTCP})
		instance, instanceErr := registry.NewInstance(registry.InstanceInput{ID: address, Endpoints: []registry.NetworkEndpoint{endpoint}, Ready: true, Serving: true})
		if instanceErr != nil {
			t.Fatal(instanceErr)
		}
		instances = append(instances, instance)
	}
	revision, _ := registry.NewRevision(registry.RevisionInput{SourceTokens: []string{"stack-1"}})
	snapshot, err := registry.NewInstanceSnapshot(registry.InstanceSnapshotInput{Target: target, Revision: revision, State: registry.SnapshotStatePopulated, Instances: instances})
	if err != nil {
		t.Fatal(err)
	}
	return target, snapshot
}
