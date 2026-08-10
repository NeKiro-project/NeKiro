package main

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/NeKiro-project/NeKiro/apps/a2a-router/internal/auth"
	"github.com/NeKiro-project/NeKiro/apps/a2a-router/internal/config"
	"github.com/NeKiro-project/NeKiro/apps/a2a-router/internal/credential"
	"github.com/NeKiro-project/NeKiro/apps/a2a-router/internal/nested"
	a2atransport "github.com/NeKiro-project/NeKiro/apps/a2a-router/internal/transport/a2a"
	configcenter "github.com/NeKiro-project/NeKiro/config_center"
	"github.com/NeKiro-project/NeKiro/contracts"
	"github.com/NeKiro-project/NeKiro/registry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthv1 "google.golang.org/grpc/health/grpc_health_v1"
)

type failingDoer struct{}

type topologySelectorStub struct{}

func (topologySelectorStub) Select(_ context.Context, target a2atransport.Target, _ a2atransport.ContextHeaders) (a2atransport.Target, error) {
	return target, nil
}

func (topologySelectorStub) TopologyStatus() contracts.RouterTopologyStatusV1 {
	return contracts.RouterTopologyStatusV1{
		SchemaVersion: contracts.RouterTopologyStatusSchemaVersion,
		Provider:      "nacos",
		Observations:  []contracts.RouterTopologyStatusObservationV1{},
	}
}

func (failingDoer) Do(*http.Request) (*http.Response, error) {
	panic("readiness must not probe dependencies")
}

type ledgerAppenderStub struct{}

func (ledgerAppenderStub) Append(context.Context, contracts.InvocationEventV03) error { return nil }
func (ledgerAppenderStub) GetInvocation(context.Context, string, string) (contracts.InvocationDetailResponseV4, error) {
	return contracts.InvocationDetailResponseV4{}, nil
}
func (ledgerAppenderStub) GetTrace(context.Context, string, contracts.TraceID) (contracts.TraceResponseV4, error) {
	return contracts.TraceResponseV4{}, nil
}
func (ledgerAppenderStub) GetInvocationByParentID(context.Context, string) (contracts.InvocationDetailResponseV4, error) {
	return contracts.InvocationDetailResponseV4{}, nil
}

func TestRunRequiresExplicitCommandAndMigrationDirection(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "missing command", want: "command is required: serve or migrate"},
		{name: "unknown command", args: []string{"status"}, want: `unknown command "status"`},
		{name: "migration direction", args: []string{"migrate"}, want: "migrate requires exactly one direction: up"},
		{name: "migration down", args: []string{"migrate", "down"}, want: "migrate requires exactly one direction: up"},
		{name: "serve arguments", args: []string{"serve", "extra"}, want: "serve accepts no arguments"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := run(context.Background(), test.args, nil)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error=%v want substring=%q", err, test.want)
			}
		})
	}
}

func TestNewHandlerAssemblesReadinessWithoutDependencyProbe(t *testing.T) {
	handler, err := newHandler(config.Config{
		ListenAddress:                  "127.0.0.1:9090",
		RouterPrincipals:               []auth.Principal{{ID: "router", TokenSHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}},
		AgentPrincipals:                []nested.AgentPrincipal{{WorkspaceID: "workspace-a", AgentID: "runtime-a", TokenSHA256: "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"}},
		ControlPlaneResolveURL:         "https://control.internal/internal/v2/resolve-agent",
		ControlPlaneVersionURL:         "https://control.internal/internal/v3/resolve-installed-version",
		ControlPlaneServiceToken:       "control-token",
		InternalRequestLimitBytes:      1024,
		AgentRequestLimitBytes:         1024,
		ControlPlaneResponseLimitBytes: 2048,
		AgentResponseLimitBytes:        4096,
		A2AEventLimitBytes:             4096,
		SSEEventLimitBytes:             4096,
		ResolutionDeadline:             time.Second,
		AgentDeadline:                  time.Second,
		AgentCredential:                credential.Config{Issuer: "https://a2a-router.nekiro.test", KeyID: "router-key-1", PrivateKey: ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize)), TTL: 30 * time.Second},
	}, failingDoer{}, &http.Client{}, ledgerAppenderStub{})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d", response.Code)
	}
	readResponse := httptest.NewRecorder()
	handler.ServeHTTP(readResponse, httptest.NewRequest(http.MethodGet, "/internal/v3/workspaces/workspace-a/invocations/inv-a", nil))
	if readResponse.Code != http.StatusUnauthorized {
		t.Fatalf("metadata read route status=%d, want 401", readResponse.Code)
	}
}

