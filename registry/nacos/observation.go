package nacos

import (
	"context"
	"errors"
	"sync"

	"github.com/NeKiro-project/NeKiro/registry"
)

type observationSession struct {
	directory *Directory
	target    registry.ReleaseTarget
	binding   Binding
	source    NamingSubscription
	inner     registry.InstanceWatch
	publisher registry.InstanceWatchPublisher
	watch     *nacosWatch
	ctx       context.Context
	cancel    context.CancelFunc

	mu       sync.Mutex
	current  registry.InstanceSnapshot
	terminal error
	stopOnce sync.Once
}

func newObservationSession(
	directory *Directory,
	target registry.ReleaseTarget,
	binding Binding,
	initial registry.InstanceSnapshot,
	source NamingSubscription,
	inner registry.InstanceWatch,
	publisher registry.InstanceWatchPublisher,
) *observationSession {
	ctx, cancel := context.WithCancel(context.Background())
	session := &observationSession{
		directory: directory, target: target, binding: binding, source: source,
		inner: inner, publisher: publisher, ctx: ctx, cancel: cancel, current: initial,
	}
	session.watch = &nacosWatch{session: session, inner: inner}
	return session
}

func (session *observationSession) run() {
	for {
		payload, err := session.source.Next(session.ctx)
		if err != nil {
			if session.ctx.Err() != nil {
				return
			}
			session.fail(registry.NewOutcomeError(registry.OutcomeWatchInterrupted, registry.CauseStreamEOF))
			return
		}
		if int64(len(payload)) > session.directory.maxResponseBytes {
			session.fail(registry.NewOutcomeError(registry.OutcomeWatchInterrupted, registry.CauseWatchEventTooLarge))
			return
		}
		if err := session.accept(payload); err != nil {
			session.fail(err)
			return
		}
	}
}

func (session *observationSession) accept(payload []byte) error {
	session.mu.Lock()
	defer session.mu.Unlock()
	if session.terminal != nil {
		return nil
	}
	nextOrder := session.current.Revision().LocalOrder() + 1
	next, err := snapshotFromPayload(payload, session.binding, session.directory.portName, nextOrder)
	if err != nil {
		return registry.NewOutcomeError(registry.OutcomeWatchInterrupted, registry.CauseWatchEventInvalid)
	}
	if samePublicTopology(session.current, next) {
		return nil
	}
	upserts, deleted := topologyDelta(session.current, next)
	change, err := registry.NewInstanceChange(registry.InstanceChangeInput{
		Kind: registry.InstanceChangeInstancesChanged, Revision: next.Revision(),
		Upserts: upserts, DeletedInstanceIDs: deleted, Snapshot: next,
	})
	if err != nil {
		return registry.NewOutcomeError(registry.OutcomeWatchInterrupted, registry.CauseWatchEventInvalid)
	}
	if err := session.publisher.Publish(change); err != nil {
		return err
	}
	session.current = next
	return nil
}

func (session *observationSession) fail(err error) {
	session.mu.Lock()
	if session.terminal == nil {
		session.terminal = safeWatchTerminal(err)
	}
	terminal := session.terminal
	session.mu.Unlock()
	session.stopOnce.Do(func() {
		session.cancel()
		session.publisher.Terminate(terminal)
		_ = session.source.Close()
		session.directory.unregisterSession(session)
	})
}

func (session *observationSession) close() error {
	session.mu.Lock()
	if session.terminal == nil {
		session.terminal = registry.ErrClosed
	}
	terminal := session.terminal
	session.mu.Unlock()
	session.stopOnce.Do(func() {
		session.cancel()
		session.publisher.Terminate(terminal)
		_ = session.source.Close()
		session.directory.unregisterSession(session)
	})
	return nil
}

type nacosWatch struct {
	session *observationSession
	inner   registry.InstanceWatch
}

func (watch *nacosWatch) Next(ctx context.Context) (registry.InstanceChange, error) {
	change, err := watch.inner.Next(ctx)
	if err != nil {
		if outcome, ok := registry.OutcomeOf(err); ok && outcome != registry.OutcomeCanceled && outcome != registry.OutcomeInvalid {
			_ = watch.session.close()
		}
		return registry.InstanceChange{}, err
	}
	return change, nil
}

func (watch *nacosWatch) Close() error { return watch.session.close() }

func safeSubscribeError(err error) error {
	var outcome *registry.OutcomeError
	if errors.As(err, &outcome) && outcome != nil {
		return registry.NewOutcomeError(outcome.Outcome(), outcome.Cause())
	}
	return registry.NewOutcomeError(registry.OutcomeUnavailable, registry.CauseProviderUnavailable)
}

func safeWatchTerminal(err error) error {
	var outcome *registry.OutcomeError
	if errors.As(err, &outcome) && outcome != nil && outcome.Outcome() == registry.OutcomeWatchInterrupted {
		return registry.NewOutcomeError(outcome.Outcome(), outcome.Cause())
	}
	return registry.NewOutcomeError(registry.OutcomeWatchInterrupted, registry.CauseStreamEOF)
}

func samePublicTopology(left, right registry.InstanceSnapshot) bool {
	if left.State() != right.State() {
		return false
	}
	leftInstances := left.Instances()
	rightInstances := right.Instances()
	if len(leftInstances) != len(rightInstances) {
		return false
	}
	for index := range leftInstances {
		if !leftInstances[index].Equal(rightInstances[index]) {
			return false
		}
	}
	return true
}

func topologyDelta(previous, next registry.InstanceSnapshot) ([]registry.Instance, []string) {
	previousByID := make(map[string]registry.Instance)
	for _, instance := range previous.Instances() {
		previousByID[instance.ID()] = instance
	}
	nextByID := make(map[string]registry.Instance)
	upserts := make([]registry.Instance, 0)
	for _, instance := range next.Instances() {
		nextByID[instance.ID()] = instance
		if prior, found := previousByID[instance.ID()]; !found || !prior.Equal(instance) {
			upserts = append(upserts, instance)
		}
	}
	deleted := make([]string, 0)
	for id := range previousByID {
		if _, found := nextByID[id]; !found {
			deleted = append(deleted, id)
		}
	}
	return upserts, deleted
}

var _ registry.InstanceWatch = (*nacosWatch)(nil)
