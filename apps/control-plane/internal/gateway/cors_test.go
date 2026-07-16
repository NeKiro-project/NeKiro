package gateway

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSHandlerAnswersAllowedPreflightWithoutCallingNext(t *testing.T) {
	called := false
	handler := NewCORSHandler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		called = true
	}), []string{"http://127.0.0.1:3000"})
	request := httptest.NewRequest(http.MethodOptions, "/v3/agents", nil)
	request.Header.Set("Origin", "http://127.0.0.1:3000")
	request.Header.Set("Access-Control-Request-Method", "GET")
	request.Header.Set("Access-Control-Request-Headers", "authorization")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if called {
		t.Fatal("preflight reached downstream handler")
	}
	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", response.Code)
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "http://127.0.0.1:3000" {
		t.Fatalf("allow origin = %q", response.Header().Get("Access-Control-Allow-Origin"))
	}
	if response.Header().Get("Access-Control-Allow-Headers") != corsAllowHeaders {
		t.Fatalf("allow headers = %q", response.Header().Get("Access-Control-Allow-Headers"))
	}
	if response.Header().Get("Access-Control-Expose-Headers") != TraceHeader {
		t.Fatalf("expose headers = %q", response.Header().Get("Access-Control-Expose-Headers"))
	}
}

func TestCORSHandlerAddsHeadersToAllowedActualRequest(t *testing.T) {
	handler := NewCORSHandler(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusAccepted)
	}), []string{"http://127.0.0.1:3000"})
	request := httptest.NewRequest(http.MethodGet, "/v3/agents", nil)
	request.Header.Set("Origin", "http://127.0.0.1:3000")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", response.Code)
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "http://127.0.0.1:3000" {
		t.Fatalf("allow origin = %q", response.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORSHandlerDoesNotReflectDisallowedOrigin(t *testing.T) {
	handler := NewCORSHandler(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusOK)
	}), []string{"http://127.0.0.1:3000"})
	request := httptest.NewRequest(http.MethodGet, "/v3/agents", nil)
	request.Header.Set("Origin", "http://evil.example")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", response.Code)
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("disallowed origin was reflected as %q", response.Header().Get("Access-Control-Allow-Origin"))
	}
}
