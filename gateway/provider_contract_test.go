package gateway

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestProviderNeutralValueObjectsAndResults(t *testing.T) {
	name, err := NewProviderName("envoy-v1")
	if err != nil || name.String() != "envoy-v1" {
		t.Fatalf("provider name = %q, %v", name, err)
	}
	key := mustRouteKey(t, "route-a")
	revision := mustRouteRevision(t, "rev-1")
	observed := mustRouteRevision(t, "provider-1")
	status, err := NewRouteStatus(RouteStatusInput{Key: key, State: RouteStateProgrammed, DesiredRevision: revision, ObservedRevision: &observed})
	if err != nil {
		t.Fatalf("NewRouteStatus: %v", err)
	}
	if status.Key() != key || status.State() != RouteStateProgrammed || !status.DesiredRevision().Equal(revision) {
		t.Fatalf("status accessors lost identity: %#v", status)
	}
	clone, err := NewReconcileResult(status)
	if err != nil || !clone.Equal(clone) || !clone.Status().Equal(status) {
		t.Fatalf("reconcile result clone = %#v, %v", clone, err)
	}
	if got, ok := clone.ObservedRevision(); !ok || !got.Equal(observed) {
		t.Fatalf("result observed revision = %q, %v", got, ok)
	}
	if clone.Key() != key || clone.State() != RouteStateProgrammed || !clone.DesiredRevision().Equal(revision) || clone.Validate() != nil {
		t.Fatal("reconcile result accessors did not preserve status")
	}
	drain, err := NewDrainRequest(revision)
	if err != nil || !drain.ExpectedDesiredRevision().Equal(revision) || drain.Validate() != nil {
		t.Fatalf("drain request = %#v, %v", drain, err)
	}
	deleteRequest, err := NewDeleteRequest(revision)
	if err != nil || !deleteRequest.ExpectedDesiredRevision().Equal(revision) || deleteRequest.Validate() != nil {
		t.Fatalf("delete request = %#v, %v", deleteRequest, err)
	}
}

func TestRouteStatusAcceptsEveryLifecycleStateAndRejectsInvalidInputs(t *testing.T) {
	key := mustRouteKey(t, "route-a")
	revision := mustRouteRevision(t, "rev-1")
	for _, state := range []RouteState{RouteStateAbsent, RouteStateAccepted, RouteStateProgrammed, RouteStateNotReady, RouteStateRejected, RouteStateEmptyUpstream, RouteStateDraining, RouteStateDeleting, RouteStateDeleted, RouteStateStaleRevision} {
		observed := mustRouteRevision(t, "provider-1")
		if state != RouteStateStaleRevision {
			if _, err := NewRouteStatus(RouteStatusInput{Key: key, State: state, DesiredRevision: revision}); err != nil {
				t.Errorf("state %q: %v", state, err)
			}
			continue
		}
		if _, err := NewRouteStatus(RouteStatusInput{Key: key, State: state, DesiredRevision: revision, ObservedRevision: &observed}); err != nil {
			t.Errorf("state %q: %v", state, err)
		}
	}
	bad := []RouteStatusInput{{Key: key, State: RouteState("bogus"), DesiredRevision: revision}, {Key: key, State: RouteStateStaleRevision, DesiredRevision: revision}, {Key: key, State: RouteStateStaleRevision, DesiredRevision: revision, ObservedRevision: &revision}}
	for i, input := range bad {
		if _, err := NewRouteStatus(input); !errors.Is(err, ErrInvalid) {
			t.Errorf("bad status %d error = %v, want invalid", i, err)
		}
	}
	withoutObserved, err := NewRouteStatus(RouteStatusInput{Key: key, State: RouteStateAccepted, DesiredRevision: revision})
	if err != nil {
		t.Fatalf("status without observation: %v", err)
	}
	if _, ok := withoutObserved.ObservedRevision(); ok || withoutObserved.Equal(RouteStatus{}) || statusEqualWithDifferentState(t, withoutObserved, revision) {
		t.Fatal("status equality collapsed distinct observations or states")
	}
	if _, err := NewDrainRequest(RouteRevision{}); !errors.Is(err, ErrInvalid) {
		t.Errorf("empty drain revision = %v", err)
	}
	if _, err := NewDeleteRequest(RouteRevision{}); !errors.Is(err, ErrInvalid) {
		t.Errorf("empty delete revision = %v", err)
	}
}

