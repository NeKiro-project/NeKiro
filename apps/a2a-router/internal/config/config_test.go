package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"
)

func TestLoadRequiresStrictRouterConfig(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		config, err := loadWithEnv(t, validEnv())
		if err != nil {
			t.Fatalf("valid config rejected: %v", err)
		}
		if config.ListenAddress != "127.0.0.1:9090" || config.DatabaseURL != "postgresql://router:secret@postgres:5432/nekiro?sslmode=disable" || config.ControlPlaneResolveURL != "https://control.internal/internal/v2/resolve-agent" || config.ControlPlaneVersionURL != "https://control.internal/internal/v3/resolve-installed-version" || len(config.AgentPrincipals) != 1 || config.AgentPrincipals[0].WorkspaceID != "workspace-a" || config.InternalRequestLimitBytes != 1024 || config.AgentRequestLimitBytes != 1024 || config.ControlPlaneResponseLimitBytes != 2048 || config.AgentResponseLimitBytes != 4096 || config.A2AEventLimitBytes != 4096 || config.SSEEventLimitBytes != 4096 || config.ResolutionDeadline.Milliseconds() != 5000 || config.AgentDeadline.Milliseconds() != 5000 {
			t.Fatalf("config=%#v", config)
		}
	})

	tests := []struct {
		name  string
		key   string
		value *string
	}{
		{name: "missing listen", key: "NEKIRO_ROUTER_LISTEN_ADDRESS", value: nil},
		{name: "missing database", key: "NEKIRO_DATABASE_URL", value: nil},
		{name: "database wrong scheme", key: "NEKIRO_DATABASE_URL", value: ptr("https://postgres/nekiro")},
		{name: "database missing port", key: "NEKIRO_DATABASE_URL", value: ptr("postgresql://router:secret@postgres/nekiro?sslmode=disable")},
		{name: "database missing password", key: "NEKIRO_DATABASE_URL", value: ptr("postgresql://router@postgres:5432/nekiro?sslmode=disable")},
		{name: "database missing sslmode", key: "NEKIRO_DATABASE_URL", value: ptr("postgresql://router:secret@postgres:5432/nekiro")},
		{name: "blank token", key: "NEKIRO_CONTROL_PLANE_SERVICE_TOKEN", value: ptr(" ")},
		{name: "missing Agent principals", key: "NEKIRO_ROUTER_AGENT_PRINCIPALS_JSON", value: nil},
		{name: "duplicate Agent principal field", key: "NEKIRO_ROUTER_AGENT_PRINCIPALS_JSON", value: ptr(`[{"workspaceId":"workspace-a","workspaceId":"workspace-b","agentId":"runtime-a","tokenSha256":"` + digest("agent-token") + `"}]`)},
		{name: "invalid Agent principal digest", key: "NEKIRO_ROUTER_AGENT_PRINCIPALS_JSON", value: ptr(`[{"workspaceId":"workspace-a","agentId":"runtime-a","tokenSha256":"bad"}]`)},
		{name: "whitespace token", key: "NEKIRO_CONTROL_PLANE_SERVICE_TOKEN", value: ptr(" token")},
		{name: "control plane userinfo", key: "NEKIRO_CONTROL_PLANE_RESOLVE_URL", value: ptr("https://user@control.internal/internal/v2/resolve-agent")},
		{name: "control plane wrong path", key: "NEKIRO_CONTROL_PLANE_RESOLVE_URL", value: ptr("https://control.internal/internal/v2/other")},
		{name: "control plane query", key: "NEKIRO_CONTROL_PLANE_RESOLVE_URL", value: ptr("https://control.internal/internal/v2/resolve-agent?x=1")},
		{name: "control plane empty query", key: "NEKIRO_CONTROL_PLANE_RESOLVE_URL", value: ptr("https://control.internal/internal/v2/resolve-agent?")},
		{name: "control plane empty fragment", key: "NEKIRO_CONTROL_PLANE_RESOLVE_URL", value: ptr("https://control.internal/internal/v2/resolve-agent#")},
		{name: "control plane port out of range", key: "NEKIRO_CONTROL_PLANE_RESOLVE_URL", value: ptr("https://control.internal:99999/internal/v2/resolve-agent")},
		{name: "version URL wrong path", key: "NEKIRO_CONTROL_PLANE_VERSION_URL", value: ptr("https://control.internal/internal/v2/resolve-agent")},
		{name: "version URL port out of range", key: "NEKIRO_CONTROL_PLANE_VERSION_URL", value: ptr("https://control.internal:99999/internal/v3/resolve-installed-version")},
		{name: "negative limit", key: "NEKIRO_ROUTER_INTERNAL_REQUEST_LIMIT_BYTES", value: ptr("-1")},
		{name: "zero limit", key: "NEKIRO_ROUTER_INTERNAL_REQUEST_LIMIT_BYTES", value: ptr("0")},
		{name: "fractional limit", key: "NEKIRO_ROUTER_INTERNAL_REQUEST_LIMIT_BYTES", value: ptr("1.5")},
		{name: "exponent limit", key: "NEKIRO_ROUTER_INTERNAL_REQUEST_LIMIT_BYTES", value: ptr("1e3")},
		{name: "overflow limit", key: "NEKIRO_ROUTER_INTERNAL_REQUEST_LIMIT_BYTES", value: ptr("2147483648")},
		{name: "missing Agent response limit", key: "NEKIRO_ROUTER_AGENT_RESPONSE_LIMIT_BYTES", value: nil},
		{name: "missing Agent request limit", key: "NEKIRO_ROUTER_AGENT_REQUEST_LIMIT_BYTES", value: nil},
		{name: "missing A2A event limit", key: "NEKIRO_ROUTER_A2A_EVENT_LIMIT_BYTES", value: nil},
		{name: "missing SSE event limit", key: "NEKIRO_ROUTER_SSE_EVENT_LIMIT_BYTES", value: nil},
		{name: "zero deadline", key: "NEKIRO_ROUTER_RESOLUTION_DEADLINE_MS", value: ptr("0")},
		{name: "overflow deadline", key: "NEKIRO_ROUTER_RESOLUTION_DEADLINE_MS", value: ptr("600001")},
		{name: "zero Agent deadline", key: "NEKIRO_ROUTER_AGENT_DEADLINE_MS", value: ptr("0")},
		{name: "duplicate principal field", key: "NEKIRO_ROUTER_SERVICE_PRINCIPALS_JSON", value: ptr(`[{"id":"router","id":"other","tokenSha256":"` + digest("router-token") + `"}]`)},
		{name: "duplicate principal digest", key: "NEKIRO_ROUTER_SERVICE_PRINCIPALS_JSON", value: ptr(`[{"id":"router","tokenSha256":"` + digest("router-token") + `"},{"id":"other","tokenSha256":"` + digest("router-token") + `"}]`)},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			env := validEnv()
			if test.value == nil {
				delete(env, test.key)
			} else {
				env[test.key] = *test.value
			}
			if _, err := loadWithEnv(t, env); err == nil {
				t.Fatal("invalid config accepted")
			}
		})
	}
}

