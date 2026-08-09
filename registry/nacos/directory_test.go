package nacos

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NeKiro-project/NeKiro/registry"
)

type bindingSource struct {
	binding Binding
	err     error
}

func (source bindingSource) Binding(context.Context, registry.ReleaseTarget) (Binding, error) {
	return source.binding, source.err
}

type requestExecutorFunc func(*http.Request) (*http.Response, error)

func (function requestExecutorFunc) Do(request *http.Request) (*http.Response, error) {
	return function(request)
}

type failingReader struct{}

func (failingReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (failingReader) Close() error             { return nil }

type invalidGuaranteeSubscriber struct{}

func (invalidGuaranteeSubscriber) Guarantees() NamingSubscriptionGuarantees {
	return NamingSubscriptionGuarantees{}
}

func (invalidGuaranteeSubscriber) Subscribe(context.Context, NamingSubscribeRequest) (NamingSubscription, error) {
	return NamingSubscription{}, registry.ErrUnavailable
}

func TestDirectoryRejectsImplicitObservationPolicy(t *testing.T) {
	target := testTarget(t)
	binding, _ := NewBinding(BindingInput{Target: target, ServiceName: "runtime-b", GroupName: "NEKIRO", ClusterName: "DEFAULT"})
	base := DirectoryConfig{APIOrigin: "http://nacos.test/nacos", NamespaceID: "public", PortName: "a2a", MaxResponseBytes: 4096, AuthMode: AuthNone, Executor: http.DefaultClient, Bindings: bindingSource{binding: binding}}
	base.PendingChanges = 1
	if _, err := NewDirectory(base); !errors.Is(err, registry.ErrInvalid) {
		t.Fatalf("queue without subscriber error=%v", err)
	}
	base.Subscriber = invalidGuaranteeSubscriber{}
	if _, err := NewDirectory(base); !errors.Is(err, registry.ErrInvalid) {
		t.Fatalf("invalid subscriber guarantees error=%v", err)
	}
	var typedNil *fixtureSubscriber
	base.Subscriber = typedNil
	if _, err := NewDirectory(base); !errors.Is(err, registry.ErrInvalid) {
		t.Fatalf("typed nil subscriber error=%v", err)
	}
}

func TestDirectoryReadsOneExactNacosSnapshot(t *testing.T) {
	target := testTarget(t)
	binding, _ := NewBinding(BindingInput{Target: target, ServiceName: "runtime-b", GroupName: "NEKIRO", ClusterName: "DEFAULT"})
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		query := request.URL.Query()
		if request.URL.Path != "/nacos/v1/ns/instance/list" || query.Get("serviceName") != "runtime-b" || query.Get("groupName") != "NEKIRO" || query.Get("clusters") != "DEFAULT" || query.Get("namespaceId") != "public" || query.Get("healthyOnly") != "false" {
			t.Errorf("request URL=%s", request.URL.String())
		}
		_, _ = writer.Write([]byte(`{"hosts":[{"instanceId":"provider-generated","ip":"172.28.0.12","port":8092,"healthy":true,"enabled":true,"ephemeral":true,"clusterName":"DEFAULT","metadata":{"nekiro.instanceId":"runtime-b-directory"}}]}`))
	}))
	t.Cleanup(server.Close)
	directory, err := NewDirectory(DirectoryConfig{APIOrigin: server.URL + "/nacos", NamespaceID: "public", PortName: "a2a", MaxResponseBytes: 4096, AuthMode: AuthNone, Executor: server.Client(), Bindings: bindingSource{binding: binding}})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := directory.Snapshot(t.Context(), target)
	if err != nil || requests != 1 || snapshot.State() != registry.SnapshotStatePopulated || len(snapshot.Instances()) != 1 || snapshot.Instances()[0].ID() != "runtime-b-directory" || snapshot.Instances()[0].State() != registry.InstanceStateReady {
		t.Fatalf("Snapshot=%#v requests=%d error=%v", snapshot, requests, err)
	}
}

