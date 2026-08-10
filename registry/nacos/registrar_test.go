package nacos

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/NeKiro-project/NeKiro/registry"
)

func TestRegistrarPublishesHeartbeatsAndDeregistersExactInstance(t *testing.T) {
	var mu sync.Mutex
	methods := make([]string, 0, 3)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		mu.Lock()
		methods = append(methods, request.Method)
		mu.Unlock()
		assertRegistrationRequest(t, request)
		if request.Method == http.MethodPut {
			_, _ = writer.Write([]byte(`{"clientBeatInterval":1000}`))
			return
		}
		_, _ = writer.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)
	registrar, registration := testRegistrar(t, server, server.Client())
	lease, err := registrar.Register(t.Context(), registration)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := registrar.Register(t.Context(), registration); !errors.Is(err, registry.ErrInvalid) {
		t.Fatalf("duplicate registration error=%v", err)
	}
	eventuallyRegistrar(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(methods) >= 2
	})
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if err := lease.Close(ctx); err != nil {
		t.Fatal(err)
	}
	<-lease.Done()
	if !errors.Is(lease.Err(), registry.ErrClosed) {
		t.Fatalf("lease terminal=%v", lease.Err())
	}
	mu.Lock()
	defer mu.Unlock()
	if len(methods) != 3 || methods[0] != http.MethodPost || methods[1] != http.MethodPut || methods[2] != http.MethodDelete {
		t.Fatalf("methods=%v", methods)
	}
}

func TestRegistrarHeartbeatFailureTerminatesLeaseWithoutRetry(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests++
		if request.Method == http.MethodPut {
			writer.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = writer.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)
	registrar, registration := testRegistrar(t, server, server.Client())
	lease, err := registrar.Register(t.Context(), registration)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-lease.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("heartbeat failure did not terminate lease")
	}
	if !errors.Is(lease.Err(), registry.ErrUnavailable) || requests != 2 {
		t.Fatalf("lease terminal=%v requests=%d", lease.Err(), requests)
	}
}

func TestRegistrarRejectsInvalidAndCanceledRegistration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) { _, _ = writer.Write([]byte("ok")) }))
	t.Cleanup(server.Close)
	registrar, registration := testRegistrar(t, server, server.Client())
	if _, err := registrar.Register(nil, registration); !errors.Is(err, registry.ErrInvalid) {
		t.Fatalf("nil context error=%v", err)
	}
	canceledContext, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := registrar.Register(canceledContext, registration); !errors.Is(err, registry.ErrCanceled) {
		t.Fatalf("canceled context error=%v", err)
	}
	otherTarget, err := registry.NewReleaseTarget(registry.ReleaseTargetInput{
		AgentID: "runtime-c", AgentCardVersion: "1.0.0", ReleaseID: "release-c",
		CardDigest: strings.Repeat("c", 64), CanonicalEndpoint: "http://runtime-c:8093/a2a", Audience: "http://runtime-c:8093",
	})
	if err != nil {
		t.Fatal(err)
	}
	other, _ := registry.NewRegistration(registry.RegistrationInput{Target: otherTarget, Instance: registration.Instance()})
	if _, err := registrar.Register(t.Context(), other); !errors.Is(err, registry.ErrInvalid) {
		t.Fatalf("mismatched target error=%v", err)
	}
	reservedInstance, _ := registry.NewInstance(registry.InstanceInput{
		ID: "runtime-b-instance", Endpoints: registration.Instance().Endpoints(), Ready: true, Serving: true,
		Metadata: map[string]string{heartbeatTimeoutMetadataKey: "1"},
	})
	reserved, _ := registry.NewRegistration(registry.RegistrationInput{Target: registration.Target(), Instance: reservedInstance})
	if _, err := registrar.Register(t.Context(), reserved); !errors.Is(err, registry.ErrInvalid) {
		t.Fatalf("reserved metadata error=%v", err)
	}
	if err := registrar.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := registrar.Register(t.Context(), registration); !errors.Is(err, registry.ErrClosed) {
		t.Fatalf("closed registrar error=%v", err)
	}
}

