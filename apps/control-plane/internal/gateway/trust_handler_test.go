package gateway

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Nene7ko/NeKiro/apps/control-plane/internal/catalog"
	"github.com/Nene7ko/NeKiro/contracts"
)

func TestTrustHandlerCreatesBindingThroughAuthenticatedProvider(t *testing.T) {
	now := time.Date(2026, 7, 22, 0, 0, 0, 0, time.UTC)
	service := &fakeTrustCatalog{binding: catalog.EndpointBinding{BindingID: "binding-1", ProviderID: "provider-1", AgentID: "agent-1", AgentCardVersion: "1.0.0", Endpoint: "https://agent.example/a2a", VerificationMethod: catalog.VerificationMethodHTTPWellKnown, VerificationStatus: catalog.VerificationPending, CreatedAt: now, UpdatedAt: now}}
	handler := newTrustTestHandler(t, fakeAuthenticator{caller: catalog.AuthenticatedCaller{ID: "provider-1"}}, service)
	request := httptest.NewRequest(http.MethodPost, "/v4/providers/provider-1/agents/agent-1/endpoint-bindings", strings.NewReader(`{"endpoint":"https://agent.example/a2a","method":"http_well_known","version":"1.0.0"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if service.caller.ID != "provider-1" || service.providerID != "provider-1" || service.agentID != "agent-1" || service.version != "1.0.0" {
		t.Fatalf("service call=%#v", service)
	}
	var response contracts.EndpointBindingResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.BindingID != "binding-1" || response.VerificationStatus != "pending" || response.VerificationEvidenceDigest != nil {
		t.Fatalf("binding response=%#v", response)
	}
}

func TestTrustHandlerMapsChallengeFailureWithoutLeakingProof(t *testing.T) {
	service := &fakeTrustCatalog{completeErr: catalog.ErrWrongProof}
	handler := newTrustTestHandler(t, fakeAuthenticator{caller: catalog.AuthenticatedCaller{ID: "provider-1"}}, service)
	request := httptest.NewRequest(http.MethodPost, "/v4/providers/provider-1/endpoint-bindings/binding-1/challenges/challenge-1/complete", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var publicError contracts.TrustedPublicationError
	if err := json.Unmarshal(recorder.Body.Bytes(), &publicError); err != nil {
		t.Fatal(err)
	}
	if publicError.Code != contracts.TrustedErrorWrongProof || strings.Contains(recorder.Body.String(), catalog.ErrWrongProof.Error()) {
		t.Fatalf("public error leaked dependency details: %s", recorder.Body.String())
	}
}

func TestTrustHandlerMapsEndpointUnavailableToServiceUnavailable(t *testing.T) {
	service := &fakeTrustCatalog{completeErr: catalog.ErrEndpointUnavailable}
	handler := newTrustTestHandler(t, fakeAuthenticator{caller: catalog.AuthenticatedCaller{ID: "provider-1"}}, service)
	request := httptest.NewRequest(http.MethodPost, "/v4/providers/provider-1/endpoint-bindings/binding-1/challenges/challenge-1/complete", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var publicError contracts.TrustedPublicationError
	if err := json.Unmarshal(recorder.Body.Bytes(), &publicError); err != nil {
		t.Fatal(err)
	}
	if publicError.Code != contracts.TrustedErrorEndpointUnavailable {
		t.Fatalf("error=%#v", publicError)
	}
}

func newTrustTestHandler(t *testing.T, authenticator Authenticator, service TrustCatalogService) http.Handler {
	t.Helper()
	traces, err := NewTraceGenerator()
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewTrustHandler(authenticator, service, traces, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	return mux
}

type fakeTrustCatalog struct {
	binding     catalog.EndpointBinding
	completeErr error
	caller      catalog.AuthenticatedCaller
	providerID  string
	agentID     string
	version     string
}

func (service *fakeTrustCatalog) CreateBindingForCaller(_ context.Context, caller catalog.AuthenticatedCaller, providerID, agentID, version, _, _ string) (catalog.EndpointBinding, error) {
	service.caller, service.providerID, service.agentID, service.version = caller, providerID, agentID, version
	return service.binding, nil
}

func (service *fakeTrustCatalog) CreateChallengeForCaller(context.Context, catalog.AuthenticatedCaller, string, string) (contracts.VerificationChallengeResponse, error) {
	return contracts.VerificationChallengeResponse{}, nil
}

func (service *fakeTrustCatalog) CompleteChallengeForCaller(context.Context, catalog.AuthenticatedCaller, string, string, string) (catalog.EndpointBinding, error) {
	return service.binding, service.completeErr
}

func (service *fakeTrustCatalog) GetBindingForCaller(context.Context, catalog.AuthenticatedCaller, string, string) (catalog.EndpointBinding, error) {
	return service.binding, nil
}
