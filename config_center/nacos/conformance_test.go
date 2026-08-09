package nacos

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	configcenter "github.com/NeKiro-project/NeKiro/config_center"
	"github.com/NeKiro-project/NeKiro/config_center/testkit"
)

func TestNacosConformance(t *testing.T) {
	testkit.Run(t, newConformanceHarness(t))
}

func TestNacosReaderCloseConformance(t *testing.T) {
	testkit.RunReaderClose(t, newConformanceHarness(t))
}

func newConformanceHarness(t *testing.T) testkit.Harness {
	t.Helper()
	provider := newFakeNacos()
	server := httptest.NewServer(provider)
	reader, err := NewReader(ReaderConfig{
		APIOrigin: server.URL + "/nacos", NamespaceID: "nekiro", GroupName: "NEKIRO",
		MaxPayloadBytes: 1024, WatchEnabled: true, LongPollTimeout: 100 * time.Millisecond,
		SubscriptionBuffer: 8, AuthMode: AuthNone, Executor: server.Client(),
	})
	if err != nil {
		server.Close()
		t.Fatal(err)
	}
	publisher := &fakePublisher{provider: provider}
	return testkit.Harness{
		Reader: reader, Publisher: publisher,
		Publish: publisher.Publish, Delete: publisher.Delete,
		Interrupt: func(context.Context) error {
			reader.interruptAll("")
			return nil
		},
		Cleanup: func() error {
			err := reader.Close()
			server.Close()
			return err
		},
	}
}

type fakeNacos struct {
	mu      sync.Mutex
	states  map[string]remoteState
	changed chan struct{}
}

func newFakeNacos() *fakeNacos {
	return &fakeNacos{states: make(map[string]remoteState), changed: make(chan struct{})}
}

func (provider *fakeNacos) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if request.URL.Path == "/nacos/v1/cs/configs" && request.Method == http.MethodGet {
		provider.serveGet(writer, request)
		return
	}
	if request.URL.Path == "/nacos/v1/cs/configs/listener" && request.Method == http.MethodPost {
		provider.serveListener(writer, request)
		return
	}
	http.Error(writer, "unsupported", http.StatusNotFound)
}

func (provider *fakeNacos) serveGet(writer http.ResponseWriter, request *http.Request) {
	provider.mu.Lock()
	state := provider.states[request.URL.Query().Get("dataId")]
	provider.mu.Unlock()
	if !state.present {
		http.Error(writer, "missing", http.StatusNotFound)
		return
	}
	_, _ = writer.Write(state.content)
}

func (provider *fakeNacos) serveListener(writer http.ResponseWriter, request *http.Request) {
	if err := request.ParseForm(); err != nil {
		http.Error(writer, "invalid form", http.StatusBadRequest)
		return
	}
	record := request.Form.Get("Listening-Configs")
	parts := strings.Split(strings.TrimSuffix(record, listenerRecordSeparator), listenerFieldSeparator)
	if len(parts) != 4 || parts[1] != "NEKIRO" || parts[3] != "nekiro" {
		http.Error(writer, "invalid listener", http.StatusBadRequest)
		return
	}
	dataID, digest := parts[0], parts[2]

	for {
		provider.mu.Lock()
		state := provider.states[dataID]
		changed := provider.changed
		provider.mu.Unlock()
		if contentDigest(state) != digest {
			response := strings.Join([]string{dataID, "NEKIRO", "nekiro"}, listenerFieldSeparator) + listenerRecordSeparator
			_, _ = writer.Write([]byte(url.QueryEscape(response)))
			return
		}
		select {
		case <-changed:
		case <-request.Context().Done():
			return
		case <-time.After(100 * time.Millisecond):
			return
		}
	}
}

func (provider *fakeNacos) publish(key configcenter.Key, content []byte) {
	provider.mu.Lock()
	provider.states[dataIDForKey(key)] = remoteState{present: true, content: append([]byte(nil), content...)}
	close(provider.changed)
	provider.changed = make(chan struct{})
	provider.mu.Unlock()
}

func (provider *fakeNacos) delete(key configcenter.Key) bool {
	provider.mu.Lock()
	defer provider.mu.Unlock()
	dataID := dataIDForKey(key)
	if !provider.states[dataID].present {
		return false
	}
	delete(provider.states, dataID)
	close(provider.changed)
	provider.changed = make(chan struct{})
	return true
}

type fakePublisher struct{ provider *fakeNacos }

func (publisher *fakePublisher) Publish(_ context.Context, key configcenter.Key, content []byte) error {
	publisher.provider.publish(key, content)
	return nil
}

func (publisher *fakePublisher) Delete(_ context.Context, key configcenter.Key) error {
	if !publisher.provider.delete(key) {
		return readerError(configcenter.CodeMissing, key, configcenter.OperationDelete)
	}
	return nil
}

func (*fakePublisher) Close() error { return nil }

var _ configcenter.ConfigurationPublisher = (*fakePublisher)(nil)
