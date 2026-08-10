package routing

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/NeKiro-project/NeKiro/apps/a2a-router/internal/transport/a2a"
	"github.com/NeKiro-project/NeKiro/contracts"
	"github.com/NeKiro-project/NeKiro/registry"
	"github.com/NeKiro-project/NeKiro/registry/testkit"
)

type countingDirectory struct {
	*testkit.FakeDirectory
	observes  atomic.Int32
	snapshots atomic.Int32
}

type failingObserveDirectory struct {
	capabilityDirectory
	observes atomic.Int32
}

type mismatchedInitialDirectory struct {
	capabilityDirectory
	initial registry.InstanceSnapshot
	watch   registry.InstanceWatch
}

type blockingObserveDirectory struct {
	capabilityDirectory
	started     chan struct{}
	release     chan struct{}
	observation registry.InstanceObservation
}

func (directory *blockingObserveDirectory) Observe(context.Context, registry.ReleaseTarget) (registry.InstanceObservation, error) {
	close(directory.started)
	<-directory.release
	return directory.observation, nil
}

func (directory *mismatchedInitialDirectory) Observe(context.Context, registry.ReleaseTarget) (registry.InstanceObservation, error) {
	return registry.NewInstanceObservation(directory.initial, directory.watch)
}

func (directory *failingObserveDirectory) Observe(context.Context, registry.ReleaseTarget) (registry.InstanceObservation, error) {
	directory.observes.Add(1)
	return registry.InstanceObservation{}, registry.ErrUnavailable
}

func (directory *countingDirectory) Observe(ctx context.Context, target registry.ReleaseTarget) (registry.InstanceObservation, error) {
	directory.observes.Add(1)
	return directory.FakeDirectory.Observe(ctx, target)
}

func (directory *countingDirectory) Snapshot(ctx context.Context, target registry.ReleaseTarget) (registry.InstanceSnapshot, error) {
	directory.snapshots.Add(1)
	return directory.FakeDirectory.Snapshot(ctx, target)
}

func TestWatchSelectorSharesObservationAndAppliesChanges(t *testing.T) {
	target, initial := selectionFixture(t, []string{"runtime-b-old"})
	directory := newCountingDirectory(t, target, initial)
	selector, err := NewWatchSelector(directory, "nacos", "a2a", 4)
	if err != nil {
		t.Fatal(err)
	}
	defer selector.Close()

	input := a2aTarget(target)
	const callers = 16
	results := make(chan a2a.Target, callers)
	errors := make(chan error, callers)
	var group sync.WaitGroup
	for range callers {
		group.Add(1)
		go func() {
			defer group.Done()
			selected, selectErr := selector.Select(t.Context(), input, a2a.ContextHeaders{})
			results <- selected
			errors <- selectErr
		}()
	}
	group.Wait()
	close(results)
	close(errors)
	for selectErr := range errors {
		if selectErr != nil {
			t.Fatalf("Select failed: %v", selectErr)
		}
	}
	for selected := range results {
		if selected.Endpoint != "http://runtime-b-old:8092" {
			t.Fatalf("initial selected endpoint=%s", selected.Endpoint)
		}
	}
	if directory.observes.Load() != 1 || directory.snapshots.Load() != 0 {
		t.Fatalf("directory calls: observe=%d snapshot=%d", directory.observes.Load(), directory.snapshots.Load())
	}

	change := replacementChange(t, target, initial, "runtime-b-new")
	if err := directory.Emit(target, change); err != nil {
		t.Fatal(err)
	}
	eventually(t, func() bool {
		selected, selectErr := selector.Select(t.Context(), input, a2a.ContextHeaders{})
		return selectErr == nil && selected.Endpoint == "http://runtime-b-new:8092"
	})
	if directory.observes.Load() != 1 || directory.snapshots.Load() != 0 {
		t.Fatalf("directory calls after change: observe=%d snapshot=%d", directory.observes.Load(), directory.snapshots.Load())
	}
}