func TestLoadDatabaseURLIsMigrationScoped(t *testing.T) {
	t.Setenv("NEKIRO_DATABASE_URL", "postgresql://router:secret@postgres:5432/nekiro?sslmode=disable")
	value, err := LoadDatabaseURL()
	if err != nil {
		t.Fatalf("valid database URL rejected: %v", err)
	}
	if value != "postgresql://router:secret@postgres:5432/nekiro?sslmode=disable" {
		t.Fatalf("database URL=%q", value)
	}

	t.Setenv("NEKIRO_DATABASE_URL", " ")
	if _, err := LoadDatabaseURL(); err == nil {
		t.Fatal("blank database URL accepted")
	}
}

func validEnv() map[string]string {
	return map[string]string{
		"NEKIRO_ROUTER_LISTEN_ADDRESS":                         "127.0.0.1:9090",
		"NEKIRO_ROUTER_SERVICE_PRINCIPALS_JSON":                fmt.Sprintf(`[{"id":"router","tokenSha256":"%s"}]`, digest("router-token")),
		"NEKIRO_ROUTER_AGENT_PRINCIPALS_JSON":                  fmt.Sprintf(`[{"workspaceId":"workspace-a","agentId":"runtime-a","tokenSha256":"%s"}]`, digest("agent-token")),
		"NEKIRO_DATABASE_URL":                                  "postgresql://router:secret@postgres:5432/nekiro?sslmode=disable",
		"NEKIRO_CONTROL_PLANE_RESOLVE_URL":                     "https://control.internal/internal/v2/resolve-agent",
		"NEKIRO_CONTROL_PLANE_VERSION_URL":                     "https://control.internal/internal/v3/resolve-installed-version",
		"NEKIRO_CONTROL_PLANE_SERVICE_TOKEN":                   "control-token",
		"NEKIRO_ROUTER_INTERNAL_REQUEST_LIMIT_BYTES":           "1024",
		"NEKIRO_ROUTER_AGENT_REQUEST_LIMIT_BYTES":              "1024",
		"NEKIRO_ROUTER_CONTROL_PLANE_RESPONSE_LIMIT_BYTES":     "2048",
		"NEKIRO_ROUTER_AGENT_RESPONSE_LIMIT_BYTES":             "4096",
		"NEKIRO_ROUTER_A2A_EVENT_LIMIT_BYTES":                  "4096",
		"NEKIRO_ROUTER_SSE_EVENT_LIMIT_BYTES":                  "4096",
		"NEKIRO_ROUTER_RESOLUTION_DEADLINE_MS":                 "5000",
		"NEKIRO_ROUTER_AGENT_DEADLINE_MS":                      "5000",
		"NEKIRO_ROUTER_AGENT_CREDENTIAL_ISSUER":                "https://a2a-router.nekiro.test",
		"NEKIRO_ROUTER_AGENT_CREDENTIAL_KEY_ID":                "router-key-1",
		"NEKIRO_ROUTER_AGENT_CREDENTIAL_PRIVATE_KEY_BASE64URL": "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8DoQe_884Qvh1w3RjnS8CZZ-TWMJulDV8d3IZkElUxuA",
		"NEKIRO_ROUTER_AGENT_CREDENTIAL_TTL_SECONDS":           "30",
		"NEKIRO_ROUTER_INSTANCE_ROUTING_MODE":                  "direct",
	}
}