func TestDirectoryKeepsUnhealthyNacosInstanceUnavailable(t *testing.T) {
	target := testTarget(t)
	binding, _ := NewBinding(BindingInput{Target: target, ServiceName: "runtime-b", GroupName: "NEKIRO", ClusterName: "DEFAULT"})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`{"hosts":[{"instanceId":"provider-generated","ip":"172.28.0.12","port":8092,"healthy":false,"enabled":true,"ephemeral":true,"clusterName":"DEFAULT","metadata":{"nekiro.instanceId":"runtime-b-directory"}}]}`))
	}))
	defer server.Close()
	directory, _ := NewDirectory(DirectoryConfig{APIOrigin: server.URL + "/nacos", NamespaceID: "public", PortName: "a2a", MaxResponseBytes: 4096, AuthMode: AuthNone, Executor: server.Client(), Bindings: bindingSource{binding: binding}})
	snapshot, err := directory.Snapshot(t.Context(), target)
	if err != nil || snapshot.Instances()[0].State() != registry.InstanceStateUnavailable {
		t.Fatalf("Snapshot=%#v error=%v", snapshot, err)
	}
}

func TestDirectoryRejectsProviderIdentityAndFailureChanges(t *testing.T) {
	target := testTarget(t)
	binding, _ := NewBinding(BindingInput{Target: target, ServiceName: "runtime-b", GroupName: "NEKIRO", ClusterName: "DEFAULT"})
	for name, test := range map[string]struct {
		status int
		body   string
		want   error
	}{
		"unauthorized":        {status: http.StatusUnauthorized, want: registry.ErrUnauthorized},
		"unavailable":         {status: http.StatusServiceUnavailable, want: registry.ErrUnavailable},
		"persistent":          {status: http.StatusOK, body: `{"hosts":[{"ip":"172.28.0.12","port":8092,"healthy":true,"enabled":true,"ephemeral":false,"clusterName":"DEFAULT","metadata":{"nekiro.instanceId":"runtime-b-directory"}}]}`, want: registry.ErrInvalid},
		"wrong cluster":       {status: http.StatusOK, body: `{"hosts":[{"ip":"172.28.0.12","port":8092,"healthy":true,"enabled":true,"ephemeral":true,"clusterName":"OTHER","metadata":{"nekiro.instanceId":"runtime-b-directory"}}]}`, want: registry.ErrInvalid},
		"missing instance ID": {status: http.StatusOK, body: `{"hosts":[{"ip":"172.28.0.12","port":8092,"healthy":true,"enabled":true,"ephemeral":true,"clusterName":"DEFAULT","metadata":{}}]}`, want: registry.ErrInvalid},
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(test.status)
				_, _ = writer.Write([]byte(test.body))
			}))
			defer server.Close()
			directory, _ := NewDirectory(DirectoryConfig{APIOrigin: server.URL + "/nacos", NamespaceID: "public", PortName: "a2a", MaxResponseBytes: 4096, AuthMode: AuthNone, Executor: server.Client(), Bindings: bindingSource{binding: binding}})
			_, err := directory.Snapshot(t.Context(), target)
			if !errors.Is(err, test.want) {
				t.Fatalf("Snapshot error=%v want=%v", err, test.want)
			}
		})
	}
}

