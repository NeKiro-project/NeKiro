package gateway

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Nene7ko/NeKiro/apps/control-plane/internal/catalog"
	"github.com/Nene7ko/NeKiro/apps/control-plane/internal/config"
	"github.com/Nene7ko/NeKiro/contracts"
)

type fakeAuthenticator struct {
	caller catalog.AuthenticatedCaller
	err    error
}

func (authenticator fakeAuthenticator) Authenticate(*http.Request) (catalog.AuthenticatedCaller, error) {
	return authenticator.caller, authenticator.err
}

type fakeReadiness struct{ err error }

func (readiness fakeReadiness) Check(context.Context) error { return readiness.err }

type fakeCatalogService struct {
	registerCaller catalog.AuthenticatedCaller
	registerBody   []byte
	entry          contracts.CatalogEntry
	searchResult   catalog.SearchResult
	err            error
	registerCalls  int
}

func (service *fakeCatalogService) Register(_ context.Context, caller catalog.AuthenticatedCaller, body []byte) (contracts.CatalogEntry, error) {
	service.registerCaller = caller
	service.registerBody = append([]byte(nil), body...)
	service.registerCalls++
	return service.entry, service.err
}
func (service *fakeCatalogService) Get(context.Context, catalog.AuthenticatedCaller, string, string) (contracts.CatalogEntry, error) {
	return service.entry, service.err
}
func (service *fakeCatalogService) Publish(context.Context, catalog.AuthenticatedCaller, string, string) (contracts.CatalogEntry, error) {
	return service.entry, service.err
}
func (service *fakeCatalogService) Disable(context.Context, catalog.AuthenticatedCaller, string, string) (contracts.CatalogEntry, error) {
	return service.entry, service.err
}
func (service *fakeCatalogService) Search(context.Context, contracts.SearchAgentsQuery) (catalog.SearchResult, error) {
	return service.searchResult, service.err
}

func TestDevelopmentStaticAuthenticatorUsesBearerDigestOnly(t *testing.T) {
	token := "local-secret-token"
	digest := sha256.Sum256([]byte(token))
	authenticator, err := NewDevelopmentStaticAuthenticator([]config.StaticPrincipal{{
		ID: "owner-a", TokenSHA256: hex.EncodeToString(digest[:]),
	}})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/v2/agents", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("x-caller-id", "forged-owner")
	caller, err := authenticator.Authenticate(request)
	if err != nil {
		t.Fatalf("authenticate valid token: %v", err)
	}
	if caller.ID != "owner-a" || caller.AuthenticationKind != config.DevelopmentStaticAuthMode {
		t.Fatalf("caller = %#v", caller)
	}

	for _, authorization := range []string{"", "Bearer", "Bearer wrong", "Bearer " + token + " extra"} {
		request := httptest.NewRequest(http.MethodGet, "/v2/agents", nil)
		if authorization != "" {
			request.Header.Set("Authorization", authorization)
		}
		if _, err := authenticator.Authenticate(request); !errors.Is(err, ErrUnauthenticated) {
			t.Fatalf("authorization %q error = %v", authorization, err)
		}
	}
	lowercaseScheme := httptest.NewRequest(http.MethodGet, "/v2/agents", nil)
	lowercaseScheme.Header.Set("Authorization", "bearer "+token)
	if _, err := authenticator.Authenticate(lowercaseScheme); err != nil {
		t.Fatalf("case-insensitive Bearer scheme was rejected: %v", err)
	}
}

func TestHandlerAuthenticationErrorHasMatchingTrace(t *testing.T) {
	handler := newTestHandler(t, fakeAuthenticator{err: ErrUnauthenticated}, &fakeCatalogService{}, fakeReadiness{})
	request := httptest.NewRequest(http.MethodGet, "/v2/agents", nil)
	response := httptest.NewRecorder()
	handler.Routes().ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", response.Code)
	}
	var platformError contracts.PlatformError
	if err := json.Unmarshal(response.Body.Bytes(), &platformError); err != nil {
		t.Fatalf("decode Platform Error: %v", err)
	}
	if platformError.Code != contracts.ErrorCodeUnauthenticated || string(platformError.TraceID) != response.Header().Get(TraceHeader) {
		t.Fatalf("error/header correlation = %#v / %q", platformError, response.Header().Get(TraceHeader))
	}
}

