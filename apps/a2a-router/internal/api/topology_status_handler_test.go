package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/NeKiro-project/NeKiro/apps/a2a-router/internal/auth"
	"github.com/NeKiro-project/NeKiro/contracts"
)

type topologyStatusReaderStub struct {
	status contracts.RouterTopologyStatusV1
}

func (reader topologyStatusReaderStub) TopologyStatus() contracts.RouterTopologyStatusV1 {
	return reader.status
}

func TestTopologyStatusHandlerServesAuthenticatedSafeContract(t *testing.T) {
	handler, err := NewTopologyStatusHandler(topologyStatusReaderStub{status: topologyStatusFixture()})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	if err := handler.RegisterRoutes(mux, authStub{caller: auth.Caller{ID: "stack-acceptance"}}); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, RouterTopologyStatusPath, nil))
	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/json" ||
		response.Header().Get("Cache-Control") != "no-store" || response.Header().Get(TraceHeader) == "" {
		t.Fatalf("response = %d headers=%v", response.Code, response.Header())
	}
	var status contracts.RouterTopologyStatusV1
	if err := json.Unmarshal(response.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if err := contracts.ValidateRouterTopologyStatusV1(status); err != nil || status.Observations[0].State != contracts.RouterTopologyStateEmpty {
		t.Fatalf("decoded status = %#v error=%v", status, err)
	}
}

func TestTopologyStatusHandlerMapsAuthenticationAndInvalidProjection(t *testing.T) {
	for name, test := range map[string]struct {
		authErr error
		status  contracts.RouterTopologyStatusV1
		want    int
		code    contracts.PlatformErrorCode
	}{
		"unauthenticated": {authErr: auth.ErrUnauthenticated, status: topologyStatusFixture(), want: http.StatusUnauthorized, code: contracts.ErrorCodeUnauthenticated},
		"forbidden":       {authErr: auth.ErrForbidden, status: topologyStatusFixture(), want: http.StatusForbidden, code: contracts.ErrorCodeForbidden},
		"invalid reader":  {status: contracts.RouterTopologyStatusV1{}, want: http.StatusServiceUnavailable, code: contracts.ErrorCodeDependency},
	} {
		t.Run(name, func(t *testing.T) {
			handler, err := NewTopologyStatusHandler(topologyStatusReaderStub{status: test.status})
			if err != nil {
				t.Fatal(err)
			}
			mux := http.NewServeMux()
			if err := handler.RegisterRoutes(mux, authStub{err: test.authErr}); err != nil {
				t.Fatal(err)
			}
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, RouterTopologyStatusPath, nil))
			var platformError contracts.PreCorrelationPlatformErrorV4
			if err := json.Unmarshal(response.Body.Bytes(), &platformError); err != nil {
				t.Fatal(err)
			}
			if response.Code != test.want || platformError.Code != test.code || platformError.TraceID == "" || response.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("response = %d %#v headers=%v", response.Code, platformError, response.Header())
			}
		})
	}
}

func TestTopologyStatusHandlerSupportsConcurrentReadOnlyRequests(t *testing.T) {
	handler, err := NewTopologyStatusHandler(topologyStatusReaderStub{status: topologyStatusFixture()})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	if err := handler.RegisterRoutes(mux, authStub{}); err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	failures := make(chan error, 32)
	for range 32 {
		group.Add(1)
		go func() {
			defer group.Done()
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, httptest.NewRequest(http.MethodGet, RouterTopologyStatusPath, nil))
			if response.Code != http.StatusOK {
				failures <- errors.New("unexpected topology status response")
			}
		}()
	}
	group.Wait()
	close(failures)
	for err := range failures {
		t.Fatal(err)
	}
}

func TestTopologyStatusHandlerRequiresDependencies(t *testing.T) {
	if _, err := NewTopologyStatusHandler(nil); err == nil {
		t.Fatal("nil topology reader accepted")
	}
	handler, _ := NewTopologyStatusHandler(topologyStatusReaderStub{status: topologyStatusFixture()})
	if err := handler.RegisterRoutes(nil, authStub{}); err == nil {
		t.Fatal("nil mux accepted")
	}
	if err := handler.RegisterRoutes(http.NewServeMux(), nil); err == nil {
		t.Fatal("nil authenticator accepted")
	}
}

func topologyStatusFixture() contracts.RouterTopologyStatusV1 {
	return contracts.RouterTopologyStatusV1{
		SchemaVersion: contracts.RouterTopologyStatusSchemaVersion,
		Provider:      "nacos",
		Observations: []contracts.RouterTopologyStatusObservationV1{{
			AgentID: "runtime-b", AgentCardVersion: "1.0.0", ReleaseID: "release-b",
			State: contracts.RouterTopologyStateEmpty, LocalRevision: 2,
			ObservedAt: time.Date(2026, 8, 10, 5, 0, 0, 0, time.UTC),
		}},
	}
}
