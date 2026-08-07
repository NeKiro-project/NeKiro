package registry

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestCapabilitiesValidateCopyAndCompare(t *testing.T) {
	all := []Capability{
		CapabilitySnapshot,
		CapabilityObserve,
		CapabilityRegistration,
		CapabilityDeregistration,
		CapabilityLease,
		CapabilityHeartbeat,
	}
	capabilities, err := NewCapabilities(all...)
	if err != nil {
		t.Fatalf("NewCapabilities: %v", err)
	}
	want := []Capability{
		CapabilityDeregistration,
		CapabilityHeartbeat,
		CapabilityLease,
		CapabilityObserve,
		CapabilityRegistration,
		CapabilitySnapshot,
	}
	if got := capabilities.Values(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Values = %v, want %v", got, want)
	}
	values := capabilities.Values()
	values[0] = "changed"
	if !capabilities.Supports(CapabilitySnapshot) || capabilities.Supports("unknown") || !reflect.DeepEqual(capabilities.Values(), want) {
		t.Fatal("Capabilities did not preserve an immutable exact set")
	}
	copySet, err := NewCapabilities(all...)
	if err != nil || !capabilities.Equal(copySet) || !copySet.Equal(capabilities) {
		t.Fatalf("equal capabilities = %v / %v", err, capabilities.Equal(copySet))
	}
	subset, err := NewCapabilities(CapabilitySnapshot)
	if err != nil {
		t.Fatalf("NewCapabilities subset: %v", err)
	}
	if capabilities.Equal(subset) || subset.Equal(capabilities) {
		t.Fatal("different capability sets compare equal")
	}
	for name, values := range map[string][]Capability{
		"unknown":   {"unknown"},
		"duplicate": {CapabilityObserve, CapabilityObserve},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := NewCapabilities(values...); !errors.Is(err, ErrInvalid) {
				t.Fatalf("NewCapabilities error = %v, want invalid", err)
			}
		})
	}
}

func TestInstanceObservationConstructionAndAccessors(t *testing.T) {
	target := mustTarget(t, validTargetInput())
	initial := mustSnapshot(t, target, mustRevision(t, []string{"service-rv", "slice-rv"}, 0), SnapshotStateEmpty, nil)
	watch, _, err := NewInstanceWatch(1)
	if err != nil {
		t.Fatalf("NewInstanceWatch: %v", err)
	}
	observation, err := NewInstanceObservation(initial, watch)
	if err != nil {
		t.Fatalf("NewInstanceObservation: %v", err)
	}
	if !observation.Initial().Equal(initial) || observation.Watch() != watch {
		t.Fatal("observation accessors changed the supplied values")
	}
	if _, err := NewInstanceObservation(InstanceSnapshot{}, watch); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid initial error = %v, want invalid", err)
	}
	if _, err := NewInstanceObservation(initial, nil); !errors.Is(err, ErrInvalid) {
		t.Fatalf("nil watch error = %v, want invalid", err)
	}
}

func TestOutcomeErrorAccessorsAndSanitization(t *testing.T) {
	var nilOutcome *OutcomeError
	if nilOutcome.Error() != "<nil>" || nilOutcome.Unwrap() != nil || nilOutcome.Outcome() != "" || nilOutcome.Code() != "" || nilOutcome.Cause() != "" {
		t.Fatal("nil OutcomeError accessors returned non-zero values")
	}
	if ErrClosed.Error() != "instance registry: closed" {
		t.Fatalf("closed error text = %q", ErrClosed.Error())
	}
	err := NewOutcomeError(OutcomeUnavailable, CauseRateLimited)
	if err.Outcome() != OutcomeUnavailable || err.Code() != OutcomeUnavailable || err.Cause() != CauseRateLimited || err.Unwrap() != nil {
		t.Fatal("OutcomeError accessors changed the typed result")
	}
	if got := NewOutcomeError("unknown", CauseNone); !errors.Is(got, ErrInvalid) || got.Cause() != CauseUnknownOutcome {
		t.Fatalf("unknown outcome = %v/%q", got, got.Cause())
	}
	canceled := newCanceledError(context.DeadlineExceeded)
	if !errors.Is(canceled, ErrCanceled) || !errors.Is(canceled, context.DeadlineExceeded) {
		t.Fatalf("canceled error = %v", canceled)
	}
	if got, ok := OutcomeOf(errors.New("plain")); ok || got != "" {
		t.Fatalf("OutcomeOf plain error = %q/%v", got, ok)
	}
	if IsOutcome(ErrClosed, "unknown") {
		t.Fatal("IsOutcome accepted an unknown outcome")
	}
}

