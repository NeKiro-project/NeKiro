package nacos

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	configcenter "github.com/NeKiro-project/NeKiro/config_center"
)

func TestReaderGetsExactNacosConfigWithoutFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/nacos/v1/cs/configs" || request.URL.Query().Get("dataId") != "router.nacos-bindings" || request.URL.Query().Get("group") != "NEKIRO" || request.URL.Query().Get("tenant") != "public" || request.URL.Query().Get("accessToken") != "token-a" {
			t.Fatalf("request URL=%s", request.URL.String())
		}
		_, _ = writer.Write([]byte("binding"))
	}))
	t.Cleanup(server.Close)
	reader, err := NewReader(ReaderConfig{APIOrigin: server.URL + "/nacos", NamespaceID: "public", GroupName: "NEKIRO", MaxPayloadBytes: 64, AuthMode: AuthAccessToken, AccessToken: "token-a", Executor: server.Client()})
	if err != nil {
		t.Fatal(err)
	}
	key, _ := configcenter.ParseKey("router.nacos-bindings")
	snapshot, err := reader.Get(t.Context(), key)
	if err != nil || !snapshot.Present() || string(snapshot.Content()) != "binding" {
		t.Fatalf("Get = %#v, %v", snapshot, err)
	}
	if _, err := reader.Observe(t.Context(), key); !errors.Is(err, configcenter.ErrUnsupported) {
		t.Fatalf("Observe error=%v", err)
	}
}

func TestReaderClassifiesNacosFailuresAndBounds(t *testing.T) {
	key, _ := configcenter.ParseKey("router.nacos-bindings")
	for name, test := range map[string]struct {
		status int
		body   string
		want   error
	}{
		"missing":      {status: http.StatusNotFound, want: configcenter.ErrMissing},
		"unauthorized": {status: http.StatusForbidden, want: configcenter.ErrUnauthorized},
		"unavailable":  {status: http.StatusServiceUnavailable, want: configcenter.ErrUnavailable},
		"too large":    {status: http.StatusOK, body: "12345", want: configcenter.ErrPayloadTooLarge},
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(test.status)
				_, _ = writer.Write([]byte(test.body))
			}))
			defer server.Close()
			reader, _ := NewReader(ReaderConfig{APIOrigin: server.URL + "/nacos", NamespaceID: "public", GroupName: "NEKIRO", MaxPayloadBytes: 4, AuthMode: AuthNone, Executor: server.Client()})
			_, err := reader.Get(t.Context(), key)
			if !errors.Is(err, test.want) {
				t.Fatalf("Get error=%v want=%v", err, test.want)
			}
		})
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	reader, _ := NewReader(ReaderConfig{APIOrigin: "http://nacos.test/nacos", NamespaceID: "public", GroupName: "NEKIRO", MaxPayloadBytes: 4, AuthMode: AuthNone, Executor: http.DefaultClient})
	if _, err := reader.Get(ctx, key); !errors.Is(err, context.Canceled) || !errors.Is(err, configcenter.ErrCanceled) {
		t.Fatalf("canceled Get error=%v", err)
	}
	_ = reader.Close()
	if _, err := reader.Get(t.Context(), key); !errors.Is(err, configcenter.ErrReaderClosed) {
		t.Fatalf("closed Get error=%v", err)
	}
}

func TestReaderMapsProviderNeutralKeysWithoutCollisions(t *testing.T) {
	exact, _ := configcenter.ParseKey("router.nacos-bindings")
	mapped, _ := configcenter.ParseKey("router/nacos-bindings")
	reserved, _ := configcenter.ParseKey("nekiro.key.v1.literal")
	if got := dataIDForKey(exact); got != exact.String() {
		t.Fatalf("exact dataId=%q", got)
	}
	if got := dataIDForKey(mapped); got == mapped.String() || !ValidDataID(got) {
		t.Fatalf("mapped dataId=%q", got)
	}
	if got := dataIDForKey(reserved); got == reserved.String() || got == dataIDForKey(mapped) || !ValidDataID(got) {
		t.Fatalf("reserved dataId=%q", got)
	}
}

type requestExecutorFunc func(*http.Request) (*http.Response, error)

func (function requestExecutorFunc) Do(request *http.Request) (*http.Response, error) {
	return function(request)
}

