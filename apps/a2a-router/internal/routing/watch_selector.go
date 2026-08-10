package routing

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/NeKiro-project/NeKiro/apps/a2a-router/internal/transport/a2a"
	"github.com/NeKiro-project/NeKiro/contracts"
	"github.com/NeKiro-project/NeKiro/registry"
)

// WatchSelector establishes one continuous observation for each exact Release
// on first use. Every Invocation reads one immutable snapshot and remains
// pinned to the selected endpoint for its complete lifecycle.
type WatchSelector struct {
	directory       registry.InstanceDirectory
	provider        string
	portName        string
	maxObservations int
	now             func() time.Time
	ctx             context.Context
	cancel          context.CancelFunc

	mu       sync.Mutex
	closed   bool
	sessions map[registry.ReleaseTarget]*observationSession
	wg       sync.WaitGroup
}

type observationSession struct {
	ready     chan struct{}
	readyOnce sync.Once

	mu         sync.RWMutex
	snapshot   registry.InstanceSnapshot
	available  bool
	watch      registry.InstanceWatch
	state      contracts.RouterTopologyObservationState
	revision   uint64
	observedAt time.Time
}

func NewWatchSelector(directory registry.InstanceDirectory, provider, portName string, maxObservations int) (*WatchSelector, error) {
	status := contracts.RouterTopologyStatusV1{
		SchemaVersion: contracts.RouterTopologyStatusSchemaVersion, Provider: provider,
		Observations: []contracts.RouterTopologyStatusObservationV1{},
	}
	if directory == nil || contracts.ValidateRouterTopologyStatusV1(status) != nil || portName == "" ||
		maxObservations <= 0 || maxObservations > contracts.RouterTopologyStatusObservationMaximum ||
		!directory.Capabilities().Supports(registry.CapabilityObserve) {
		return nil, errors.New("watch selector dependencies are required")
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &WatchSelector{
		directory: directory, provider: provider, portName: portName, maxObservations: maxObservations, now: time.Now,
		ctx: ctx, cancel: cancel, sessions: make(map[registry.ReleaseTarget]*observationSession),
	}, nil
}

func (selector *WatchSelector) Select(ctx context.Context, target a2a.Target, _ a2a.ContextHeaders) (a2a.Target, error) {
	if ctx == nil {
		return a2a.Target{}, errors.New("selection context is required")
	}
	if err := ctx.Err(); err != nil {
		return a2a.Target{}, err
	}
	release, err := releaseTarget(target)
	if err != nil {
		return a2a.Target{}, err
	}
	session, err := selector.session(release)
	if err != nil {
		return a2a.Target{}, err
	}
	select {
	case <-ctx.Done():
		return a2a.Target{}, ctx.Err()
	case <-session.ready:
	}
	snapshot, ok := session.current()
	if !ok {
		return a2a.Target{}, errors.New("instance observation is unavailable")
	}
	return selectSnapshotTarget(target, snapshot, selector.portName)
}

func (selector *WatchSelector) session(target registry.ReleaseTarget) (*observationSession, error) {
	selector.mu.Lock()
	defer selector.mu.Unlock()
	if selector.closed {
		return nil, errors.New("watch selector is closed")
	}
	if session := selector.sessions[target]; session != nil {
		return session, nil
	}
	if len(selector.sessions) >= selector.maxObservations {
		return nil, errors.New("instance observation limit reached")
	}
	session := &observationSession{
		ready: make(chan struct{}), state: contracts.RouterTopologyStateInitializing,
		observedAt: selector.now().UTC(),
	}
	selector.sessions[target] = session
	selector.wg.Add(1)
	go selector.observe(target, session)
	return session, nil
}

func (selector *WatchSelector) observe(target registry.ReleaseTarget, session *observationSession) {
	defer selector.wg.Done()
	observation, err := selector.directory.Observe(selector.ctx, target)
	if err != nil {
		session.fail(selector.now().UTC())
		return
	}
	watch := observation.Watch()
	if !session.initialize(target, observation.Initial(), watch, selector.now().UTC()) {
		if watch != nil {
			_ = watch.Close()
		}
		return
	}
	for {
		change, err := watch.Next(selector.ctx)
		if err != nil || !session.apply(target, change, selector.now().UTC()) {
			session.fail(selector.now().UTC())
			_ = watch.Close()
			return
		}
	}
}

func (selector *WatchSelector) Close() error {
	selector.mu.Lock()
	if selector.closed {
		selector.mu.Unlock()
		return nil
	}
	selector.closed = true
	selector.cancel()
	sessions := make([]*observationSession, 0, len(selector.sessions))
	for _, session := range selector.sessions {
		sessions = append(sessions, session)
	}
	selector.mu.Unlock()

	for _, session := range sessions {
		session.closeWatch()
	}
	selector.wg.Wait()
	return nil
}

func (selector *WatchSelector) TopologyStatus() contracts.RouterTopologyStatusV1 {
	selector.mu.Lock()
	observations := make([]contracts.RouterTopologyStatusObservationV1, 0, len(selector.sessions))
	for target, session := range selector.sessions {
		state, revision, observedAt := session.status()
		observations = append(observations, contracts.RouterTopologyStatusObservationV1{
			AgentID: target.AgentID(), AgentCardVersion: target.AgentCardVersion(), ReleaseID: target.ReleaseID(),
			State: state, LocalRevision: revision, ObservedAt: observedAt,
		})
	}
	provider := selector.provider
	selector.mu.Unlock()
	sort.Slice(observations, func(i, j int) bool {
		if observations[i].AgentID != observations[j].AgentID {
			return observations[i].AgentID < observations[j].AgentID
		}
		if observations[i].AgentCardVersion != observations[j].AgentCardVersion {
			return observations[i].AgentCardVersion < observations[j].AgentCardVersion
		}
		return observations[i].ReleaseID < observations[j].ReleaseID
	})
	return contracts.RouterTopologyStatusV1{
		SchemaVersion: contracts.RouterTopologyStatusSchemaVersion,
		Provider:      provider,
		Observations:  observations,
	}
}

func (session *observationSession) initialize(target registry.ReleaseTarget, snapshot registry.InstanceSnapshot, watch registry.InstanceWatch, observedAt time.Time) bool {
	session.mu.Lock()
	defer session.mu.Unlock()
	if watch == nil || snapshot.Validate() != nil || !snapshot.Target().Equal(target) || snapshot.Revision().LocalOrder() != 0 {
		session.state = contracts.RouterTopologyStateUnavailable
		session.observedAt = observedAt
		session.readyOnce.Do(func() { close(session.ready) })
		return false
	}
	session.snapshot = snapshot
	session.available = true
	session.watch = watch
	session.state = topologyObservationState(snapshot.State())
	session.revision = snapshot.Revision().LocalOrder()
	session.observedAt = observedAt
	session.readyOnce.Do(func() { close(session.ready) })
	return true
}

func (session *observationSession) apply(target registry.ReleaseTarget, change registry.InstanceChange, observedAt time.Time) bool {
	if change.Validate() != nil {
		return false
	}
	next := change.Snapshot()
	session.mu.Lock()
	defer session.mu.Unlock()
	if !session.available || !next.Target().Equal(target) || next.Revision().LocalOrder() != session.snapshot.Revision().LocalOrder()+1 {
		return false
	}
	session.snapshot = next
	session.state = topologyObservationState(next.State())
	session.revision = next.Revision().LocalOrder()
	session.observedAt = observedAt
	return true
}

func (session *observationSession) current() (registry.InstanceSnapshot, bool) {
	session.mu.RLock()
	defer session.mu.RUnlock()
	return session.snapshot, session.available
}

func (session *observationSession) fail(observedAt time.Time) {
	session.mu.Lock()
	session.snapshot = registry.InstanceSnapshot{}
	session.available = false
	session.state = contracts.RouterTopologyStateUnavailable
	session.observedAt = observedAt
	session.mu.Unlock()
	session.readyOnce.Do(func() { close(session.ready) })
}

func (session *observationSession) status() (contracts.RouterTopologyObservationState, uint64, time.Time) {
	session.mu.RLock()
	defer session.mu.RUnlock()
	return session.state, session.revision, session.observedAt
}

func (session *observationSession) closeWatch() {
	session.mu.RLock()
	watch := session.watch
	session.mu.RUnlock()
	if watch != nil {
		_ = watch.Close()
	}
}

var _ a2a.TargetSelector = (*WatchSelector)(nil)

func topologyObservationState(state registry.SnapshotState) contracts.RouterTopologyObservationState {
	switch state {
	case registry.SnapshotStateMissing:
		return contracts.RouterTopologyStateMissing
	case registry.SnapshotStateEmpty:
		return contracts.RouterTopologyStateEmpty
	case registry.SnapshotStatePopulated:
		return contracts.RouterTopologyStatePopulated
	default:
		return contracts.RouterTopologyStateUnavailable
	}
}