func TestNetworkEndpointValidationOrderingAndAccessors(t *testing.T) {
	valid := NetworkEndpointInput{AddressType: AddressTypeIPv4, Address: "10.0.0.1", PortName: "a2a", Port: 8080, Protocol: TransportProtocolTCP}
	endpoint := mustEndpoint(t, valid)
	if endpoint.AddressType() != valid.AddressType || endpoint.Address() != valid.Address || endpoint.PortName() != valid.PortName ||
		endpoint.Port() != valid.Port || endpoint.Protocol() != valid.Protocol {
		t.Fatal("network endpoint accessors changed exact values")
	}
	cases := map[string]NetworkEndpointInput{
		"address type": {Address: valid.Address, PortName: valid.PortName, Port: valid.Port, Protocol: valid.Protocol},
		"address":      {AddressType: valid.AddressType, PortName: valid.PortName, Port: valid.Port, Protocol: valid.Protocol},
		"port name":    {AddressType: valid.AddressType, Address: valid.Address, Port: valid.Port, Protocol: valid.Protocol},
		"low port":     {AddressType: valid.AddressType, Address: valid.Address, PortName: valid.PortName, Port: 0, Protocol: valid.Protocol},
		"high port":    {AddressType: valid.AddressType, Address: valid.Address, PortName: valid.PortName, Port: 65536, Protocol: valid.Protocol},
		"protocol":     {AddressType: valid.AddressType, Address: valid.Address, PortName: valid.PortName, Port: valid.Port},
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := NewNetworkEndpoint(input); !errors.Is(err, ErrInvalid) {
				t.Fatalf("NewNetworkEndpoint error = %v, want invalid", err)
			}
		})
	}
	variants := []NetworkEndpoint{
		mustEndpoint(t, NetworkEndpointInput{AddressType: "A", Address: "x", PortName: "p", Port: 1, Protocol: "P"}),
		mustEndpoint(t, NetworkEndpointInput{AddressType: "B", Address: "x", PortName: "p", Port: 1, Protocol: "P"}),
		mustEndpoint(t, NetworkEndpointInput{AddressType: "B", Address: "y", PortName: "p", Port: 1, Protocol: "P"}),
		mustEndpoint(t, NetworkEndpointInput{AddressType: "B", Address: "y", PortName: "q", Port: 1, Protocol: "P"}),
		mustEndpoint(t, NetworkEndpointInput{AddressType: "B", Address: "y", PortName: "q", Port: 2, Protocol: "P"}),
		mustEndpoint(t, NetworkEndpointInput{AddressType: "B", Address: "y", PortName: "q", Port: 2, Protocol: "Q"}),
	}
	for index := 1; index < len(variants); index++ {
		if compareNetworkEndpoints(variants[index-1], variants[index]) >= 0 || compareNetworkEndpoints(variants[index], variants[index-1]) <= 0 {
			t.Fatalf("endpoint ordering failed at index %d", index)
		}
	}
	if compareNetworkEndpoints(endpoint, endpoint) != 0 {
		t.Fatal("equal endpoint did not compare as equal")
	}
}

