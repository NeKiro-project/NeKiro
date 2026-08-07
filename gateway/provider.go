package gateway

import "context"

// RouteState is the provider-reported lifecycle state of a known desired
// route. It is separate from Outcome because a route state is not an operation
// failure. In particular, absent means a known desired route is absent at the
// provider, while not_found means the provider has no known route key.
type RouteState string

const (
	RouteStateAbsent        RouteState = "absent"
	RouteStateAccepted      RouteState = "accepted"
	RouteStateProgrammed    RouteState = "programmed"
	RouteStateNotReady      RouteState = "not_ready"
	RouteStateRejected      RouteState = "rejected"
	RouteStateEmptyUpstream RouteState = "empty_upstream"
	RouteStateDraining      RouteState = "draining"
	RouteStateDeleting      RouteState = "deleting"
	RouteStateDeleted       RouteState = "deleted"
	RouteStateStaleRevision RouteState = "stale_revision"
)

// RouteStatusInput contains the desired route revision and the optional,
// separately reported provider-observed revision. A stale_revision status
// requires both revisions and requires them to differ.
type RouteStatusInput struct {
	Key              RouteKey
	State            RouteState
	DesiredRevision  RouteRevision
	ObservedRevision *RouteRevision
}

// RouteStatus is an immutable provider observation for one known desired
// route. Programmed never means the data plane is ready; that claim requires
// CapabilityDataPlaneReadiness and provider-specific evidence outside this
// foundation.
type RouteStatus struct {
	key              RouteKey
	state            RouteState
	desiredRevision  RouteRevision
	observedRevision *RouteRevision
}

// NewRouteStatus validates and copies one provider observation.
func NewRouteStatus(input RouteStatusInput) (RouteStatus, error) {
	status := RouteStatus{
		key:             input.Key,
		state:           input.State,
		desiredRevision: input.DesiredRevision,
	}
	if input.ObservedRevision != nil {
		observed := *input.ObservedRevision
		status.observedRevision = &observed
	}
	if err := status.Validate(); err != nil {
		return RouteStatus{}, err
	}
	return status, nil
}

// Validate verifies that this is a complete route status.
func (s RouteStatus) Validate() error {
	if err := s.key.Validate(); err != nil {
		return err
	}
	if !validRouteState(s.state) {
		return newInvalidError("route_state")
	}
	if err := s.desiredRevision.Validate(); err != nil {
		return err
	}
	if s.observedRevision != nil {
		if err := s.observedRevision.Validate(); err != nil {
			return err
		}
	}
	if s.state == RouteStateStaleRevision {
		if s.observedRevision == nil || s.desiredRevision.Equal(*s.observedRevision) {
			return newInvalidError("stale_revision")
		}
	}
	return nil
}

func validRouteState(state RouteState) bool {
	switch state {
	case RouteStateAbsent, RouteStateAccepted, RouteStateProgrammed, RouteStateNotReady,
		RouteStateRejected, RouteStateEmptyUpstream, RouteStateDraining, RouteStateDeleting,
		RouteStateDeleted, RouteStateStaleRevision:
		return true
	default:
		return false
	}
}

func (s RouteStatus) Key() RouteKey                  { return s.key }
func (s RouteStatus) State() RouteState              { return s.state }
func (s RouteStatus) DesiredRevision() RouteRevision { return s.desiredRevision }

// ObservedRevision returns the independently reported provider revision, if
// the provider has one. An absent observed revision is distinct from an empty
// or substituted revision.
func (s RouteStatus) ObservedRevision() (RouteRevision, bool) {
	if s.observedRevision == nil {
		return RouteRevision{}, false
	}
	return *s.observedRevision, true
}

// Equal reports equality across the status state and both revision channels.
func (s RouteStatus) Equal(other RouteStatus) bool {
	if !s.key.Equal(other.key) || s.state != other.state || !s.desiredRevision.Equal(other.desiredRevision) {
		return false
	}
	if s.observedRevision == nil || other.observedRevision == nil {
		return s.observedRevision == nil && other.observedRevision == nil
	}
	return s.observedRevision.Equal(*other.observedRevision)
}