func TestNewHandlerRegistersTopologyStatusOnlyForObservedSelector(t *testing.T) {
	cfg := config.Config{
		RouterPrincipals:               []auth.Principal{{ID: "router", TokenSHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}},
		AgentPrincipals:                []nested.AgentPrincipal{{WorkspaceID: "workspace-a", AgentID: "runtime-a", TokenSHA256: "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"}},
		ControlPlaneResolveURL:         "https://control.internal/internal/v2/resolve-agent",
		ControlPlaneVersionURL:         "https://control.internal/internal/v3/resolve-installed-version",
		ControlPlaneServiceToken:       "control-token",
		InternalRequestLimitBytes:      1024,
		AgentRequestLimitBytes:         1024,
		ControlPlaneResponseLimitBytes: 2048,
		AgentResponseLimitBytes:        4096,
		A2AEventLimitBytes:             4096,
		SSEEventLimitBytes:             4096,
		ResolutionDeadline:             time.Second,
		AgentDeadline:                  time.Second,
		AgentCredential:                credential.Config{Issuer: "https://a2a-router.nekiro.test", KeyID: "router-key-1", PrivateKey: ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize)), TTL: 30 * time.Second},
	}
	handler, err := newHandlerWithTargetSelector(cfg, failingDoer{}, &http.Client{}, ledgerAppenderStub{}, topologySelectorStub{})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/internal/v1/instance-topology/status", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("observed topology route status=%d, want 401", response.Code)
	}

	direct, err := newHandlerWithTargetSelector(cfg, failingDoer{}, &http.Client{}, ledgerAppenderStub{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	response = httptest.NewRecorder()
	direct.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/internal/v1/instance-topology/status", nil))
	if response.Code != http.StatusNotFound {
		t.Fatalf("direct topology route status=%d, want 404", response.Code)
	}
}

func TestOpenInstanceDirectoryOwnsEachConfiguredMode(t *testing.T) {
	key, err := configcenter.ParseKey("router.nacos-bindings")
	if err != nil {
		t.Fatal(err)
	}
	if directory, err := openInstanceDirectory(config.Config{InstanceRoutingMode: config.InstanceRoutingDirect}); err != nil || directory != nil {
		t.Fatalf("direct directory=%v error=%v", directory, err)
	}
	fileDirectory, err := openInstanceDirectory(config.Config{
		InstanceRoutingMode: config.InstanceRoutingConfigCenterFile, ConfigCenterFileRoot: t.TempDir(),
		ConfigCenterMaxPayloadBytes: 64, InstanceDirectoryKey: key,
	})
	if err != nil || fileDirectory == nil {
		t.Fatalf("file directory=%v error=%v", fileDirectory, err)
	}
	if err := fileDirectory.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := openInstanceDirectory(config.Config{
		InstanceRoutingMode: config.InstanceRoutingConfigCenterFile, ConfigCenterFileRoot: t.TempDir(),
		ConfigCenterMaxPayloadBytes: 64,
	}); err == nil || !strings.Contains(err.Error(), "initialize Router instance directory") {
		t.Fatalf("invalid file directory error=%v", err)
	}
	nacosConfig := config.Config{
		InstanceRoutingMode: config.InstanceRoutingNacos, InstanceDirectoryKey: key, InstancePortName: "a2a",
		NacosAPIOrigin: "http://nacos.test/nacos", NacosNamespaceID: "nekiro", NacosConfigGroup: "NEKIRO",
		NacosAuthMode: config.NacosAuthNone, NacosResponseLimitBytes: 4096, NacosRequestTimeout: time.Second,
	}
	nacosDirectory, err := openInstanceDirectory(nacosConfig)
	if err != nil || nacosDirectory == nil || !nacosDirectory.Capabilities().Supports(registry.CapabilitySnapshot) {
		t.Fatalf("Nacos directory=%v error=%v", nacosDirectory, err)
	}
	if err := nacosDirectory.Close(); err != nil {
		t.Fatal(err)
	}
	tlsMaterial := newTLSMaterial(t)
	httpsConfig := nacosConfig
	httpsConfig.NacosAPIOrigin = "https://nacos.internal:8848/nacos"
	httpsConfig.NacosHTTPTLSCAFile = tlsMaterial.caFile
	httpsConfig.NacosHTTPTLSServerName = "nacos.internal"
	httpsDirectory, err := openInstanceDirectory(httpsConfig)
	if err != nil || httpsDirectory == nil || !httpsDirectory.Capabilities().Supports(registry.CapabilitySnapshot) {
		t.Fatalf("HTTPS Nacos directory=%v error=%v", httpsDirectory, err)
	}
	if err := httpsDirectory.Close(); err != nil {
		t.Fatal(err)
	}
	invalidHTTPSTLS := httpsConfig
	invalidHTTPSTLS.NacosHTTPTLSCAFile = filepath.Join(t.TempDir(), "missing.pem")
	if _, err := openInstanceDirectory(invalidHTTPSTLS); err == nil || !strings.Contains(err.Error(), "initialize Router Nacos HTTP transport security") || strings.Contains(err.Error(), invalidHTTPSTLS.NacosHTTPTLSCAFile) {
		t.Fatalf("invalid Nacos HTTPS error=%v", err)
	}
	observedConfig := nacosConfig
	observedConfig.NacosObserveEnabled = true
	observedConfig.NacosGRPCTarget = "nacos.test:9848"
	observedConfig.NacosGRPCClientIP = "127.0.0.1"
	observedConfig.NacosGRPCRequestTimeout = time.Second
	observedConfig.NacosPendingChanges = 4
	observedConfig.NacosGRPCTransportSecurity = "insecure"
	observedDirectory, err := openInstanceDirectory(observedConfig)
	if err != nil || observedDirectory == nil || !observedDirectory.Capabilities().Supports(registry.CapabilityObserve) {
		t.Fatalf("observed Nacos directory=%v error=%v", observedDirectory, err)
	}
	if err := observedDirectory.Close(); err != nil {
		t.Fatal(err)
	}
	tlsObservedConfig := observedConfig
	tlsObservedConfig.NacosGRPCTransportSecurity = config.NacosGRPCSecurityTLS
	tlsObservedConfig.NacosGRPCTLSCAFile = tlsMaterial.caFile
	tlsObservedConfig.NacosGRPCTLSServerName = "nacos.internal"
	tlsObservedDirectory, err := openInstanceDirectory(tlsObservedConfig)
	if err != nil || tlsObservedDirectory == nil || !tlsObservedDirectory.Capabilities().Supports(registry.CapabilityObserve) {
		t.Fatalf("TLS-observed Nacos directory=%v error=%v", tlsObservedDirectory, err)
	}
	if err := tlsObservedDirectory.Close(); err != nil {
		t.Fatal(err)
	}
	invalidTLS := tlsObservedConfig
	invalidTLS.NacosGRPCTLSCAFile = filepath.Join(t.TempDir(), "missing.pem")
	if _, err := openInstanceDirectory(invalidTLS); err == nil || !strings.Contains(err.Error(), "initialize Router Nacos gRPC transport security") || strings.Contains(err.Error(), invalidTLS.NacosGRPCTLSCAFile) {
		t.Fatalf("invalid Nacos TLS error=%v", err)
	}
	invalidGRPC := observedConfig
	invalidGRPC.NacosGRPCTarget = ""
	if _, err := openInstanceDirectory(invalidGRPC); err == nil || !strings.Contains(err.Error(), "initialize Router Nacos gRPC executor") {
		t.Fatalf("invalid Nacos gRPC executor error=%v", err)
	}
	invalidReader := nacosConfig
	invalidReader.NacosAPIOrigin = "http://nacos.test/wrong"
	if _, err := openInstanceDirectory(invalidReader); err == nil || !strings.Contains(err.Error(), "open Router Nacos Config Center reader") {
		t.Fatalf("invalid Nacos reader error=%v", err)
	}
	invalidDirectory := nacosConfig
	invalidDirectory.InstancePortName = ""
	if _, err := openInstanceDirectory(invalidDirectory); err == nil || !strings.Contains(err.Error(), "initialize Router Nacos instance directory") {
		t.Fatalf("invalid Nacos directory error=%v", err)
	}
}

func TestNacosHTTPClientDisablesAmbientNetworkBehavior(t *testing.T) {
	client, err := newNacosHTTPClient(config.Config{NacosAPIOrigin: "http://nacos.test/nacos", NacosRequestTimeout: 750 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok || transport.Proxy != nil || transport.TLSClientConfig != nil || !transport.DisableKeepAlives || client.Timeout != 750*time.Millisecond {
		t.Fatalf("client=%#v transport=%#v", client, transport)
	}
	if err := client.CheckRedirect(httptest.NewRequest(http.MethodGet, "http://nacos.test/next", nil), nil); err == nil || err.Error() != "Nacos redirects are disabled" {
		t.Fatalf("redirect error=%v", err)
	}
}

func TestNacosHTTPClientAuthenticatesPrivateCAAndOptionalClient(t *testing.T) {
	material := newTLSMaterial(t)

	for _, test := range []struct {
		name         string
		config       config.Config
		serverConfig *tls.Config
		wantError    bool
	}{
		{
			name:         "TLS",
			config:       config.Config{NacosAPIOrigin: "https://nacos.internal:8848/nacos", NacosRequestTimeout: 2 * time.Second, NacosHTTPTLSCAFile: material.caFile, NacosHTTPTLSServerName: "nacos.internal"},
			serverConfig: material.serverTLS(false),
		},
		{
			name:         "mTLS",
			config:       config.Config{NacosAPIOrigin: "https://nacos.internal:8848/nacos", NacosRequestTimeout: 2 * time.Second, NacosHTTPTLSCAFile: material.caFile, NacosHTTPTLSServerName: "nacos.internal", NacosHTTPTLSClientCertFile: material.clientCertFile, NacosHTTPTLSClientKeyFile: material.clientKeyFile},
			serverConfig: material.serverTLS(true),
		},
		{
			name:         "wrong CA",
			config:       config.Config{NacosAPIOrigin: "https://nacos.internal:8848/nacos", NacosRequestTimeout: 2 * time.Second, NacosHTTPTLSCAFile: newTLSMaterial(t).caFile, NacosHTTPTLSServerName: "nacos.internal"},
			serverConfig: material.serverTLS(false), wantError: true,
		},
		{
			name:         "wrong server name",
			config:       config.Config{NacosAPIOrigin: "https://nacos.internal:8848/nacos", NacosRequestTimeout: 2 * time.Second, NacosHTTPTLSCAFile: material.caFile, NacosHTTPTLSServerName: "other.internal"},
			serverConfig: material.serverTLS(false), wantError: true,
		},
		{
			name:         "missing client certificate",
			config:       config.Config{NacosAPIOrigin: "https://nacos.internal:8848/nacos", NacosRequestTimeout: 2 * time.Second, NacosHTTPTLSCAFile: material.caFile, NacosHTTPTLSServerName: "nacos.internal"},
			serverConfig: material.serverTLS(true), wantError: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewUnstartedServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				response.WriteHeader(http.StatusNoContent)
			}))
			server.TLS = test.serverConfig
			server.StartTLS()
			defer server.Close()

			client, err := newNacosHTTPClient(test.config)
			if err != nil {
				t.Fatalf("construct Nacos HTTP client: %v", err)
			}
			response, err := client.Get(server.URL)
			if response != nil {
				_ = response.Body.Close()
			}
			if (err != nil) != test.wantError {
				t.Fatalf("HTTPS request error=%v wantError=%v", err, test.wantError)
			}
			if !test.wantError && response.StatusCode != http.StatusNoContent {
				t.Fatalf("HTTPS status=%d", response.StatusCode)
			}
		})
	}
}

