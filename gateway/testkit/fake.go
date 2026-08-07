// Package testkit provides deterministic provider-neutral Gateway fixtures.
// The fake has no network client, proxy, polling loop, retry policy, or
// provider fallback.
package testkit

import (
	"context"
	"sync"

	"github.com/NeKiro-project/NeKiro/gateway"
)

// FakeConfig configures one deterministic fake. Name is intentionally
// required; the testkit does not invent a provider identity.
type FakeConfig struct {
	Name gateway.ProviderName
}

// FakeProvider is an in-memory Gateway Provider. It advertises no real
// Gateway data-plane capabilities. Provider-observed statuses can only be
// supplied explicitly through SetObservedStatus for conformance fixtures.
type FakeProvider struct {
	mu     sync.RWMutex
	name   gateway.ProviderName
	caps   gateway.Capabilities
	closed bool
	routes map[gateway.RouteKey]fakeRoute
	// provenance retains every accepted desired revision for each route key so
	// a later revision cannot make an earlier revision reusable with new facts.
	provenance map[gateway.RouteKey]map[gateway.RouteRevision]gateway.RouteSpec
}

type fakeRoute struct {
	spec   gateway.RouteSpec
	status gateway.RouteStatus
}

// NewFakeProvider constructs a fake with an explicit provider identity. It
// reports an empty capability set because this deterministic fixture cannot
// prove any external Gateway data-plane behavior.
func NewFakeProvider(config FakeConfig) (*FakeProvider, error) {
	if err := config.Name.Validate(); err != nil {
		return nil, err
	}
	caps, err := gateway.NewCapabilities()
	if err != nil {
		return nil, err
	}
	return &FakeProvider{
		name:       config.Name,
		caps:       caps,
		routes:     make(map[gateway.RouteKey]fakeRoute),
		provenance: make(map[gateway.RouteKey]map[gateway.RouteRevision]gateway.RouteSpec),
	}, nil
}

// NewFake is a short alias for NewFakeProvider.
func NewFake(config FakeConfig) (*FakeProvider, error) { return NewFakeProvider(config) }

// Provider returns the fake through the provider-neutral interface.
func (f *FakeProvider) Provider() gateway.Provider { return f }

// Name returns the explicit fake provider name.
func (f *FakeProvider) Name() gateway.ProviderName { return f.name }

// Capabilities returns an immutable empty set. The fake performs no network
// I/O and cannot claim forwarding, readiness, drain completion, or affinity.
func (f *FakeProvider) Capabilities() gateway.Capabilities {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return cloneCapabilities(f.caps)
}

// Reconcile records one exact desired route and returns accepted. It never
// reports programmed, ready, forwarding, or an observed provider revision.
func (f *FakeProvider) Reconcile(ctx context.Context, spec gateway.RouteSpec) (gateway.ReconcileResult, error) {
	if err := validateContext(ctx); err != nil {
		return gateway.ReconcileResult{}, err
	}
	if err := spec.Validate(); err != nil {
		return gateway.ReconcileResult{}, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return gateway.ReconcileResult{}, gateway.ErrClosed
	}
	if spec.DiscoveryOwner() == gateway.DiscoveryOwnerRouter {
		return gateway.ReconcileResult{}, gateway.NewOutcomeError(gateway.OutcomeUnsupported, gateway.CauseRouterDiscoveryUnsupported)
	}
	if missing := f.caps.Missing(spec.RequiredCapabilities()); len(missing) != 0 {
		return gateway.ReconcileResult{}, gateway.NewOutcomeError(gateway.OutcomeUnsupported, gateway.CauseRequiredCapability)
	}
	if prior, found := f.provenance[spec.Key()][spec.Revision()]; found && !prior.Equal(spec) {
		return gateway.ReconcileResult{}, gateway.NewOutcomeError(gateway.OutcomeInvalid, gateway.CauseRevisionReused)
	}

	if current, found := f.routes[spec.Key()]; found {
		if current.spec.Revision().Equal(spec.Revision()) && !current.spec.Equal(spec) {
			return gateway.ReconcileResult{}, gateway.NewOutcomeError(gateway.OutcomeInvalid, gateway.CauseRevisionReused)
		}
		if current.spec.Revision().Equal(spec.Revision()) {
			result, err := gateway.NewReconcileResult(current.status)
			if err != nil {
				return gateway.ReconcileResult{}, err
			}
			return result, nil
		}
	}

	status, err := gateway.NewRouteStatus(gateway.RouteStatusInput{
		Key:             spec.Key(),
		State:           gateway.RouteStateAccepted,
		DesiredRevision: spec.Revision(),
	})
	if err != nil {
		return gateway.ReconcileResult{}, err
	}
	if f.provenance[spec.Key()] == nil {
		f.provenance[spec.Key()] = make(map[gateway.RouteRevision]gateway.RouteSpec)
	}
	f.provenance[spec.Key()][spec.Revision()] = spec
	f.routes[spec.Key()] = fakeRoute{spec: spec, status: status}
	return gateway.NewReconcileResult(status)
}