func TestWatchSelectorFailsClosedAfterWatchTermination(t *testing.T) {
	target, initial := selectionFixture(t, []string{"runtime-b-old"})
	directory := newCountingDirectory(t, target, initial)
	selector, _ := NewWatchSelector(directory, "nacos", "a2a", 1)
	defer selector.Close()
	input := a2aTarget(target)
	if _, err := selector.Select(t.Context(), input, a2a.ContextHeaders{}); err != nil {
		t.Fatal(err)
	}
	if err := directory.Terminate(target, registry.NewOutcomeError(registry.OutcomeWatchInterrupted, registry.CauseStreamEOF)); err != nil {
		t.Fatal(err)
	}
	eventually(t, func() bool {
		_, selectErr := selector.Select(t.Context(), input, a2a.ContextHeaders{})
		return selectErr != nil
	})
	status := selector.TopologyStatus()
	if len(status.Observations) != 1 || status.Observations[0].State != contracts.RouterTopologyStateUnavailable || status.Observations[0].LocalRevision != 0 {
		t.Fatalf("terminal observation status = %#v", status)
	}
	if directory.snapshots.Load() != 0 {
		t.Fatalf("terminal watch fell back to Snapshot %d times", directory.snapshots.Load())
	}
}

func TestWatchSelectorEnforcesObservationLimitAndClose(t *testing.T) {
	first, firstSnapshot := selectionFixture(t, []string{"runtime-b-old"})
	directory := newCountingDirectory(t, first, firstSnapshot)
	second, secondSnapshot := releaseFixture(t, "runtime-c", "release-c", "runtime-c-old", 8093)
	if err := directory.Bind(second, secondSnapshot); err != nil {
		t.Fatal(err)
	}
	selector, _ := NewWatchSelector(directory, "nacos", "a2a", 1)
	if _, err := selector.Select(t.Context(), a2aTarget(first), a2a.ContextHeaders{}); err != nil {
		t.Fatal(err)
	}
	if _, err := selector.Select(t.Context(), a2aTarget(second), a2a.ContextHeaders{}); err == nil {
		t.Fatal("observation limit accepted another exact Release")
	}
	if err := selector.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := selector.Select(t.Context(), a2aTarget(first), a2a.ContextHeaders{}); err == nil {
		t.Fatal("closed selector accepted selection")
	}
	if err := selector.Close(); err != nil {
		t.Fatalf("second Close failed: %v", err)
	}
}

func TestWatchSelectorLatchesInitializationFailure(t *testing.T) {
	capabilities, _ := registry.NewCapabilities(registry.CapabilityObserve)
	directory := &failingObserveDirectory{capabilityDirectory: capabilityDirectory{capabilities: capabilities}}
	selector, _ := NewWatchSelector(directory, "nacos", "a2a", 1)
	defer selector.Close()
	target, _ := selectionFixture(t, []string{"runtime-b-old"})
	for range 2 {
		if _, err := selector.Select(t.Context(), a2aTarget(target), a2a.ContextHeaders{}); err == nil {
			t.Fatal("failed observation produced a target")
		}
	}
	if directory.observes.Load() != 1 {
		t.Fatalf("failed observation retried %d times", directory.observes.Load())
	}
}

func TestWatchSelectorRejectsMismatchedInitialSnapshot(t *testing.T) {
	target, _ := selectionFixture(t, []string{"runtime-b-old"})
	_, otherInitial := releaseFixture(t, "runtime-c", "release-c", "runtime-c-old", 8093)
	watch, _, err := registry.NewInstanceWatch(1)
	if err != nil {
		t.Fatal(err)
	}
	directory := &mismatchedInitialDirectory{
		capabilityDirectory: capabilityDirectory{capabilities: func() registry.Capabilities {
			capabilities, _ := registry.NewCapabilities(registry.CapabilityObserve)
			return capabilities
		}()},
		initial: otherInitial,
		watch:   watch,
	}
	selector, err := NewWatchSelector(directory, "nacos", "a2a", 1)
	if err != nil {
		t.Fatal(err)
	}
	defer selector.Close()
	if _, err := selector.Select(t.Context(), a2aTarget(target), a2a.ContextHeaders{}); err == nil {
		t.Fatal("selector accepted an initial snapshot for another Release")
	}
}

func TestObservationSessionRejectsRevisionGap(t *testing.T) {
	target, initial := selectionFixture(t, []string{"runtime-b-old"})
	session := &observationSession{snapshot: initial, available: true}
	if session.apply(target, replacementChangeAtOrder(t, target, initial, "runtime-b-new", 2), time.Now()) {
		t.Fatal("revision gap was applied")
	}
	current, available := session.current()
	if !available || !current.Equal(initial) {
		t.Fatal("rejected revision changed the active snapshot")
	}
}