func TestOpenInstanceDirectoryUsesAuthenticatedHTTPSForBindingAndSnapshot(t *testing.T) {
	material := newTLSMaterial(t)
	digest := strings.Repeat("a", 64)
	binding := `{"schemaVersion":"1","revision":"test-1","targets":[{"agentId":"runtime-a","agentCardVersion":"1.0.0","releaseId":"release-a","cardDigest":"` + digest + `","canonicalEndpoint":"http://runtime-a:8091/","audience":"http://runtime-a:8091","serviceName":"runtime-a","groupName":"NEKIRO","clusterName":"DEFAULT"}]}`
	requests := map[string]int{}
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requests[request.URL.Path]++
		switch request.URL.Path {
		case "/nacos/v1/cs/configs":
			query := request.URL.Query()
			if query.Get("dataId") != "router.nacos-bindings" || query.Get("group") != "NEKIRO" || query.Get("tenant") != "public" {
				t.Errorf("binding query=%s", request.URL.RawQuery)
			}
			_, _ = response.Write([]byte(binding))
		case "/nacos/v1/ns/instance/list":
			query := request.URL.Query()
			if query.Get("serviceName") != "runtime-a" || query.Get("groupName") != "NEKIRO" || query.Get("clusters") != "DEFAULT" || query.Get("namespaceId") != "public" {
				t.Errorf("snapshot query=%s", request.URL.RawQuery)
			}
			_, _ = response.Write([]byte(`{"name":"NEKIRO@@runtime-a","groupName":"NEKIRO","clusters":"DEFAULT","hosts":[{"instanceId":"provider-generated","ip":"127.0.0.1","port":8091,"healthy":true,"enabled":true,"ephemeral":true,"clusterName":"DEFAULT","metadata":{"nekiro.instanceId":"runtime-a-1"}}]}`))
		default:
			http.NotFound(response, request)
		}
	}))
	server.TLS = material.serverTLS(false)
	server.StartTLS()
	defer server.Close()
	key, err := configcenter.ParseKey("router.nacos-bindings")
	if err != nil {
		t.Fatal(err)
	}
	directory, err := openInstanceDirectory(config.Config{
		InstanceRoutingMode:     config.InstanceRoutingNacos,
		InstanceDirectoryKey:    key,
		InstancePortName:        "a2a",
		NacosAPIOrigin:          server.URL + "/nacos",
		NacosNamespaceID:        "public",
		NacosConfigGroup:        "NEKIRO",
		NacosAuthMode:           config.NacosAuthNone,
		NacosResponseLimitBytes: 4096,
		NacosRequestTimeout:     2 * time.Second,
		NacosHTTPTLSCAFile:      material.caFile,
		NacosHTTPTLSServerName:  "nacos.internal",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer directory.Close()
	target, err := registry.NewReleaseTarget(registry.ReleaseTargetInput{
		AgentID: "runtime-a", AgentCardVersion: "1.0.0", ReleaseID: "release-a",
		CardDigest: digest, CanonicalEndpoint: "http://runtime-a:8091/", Audience: "http://runtime-a:8091",
	})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err := directory.Snapshot(t.Context(), target)
	if err != nil || snapshot.State() != registry.SnapshotStatePopulated || len(snapshot.Instances()) != 1 || snapshot.Instances()[0].ID() != "runtime-a-1" {
		t.Fatalf("snapshot=%#v error=%v", snapshot, err)
	}
	if requests["/nacos/v1/cs/configs"] != 1 || requests["/nacos/v1/ns/instance/list"] != 1 {
		t.Fatalf("HTTPS requests=%v", requests)
	}
}

func TestNacosHTTPClientRejectsSchemeAndTLSFieldMismatch(t *testing.T) {
	material := newTLSMaterial(t)
	for _, cfg := range []config.Config{
		{NacosAPIOrigin: "://invalid"},
		{NacosAPIOrigin: "ftp://nacos.internal/nacos"},
		{NacosAPIOrigin: "http://nacos.internal/nacos", NacosHTTPTLSCAFile: material.caFile},
		{NacosAPIOrigin: "https://nacos.internal/nacos", NacosHTTPTLSCAFile: material.caFile},
		{NacosAPIOrigin: "https://nacos.internal/nacos", NacosHTTPTLSCAFile: material.caFile, NacosHTTPTLSServerName: "nacos.internal", NacosHTTPTLSClientCertFile: material.clientCertFile},
	} {
		if _, err := newNacosHTTPClient(cfg); err == nil {
			t.Fatalf("invalid Nacos HTTP transport accepted: %#v", cfg)
		}
	}
}

func TestNacosGRPCTransportCredentialsHandshake(t *testing.T) {
	material := newTLSMaterial(t)

	for _, test := range []struct {
		name         string
		config       config.Config
		serverConfig *tls.Config
		wantError    bool
	}{
		{
			name:         "TLS",
			config:       config.Config{NacosGRPCTransportSecurity: config.NacosGRPCSecurityTLS, NacosGRPCTLSCAFile: material.caFile, NacosGRPCTLSServerName: "nacos.internal"},
			serverConfig: material.serverTLS(false),
		},
		{
			name:         "mTLS",
			config:       config.Config{NacosGRPCTransportSecurity: config.NacosGRPCSecurityMTLS, NacosGRPCTLSCAFile: material.caFile, NacosGRPCTLSServerName: "nacos.internal", NacosGRPCTLSClientCertFile: material.clientCertFile, NacosGRPCTLSClientKeyFile: material.clientKeyFile},
			serverConfig: material.serverTLS(true),
		},
		{
			name:         "wrong server name",
			config:       config.Config{NacosGRPCTransportSecurity: config.NacosGRPCSecurityTLS, NacosGRPCTLSCAFile: material.caFile, NacosGRPCTLSServerName: "other.internal"},
			serverConfig: material.serverTLS(false), wantError: true,
		},
		{
			name:         "private CA mismatch",
			config:       config.Config{NacosGRPCTransportSecurity: config.NacosGRPCSecurityTLS, NacosGRPCTLSCAFile: newTLSMaterial(t).caFile, NacosGRPCTLSServerName: "nacos.internal"},
			serverConfig: material.serverTLS(false), wantError: true,
		},
		{
			name:         "server requires client certificate",
			config:       config.Config{NacosGRPCTransportSecurity: config.NacosGRPCSecurityTLS, NacosGRPCTLSCAFile: material.caFile, NacosGRPCTLSServerName: "nacos.internal"},
			serverConfig: material.serverTLS(true), wantError: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			transportCredentials, err := newNacosGRPCTransportCredentials(test.config)
			if err != nil {
				t.Fatalf("construct credentials: %v", err)
			}
			err = handshakeNacosTLS(transportCredentials, test.serverConfig)
			if (err != nil) != test.wantError {
				t.Fatalf("handshake error=%v wantError=%v", err, test.wantError)
			}
		})
	}
}

