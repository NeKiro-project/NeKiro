package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Nene7ko/NeKiro/apps/a2a-router/internal/auth"
	"github.com/Nene7ko/NeKiro/apps/a2a-router/internal/config"
)

type failingDoer struct{}

func (failingDoer) Do(*http.Request) (*http.Response, error) {
	panic("readiness must not probe dependencies")
}

func TestNewHandlerAssemblesReadinessWithoutDependencyProbe(t *testing.T) {
	handler, err := newHandler(config.Config{
		ListenAddress:                  "127.0.0.1:9090",
		RouterPrincipals:               []auth.Principal{{ID: "router", TokenSHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}},
		ControlPlaneResolveURL:         "https://control.internal/internal/v2/resolve-agent",
		ControlPlaneServiceToken:       "control-token",
		InternalRequestLimitBytes:      1024,
		ControlPlaneResponseLimitBytes: 2048,
		ResolutionDeadline:             time.Second,
	}, failingDoer{}, &http.Client{})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d", response.Code)
	}
}