func TestRegistrarRejectsProviderHeartbeatIntervalOverride(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodPut {
			_, _ = writer.Write([]byte(`{"clientBeatInterval":2000}`))
			return
		}
		_, _ = writer.Write([]byte("ok"))
	}))
	defer server.Close()
	registrar, registration := testRegistrar(t, server, server.Client())
	lease, err := registrar.Register(t.Context(), registration)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-lease.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("heartbeat interval override did not terminate lease")
	}
	if !errors.Is(lease.Err(), registry.ErrInvalid) {
		t.Fatalf("lease terminal=%v", lease.Err())
	}
}

func TestRegistrarClassifiesInitialRegistrationFailures(t *testing.T) {
	for name, status := range map[string]int{
		"unauthorized": http.StatusUnauthorized,
		"forbidden":    http.StatusForbidden,
		"unavailable":  http.StatusServiceUnavailable,
		"rate limited": http.StatusTooManyRequests,
	} {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) { writer.WriteHeader(status) }))
			defer server.Close()
			registrar, registration := testRegistrar(t, server, server.Client())
			_, err := registrar.Register(t.Context(), registration)
			want := registry.ErrUnavailable
			if status == http.StatusUnauthorized || status == http.StatusForbidden {
				want = registry.ErrUnauthorized
			}
			if !errors.Is(err, want) {
				t.Fatalf("Register error=%v want=%v", err, want)
			}
		})
	}
	t.Run("malformed acknowledgement", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) { _, _ = writer.Write([]byte("unexpected")) }))
		defer server.Close()
		registrar, registration := testRegistrar(t, server, server.Client())
		if _, err := registrar.Register(t.Context(), registration); !errors.Is(err, registry.ErrInvalid) {
			t.Fatalf("Register error=%v", err)
		}
	})
}

func TestRegistrarCloseDuringInitialRegistrationDoesNotPublishLease(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	methods := make(chan string, 2)
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	executor := requestExecutorFunc(func(request *http.Request) (*http.Response, error) {
		methods <- request.Method
		if request.Method == http.MethodPost {
			close(entered)
			<-release
		}
		if request.Method == http.MethodDelete && request.Context().Err() != nil {
			return nil, request.Context().Err()
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("ok"))}, nil
	})
	registrar, registration := testRegistrar(t, server, executor)
	registerContext, cancelRegister := context.WithCancel(t.Context())
	result := make(chan error, 1)
	go func() {
		_, err := registrar.Register(registerContext, registration)
		result <- err
	}()
	<-entered
	closeResult := make(chan error, 1)
	go func() { closeResult <- registrar.Close() }()
	select {
	case err := <-closeResult:
		t.Fatalf("Registrar.Close returned before initial registration cleanup: %v", err)
	case <-time.After(25 * time.Millisecond):
	}
	cancelRegister()
	close(release)
	if err := <-closeResult; err != nil {
		t.Fatal(err)
	}
	if err := <-result; !errors.Is(err, registry.ErrClosed) || errors.Is(err, registry.ErrCanceled) {
		t.Fatalf("Register error=%v", err)
	}
	if first, second := <-methods, <-methods; first != http.MethodPost || second != http.MethodDelete {
		t.Fatalf("methods=%s,%s", first, second)
	}
}

func TestLeaseClosePreservesDeregistrationFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodDelete {
			writer.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = writer.Write([]byte("ok"))
	}))
	defer server.Close()
	registrar, registration := testRegistrar(t, server, server.Client())
	lease, err := registrar.Register(t.Context(), registration)
	if err != nil {
		t.Fatal(err)
	}
	if err := lease.Close(t.Context()); !errors.Is(err, registry.ErrUnavailable) {
		t.Fatalf("Close error=%v", err)
	}
	<-lease.Done()
	if !errors.Is(lease.Err(), registry.ErrUnavailable) {
		t.Fatalf("lease terminal=%v", lease.Err())
	}
}