// Status returns the current explicitly stored status. An unknown route key
// returns not_found; absent is reserved for a known desired route whose
// provider observation says it is absent.
func (f *FakeProvider) Status(ctx context.Context, key gateway.RouteKey) (gateway.RouteStatus, error) {
	if err := validateContext(ctx); err != nil {
		return gateway.RouteStatus{}, err
	}
	if err := key.Validate(); err != nil {
		return gateway.RouteStatus{}, err
	}

	f.mu.RLock()
	defer f.mu.RUnlock()
	if f.closed {
		return gateway.RouteStatus{}, gateway.ErrClosed
	}
	route, found := f.routes[key]
	if !found {
		return gateway.RouteStatus{}, gateway.ErrNotFound
	}
	return cloneStatus(route.status), nil
}

// BeginDrain refuses because the fake advertises no drain capability. The
// request is still fully validated, and no route state is changed.
func (f *FakeProvider) BeginDrain(ctx context.Context, key gateway.RouteKey, request gateway.DrainRequest) (gateway.ReconcileResult, error) {
	if err := validateContext(ctx); err != nil {
		return gateway.ReconcileResult{}, err
	}
	if err := key.Validate(); err != nil {
		return gateway.ReconcileResult{}, err
	}
	if err := request.Validate(); err != nil {
		return gateway.ReconcileResult{}, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return gateway.ReconcileResult{}, gateway.ErrClosed
	}
	route, found := f.routes[key]
	if !found {
		return gateway.ReconcileResult{}, gateway.ErrNotFound
	}
	if !route.spec.Revision().Equal(request.ExpectedDesiredRevision()) {
		return gateway.ReconcileResult{}, gateway.ErrStale
	}
	return gateway.ReconcileResult{}, gateway.NewOutcomeError(gateway.OutcomeUnsupported, gateway.CauseRequiredCapability)
}

// Delete records the desired deletion and returns deleting. It never claims
// that the provider or data plane has completed deletion; a deleted status
// must be supplied as an explicit provider observation.
func (f *FakeProvider) Delete(ctx context.Context, key gateway.RouteKey, request gateway.DeleteRequest) (gateway.ReconcileResult, error) {
	if err := validateContext(ctx); err != nil {
		return gateway.ReconcileResult{}, err
	}
	if err := key.Validate(); err != nil {
		return gateway.ReconcileResult{}, err
	}
	if err := request.Validate(); err != nil {
		return gateway.ReconcileResult{}, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return gateway.ReconcileResult{}, gateway.ErrClosed
	}
	route, found := f.routes[key]
	if !found {
		return gateway.ReconcileResult{}, gateway.ErrNotFound
	}
	if !route.spec.Revision().Equal(request.ExpectedDesiredRevision()) {
		return gateway.ReconcileResult{}, gateway.ErrStale
	}
	if route.status.State() == gateway.RouteStateDeleted {
		result, err := gateway.NewReconcileResult(route.status)
		if err != nil {
			return gateway.ReconcileResult{}, err
		}
		return result, nil
	}
	if route.status.State() == gateway.RouteStateDeleting {
		return gateway.NewReconcileResult(route.status)
	}
	status, err := gateway.NewRouteStatus(gateway.RouteStatusInput{
		Key:              key,
		State:            gateway.RouteStateDeleting,
		DesiredRevision:  route.spec.Revision(),
		ObservedRevision: observedRevisionPointer(route.status),
	})
	if err != nil {
		return gateway.ReconcileResult{}, err
	}
	route.status = status
	f.routes[key] = route
	return gateway.NewReconcileResult(status)
}

// SetObservedStatus injects one explicit provider observation for a route in a
// deterministic fixture. It cannot replace the route's desired revision.
func (f *FakeProvider) SetObservedStatus(status gateway.RouteStatus) error {
	if err := status.Validate(); err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return gateway.ErrClosed
	}
	route, found := f.routes[status.Key()]
	if !found {
		return gateway.ErrNotFound
	}
	if !route.spec.Revision().Equal(status.DesiredRevision()) {
		return gateway.ErrStale
	}
	route.status = cloneStatus(status)
	f.routes[status.Key()] = route
	return nil
}

// SetStatus is an alias for SetObservedStatus used by generic conformance
// drivers.
func (f *FakeProvider) SetStatus(status gateway.RouteStatus) error {
	return f.SetObservedStatus(status)
}

// Close is idempotent and only closes this in-memory provider.
func (f *FakeProvider) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.closed {
		return nil
	}
	f.closed = true
	return nil
}

func validateContext(ctx context.Context) error {
	if ctx == nil {
		return gateway.NewOutcomeError(gateway.OutcomeInvalid, gateway.CauseInvalidInput)
	}
	if err := ctx.Err(); err != nil {
		return gateway.NewCanceledError(err)
	}
	return nil
}

func observedRevisionPointer(status gateway.RouteStatus) *gateway.RouteRevision {
	revision, ok := status.ObservedRevision()
	if !ok {
		return nil
	}
	return &revision
}

func cloneStatus(status gateway.RouteStatus) gateway.RouteStatus {
	copyStatus, err := gateway.NewRouteStatus(gateway.RouteStatusInput{
		Key:              status.Key(),
		State:            status.State(),
		DesiredRevision:  status.DesiredRevision(),
		ObservedRevision: observedRevisionPointer(status),
	})
	if err != nil {
		panic("gateway testkit received invalid status")
	}
	return copyStatus
}

func cloneCapabilities(capabilities gateway.Capabilities) gateway.Capabilities {
	copyCapabilities, err := gateway.NewCapabilities(capabilities.Values()...)
	if err != nil {
		panic("gateway testkit received invalid capabilities")
	}
	return copyCapabilities
}