func TestInstanceRevisionSnapshotAndChangeAccessors(t *testing.T) {
	target := mustTarget(t, validTargetInput())
	instance := mustInstance(t, "uid-a", true, true, false)
	if !instance.Ready() || !instance.Serving() || instance.Terminating() || instance.State() != LifecycleStateReady || instance.Lifecycle() != LifecycleStateReady {
		t.Fatal("instance lifecycle accessors changed the source conditions")
	}
	if _, ok := instance.Zone(); ok {
		t.Fatal("absent zone reported as present")
	}
	if _, ok := instance.Weight(); ok {
		t.Fatal("absent weight reported as present")
	}
	if !reflect.DeepEqual(instance.SafeMetadata(), instance.Metadata()) {
		t.Fatal("SafeMetadata differs from Metadata")
	}

	revision := mustRevision(t, []string{"service-rv", "slice-rv"}, 1)
	if revision.LocalOrder() != 1 {
		t.Fatalf("LocalOrder = %d, want 1", revision.LocalOrder())
	}
	if _, err := NewRevision(RevisionInput{}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("empty revision error = %v, want invalid", err)
	}
	if _, err := NewRevision(RevisionInput{SourceTokens: []string{" bad"}}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unsafe revision error = %v, want invalid", err)
	}
	if revision.Equal(mustRevision(t, []string{"service-rv"}, 1)) || revision.Equal(mustRevision(t, []string{"service-rv", "other"}, 1)) ||
		revision.Equal(mustRevision(t, []string{"service-rv", "slice-rv"}, 2)) {
		t.Fatal("different revisions compare equal")
	}

	snapshot, err := NewSnapshot(SnapshotInput{Target: target, Revision: revision, State: SnapshotStatePopulated, Instances: []Instance{instance}})
	if err != nil {
		t.Fatalf("NewSnapshot: %v", err)
	}
	if !snapshot.Target().Equal(target) || snapshot.State() != SnapshotStatePopulated {
		t.Fatal("snapshot accessors changed target or state")
	}
	change, err := NewInstanceChange(InstanceChangeInput{
		Kind:     InstanceChangeInstancesChanged,
		Revision: revision,
		Upserts:  []Instance{instance},
		Snapshot: snapshot,
	})
	if err != nil {
		t.Fatalf("NewInstanceChange: %v", err)
	}
	if change.Kind() != InstanceChangeInstancesChanged || !change.Revision().Equal(revision) || !change.Snapshot().Equal(snapshot) {
		t.Fatal("change accessors changed the transition")
	}
}

func TestInstanceValidateRejectsCorruptedRetainedState(t *testing.T) {
	first := mustEndpoint(t, NetworkEndpointInput{AddressType: AddressTypeIPv4, Address: "10.0.0.1", PortName: "a2a", Port: 8080, Protocol: TransportProtocolTCP})
	second := mustEndpoint(t, NetworkEndpointInput{AddressType: AddressTypeIPv4, Address: "10.0.0.2", PortName: "a2a", Port: 8080, Protocol: TransportProtocolTCP})
	valid := mustInstance(t, "uid-a", true, true, false)
	badZone := " bad"
	for name, mutate := range map[string]func(*Instance){
		"id":        func(instance *Instance) { instance.id = "" },
		"endpoints": func(instance *Instance) { instance.endpoints = nil },
		"invalid endpoint": func(instance *Instance) {
			instance.endpoints = []NetworkEndpoint{{}}
		},
		"unsorted endpoints": func(instance *Instance) {
			instance.endpoints = []NetworkEndpoint{second, first}
		},
		"duplicate endpoints": func(instance *Instance) {
			instance.endpoints = []NetworkEndpoint{first, first}
		},
		"derived state": func(instance *Instance) { instance.state = LifecycleStateDraining },
		"zone":          func(instance *Instance) { instance.zone = &badZone },
		"metadata key":  func(instance *Instance) { instance.metadata = map[string]string{" bad": "value"} },
		"metadata value": func(instance *Instance) {
			instance.metadata = map[string]string{"key": " bad"}
		},
	} {
		t.Run(name, func(t *testing.T) {
			corrupted := cloneInstances([]Instance{valid})[0]
			mutate(&corrupted)
			if err := corrupted.Validate(); !errors.Is(err, ErrInvalid) {
				t.Fatalf("Validate error = %v, want invalid", err)
			}
		})
	}
	if _, err := NewInstance(InstanceInput{ID: "uid-a", Endpoints: []NetworkEndpoint{{}}}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("NewInstance invalid endpoint error = %v", err)
	}
	if _, err := NewInstance(InstanceInput{ID: "uid-a", Endpoints: []NetworkEndpoint{first}, Ready: true, Serving: true, State: LifecycleStateDraining}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("NewInstance mismatched state error = %v", err)
	}
}