func TestBindingAndDirectorySurfaceRejectInvalidState(t *testing.T) {
	target := testTarget(t)
	binding, err := NewBinding(BindingInput{Target: target, ServiceName: "runtime-b", GroupName: "NEKIRO", ClusterName: "DEFAULT"})
	if err != nil || !binding.Target().Equal(target) || binding.ServiceName() != "runtime-b" || binding.GroupName() != "NEKIRO" || binding.ClusterName() != "DEFAULT" {
		t.Fatalf("binding=%#v error=%v", binding, err)
	}
	if _, err := NewBinding(BindingInput{Target: target, ServiceName: " bad", GroupName: "NEKIRO", ClusterName: "DEFAULT"}); !errors.Is(err, registry.ErrInvalid) {
		t.Fatalf("invalid binding error=%v", err)
	}
	valid := DirectoryConfig{APIOrigin: "http://nacos.test/nacos", NamespaceID: "public", PortName: "a2a", MaxResponseBytes: 64, AuthMode: AuthNone, Executor: http.DefaultClient, Bindings: bindingSource{binding: binding}}
	for name, mutate := range map[string]func(*DirectoryConfig){
		"origin":        func(value *DirectoryConfig) { value.APIOrigin = "http://nacos.test/wrong" },
		"namespace":     func(value *DirectoryConfig) { value.NamespaceID = " public" },
		"port name":     func(value *DirectoryConfig) { value.PortName = "" },
		"response size": func(value *DirectoryConfig) { value.MaxResponseBytes = 0 },
		"executor":      func(value *DirectoryConfig) { value.Executor = nil },
		"bindings":      func(value *DirectoryConfig) { value.Bindings = nil },
		"auth mode":     func(value *DirectoryConfig) { value.AuthMode = "implicit" },
		"token none":    func(value *DirectoryConfig) { value.AccessToken = "token" },
		"empty token": func(value *DirectoryConfig) {
			value.AuthMode = AuthAccessToken
			value.AccessToken = ""
		},
	} {
		t.Run(name, func(t *testing.T) {
			candidate := valid
			mutate(&candidate)
			if _, err := NewDirectory(candidate); !errors.Is(err, registry.ErrInvalid) {
				t.Fatalf("NewDirectory error=%v", err)
			}
		})
	}
	directory, err := NewDirectory(valid)
	if err != nil || !directory.Capabilities().Supports(registry.CapabilitySnapshot) {
		t.Fatalf("directory=%v error=%v", directory, err)
	}
	if _, err := directory.Observe(t.Context(), target); !errors.Is(err, registry.ErrInvalid) {
		t.Fatalf("Observe error=%v", err)
	}
	if _, err := directory.Snapshot(nil, target); !errors.Is(err, registry.ErrInvalid) {
		t.Fatalf("nil-context Snapshot error=%v", err)
	}
	canceledContext, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := directory.Snapshot(canceledContext, target); !errors.Is(err, registry.ErrCanceled) || !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled Snapshot error=%v", err)
	}
	if err := directory.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := directory.Snapshot(t.Context(), target); !errors.Is(err, registry.ErrClosed) {
		t.Fatalf("closed Snapshot error=%v", err)
	}
}

func TestDirectoryClassifiesTransportAndBindingFailures(t *testing.T) {
	target := testTarget(t)
	binding, _ := NewBinding(BindingInput{Target: target, ServiceName: "runtime-b", GroupName: "NEKIRO", ClusterName: "DEFAULT"})
	config := DirectoryConfig{APIOrigin: "http://nacos.test/nacos", NamespaceID: "public", PortName: "a2a", MaxResponseBytes: 4, AuthMode: AuthNone, Bindings: bindingSource{binding: binding}}
	config.Executor = requestExecutorFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("offline") })
	directory, _ := NewDirectory(config)
	if _, err := directory.Snapshot(t.Context(), target); !errors.Is(err, registry.ErrUnavailable) {
		t.Fatalf("unavailable Snapshot error=%v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	config.Executor = requestExecutorFunc(func(*http.Request) (*http.Response, error) {
		cancel()
		return nil, errors.New("canceled")
	})
	directory, _ = NewDirectory(config)
	if _, err := directory.Snapshot(ctx, target); !errors.Is(err, registry.ErrCanceled) || !errors.Is(err, context.Canceled) {
		t.Fatalf("transport-canceled Snapshot error=%v", err)
	}
	config.Executor = requestExecutorFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: failingReader{}}, nil
	})
	directory, _ = NewDirectory(config)
	if _, err := directory.Snapshot(t.Context(), target); !errors.Is(err, registry.ErrUnavailable) {
		t.Fatalf("read-failed Snapshot error=%v", err)
	}
	config.Executor = requestExecutorFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("12345"))}, nil
	})
	directory, _ = NewDirectory(config)
	if _, err := directory.Snapshot(t.Context(), target); !errors.Is(err, registry.ErrInvalid) {
		t.Fatalf("oversized Snapshot error=%v", err)
	}
	config.MaxResponseBytes = 4096
	config.Executor = http.DefaultClient
	config.Bindings = bindingSource{err: registry.ErrMissing}
	directory, _ = NewDirectory(config)
	if _, err := directory.Snapshot(t.Context(), target); !errors.Is(err, registry.ErrMissing) {
		t.Fatalf("binding failure Snapshot error=%v", err)
	}
	otherTarget, _ := registry.NewReleaseTarget(registry.ReleaseTargetInput{AgentID: "runtime-b", AgentCardVersion: "1.0.0", ReleaseID: "other", CardDigest: strings.Repeat("a", 64), CanonicalEndpoint: "http://runtime-b:8092/", Audience: "http://runtime-b:8092"})
	otherBinding, _ := NewBinding(BindingInput{Target: otherTarget, ServiceName: "runtime-b", GroupName: "NEKIRO", ClusterName: "DEFAULT"})
	config.Bindings = bindingSource{binding: otherBinding}
	directory, _ = NewDirectory(config)
	if _, err := directory.Snapshot(t.Context(), target); !errors.Is(err, registry.ErrInvalid) {
		t.Fatalf("mismatched binding Snapshot error=%v", err)
	}
	if err := canceled(errors.New("not a context error")); !errors.Is(err, registry.ErrCanceled) || !errors.Is(err, context.Canceled) {
		t.Fatalf("normalized canceled error=%v", err)
	}
}