func TestLoadFromUsesOnlyInjectedSource(t *testing.T) {
	env := validEnv()
	config, err := LoadFrom(func(name string) (string, bool) {
		value, ok := env[name]
		return value, ok
	})
	if err != nil {
		t.Fatalf("LoadFrom rejected injected source: %v", err)
	}
	if config.InstanceRoutingMode != InstanceRoutingDirect {
		t.Fatalf("routing mode = %q", config.InstanceRoutingMode)
	}
	if _, err := LoadFrom(nil); err == nil {
		t.Fatal("LoadFrom(nil) succeeded")
	}
}

func TestLoadFromRequiresFileRoutingInputs(t *testing.T) {
	env := validEnv()
	env["NEKIRO_ROUTER_INSTANCE_ROUTING_MODE"] = InstanceRoutingConfigCenterFile
	env["NEKIRO_ROUTER_CONFIG_CENTER_FILE_ROOT"] = `C:\nekiro\config`
	env["NEKIRO_ROUTER_CONFIG_CENTER_MAX_PAYLOAD_BYTES"] = "65536"
	env["NEKIRO_ROUTER_INSTANCE_DIRECTORY_KEY"] = "router/instance-directory"
	env["NEKIRO_ROUTER_INSTANCE_PORT_NAME"] = "a2a"
	config, err := LoadFrom(func(name string) (string, bool) {
		value, ok := env[name]
		return value, ok
	})
	if err != nil {
		t.Fatalf("file routing config rejected: %v", err)
	}
	if config.InstanceDirectoryKey.String() != "router/instance-directory" || config.InstancePortName != "a2a" {
		t.Fatalf("file routing config = %#v", config)
	}
	delete(env, "NEKIRO_ROUTER_INSTANCE_DIRECTORY_KEY")
	if _, err := LoadFrom(func(name string) (string, bool) { value, ok := env[name]; return value, ok }); err == nil {
		t.Fatal("missing directory key accepted")
	}
	env["NEKIRO_ROUTER_INSTANCE_DIRECTORY_KEY"] = "router/instance-directory"
	env["NEKIRO_ROUTER_INSTANCE_PORT_NAME"] = "a2a port"
	if _, err := LoadFrom(func(name string) (string, bool) { value, ok := env[name]; return value, ok }); err == nil {
		t.Fatal("invalid directory port name accepted")
	}
}