func statusEqualWithDifferentState(t testing.TB, status RouteStatus, revision RouteRevision) bool {
	t.Helper()
	other, err := NewRouteStatus(RouteStatusInput{Key: status.Key(), State: RouteStateDeleting, DesiredRevision: revision})
	if err != nil {
		t.Fatalf("different status: %v", err)
	}
	return status.Equal(other)
}

func TestRouteAndReleaseAccessorsAndEquality(t *testing.T) {
	input := validRouteInput(t)
	input.RequiredCapabilities = []Capability{CapabilityRetryPolicyControl}
	spec := mustRouteSpec(t, input)
	if spec.Key().Value() != "route-a" || spec.Key().String() != "route-a" || spec.Revision().Value() != "rev-1" || spec.Revision().String() != "rev-1" {
		t.Fatal("route accessors did not preserve values")
	}
	if spec.ReleaseID() != "release-a" || spec.CardDigest() != strings.Repeat("a", 64) || spec.Release().ReleaseID() != "release-a" || spec.Release().CardDigest() == "" {
		t.Fatal("release accessors did not preserve provenance")
	}
	if spec.AgentID() != "agent-a" || spec.AgentVersion() != "1.0.0" || spec.EndpointOrigin() != "https://agent.example" || spec.EndpointPath() != "/a2a" || spec.BackendRef().String() != "backend-a" {
		t.Fatal("route accessors did not preserve facts")
	}
	if !spec.Equal(spec) || !spec.Release().Equal(spec.Release()) || !spec.Key().Equal(spec.Key()) || !spec.Revision().Equal(spec.Revision()) {
		t.Fatal("identical values should compare equal")
	}
	otherInput := input
	otherInput.BackendRef = mustBackendRef(t, "backend-b")
	other := mustRouteSpec(t, otherInput)
	if spec.Equal(other) || !spec.Release().Equal(other.Release()) {
		t.Fatal("route equality did not distinguish backend from release provenance")
	}
}

func TestOutcomeClassificationAndCancellation(t *testing.T) {
	for _, outcome := range []Outcome{OutcomeInvalid, OutcomeUnsupported, OutcomeUnauthorized, OutcomeUnavailable, OutcomeRejected, OutcomeNotReady, OutcomeNotFound, OutcomeStale, OutcomeCanceled, OutcomeClosed} {
		err := NewOutcomeError(outcome, CauseNone)
		if got, ok := OutcomeOf(fmt.Errorf("wrapped: %w", err)); !ok || got != outcome || !IsOutcome(err, outcome) {
			t.Errorf("outcome %q = %q, %v", outcome, got, ok)
		}
	}
	if got := (*OutcomeError)(nil).Error(); got != "<nil>" || (*OutcomeError)(nil).Outcome() != "" || (*OutcomeError)(nil).Cause() != "" || (*OutcomeError)(nil).Unwrap() != nil {
		t.Fatal("nil OutcomeError methods are not safe")
	}
	if _, ok := OutcomeOf(nil); ok || IsOutcome(errors.New("plain"), OutcomeInvalid) || IsOutcome(nil, Outcome("bogus")) {
		t.Fatal("plain/unknown errors incorrectly classified")
	}
	if err := NewCanceledError(context.Canceled); !errors.Is(err, context.Canceled) || !errors.Is(err, ErrCanceled) {
		t.Fatalf("canceled error = %v", err)
	}
}
