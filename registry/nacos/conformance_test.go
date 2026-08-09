package nacos

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/NeKiro-project/NeKiro/registry"
	"github.com/NeKiro-project/NeKiro/registry/testkit"
)

const initialServiceInfo = `{"hosts":[{"ip":"172.28.0.12","port":8092,"healthy":true,"enabled":true,"ephemeral":true,"clusterName":"DEFAULT","metadata":{"nekiro.instanceId":"runtime-b-1"}}]}`
const changedServiceInfo = `{"hosts":[{"ip":"172.28.0.12","port":8092,"healthy":true,"enabled":true,"ephemeral":true,"clusterName":"DEFAULT","metadata":{"nekiro.instanceId":"runtime-b-1"}},{"ip":"172.28.0.13","port":8092,"healthy":true,"enabled":true,"ephemeral":true,"clusterName":"DEFAULT","metadata":{"nekiro.instanceId":"runtime-b-2"}}]}`

func TestNacosDirectoryConformance(t *testing.T) {
	target := testTarget(t)
	unbound, err := registry.NewReleaseTarget(registry.ReleaseTargetInput{
		AgentID: "unbound", AgentCardVersion: "1.0.0", ReleaseID: "release-unbound",
		CardDigest: target.CardDigest(), CanonicalEndpoint: "http://unbound:8092/", Audience: "http://unbound:8092",
	})
	if err != nil {
		t.Fatal(err)
	}
	binding, _ := NewBinding(BindingInput{Target: target, ServiceName: "runtime-b", GroupName: "NEKIRO", ClusterName: "DEFAULT"})
	subscriber := newFixtureSubscriber([]byte(initialServiceInfo))
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(initialServiceInfo))
	}))
	defer server.Close()
	directory, err := NewDirectory(DirectoryConfig{
		APIOrigin: server.URL + "/nacos", NamespaceID: "public", PortName: "a2a", MaxResponseBytes: 4096,
		AuthMode: AuthNone, Executor: server.Client(), Bindings: exactBindingSource{binding: binding},
		Subscriber: subscriber, PendingChanges: 4,
	})
	if err != nil {
		t.Fatal(err)
	}
	initial, err := snapshotFromPayload([]byte(initialServiceInfo), binding, "a2a", 0)
	if err != nil {
		t.Fatal(err)
	}
	next, err := snapshotFromPayload([]byte(changedServiceInfo), binding, "a2a", 1)
	if err != nil {
		t.Fatal(err)
	}
	upserts, deleted := topologyDelta(initial, next)
	change, err := registry.NewInstanceChange(registry.InstanceChangeInput{
		Kind: registry.InstanceChangeInstancesChanged, Revision: next.Revision(), Upserts: upserts,
		DeletedInstanceIDs: deleted, Snapshot: next,
	})
	if err != nil {
		t.Fatal(err)
	}
	driver := &nacosConformanceDriver{directory: directory, subscriber: subscriber, changePayload: []byte(changedServiceInfo)}
	testkit.RunDirectoryConformance(t, driver, testkit.DirectoryConformanceFixture{
		Target: target, UnboundTarget: unbound, Initial: initial, Change: change,
		Terminal: registry.NewOutcomeError(registry.OutcomeWatchInterrupted, registry.CauseStreamEOF),
	})
}

type exactBindingSource struct{ binding Binding }

func (source exactBindingSource) Binding(_ context.Context, target registry.ReleaseTarget) (Binding, error) {
	if !source.binding.target.Equal(target) {
		return Binding{}, registry.ErrMissing
	}
	return source.binding, nil
}

type fixtureSubscriber struct {
	mu      sync.Mutex
	initial []byte
	streams []*fixturePushStream
}

func newFixtureSubscriber(initial []byte) *fixtureSubscriber {
	return &fixtureSubscriber{initial: append([]byte(nil), initial...)}
}

func (*fixtureSubscriber) Guarantees() NamingSubscriptionGuarantees {
	return validSubscriptionGuarantees()
}

func (subscriber *fixtureSubscriber) Subscribe(_ context.Context, request NamingSubscribeRequest) (NamingSubscription, error) {
	if request.NamespaceID() != "public" || request.ServiceName() != "runtime-b" || request.GroupName() != "NEKIRO" || request.ClusterName() != "DEFAULT" {
		return NamingSubscription{}, registry.ErrInvalid
	}
	stream := newFixturePushStream()
	subscriber.mu.Lock()
	subscriber.streams = append(subscriber.streams, stream)
	subscriber.mu.Unlock()
	return NewNamingSubscription(subscriber.initial, stream)
}

func (subscriber *fixtureSubscriber) latest() *fixturePushStream {
	subscriber.mu.Lock()
	defer subscriber.mu.Unlock()
	return subscriber.streams[len(subscriber.streams)-1]
}

type fixturePushStream struct {
	events   chan []byte
	terminal chan error
	done     chan struct{}
	once     sync.Once
}

func newFixturePushStream() *fixturePushStream {
	return &fixturePushStream{events: make(chan []byte, 8), terminal: make(chan error, 1), done: make(chan struct{})}
}

func (stream *fixturePushStream) Next(ctx context.Context) ([]byte, error) {
	select {
	case payload := <-stream.events:
		return append([]byte(nil), payload...), nil
	case err := <-stream.terminal:
		return nil, err
	case <-stream.done:
		return nil, registry.ErrClosed
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (stream *fixturePushStream) Close() error {
	stream.once.Do(func() { close(stream.done) })
	return nil
}

type nacosConformanceDriver struct {
	directory     *Directory
	subscriber    *fixtureSubscriber
	changePayload []byte
}

func (driver *nacosConformanceDriver) Directory() registry.InstanceDirectory { return driver.directory }

func (driver *nacosConformanceDriver) Emit(_ registry.ReleaseTarget, _ registry.InstanceChange) error {
	driver.subscriber.latest().events <- append([]byte(nil), driver.changePayload...)
	return nil
}

func (driver *nacosConformanceDriver) Terminate(_ registry.ReleaseTarget, err error) error {
	if !errors.Is(err, registry.ErrWatchInterrupted) {
		return registry.ErrInvalid
	}
	driver.subscriber.latest().terminal <- err
	return nil
}

func validSubscriptionGuarantees() NamingSubscriptionGuarantees {
	return NamingSubscriptionGuarantees{
		Version: NamingSubscriptionExecutorVersionV1, AtomicInitialPushHandoff: true,
		ExactlyOneSubscribeAttempt: true, NoRetry: true, NoReconnect: true,
		NoResponseCache: true, NoFailover: true, NoImplicitPolling: true,
		NoHiddenReauthentication: true,
	}
}

var _ NamingSubscriptionExecutor = (*fixtureSubscriber)(nil)
var _ NamingPushStream = (*fixturePushStream)(nil)
var _ testkit.DirectoryConformanceDriver = (*nacosConformanceDriver)(nil)
