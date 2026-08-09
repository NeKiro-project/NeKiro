package nacos

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	configcenter "github.com/NeKiro-project/NeKiro/config_center"
)

func TestListenerProtocolFailuresInterruptReader(t *testing.T) {
	for name, response := range map[string]struct {
		status int
		body   string
		want   error
	}{
		"unauthorized": {status: http.StatusForbidden, want: configcenter.ErrUnauthorized},
		"unavailable":  {status: http.StatusServiceUnavailable, want: configcenter.ErrUnavailable},
		"malformed":    {status: http.StatusOK, body: "not-a-listener-record", want: configcenter.ErrUnavailable},
		"oversized":    {status: http.StatusOK, body: strings.Repeat("x", 1024), want: configcenter.ErrPayloadTooLarge},
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.Method == http.MethodGet {
					writer.WriteHeader(http.StatusNotFound)
					return
				}
				writer.WriteHeader(response.status)
				_, _ = writer.Write([]byte(response.body))
			}))
			defer server.Close()
			reader, err := NewReader(ReaderConfig{
				APIOrigin: server.URL + "/nacos", NamespaceID: "nekiro", GroupName: "NEKIRO",
				MaxPayloadBytes: 64, WatchEnabled: true, LongPollTimeout: 100 * time.Millisecond,
				SubscriptionBuffer: 1, AuthMode: AuthNone, Executor: server.Client(),
			})
			if err != nil {
				t.Fatal(err)
			}
			key, _ := configcenter.ParseKey("watch/failure")
			observation, err := reader.Observe(t.Context(), key)
			if err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(t.Context(), time.Second)
			defer cancel()
			_, err = observation.Subscription.Next(ctx)
			if !errors.Is(err, configcenter.ErrWatchInterrupted) {
				t.Fatalf("Next error=%v", err)
			}
			typed, ok := err.(*configcenter.Error)
			if !ok || !errors.Is(readerError(typed.Details().CauseKind, key, configcenter.OperationWatch), response.want) {
				t.Fatalf("watch cause=%q want=%v", typed.Details().CauseKind, response.want)
			}
		})
	}
}

func TestListenerResponseRequiresExactSingleTuple(t *testing.T) {
	valid := url.QueryEscape("key\x02GROUP\x02tenant\x01")
	if changed, err := parseListenerResponse("", "key", "GROUP", "tenant"); err != nil || changed {
		t.Fatalf("empty response=(%v,%v)", changed, err)
	}
	if changed, err := parseListenerResponse(valid, "key", "GROUP", "tenant"); err != nil || !changed {
		t.Fatalf("valid response=(%v,%v)", changed, err)
	}
	for _, body := range []string{
		"%zz", "key%02GROUP%02tenant", "key%02OTHER%02tenant%01",
		valid + valid,
	} {
		if _, err := parseListenerResponse(body, "key", "GROUP", "tenant"); !errors.Is(err, configcenter.ErrUnavailable) {
			t.Fatalf("parse %q error=%v", body, err)
		}
	}
}

func TestSubscriptionSuppressesDuplicateStateAndRejectsFullQueue(t *testing.T) {
	reader := &Reader{subscriptionBuffer: 1}
	key, _ := configcenter.ParseKey("watch/state")
	revision := configcenter.NewObservationRevision()
	subscription := newSubscription(reader, key, remoteState{present: true, content: []byte("a")}, revision)
	defer subscription.cancel()
	if !subscription.apply(remoteState{present: true, content: []byte("a")}) || subscription.Stats().SuppressedNotifications != 1 {
		t.Fatalf("duplicate stats=%#v", subscription.Stats())
	}
	if !subscription.apply(remoteState{present: true, content: []byte("b")}) {
		t.Fatal("first transition was not queued")
	}
	if subscription.apply(remoteState{present: false}) {
		t.Fatal("transition was queued into a full buffer")
	}
}

func TestObserveRejectsInvalidLifecycleInputs(t *testing.T) {
	reader, err := NewReader(ReaderConfig{
		APIOrigin: "http://nacos.test/nacos", NamespaceID: "nekiro", GroupName: "NEKIRO",
		MaxPayloadBytes: 64, AuthMode: AuthNone, Executor: http.DefaultClient,
	})
	if err != nil {
		t.Fatal(err)
	}
	key, _ := configcenter.ParseKey("watch/state")
	if _, err := reader.Observe(nil, key); !errors.Is(err, configcenter.ErrInvalid) {
		t.Fatalf("nil context error=%v", err)
	}
	canceled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := reader.Observe(canceled, key); !errors.Is(err, configcenter.ErrCanceled) {
		t.Fatalf("canceled context error=%v", err)
	}
	if _, err := reader.Observe(t.Context(), key); !errors.Is(err, configcenter.ErrUnsupported) {
		t.Fatalf("snapshot-only Observe error=%v", err)
	}
	reader.interruptAll("")
	if _, err := reader.Observe(t.Context(), key); !errors.Is(err, configcenter.ErrWatchInterrupted) {
		t.Fatalf("interrupted Observe error=%v", err)
	}
	_ = reader.Close()
	if _, err := reader.Observe(t.Context(), key); !errors.Is(err, configcenter.ErrReaderClosed) {
		t.Fatalf("closed Observe error=%v", err)
	}
}