func TestWatchSelectorValidatesDependenciesAndContext(t *testing.T) {
	capabilities, _ := registry.NewCapabilities(registry.CapabilitySnapshot)
	if _, err := NewWatchSelector(capabilityDirectory{capabilities: capabilities}, "nacos", "a2a", 1); err == nil {
		t.Fatal("directory without observe capability accepted")
	}
	if _, err := NewWatchSelector(nil, "nacos", "a2a", 1); err == nil {
		t.Fatal("nil directory accepted")
	}
	observeCapabilities, _ := registry.NewCapabilities(registry.CapabilityObserve)
	if _, err := NewWatchSelector(capabilityDirectory{capabilities: observeCapabilities}, "nacos", "a2a", contracts.RouterTopologyStatusObservationMaximum+1); err == nil {
		t.Fatal("observation limit beyond status contract accepted")
	}
	if _, err := NewWatchSelector(capabilityDirectory{capabilities: func() registry.Capabilities {
		capabilities, _ := registry.NewCapabilities(registry.CapabilityObserve)
		return capabilities
	}()}, "not safe", "a2a", 1); err == nil {
		t.Fatal("invalid provider accepted")
	}
	target, initial := selectionFixture(t, []string{"runtime-b-old"})
	directory := newCountingDirectory(t, target, initial)
	selector, _ := NewWatchSelector(directory, "nacos", "a2a", 1)
	defer selector.Close()
	if _, err := selector.Select(nil, a2aTarget(target), a2a.ContextHeaders{}); err == nil {
		t.Fatal("nil context accepted")
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := selector.Select(canceled, a2aTarget(target), a2a.ContextHeaders{}); err == nil {
		t.Fatal("canceled context accepted")
	}
	if directory.observes.Load() != 0 {
		t.Fatal("canceled selection established an observation")
	}
}

func TestWatchSelectorTopologyStatusTracksSafeSortedState(t *testing.T) {
	first, firstInitial := selectionFixture(t, []string{"runtime-b-old"})
	directory := newCountingDirectory(t, first, firstInitial)
	second, secondInitial := releaseFixture(t, "runtime-a", "release-a", "runtime-a-old", 8091)
	if err := directory.Bind(second, secondInitial); err != nil {
		t.Fatal(err)
	}
	third, thirdInitial := releaseFixture(t, "runtime-a", "release-a2", "runtime-a-second", 8091)
	if err := directory.Bind(third, thirdInitial); err != nil {
		t.Fatal(err)
	}
	fourth, fourthInitial := releaseFixtureVersion(t, "runtime-a", "2.0.0", "release-a3", "runtime-a-v2", 8091)
	if err := directory.Bind(fourth, fourthInitial); err != nil {
		t.Fatal(err)
	}
	selector, err := NewWatchSelector(directory, "nacos", "a2a", 4)
	if err != nil {
		t.Fatal(err)
	}
	defer selector.Close()
	var clock atomic.Int64
	clock.Store(time.Date(2026, 8, 10, 5, 0, 0, 0, time.UTC).UnixNano())
	selector.now = func() time.Time { return time.Unix(0, clock.Load()).UTC() }

	if _, err := selector.Select(t.Context(), a2aTarget(first), a2a.ContextHeaders{}); err != nil {
		t.Fatal(err)
	}
	if _, err := selector.Select(t.Context(), a2aTarget(second), a2a.ContextHeaders{}); err != nil {
		t.Fatal(err)
	}
	if _, err := selector.Select(t.Context(), a2aTarget(third), a2a.ContextHeaders{}); err != nil {
		t.Fatal(err)
	}
	if _, err := selector.Select(t.Context(), a2aTarget(fourth), a2a.ContextHeaders{}); err != nil {
		t.Fatal(err)
	}
	status := selector.TopologyStatus()
	if err := contracts.ValidateRouterTopologyStatusV1(status); err != nil {
		t.Fatalf("status contract invalid: %v", err)
	}
	if status.Provider != "nacos" || len(status.Observations) != 4 || status.Observations[0].ReleaseID != "release-a" ||
		status.Observations[1].ReleaseID != "release-a2" || status.Observations[2].AgentCardVersion != "2.0.0" ||
		status.Observations[3].AgentID != first.AgentID() || status.Observations[3].State != contracts.RouterTopologyStatePopulated {
		t.Fatalf("initial sorted status = %#v", status)
	}

	clock.Store(time.Date(2026, 8, 10, 5, 1, 0, 0, time.UTC).UnixNano())
	if err := directory.Emit(first, emptyChange(t, first, firstInitial)); err != nil {
		t.Fatal(err)
	}
	eventually(t, func() bool {
		current := selector.TopologyStatus()
		return len(current.Observations) == 4 && current.Observations[3].State == contracts.RouterTopologyStateEmpty &&
			current.Observations[3].LocalRevision == 1 && current.Observations[3].ObservedAt.Equal(time.Unix(0, clock.Load()).UTC())
	})

	encoded, err := json.Marshal(selector.TopologyStatus())
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"runtime-b-old", "stack-1", "cardDigest", "canonicalEndpoint", "audience", "instanceId"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("topology status leaked %q: %s", forbidden, encoded)
		}
	}
}

