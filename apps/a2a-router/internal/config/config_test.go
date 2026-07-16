package config

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
)

func TestLoadRequiresStrictRouterConfig(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		config, err := loadWithEnv(t, validEnv())
		if err != nil {
			t.Fatalf("valid config rejected: %v", err)
		}
		if config.ListenAddress != "127.0.0.1:9090" || config.ControlPlaneResolveURL != "https://control.internal/internal/v2/resolve-agent" || config.InternalRequestLimitBytes != 1024 || config.ControlPlaneResponseLimitBytes != 2048 || config.ResolutionDeadline.Milliseconds() != 5000 {
			t.Fatalf("config=%#v", config)
		}
	})

	tests := []struct {
		name  string
		key   string
		value *string
	}{
		{name: "missing listen", key: "NEKIRO_ROUTER_LISTEN_ADDRESS", value: nil},
		{name: "blank token", key: "NEKIRO_CONTROL_PLANE_SERVICE_TOKEN", value: ptr(" ")},
		{name: "whitespace token", key: "NEKIRO_CONTROL_PLANE_SERVICE_TOKEN", value: ptr(" token")},
		{name: "control plane userinfo", key: "NEKIRO_CONTROL_PLANE_RESOLVE_URL", value: ptr("https://user@control.internal/internal/v2/resolve-agent")},
		{name: "control plane wrong path", key: "NEKIRO_CONTROL_PLANE_RESOLVE_URL", value: ptr("https://control.internal/internal/v2/other")},
		{name: "control plane query", key: "NEKIRO_CONTROL_PLANE_RESOLVE_URL", value: ptr("https://control.internal/internal/v2/resolve-agent?x=1")},
		{name: "negative limit", key: "NEKIRO_ROUTER_INTERNAL_REQUEST_LIMIT_BYTES", value: ptr("-1")},
		{name: "zero limit", key: "NEKIRO_ROUTER_INTERNAL_REQUEST_LIMIT_BYTES", value: ptr("0")},
		{name: "fractional limit", key: "NEKIRO_ROUTER_INTERNAL_REQUEST_LIMIT_BYTES", value: ptr("1.5")},
		{name: "exponent limit", key: "NEKIRO_ROUTER_INTERNAL_REQUEST_LIMIT_BYTES", value: ptr("1e3")},
		{name: "overflow limit", key: "NEKIRO_ROUTER_INTERNAL_REQUEST_LIMIT_BYTES", value: ptr("2147483648")},
		{name: "zero deadline", key: "NEKIRO_ROUTER_RESOLUTION_DEADLINE_MS", value: ptr("0")},
		{name: "overflow deadline", key: "NEKIRO_ROUTER_RESOLUTION_DEADLINE_MS", value: ptr("600001")},
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

func validEnv() map[string]string {
	return map[string]string{
		"NEKIRO_ROUTER_LISTEN_ADDRESS":                     "127.0.0.1:9090",
		"NEKIRO_ROUTER_SERVICE_PRINCIPALS_JSON":            fmt.Sprintf(`[{"id":"router","tokenSha256":"%s"}]`, digest("router-token")),
		"NEKIRO_CONTROL_PLANE_RESOLVE_URL":                 "https://control.internal/internal/v2/resolve-agent",
		"NEKIRO_CONTROL_PLANE_SERVICE_TOKEN":               "control-token",
		"NEKIRO_ROUTER_INTERNAL_REQUEST_LIMIT_BYTES":       "1024",
		"NEKIRO_ROUTER_CONTROL_PLANE_RESPONSE_LIMIT_BYTES": "2048",
		"NEKIRO_ROUTER_RESOLUTION_DEADLINE_MS":             "5000",
	}
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