func TestDirectoryHandlesMissingEmptyAndIPv6Snapshots(t *testing.T) {
	target := testTarget(t)
	binding, _ := NewBinding(BindingInput{Target: target, ServiceName: "runtime-b", GroupName: "NEKIRO", ClusterName: "DEFAULT"})
	for name, test := range map[string]struct {
		status int
		body   string
		want   error
		empty  bool
	}{
		"missing":       {status: http.StatusNotFound, want: registry.ErrMissing},
		"malformed":     {status: http.StatusOK, body: `{`, want: registry.ErrInvalid},
		"missing hosts": {status: http.StatusOK, body: `{}`, want: registry.ErrInvalid},
		"invalid IP":    {status: http.StatusOK, body: `{"hosts":[{"ip":"not-an-ip","port":8092,"healthy":true,"enabled":true,"ephemeral":true,"clusterName":"DEFAULT","metadata":{"nekiro.instanceId":"runtime-b-directory"}}]}`, want: registry.ErrInvalid},
		"invalid port":  {status: http.StatusOK, body: `{"hosts":[{"ip":"172.28.0.12","port":0,"healthy":true,"enabled":true,"ephemeral":true,"clusterName":"DEFAULT","metadata":{"nekiro.instanceId":"runtime-b-directory"}}]}`, want: registry.ErrInvalid},
		"invalid ID":    {status: http.StatusOK, body: `{"hosts":[{"ip":"172.28.0.12","port":8092,"healthy":true,"enabled":true,"ephemeral":true,"clusterName":"DEFAULT","metadata":{"nekiro.instanceId":" bad"}}]}`, want: registry.ErrInvalid},
		"empty":         {status: http.StatusOK, body: `{"hosts":[]}`, empty: true},
		"ipv6":          {status: http.StatusOK, body: `{"hosts":[{"ip":"2001:db8::1","port":8092,"healthy":true,"enabled":true,"ephemeral":true,"clusterName":"DEFAULT","metadata":{"nekiro.instanceId":"runtime-b-v6"}}]}`},
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if request.URL.Query().Get("accessToken") != "token-a" {
					t.Errorf("access token missing")
				}
				writer.WriteHeader(test.status)
				_, _ = writer.Write([]byte(test.body))
			}))
			defer server.Close()
			directory, _ := NewDirectory(DirectoryConfig{APIOrigin: server.URL + "/nacos", NamespaceID: "public", PortName: "a2a", MaxResponseBytes: 4096, AuthMode: AuthAccessToken, AccessToken: "token-a", Executor: server.Client(), Bindings: bindingSource{binding: binding}})
			snapshot, err := directory.Snapshot(t.Context(), target)
			if !errors.Is(err, test.want) {
				t.Fatalf("Snapshot error=%v want=%v", err, test.want)
			}
			if err == nil && test.empty && snapshot.State() != registry.SnapshotStateEmpty {
				t.Fatalf("empty Snapshot state=%v", snapshot.State())
			}
			if err == nil && !test.empty && (len(snapshot.Instances()) != 1 || snapshot.Instances()[0].Endpoints()[0].AddressType() != registry.AddressTypeIPv6) {
				t.Fatalf("IPv6 Snapshot=%#v", snapshot)
			}
		})
	}
}

func testTarget(t *testing.T) registry.ReleaseTarget {
	t.Helper()
	target, err := registry.NewReleaseTarget(registry.ReleaseTargetInput{AgentID: "runtime-b", AgentCardVersion: "1.0.0", ReleaseID: "release-b", CardDigest: strings.Repeat("a", 64), CanonicalEndpoint: "http://runtime-b:8092/", Audience: "http://runtime-b:8092"})
	if err != nil {
		t.Fatal(err)
	}
	return target
}