func TestReaderRejectsImplicitNacosConfiguration(t *testing.T) {
	for _, config := range []ReaderConfig{
		{},
		{APIOrigin: "http://nacos.test/nacos", NamespaceID: "public", GroupName: "NEKIRO", MaxPayloadBytes: 1, AuthMode: AuthNone},
		{APIOrigin: "http://nacos.test/nacos", NamespaceID: "public", GroupName: "NEKIRO", MaxPayloadBytes: 1, AuthMode: AuthNone, AccessToken: "unexpected", Executor: http.DefaultClient},
		{APIOrigin: "http://nacos.test/nacos", NamespaceID: "public", GroupName: "NEKIRO", MaxPayloadBytes: 1, WatchEnabled: true, LongPollTimeout: 99 * time.Millisecond, SubscriptionBuffer: 1, AuthMode: AuthNone, Executor: http.DefaultClient},
		{APIOrigin: "http://nacos.test/nacos", NamespaceID: "public", GroupName: "NEKIRO", MaxPayloadBytes: 1, WatchEnabled: true, LongPollTimeout: 100 * time.Millisecond, SubscriptionBuffer: 0, AuthMode: AuthNone, Executor: http.DefaultClient},
		{APIOrigin: "http://nacos.test/nacos", NamespaceID: "public", GroupName: "NEKIRO", MaxPayloadBytes: 1, LongPollTimeout: time.Second, AuthMode: AuthNone, Executor: http.DefaultClient},
	} {
		if _, err := NewReader(config); !errors.Is(err, configcenter.ErrInvalid) {
			t.Fatalf("NewReader(%#v) error=%v", config, err)
		}
	}
}

func TestReaderClassifiesRequestLifecycleFailures(t *testing.T) {
	key, _ := configcenter.ParseKey("router.nacos-bindings")
	base := ReaderConfig{APIOrigin: "http://nacos.test/nacos", NamespaceID: "public", GroupName: "NEKIRO", MaxPayloadBytes: 64, AuthMode: AuthNone}
	base.Executor = requestExecutorFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("offline") })
	reader, _ := NewReader(base)
	if _, err := reader.Get(t.Context(), key); !errors.Is(err, configcenter.ErrUnavailable) {
		t.Fatalf("unavailable Get error=%v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	base.Executor = requestExecutorFunc(func(*http.Request) (*http.Response, error) {
		cancel()
		return nil, errors.New("canceled")
	})
	reader, _ = NewReader(base)
	if _, err := reader.Get(ctx, key); !errors.Is(err, configcenter.ErrCanceled) || !errors.Is(err, context.Canceled) {
		t.Fatalf("transport-canceled Get error=%v", err)
	}
	base.Executor = requestExecutorFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: failingReadCloser{}}, nil
	})
	reader, _ = NewReader(base)
	if _, err := reader.Get(t.Context(), key); !errors.Is(err, configcenter.ErrUnavailable) {
		t.Fatalf("read-failed Get error=%v", err)
	}
	var closingReader *Reader
	base.Executor = requestExecutorFunc(func(*http.Request) (*http.Response, error) {
		_ = closingReader.Close()
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("binding"))}, nil
	})
	closingReader, _ = NewReader(base)
	if _, err := closingReader.Get(t.Context(), key); !errors.Is(err, configcenter.ErrReaderClosed) {
		t.Fatalf("closed-during-read Get error=%v", err)
	}
	var missingClosingReader *Reader
	base.Executor = requestExecutorFunc(func(*http.Request) (*http.Response, error) {
		_ = missingClosingReader.Close()
		return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("missing"))}, nil
	})
	missingClosingReader, _ = NewReader(base)
	if _, err := missingClosingReader.Get(t.Context(), key); !errors.Is(err, configcenter.ErrReaderClosed) {
		t.Fatalf("closed-during-missing-read Get error=%v", err)
	}
	if _, err := reader.Get(nil, key); !errors.Is(err, configcenter.ErrInvalid) {
		t.Fatalf("nil-context Get error=%v", err)
	}
	if _, err := reader.Get(t.Context(), configcenter.Key{}); !errors.Is(err, configcenter.ErrInvalid) {
		t.Fatalf("invalid-key Get error=%v", err)
	}
	if err := canceledError(key, configcenter.OperationGet, errors.New("not a context error")); !errors.Is(err, configcenter.ErrCanceled) || !errors.Is(err, context.Canceled) {
		t.Fatalf("normalized canceled error=%v", err)
	}
}

func TestValidDataIDAndConfigurationTextBoundaries(t *testing.T) {
	if !ValidDataID("A-z_0.:-9") {
		t.Fatal("safe data ID was rejected")
	}
	for _, value := range []string{"", strings.Repeat("a", 256), "router/bindings", "bad value"} {
		if ValidDataID(value) {
			t.Fatalf("unsafe data ID %q was accepted", value)
		}
	}
	for _, config := range []ReaderConfig{
		{APIOrigin: "http://user@nacos.test/nacos", NamespaceID: "public", GroupName: "NEKIRO", MaxPayloadBytes: 1, AuthMode: AuthNone, Executor: http.DefaultClient},
		{APIOrigin: "http://nacos.test/nacos", NamespaceID: "public\n", GroupName: "NEKIRO", MaxPayloadBytes: 1, AuthMode: AuthNone, Executor: http.DefaultClient},
		{APIOrigin: "http://nacos.test/nacos", NamespaceID: "public", GroupName: strings.Repeat("g", 257), MaxPayloadBytes: 1, AuthMode: AuthNone, Executor: http.DefaultClient},
	} {
		if _, err := NewReader(config); !errors.Is(err, configcenter.ErrInvalid) {
			t.Fatalf("NewReader(%#v) error=%v", config, err)
		}
	}
}

type failingReadCloser struct{}

func (failingReadCloser) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (failingReadCloser) Close() error             { return nil }