func TestWatchSelectorTopologyStatusIsReadOnly(t *testing.T) {
	target, initial := selectionFixture(t, []string{"runtime-b-old"})
	directory := newCountingDirectory(t, target, initial)
	selector, err := NewWatchSelector(directory, "nacos", "a2a", 1)
	if err != nil {
		t.Fatal(err)
	}
	defer selector.Close()
	for range 3 {
		status := selector.TopologyStatus()
		if status.Observations == nil || len(status.Observations) != 0 {
			t.Fatalf("unobserved status = %#v", status)
		}
	}
	if directory.observes.Load() != 0 || directory.snapshots.Load() != 0 {
		t.Fatalf("status read touched directory: observe=%d snapshot=%d", directory.observes.Load(), directory.snapshots.Load())
	}
}

func TestWatchSelectorTopologyStatusExposesInitializingWithoutProbing(t *testing.T) {
	target, initial := selectionFixture(t, []string{"runtime-b-old"})
	watch, _, err := registry.NewInstanceWatch(1)
	if err != nil {
		t.Fatal(err)
	}
	observation, err := registry.NewInstanceObservation(initial, watch)
	if err != nil {
		t.Fatal(err)
	}
	capabilities, _ := registry.NewCapabilities(registry.CapabilityObserve)
	directory := &blockingObserveDirectory{
		capabilityDirectory: capabilityDirectory{capabilities: capabilities},
		started:             make(chan struct{}), release: make(chan struct{}), observation: observation,
	}
	selector, err := NewWatchSelector(directory, "nacos", "a2a", 1)
	if err != nil {
		t.Fatal(err)
	}
	defer selector.Close()
	result := make(chan error, 1)
	go func() {
		_, selectErr := selector.Select(t.Context(), a2aTarget(target), a2a.ContextHeaders{})
		result <- selectErr
	}()
	<-directory.started
	status := selector.TopologyStatus()
	if len(status.Observations) != 1 || status.Observations[0].State != contracts.RouterTopologyStateInitializing ||
		status.Observations[0].LocalRevision != 0 || status.Observations[0].ObservedAt.IsZero() {
		t.Fatalf("initializing status = %#v", status)
	}
	close(directory.release)
	if err := <-result; err != nil {
		t.Fatal(err)
	}
}

func TestTopologyObservationStateMapping(t *testing.T) {
	for input, want := range map[registry.SnapshotState]contracts.RouterTopologyObservationState{
		registry.SnapshotStateMissing:     contracts.RouterTopologyStateMissing,
		registry.SnapshotStateEmpty:       contracts.RouterTopologyStateEmpty,
		registry.SnapshotStatePopulated:   contracts.RouterTopologyStatePopulated,
		registry.SnapshotState("invalid"): contracts.RouterTopologyStateUnavailable,
	} {
		if got := topologyObservationState(input); got != want {
			t.Fatalf("state %q mapped to %q, want %q", input, got, want)
		}
	}
}

func newCountingDirectory(t *testing.T, target registry.ReleaseTarget, snapshot registry.InstanceSnapshot) *countingDirectory {
	t.Helper()
	capabilities, _ := registry.NewCapabilities(registry.CapabilitySnapshot, registry.CapabilityObserve)
	fake, err := testkit.NewFakeDirectory(testkit.FakeConfig{Capabilities: capabilities, QueueCapacity: 4})
	if err != nil {
		t.Fatal(err)
	}
	if err := fake.Bind(target, snapshot); err != nil {
		t.Fatal(err)
	}
	return &countingDirectory{FakeDirectory: fake}
}

func replacementChange(t *testing.T, target registry.ReleaseTarget, previous registry.InstanceSnapshot, address string) registry.InstanceChange {
	return replacementChangeAtOrder(t, target, previous, address, 1)
}

