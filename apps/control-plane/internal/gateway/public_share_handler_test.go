package gateway

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Nene7ko/NeKiro/apps/control-plane/internal/catalog"
	"github.com/Nene7ko/NeKiro/contracts"
)

type publicShareResolverStub struct {
	view contracts.PublicAgentShare
	err  error
}

func (stub publicShareResolverStub) Resolve(context.Context, string) (contracts.PublicAgentShare, error) {
	return stub.view, stub.err
}

func TestPublicShareHandlerIsAnonymousAndTraceCorrelated(t *testing.T) {
	traces, err := NewTraceGenerator()
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewPublicShareHandler(publicShareResolverStub{view: contracts.PublicAgentShare{SchemaVersion: "1", PublicAgentID: "agt_0123456789abcdef0123456789abcdef", PublicURL: "https://agents.nekiro.test/a/agt_0123456789abcdef0123456789abcdef", RegisteredAt: time.Now().UTC(), Availability: contracts.PublicAgentAvailabilityNotInstallable, Releases: []contracts.PublicAgentRelease{}}}, traces, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	request := httptest.NewRequest(http.MethodGet, "/v4/public/agents/agt_0123456789abcdef0123456789abcdef", nil)
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get(TraceHeader) == "" || response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("response = %d headers=%v", response.Code, response.Header())
	}
	var view contracts.PublicAgentShare
	if err := json.Unmarshal(response.Body.Bytes(), &view); err != nil || view.Availability != contracts.PublicAgentAvailabilityNotInstallable {
		t.Fatalf("body = %s err=%v", response.Body, err)
	}
}

func TestPublicShareHandlerMapsExactFailuresWithoutAuth(t *testing.T) {
	traces, _ := NewTraceGenerator()
	for _, test := range []struct {
		name   string
		err    error
		status int
		code   contracts.PlatformErrorCode
	}{
		{"invalid", catalog.ErrInvalid, 400, contracts.ErrorCodeValidationError},
		{"unknown", catalog.ErrNotFound, 404, contracts.ErrorCodeNotFound},
		{"dependency", catalog.ErrDependency, 503, contracts.ErrorCodeDependency},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler, err := NewPublicShareHandler(publicShareResolverStub{err: test.err}, traces, slog.Default())
			if err != nil {
				t.Fatal(err)
			}
			mux := http.NewServeMux()
			handler.RegisterRoutes(mux)
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/v4/public/agents/agt_0123456789abcdef0123456789abcdef", nil)
			mux.ServeHTTP(response, request)
			var payload contracts.PlatformError
			if response.Code != test.status || json.Unmarshal(response.Body.Bytes(), &payload) != nil || payload.Code != test.code || string(payload.TraceID) != response.Header().Get(TraceHeader) {
				t.Fatalf("status=%d body=%s", response.Code, response.Body)
			}
		})
	}
}