func TestHandlerRegisterAndFixedDomainErrors(t *testing.T) {
	caller := catalog.AuthenticatedCaller{ID: "owner-a", AuthenticationKind: config.DevelopmentStaticAuthMode}
	service := &fakeCatalogService{entry: contracts.CatalogEntry{PublicationStatus: "draft", RegisteredAt: time.Now().UTC()}}
	handler := newTestHandler(t, fakeAuthenticator{caller: caller}, service, fakeReadiness{})
	request := httptest.NewRequest(http.MethodPost, "/v2/agents", bytes.NewBufferString(`{"card":{}}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.Routes().ServeHTTP(response, request)
	if response.Code != http.StatusCreated || response.Header().Get(TraceHeader) == "" {
		t.Fatalf("register response = %d, trace %q", response.Code, response.Header().Get(TraceHeader))
	}
	if service.registerCaller != caller || string(service.registerBody) != `{"card":{}}` {
		t.Fatalf("register adaptation = caller %#v body %q", service.registerCaller, service.registerBody)
	}

	tests := []struct {
		err    error
		status int
		code   contracts.PlatformErrorCode
	}{
		{catalog.ErrInvalid, 400, contracts.ErrorCodeValidationError},
		{catalog.ErrForbidden, 403, contracts.ErrorCodeForbidden},
		{catalog.ErrNotFound, 404, contracts.ErrorCodeNotFound},
		{catalog.ErrConflict, 409, contracts.ErrorCodeConflict},
		{catalog.ErrDependency, 503, contracts.ErrorCodeDependency},
	}
	for _, test := range tests {
		service.err = test.err
		request := httptest.NewRequest(http.MethodGet, "/v2/agents/agent-a/versions/1.0.0", nil)
		response := httptest.NewRecorder()
		handler.Routes().ServeHTTP(response, request)
		if response.Code != test.status {
			t.Errorf("%v status = %d, want %d", test.err, response.Code, test.status)
			continue
		}
		var body map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body["code"] != string(test.code) || body["traceId"] != response.Header().Get(TraceHeader) {
			t.Errorf("%v response = %#v, trace %q", test.err, body, response.Header().Get(TraceHeader))
		}
	}
}

func TestHandlerRejectsInvalidMediaAndSearchParameters(t *testing.T) {
	caller := catalog.AuthenticatedCaller{ID: "owner-a"}
	service := &fakeCatalogService{searchResult: catalog.SearchResult{Entries: []contracts.CatalogEntry{}}}
	handler := newTestHandler(t, fakeAuthenticator{caller: caller}, service, fakeReadiness{})

	request := httptest.NewRequest(http.MethodPost, "/v2/agents", bytes.NewBufferString(`{"card":{}}`))
	request.Header.Set("Content-Type", "text/plain")
	response := httptest.NewRecorder()
	handler.Routes().ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("invalid media status = %d", response.Code)
	}

	for _, rawQuery := range []string{"unknown=value", "limit=0", "limit=abc", "query=a&query=b", "query=%ZZ"} {
		request := httptest.NewRequest(http.MethodGet, "/v2/agents?"+rawQuery, nil)
		response := httptest.NewRecorder()
		handler.Routes().ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Errorf("query %q status = %d, want 400", rawQuery, response.Code)
		}
	}
}

func TestHandlerRejectsOversizedRegistrationBeforeCatalog(t *testing.T) {
	caller := catalog.AuthenticatedCaller{ID: "owner-a"}
	service := &fakeCatalogService{}
	handler := newTestHandler(t, fakeAuthenticator{caller: caller}, service, fakeReadiness{})
	body := io.LimitReader(repeatingReader{}, contracts.RegistrationMaximumBodyBytes+1)
	request := httptest.NewRequest(http.MethodPost, "/v2/agents", body)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.Routes().ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("oversized registration status = %d, want 400", response.Code)
	}
	if service.registerCalls != 0 {
		t.Fatalf("Catalog received %d oversized registrations", service.registerCalls)
	}
}

func TestReadinessFailureIsExplicit(t *testing.T) {
	handler := newTestHandler(t, fakeAuthenticator{}, &fakeCatalogService{}, fakeReadiness{err: errors.New("database unavailable")})
	response := httptest.NewRecorder()
	handler.Routes().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("readiness status = %d", response.Code)
	}
}

func TestTraceGeneratorFailsAtInitialization(t *testing.T) {
	if _, err := newTraceGenerator(errorReader{}); err == nil {
		t.Fatal("failed entropy source was accepted")
	}
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

type repeatingReader struct{}

func (repeatingReader) Read(buffer []byte) (int, error) {
	for index := range buffer {
		buffer[index] = 'x'
	}
	return len(buffer), nil
}

func newTestHandler(t *testing.T, authenticator Authenticator, service CatalogService, readiness ReadinessChecker) *Handler {
	t.Helper()
	traces, err := newTraceGenerator(bytes.NewReader(bytes.Repeat([]byte{1}, 16)))
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewHandler(authenticator, service, readiness, traces, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return handler
}