func TestLoadFromRequiresExplicitNacosRoutingInputs(t *testing.T) {
	env := validEnv()
	env["NEKIRO_ROUTER_INSTANCE_ROUTING_MODE"] = InstanceRoutingNacos
	env["NEKIRO_ROUTER_NACOS_API_ORIGIN"] = "http://nacos:8848/nacos"
	env["NEKIRO_ROUTER_NACOS_NAMESPACE_ID"] = "public"
	env["NEKIRO_ROUTER_NACOS_CONFIG_GROUP"] = "NEKIRO"
	env["NEKIRO_ROUTER_NACOS_AUTH_MODE"] = NacosAuthNone
	env["NEKIRO_ROUTER_NACOS_RESPONSE_LIMIT_BYTES"] = "65536"
	env["NEKIRO_ROUTER_NACOS_REQUEST_TIMEOUT_MS"] = "3000"
	env["NEKIRO_ROUTER_INSTANCE_DIRECTORY_KEY"] = "router.nacos-bindings"
	env["NEKIRO_ROUTER_INSTANCE_PORT_NAME"] = "a2a"
	loaded, err := LoadFrom(func(name string) (string, bool) { value, ok := env[name]; return value, ok })
	if err != nil {
		t.Fatalf("Nacos routing config rejected: %v", err)
	}
	if loaded.NacosAPIOrigin != env["NEKIRO_ROUTER_NACOS_API_ORIGIN"] || loaded.NacosRequestTimeout != 3*time.Second {
		t.Fatalf("Nacos config=%#v", loaded)
	}
	observed := validNacosEnv()
	observed["NEKIRO_ROUTER_NACOS_OBSERVE_ENABLED"] = "true"
	observed["NEKIRO_ROUTER_NACOS_GRPC_TARGET"] = "nacos:9848"
	observed["NEKIRO_ROUTER_NACOS_GRPC_CLIENT_IP"] = "172.30.88.12"
	observed["NEKIRO_ROUTER_NACOS_GRPC_REQUEST_TIMEOUT_MS"] = "5000"
	observed["NEKIRO_ROUTER_NACOS_PENDING_CHANGES"] = "8"
	observed["NEKIRO_ROUTER_NACOS_MAX_OBSERVATIONS"] = "1024"
	observed["NEKIRO_ROUTER_NACOS_GRPC_TRANSPORT_SECURITY"] = "insecure"
	loaded, err = LoadFrom(func(name string) (string, bool) { value, ok := observed[name]; return value, ok })
	if err != nil || !loaded.NacosObserveEnabled || loaded.NacosGRPCTarget != "nacos:9848" || loaded.NacosPendingChanges != 8 || loaded.NacosMaxObservations != 1024 {
		t.Fatalf("Nacos observation config=%#v error=%v", loaded, err)
	}
	delete(observed, "NEKIRO_ROUTER_NACOS_MAX_OBSERVATIONS")
	if _, err := LoadFrom(func(name string) (string, bool) { value, ok := observed[name]; return value, ok }); err == nil {
		t.Fatal("Nacos observation accepted a missing observation limit")
	}
	disabledWithGRPC := validNacosEnv()
	disabledWithGRPC["NEKIRO_ROUTER_NACOS_GRPC_TARGET"] = "nacos:9848"
	if _, err := LoadFrom(func(name string) (string, bool) { value, ok := disabledWithGRPC[name]; return value, ok }); err == nil {
		t.Fatal("Nacos snapshot-only mode accepted gRPC configuration")
	}
	for _, name := range []string{
		"NEKIRO_ROUTER_NACOS_GRPC_CLIENT_IP",
		"NEKIRO_ROUTER_NACOS_GRPC_REQUEST_TIMEOUT_MS",
		"NEKIRO_ROUTER_NACOS_PENDING_CHANGES",
		"NEKIRO_ROUTER_NACOS_GRPC_TRANSPORT_SECURITY",
	} {
		t.Run("snapshot only "+name, func(t *testing.T) {
			candidate := validNacosEnv()
			candidate[name] = "configured"
			if _, err := LoadFrom(func(key string) (string, bool) { value, ok := candidate[key]; return value, ok }); err == nil {
				t.Fatalf("snapshot-only mode accepted %s", name)
			}
		})
	}
	for _, test := range []struct {
		name  string
		key   string
		value *string
	}{
		{name: "observe value", key: "NEKIRO_ROUTER_NACOS_OBSERVE_ENABLED", value: ptr("yes")},
		{name: "missing target", key: "NEKIRO_ROUTER_NACOS_GRPC_TARGET"},
		{name: "invalid target", key: "NEKIRO_ROUTER_NACOS_GRPC_TARGET", value: ptr("https://nacos:9848")},
		{name: "missing client IP", key: "NEKIRO_ROUTER_NACOS_GRPC_CLIENT_IP"},
		{name: "invalid client IP", key: "NEKIRO_ROUTER_NACOS_GRPC_CLIENT_IP", value: ptr("localhost")},
		{name: "zero gRPC timeout", key: "NEKIRO_ROUTER_NACOS_GRPC_REQUEST_TIMEOUT_MS", value: ptr("0")},
		{name: "zero pending changes", key: "NEKIRO_ROUTER_NACOS_PENDING_CHANGES", value: ptr("0")},
		{name: "unsupported security", key: "NEKIRO_ROUTER_NACOS_GRPC_TRANSPORT_SECURITY", value: ptr("tls")},
	} {
		t.Run(test.name, func(t *testing.T) {
			candidate := validNacosEnv()
			candidate["NEKIRO_ROUTER_NACOS_OBSERVE_ENABLED"] = "true"
			candidate["NEKIRO_ROUTER_NACOS_GRPC_TARGET"] = "nacos:9848"
			candidate["NEKIRO_ROUTER_NACOS_GRPC_CLIENT_IP"] = "172.30.88.12"
			candidate["NEKIRO_ROUTER_NACOS_GRPC_REQUEST_TIMEOUT_MS"] = "5000"
			candidate["NEKIRO_ROUTER_NACOS_PENDING_CHANGES"] = "8"
			candidate["NEKIRO_ROUTER_NACOS_GRPC_TRANSPORT_SECURITY"] = "insecure"
			if test.value == nil {
				delete(candidate, test.key)
			} else {
				candidate[test.key] = *test.value
			}
			if _, err := LoadFrom(func(key string) (string, bool) { value, ok := candidate[key]; return value, ok }); err == nil {
				t.Fatalf("invalid observation config accepted")
			}
		})
	}
	env["NEKIRO_ROUTER_NACOS_ACCESS_TOKEN"] = "unexpected"
	if _, err := LoadFrom(func(name string) (string, bool) { value, ok := env[name]; return value, ok }); err == nil {
		t.Fatal("Nacos none auth accepted an access token")
	}
	delete(env, "NEKIRO_ROUTER_NACOS_ACCESS_TOKEN")
	env["NEKIRO_ROUTER_NACOS_AUTH_MODE"] = NacosAuthAccessToken
	if _, err := LoadFrom(func(name string) (string, bool) { value, ok := env[name]; return value, ok }); err == nil {
		t.Fatal("Nacos access-token auth accepted a missing token")
	}

	for _, test := range []struct {
		name  string
		key   string
		value string
	}{
		{name: "origin without nacos path", key: "NEKIRO_ROUTER_NACOS_API_ORIGIN", value: "http://nacos:8848"},
		{name: "origin with query", key: "NEKIRO_ROUTER_NACOS_API_ORIGIN", value: "http://nacos:8848/nacos?server=other"},
		{name: "empty namespace", key: "NEKIRO_ROUTER_NACOS_NAMESPACE_ID", value: ""},
		{name: "empty group", key: "NEKIRO_ROUTER_NACOS_CONFIG_GROUP", value: ""},
		{name: "zero response limit", key: "NEKIRO_ROUTER_NACOS_RESPONSE_LIMIT_BYTES", value: "0"},
		{name: "zero timeout", key: "NEKIRO_ROUTER_NACOS_REQUEST_TIMEOUT_MS", value: "0"},
		{name: "unsafe directory key", key: "NEKIRO_ROUTER_INSTANCE_DIRECTORY_KEY", value: "../bindings"},
		{name: "unsafe port name", key: "NEKIRO_ROUTER_INSTANCE_PORT_NAME", value: "a2a port"},
	} {
		t.Run(test.name, func(t *testing.T) {
			invalid := validNacosEnv()
			invalid[test.key] = test.value
			if _, err := LoadFrom(func(name string) (string, bool) { value, ok := invalid[name]; return value, ok }); err == nil {
				t.Fatalf("invalid %s=%q was accepted", test.key, test.value)
			}
		})
	}
}