func TestNacosGRPCTLSMaterialFailsClosedWithoutPathLeakage(t *testing.T) {
	material := newTLSMaterial(t)
	missing := filepath.Join(t.TempDir(), "private-ca.pem")
	missingClientCertificate := filepath.Join(t.TempDir(), "client.pem")
	missingClientKey := filepath.Join(t.TempDir(), "client-key.pem")
	directory := t.TempDir()
	empty := filepath.Join(t.TempDir(), "empty.pem")
	malformed := filepath.Join(t.TempDir(), "malformed.pem")
	malformedTail := filepath.Join(t.TempDir(), "malformed-tail.pem")
	oversized := filepath.Join(t.TempDir(), "oversized.pem")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(malformed, []byte("private-marker"), 0o600); err != nil {
		t.Fatal(err)
	}
	validCA, err := os.ReadFile(material.caFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(malformedTail, append(validCA, []byte("private-marker")...), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(oversized, make([]byte, nacosTLSMaterialLimit+1), 0o600); err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name      string
		cfg       config.Config
		forbidden []string
	}{
		{name: "missing", cfg: tlsCredentialConfig(missing), forbidden: []string{missing}},
		{name: "relative path", cfg: tlsCredentialConfig("private-ca.pem"), forbidden: []string{"private-ca.pem"}},
		{name: "directory", cfg: tlsCredentialConfig(directory), forbidden: []string{directory}},
		{name: "empty", cfg: tlsCredentialConfig(empty), forbidden: []string{empty}},
		{name: "oversized", cfg: tlsCredentialConfig(oversized), forbidden: []string{oversized}},
		{name: "malformed CA", cfg: tlsCredentialConfig(malformed), forbidden: []string{malformed, "private-marker"}},
		{name: "malformed CA tail", cfg: tlsCredentialConfig(malformedTail), forbidden: []string{malformedTail, "private-marker"}},
		{name: "mismatched key pair", cfg: config.Config{NacosGRPCTransportSecurity: config.NacosGRPCSecurityMTLS, NacosGRPCTLSCAFile: material.caFile, NacosGRPCTLSServerName: "nacos.internal", NacosGRPCTLSClientCertFile: material.clientCertFile, NacosGRPCTLSClientKeyFile: newTLSMaterial(t).clientKeyFile}, forbidden: []string{material.clientCertFile}},
		{name: "missing client certificate file", cfg: config.Config{NacosGRPCTransportSecurity: config.NacosGRPCSecurityMTLS, NacosGRPCTLSCAFile: material.caFile, NacosGRPCTLSServerName: "nacos.internal", NacosGRPCTLSClientCertFile: missingClientCertificate, NacosGRPCTLSClientKeyFile: material.clientKeyFile}, forbidden: []string{missingClientCertificate}},
		{name: "missing client key file", cfg: config.Config{NacosGRPCTransportSecurity: config.NacosGRPCSecurityMTLS, NacosGRPCTLSCAFile: material.caFile, NacosGRPCTLSServerName: "nacos.internal", NacosGRPCTLSClientCertFile: material.clientCertFile, NacosGRPCTLSClientKeyFile: missingClientKey}, forbidden: []string{missingClientKey}},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := newNacosGRPCTransportCredentials(test.cfg)
			if err == nil {
				t.Fatal("invalid TLS material accepted")
			}
			for _, value := range test.forbidden {
				if strings.Contains(err.Error(), value) {
					t.Fatalf("error leaks forbidden material: %v", err)
				}
			}
		})
	}

	for _, cfg := range []config.Config{
		{NacosGRPCTransportSecurity: config.NacosGRPCSecurityInsecure, NacosGRPCTLSCAFile: material.caFile},
		{NacosGRPCTransportSecurity: config.NacosGRPCSecurityTLS, NacosGRPCTLSCAFile: material.caFile, NacosGRPCTLSServerName: "nacos.internal", NacosGRPCTLSClientCertFile: material.clientCertFile},
		{NacosGRPCTransportSecurity: config.NacosGRPCSecurityMTLS, NacosGRPCTLSCAFile: material.caFile, NacosGRPCTLSServerName: "nacos.internal"},
		{NacosGRPCTransportSecurity: config.NacosGRPCSecurityTLS, NacosGRPCTLSCAFile: material.caFile},
		{NacosGRPCTransportSecurity: "ambient"},
	} {
		if _, err := newNacosGRPCTransportCredentials(cfg); err == nil {
			t.Fatalf("invalid transport config accepted: %#v", cfg)
		}
	}
}

