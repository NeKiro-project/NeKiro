package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReadinessIsLocalOnly(t *testing.T) {
	response := httptest.NewRecorder()
	NewReadinessHandler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("status=%d headers=%#v body=%s", response.Code, response.Header(), response.Body.String())
	}
}