func validNacosEnv() map[string]string {
	env := validEnv()
	env["NEKIRO_ROUTER_INSTANCE_ROUTING_MODE"] = InstanceRoutingNacos
	env["NEKIRO_ROUTER_NACOS_API_ORIGIN"] = "http://nacos:8848/nacos"
	env["NEKIRO_ROUTER_NACOS_NAMESPACE_ID"] = "public"
	env["NEKIRO_ROUTER_NACOS_CONFIG_GROUP"] = "NEKIRO"
	env["NEKIRO_ROUTER_NACOS_AUTH_MODE"] = NacosAuthNone
	env["NEKIRO_ROUTER_NACOS_RESPONSE_LIMIT_BYTES"] = "65536"
	env["NEKIRO_ROUTER_NACOS_REQUEST_TIMEOUT_MS"] = "3000"
	env["NEKIRO_ROUTER_INSTANCE_DIRECTORY_KEY"] = "router.nacos-bindings"
	env["NEKIRO_ROUTER_INSTANCE_PORT_NAME"] = "a2a"
	return env
}

func loadWithEnv(t *testing.T, env map[string]string) (Config, error) {
	t.Helper()
	for key := range validEnv() {
		t.Setenv(key, "")
	}
	for key, value := range env {
		t.Setenv(key, value)
	}
	return Load()
}

func digest(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func ptr(value string) *string { return &value }