func replacementChangeAtOrder(t *testing.T, target registry.ReleaseTarget, previous registry.InstanceSnapshot, address string, order uint64) registry.InstanceChange {
	t.Helper()
	endpoint, _ := registry.NewNetworkEndpoint(registry.NetworkEndpointInput{AddressType: registry.AddressTypeDNS, Address: address, PortName: "a2a", Port: 8092, Protocol: registry.TransportProtocolTCP})
	instance, _ := registry.NewInstance(registry.InstanceInput{ID: address, Endpoints: []registry.NetworkEndpoint{endpoint}, Ready: true, Serving: true})
	revision, _ := registry.NewRevision(registry.RevisionInput{SourceTokens: []string{"stack-2"}, LocalOrder: order})
	snapshot, err := registry.NewInstanceSnapshot(registry.InstanceSnapshotInput{Target: target, Revision: revision, State: registry.SnapshotStatePopulated, Instances: []registry.Instance{instance}})
	if err != nil {
		t.Fatal(err)
	}
	change, err := registry.NewInstanceChange(registry.InstanceChangeInput{
		Kind: registry.InstanceChangeInstancesChanged, Revision: revision,
		Upserts: []registry.Instance{instance}, DeletedInstanceIDs: []string{previous.Instances()[0].ID()}, Snapshot: snapshot,
	})
	if err != nil {
		t.Fatal(err)
	}
	return change
}

func emptyChange(t *testing.T, target registry.ReleaseTarget, previous registry.InstanceSnapshot) registry.InstanceChange {
	t.Helper()
	revision, err := registry.NewRevision(registry.RevisionInput{SourceTokens: []string{"provider-secret-revision"}, LocalOrder: 1})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := registry.NewInstanceSnapshot(registry.InstanceSnapshotInput{
		Target: target, Revision: revision, State: registry.SnapshotStateEmpty, Instances: []registry.Instance{},
	})
	if err != nil {
		t.Fatal(err)
	}
	change, err := registry.NewInstanceChange(registry.InstanceChangeInput{
		Kind: registry.InstanceChangeInstancesChanged, Revision: revision,
		DeletedInstanceIDs: []string{previous.Instances()[0].ID()}, Snapshot: snapshot,
	})
	if err != nil {
		t.Fatal(err)
	}
	return change
}

func releaseFixture(t *testing.T, agentID, releaseID, address string, port int) (registry.ReleaseTarget, registry.InstanceSnapshot) {
	return releaseFixtureVersion(t, agentID, "1.0.0", releaseID, address, port)
}

func releaseFixtureVersion(t *testing.T, agentID, version, releaseID, address string, port int) (registry.ReleaseTarget, registry.InstanceSnapshot) {
	t.Helper()
	target, err := registry.NewReleaseTarget(registry.ReleaseTargetInput{
		AgentID: agentID, AgentCardVersion: version, ReleaseID: releaseID, CardDigest: strings.Repeat("b", 64),
		CanonicalEndpoint: "http://" + agentID + ":" + strconv.Itoa(port) + "/",
		Audience:          "http://" + agentID + ":" + strconv.Itoa(port),
	})
	if err != nil {
		t.Fatal(err)
	}
	endpoint, _ := registry.NewNetworkEndpoint(registry.NetworkEndpointInput{AddressType: registry.AddressTypeDNS, Address: address, PortName: "a2a", Port: port, Protocol: registry.TransportProtocolTCP})
	instance, _ := registry.NewInstance(registry.InstanceInput{ID: address, Endpoints: []registry.NetworkEndpoint{endpoint}, Ready: true, Serving: true})
	revision, _ := registry.NewRevision(registry.RevisionInput{SourceTokens: []string{"stack-1"}})
	snapshot, err := registry.NewInstanceSnapshot(registry.InstanceSnapshotInput{Target: target, Revision: revision, State: registry.SnapshotStatePopulated, Instances: []registry.Instance{instance}})
	if err != nil {
		t.Fatal(err)
	}
	return target, snapshot
}

func a2aTarget(target registry.ReleaseTarget) a2a.Target {
	return a2a.Target{
		AgentID: target.AgentID(), Version: target.AgentCardVersion(), ReleaseID: target.ReleaseID(), CardDigest: target.CardDigest(),
		Endpoint: strings.TrimSuffix(target.CanonicalEndpoint(), "/"), Audience: target.Audience(),
	}
}

func eventually(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition did not become true")
}
