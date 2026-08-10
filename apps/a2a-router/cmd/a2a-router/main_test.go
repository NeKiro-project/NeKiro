package main

import (
	"context"
	"crypto/ed25519"
	"net/http"
	"net/http/httptest"
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
	client := newNacosHTTPClient(750 * time.Millisecond)
	transport, ok := client.Transport.(*http.Transport)
	if !ok || transport.Proxy != nil || !transport.DisableKeepAlives || client.Timeout != 750*time.Millisecond {
		t.Fatalf("client=%#v transport=%#v", client, transport)
	}
	if err := client.CheckRedirect(httptest.NewRequest(http.MethodGet, "http://nacos.test/next", nil), nil); err == nil || err.Error() != "Nacos redirects are disabled" {
		t.Fatalf("redirect error=%v", err)
	}
}