func TestLeaseCloseHonorsCancellationBeforeDeregistration(t *testing.T) {
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()
	registrar, registration := testRegistrar(t, server, requestExecutorFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("must not execute")
	}))
	endpoint, err := registrationEndpoint(registration.Instance(), registrar.portName)
	if err != nil {
		t.Fatal(err)
	}
	session := &registrationSession{registrar: registrar, registration: registration, endpoint: endpoint, stop: make(chan struct{}), stopped: make(chan struct{})}
	lease, _ := registry.NewLease(session.close)
	session.lease = lease
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := lease.Close(ctx); !errors.Is(err, registry.ErrCanceled) {
		t.Fatalf("Close error=%v", err)
	}
	if !errors.Is(lease.Err(), registry.ErrCanceled) {
		t.Fatalf("lease terminal=%v", lease.Err())
	}
}

func TestRegistrationEndpointRequiresOneMatchingIPPort(t *testing.T) {
	endpoint, _ := registry.NewNetworkEndpoint(registry.NetworkEndpointInput{AddressType: registry.AddressTypeIPv4, Address: "127.0.0.1", PortName: "other", Port: 8092, Protocol: registry.TransportProtocolTCP})
	instance, _ := registry.NewInstance(registry.InstanceInput{ID: "runtime-b", Endpoints: []registry.NetworkEndpoint{endpoint}, Ready: true, Serving: true})
	if _, err := registrationEndpoint(instance, "a2a"); !errors.Is(err, registry.ErrInvalid) {
		t.Fatalf("missing endpoint error=%v", err)
	}
	matching, _ := registry.NewNetworkEndpoint(registry.NetworkEndpointInput{AddressType: registry.AddressTypeIPv4, Address: "127.0.0.2", PortName: "a2a", Port: 8092, Protocol: registry.TransportProtocolTCP})
	second, _ := registry.NewNetworkEndpoint(registry.NetworkEndpointInput{AddressType: registry.AddressTypeIPv4, Address: "127.0.0.3", PortName: "a2a", Port: 8092, Protocol: registry.TransportProtocolTCP})
	multiple, _ := registry.NewInstance(registry.InstanceInput{ID: "runtime-b", Endpoints: []registry.NetworkEndpoint{matching, second}, Ready: true, Serving: true})
	if _, err := registrationEndpoint(multiple, "a2a"); !errors.Is(err, registry.ErrInvalid) {
		t.Fatalf("multiple endpoint error=%v", err)
	}
}