func TestNacosTLSCertPoolRejectsEveryInvalidBlock(t *testing.T) {
	material := newTLSMaterial(t)
	for _, test := range []struct {
		name string
		pem  []byte
	}{
		{name: "empty", pem: nil},
		{name: "wrong block type", pem: pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: []byte("invalid")})},
		{name: "PEM headers", pem: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Headers: map[string]string{"Private": "marker"}, Bytes: material.serverCertificate.Certificate[0]})},
		{name: "non-CA certificate", pem: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: material.serverCertificate.Certificate[0]})},
		{name: "invalid certificate DER", pem: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: []byte("invalid")})},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := newNacosTLSCertPool(test.pem); err == nil || strings.Contains(err.Error(), "marker") {
				t.Fatalf("invalid CA block error=%v", err)
			}
		})
	}
}

func tlsCredentialConfig(caFile string) config.Config {
	return config.Config{NacosGRPCTransportSecurity: config.NacosGRPCSecurityTLS, NacosGRPCTLSCAFile: caFile, NacosGRPCTLSServerName: "nacos.internal"}
}

type tlsMaterial struct {
	caFile, clientCertFile, clientKeyFile string
	serverCertificate                     tls.Certificate
	caPool                                *x509.CertPool
}

func newTLSMaterial(t *testing.T) tlsMaterial {
	t.Helper()
	directory := t.TempDir()
	caPublic, caPrivate, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	caTemplate := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "NeKiro test CA"}, NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, caPublic, caPrivate)
	if err != nil {
		t.Fatal(err)
	}
	caCertificate, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatal(err)
	}
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	caFile := filepath.Join(directory, "ca.pem")
	if err := os.WriteFile(caFile, caPEM, 0o600); err != nil {
		t.Fatal(err)
	}

	issue := func(name string, serial int64, usage x509.ExtKeyUsage, dnsNames []string) (string, string, tls.Certificate) {
		public, private, keyErr := ed25519.GenerateKey(rand.Reader)
		if keyErr != nil {
			t.Fatal(keyErr)
		}
		template := &x509.Certificate{SerialNumber: big.NewInt(serial), Subject: pkix.Name{CommonName: name}, DNSNames: dnsNames, NotBefore: now.Add(-time.Hour), NotAfter: now.Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{usage}}
		der, createErr := x509.CreateCertificate(rand.Reader, template, caCertificate, public, caPrivate)
		if createErr != nil {
			t.Fatal(createErr)
		}
		certificatePEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
		keyBytes, marshalErr := x509.MarshalPKCS8PrivateKey(private)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})
		certificate, pairErr := tls.X509KeyPair(certificatePEM, keyPEM)
		if pairErr != nil {
			t.Fatal(pairErr)
		}
		certificateFile := filepath.Join(directory, name+".pem")
		keyFile := filepath.Join(directory, name+"-key.pem")
		if writeErr := os.WriteFile(certificateFile, certificatePEM, 0o600); writeErr != nil {
			t.Fatal(writeErr)
		}
		if writeErr := os.WriteFile(keyFile, keyPEM, 0o600); writeErr != nil {
			t.Fatal(writeErr)
		}
		return certificateFile, keyFile, certificate
	}
	_, _, serverCertificate := issue("server", 2, x509.ExtKeyUsageServerAuth, []string{"nacos.internal"})
	clientCertificateFile, clientKeyFile, _ := issue("client", 3, x509.ExtKeyUsageClientAuth, nil)
	pool := x509.NewCertPool()
	pool.AddCert(caCertificate)
	return tlsMaterial{caFile: caFile, clientCertFile: clientCertificateFile, clientKeyFile: clientKeyFile, serverCertificate: serverCertificate, caPool: pool}
}

func (material tlsMaterial) serverTLS(requireClient bool) *tls.Config {
	configuration := &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{material.serverCertificate}, NextProtos: []string{"h2"}}
	if requireClient {
		configuration.ClientAuth = tls.RequireAndVerifyClientCert
		configuration.ClientCAs = material.caPool
	}
	return configuration
}

func handshakeNacosTLS(transportCredentials credentials.TransportCredentials, serverConfig *tls.Config) error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	tlsListener := tls.NewListener(listener, serverConfig)
	server := grpc.NewServer()
	healthv1.RegisterHealthServer(server, health.NewServer())
	go func() { _ = server.Serve(tlsListener) }()
	defer server.Stop()

	client, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(transportCredentials))
	if err != nil {
		return err
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err = healthv1.NewHealthClient(client).Check(ctx, &healthv1.HealthCheckRequest{})
	return err
}