func TestSnapshotValidateRejectsCorruptedRetainedState(t *testing.T) {
	target := mustTarget(t, validTargetInput())
	revision := mustRevision(t, []string{"service-rv", "slice-rv"}, 0)
	instanceA := mustInstance(t, "uid-a", true, true, false)
	instanceB := mustInstance(t, "uid-b", true, true, false)
	valid := mustSnapshot(t, target, revision, SnapshotStatePopulated, []Instance{instanceA})
	for name, mutate := range map[string]func(*InstanceSnapshot){
		"target":   func(snapshot *InstanceSnapshot) { snapshot.target = ReleaseTarget{} },
		"revision": func(snapshot *InstanceSnapshot) { snapshot.revision = Revision{} },
		"invalid instance": func(snapshot *InstanceSnapshot) {
			snapshot.instances = []Instance{{}}
		},
		"unsorted instances": func(snapshot *InstanceSnapshot) {
			snapshot.instances = []Instance{instanceB, instanceA}
		},
		"duplicate instances": func(snapshot *InstanceSnapshot) {
			snapshot.instances = []Instance{instanceA, instanceA}
		},
		"unknown state": func(snapshot *InstanceSnapshot) { snapshot.state = "unknown" },
		"missing with instances": func(snapshot *InstanceSnapshot) {
			snapshot.state = SnapshotStateMissing
		},
		"populated without instances": func(snapshot *InstanceSnapshot) {
			snapshot.instances = nil
		},
	} {
		t.Run(name, func(t *testing.T) {
			corrupted := cloneSnapshot(valid)
			mutate(&corrupted)
			if err := corrupted.Validate(); !errors.Is(err, ErrInvalid) {
				t.Fatalf("Validate error = %v, want invalid", err)
			}
		})
	}
	if _, err := NewInstanceSnapshot(InstanceSnapshotInput{Target: target, Revision: revision, State: SnapshotStatePopulated, Instances: []Instance{{}}}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("NewInstanceSnapshot invalid instance error = %v", err)
	}
}

func TestChangeValidateRejectsCorruptedRetainedState(t *testing.T) {
	target := mustTarget(t, validTargetInput())
	instance := mustInstance(t, "uid-a", true, true, false)
	revision := mustRevision(t, []string{"service-rv", "slice-rv"}, 1)
	snapshot := mustSnapshot(t, target, revision, SnapshotStatePopulated, []Instance{instance})
	valid, err := NewInstanceChange(InstanceChangeInput{Kind: InstanceChangeInstancesChanged, Revision: revision, Upserts: []Instance{instance}, Snapshot: snapshot})
	if err != nil {
		t.Fatalf("NewInstanceChange: %v", err)
	}
	for name, mutate := range map[string]func(*InstanceChange){
		"revision":    func(change *InstanceChange) { change.revision = Revision{} },
		"local order": func(change *InstanceChange) { change.revision.localOrder = 0 },
		"snapshot":    func(change *InstanceChange) { change.snapshot = InstanceSnapshot{} },
		"revision mismatch": func(change *InstanceChange) {
			change.snapshot.revision.localOrder = 2
		},
		"invalid upsert": func(change *InstanceChange) { change.upserts = []Instance{{}} },
		"duplicate upsert": func(change *InstanceChange) {
			change.upserts = []Instance{instance, instance}
		},
		"invalid deletion": func(change *InstanceChange) { change.deletedInstanceIDs = []string{" bad"} },
		"duplicate deletion": func(change *InstanceChange) {
			change.deletedInstanceIDs = []string{"uid-z", "uid-z"}
		},
		"empty delta":    func(change *InstanceChange) { change.upserts = nil },
		"previous state": func(change *InstanceChange) { change.previousState = SnapshotStateEmpty },
		"unknown kind":   func(change *InstanceChange) { change.kind = "unknown" },
	} {
		t.Run(name, func(t *testing.T) {
			corrupted := cloneChange(valid)
			mutate(&corrupted)
			if err := corrupted.Validate(); !errors.Is(err, ErrInvalid) {
				t.Fatalf("Validate error = %v, want invalid", err)
			}
		})
	}
}
