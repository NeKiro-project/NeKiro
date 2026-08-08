package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type readinessCheck func(context.Context) error

func (check readinessCheck) Check(ctx context.Context) error { return check(ctx) }

func TestReadinessIsLocalOnly(t *testing.T) {
	response := httptest.NewRecorder()
	NewReadinessHandler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("status=%d headers=%#v body=%s", response.Code, response.Header(), response.Body.String())
	}
}

func TestReadinessReportsOnlySafeUnavailableState(t *testing.T) {
	response := httptest.NewRecorder()
	NewReadinessHandler(readinessCheck(func(context.Context) error { return errors.New("secret source detail") })).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if response.Code != http.StatusServiceUnavailable || response.Body.String() != "{\"status\":\"not_ready\"}\n" {
		t.Fatalf("status=%d body=%q", response.Code, response.Body.String())
	}
}