// ReconcileResult is the immutable status returned by a successful desired
// route operation. It deliberately exposes no data-plane readiness boolean.
type ReconcileResult struct {
	status RouteStatus
}

// NewReconcileResult creates a result from one valid status.
func NewReconcileResult(status RouteStatus) (ReconcileResult, error) {
	if err := status.Validate(); err != nil {
		return ReconcileResult{}, err
	}
	return ReconcileResult{status: cloneRouteStatus(status)}, nil
}

// Validate verifies the result's immutable status.
func (r ReconcileResult) Validate() error { return r.status.Validate() }

func (r ReconcileResult) Status() RouteStatus            { return cloneRouteStatus(r.status) }
func (r ReconcileResult) Key() RouteKey                  { return r.status.Key() }
func (r ReconcileResult) State() RouteState              { return r.status.State() }
func (r ReconcileResult) DesiredRevision() RouteRevision { return r.status.DesiredRevision() }
func (r ReconcileResult) ObservedRevision() (RouteRevision, bool) {
	return r.status.ObservedRevision()
}

// Equal reports equality across the returned route status.
func (r ReconcileResult) Equal(other ReconcileResult) bool { return r.status.Equal(other.status) }

func cloneRouteStatus(status RouteStatus) RouteStatus {
	copyStatus := status
	if status.observedRevision != nil {
		observed := *status.observedRevision
		copyStatus.observedRevision = &observed
	}
	return copyStatus
}

// DrainRequest is an immutable exact desired-revision precondition for a
// drain request. It intentionally has no implicit grace period, force flag, or
// completion claim.
type DrainRequest struct {
	expectedDesiredRevision RouteRevision
}

// NewDrainRequest creates a drain request guarded by one exact desired
// revision. Callers cannot drain a route by key alone.
func NewDrainRequest(expectedDesiredRevision RouteRevision) (DrainRequest, error) {
	request := DrainRequest{expectedDesiredRevision: expectedDesiredRevision}
	if err := request.Validate(); err != nil {
		return DrainRequest{}, err
	}
	return request, nil
}

// Validate verifies the exact desired-revision precondition.
func (r DrainRequest) Validate() error { return r.expectedDesiredRevision.Validate() }

func (r DrainRequest) ExpectedDesiredRevision() RouteRevision { return r.expectedDesiredRevision }

// DeleteRequest is an immutable exact desired-revision precondition for a
// delete request. It intentionally has no force, retry, or fallback option.
type DeleteRequest struct {
	expectedDesiredRevision RouteRevision
}

// NewDeleteRequest creates a delete request guarded by one exact desired
// revision. Callers cannot delete a route by key alone.
func NewDeleteRequest(expectedDesiredRevision RouteRevision) (DeleteRequest, error) {
	request := DeleteRequest{expectedDesiredRevision: expectedDesiredRevision}
	if err := request.Validate(); err != nil {
		return DeleteRequest{}, err
	}
	return request, nil
}

// Validate verifies the exact desired-revision precondition.
func (r DeleteRequest) Validate() error { return r.expectedDesiredRevision.Validate() }

func (r DeleteRequest) ExpectedDesiredRevision() RouteRevision { return r.expectedDesiredRevision }

// Provider is the provider-neutral external Gateway control contract. Its
// operations reconcile and observe desired route state only; they do not proxy
// traffic or establish a data-plane readiness claim. Implementations must not
// infer a provider, discovery owner, endpoint, release, retry, or fallback.
type Provider interface {
	Name() ProviderName
	Capabilities() Capabilities
	// Reconcile accepts one exact desired route. Repeating the same key and
	// desired revision is idempotent; reusing that revision for different route
	// facts is invalid.
	Reconcile(context.Context, RouteSpec) (ReconcileResult, error)
	Status(context.Context, RouteKey) (RouteStatus, error)
	// BeginDrain requires CapabilityDrain and an exact desired-revision
	// precondition. Success begins drain only; it does not claim completion.
	BeginDrain(context.Context, RouteKey, DrainRequest) (ReconcileResult, error)
	// Delete requires an exact desired-revision precondition. A successful
	// result is deleting, while deleted is only an explicit provider observation.
	Delete(context.Context, RouteKey, DeleteRequest) (ReconcileResult, error)
	Close() error
}