func TestNewRegistrarRequiresExplicitLifecycleConfiguration(t *testing.T) {
	target := testTarget(t)
	binding, _ := NewBinding(BindingInput{Target: target, ServiceName: "runtime-b", GroupName: "NEKIRO", ClusterName: "DEFAULT"})
	base := RegistrarConfig{
		APIOrigin: "http://nacos.test/nacos", NamespaceID: "public", Binding: binding, PortName: "a2a",
		Weight: 100, HeartbeatInterval: time.Second, HeartbeatTimeout: 3 * time.Second, IPDeleteTimeout: 6 * time.Second,
		AuthMode: AuthNone, Executor: http.DefaultClient,
	}
	for name, mutate := range map[string]func(*RegistrarConfig){
		"origin":            func(config *RegistrarConfig) { config.APIOrigin = "http://nacos.test/other" },
		"namespace":         func(config *RegistrarConfig) { config.NamespaceID = "" },
		"binding":           func(config *RegistrarConfig) { config.Binding = Binding{} },
		"port":              func(config *RegistrarConfig) { config.PortName = "" },
		"weight":            func(config *RegistrarConfig) { config.Weight = 0 },
		"heartbeat":         func(config *RegistrarConfig) { config.HeartbeatInterval = 0 },
		"heartbeat timeout": func(config *RegistrarConfig) { config.HeartbeatTimeout = config.HeartbeatInterval },
		"delete timeout":    func(config *RegistrarConfig) { config.IPDeleteTimeout = config.HeartbeatTimeout },
		"executor":          func(config *RegistrarConfig) { config.Executor = nil },
		"auth mode":         func(config *RegistrarConfig) { config.AuthMode = "implicit" },
		"auth secret":       func(config *RegistrarConfig) { config.AuthMode = AuthAccessToken; config.AccessToken = "" },
	} {
		t.Run(name, func(t *testing.T) {
			candidate := base
			mutate(&candidate)
			if _, err := NewRegistrar(candidate); !errors.Is(err, registry.ErrInvalid) {
				t.Fatalf("NewRegistrar error=%v", err)
			}
		})
	}
	registrar, err := NewRegistrar(base)
	if err != nil {
		t.Fatal(err)
	}
	for _, capability := range []registry.Capability{registry.CapabilityRegistration, registry.CapabilityDeregistration, registry.CapabilityLease, registry.CapabilityHeartbeat} {
		if !registrar.Capabilities().Supports(capability) {
			t.Fatalf("missing capability %s", capability)
		}
	}
}

func testRegistrar(t *testing.T, server *httptest.Server, executor RequestExecutor) (*Registrar, registry.Registration) {
	t.Helper()
	target := testTarget(t)
	binding, _ := NewBinding(BindingInput{Target: target, ServiceName: "runtime-b", GroupName: "NEKIRO", ClusterName: "DEFAULT"})
	endpoint, _ := registry.NewNetworkEndpoint(registry.NetworkEndpointInput{AddressType: registry.AddressTypeIPv4, Address: "127.0.0.1", PortName: "a2a", Port: 8092, Protocol: registry.TransportProtocolTCP})
	instance, _ := registry.NewInstance(registry.InstanceInput{ID: "runtime-b-instance", Endpoints: []registry.NetworkEndpoint{endpoint}, Ready: true, Serving: true})
	registration, _ := registry.NewRegistration(registry.RegistrationInput{Target: target, Instance: instance})
	registrar, err := NewRegistrar(RegistrarConfig{
		APIOrigin: server.URL + "/nacos", NamespaceID: "public", Binding: binding, PortName: "a2a",
		Weight: 100, HeartbeatInterval: time.Second, HeartbeatTimeout: 3 * time.Second, IPDeleteTimeout: 6 * time.Second,
		AuthMode: AuthNone, Executor: executor,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = registrar.Close() })
	return registrar, registration
}

func assertRegistrationRequest(t *testing.T, request *http.Request) {
	t.Helper()
	query := request.URL.Query()
	if request.URL.Path != "/nacos/v1/ns/instance" && request.URL.Path != "/nacos/v1/ns/instance/beat" ||
		query.Get("namespaceId") != "public" || query.Get("serviceName") != "NEKIRO@@runtime-b" ||
		query.Get("groupName") != "NEKIRO" || query.Get("clusterName") != "DEFAULT" ||
		query.Get("ip") != "127.0.0.1" || query.Get("port") != strconv.Itoa(8092) || query.Get("ephemeral") != "true" ||
		query.Get("enable") != "true" || query.Get("healthy") != "true" || query.Get("weight") != "100" ||
		!strings.Contains(query.Get("metadata"), "runtime-b-instance") || !strings.Contains(query.Get("metadata"), heartbeatTimeoutMetadataKey) {
		t.Errorf("registration request=%s", request.URL.String())
	}
	if request.Method == http.MethodPut && !strings.Contains(query.Get("beat"), "runtime-b-instance") {
		t.Errorf("heartbeat request=%s", request.URL.String())
	}
}

func eventuallyRegistrar(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition was not met")
}
