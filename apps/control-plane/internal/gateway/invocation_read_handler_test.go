package gateway

import (
	"bytes"
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
	"github.com/Nene7ko/NeKiro/apps/control-plane/internal/invocation"
	"github.com/Nene7ko/NeKiro/apps/control-plane/internal/workspace"
	"github.com/Nene7ko/NeKiro/contracts"
)

type metadataReadHandlerStub struct {
	invocationResponse *invocation.RouterResponse
	traceResponse      *invocation.RouterResponse
	err                error
	invocationCalls    int
	traceCalls         int
	caller             workspace.AuthenticatedCaller
	workspaceID        string
	resourceID         string
	traceID            contracts.TraceID
}

func (stub *metadataReadHandlerStub) GetInvocation(_ context.Context, caller workspace.AuthenticatedCaller, workspaceID, invocationID string) (*invocation.RouterResponse, error) {
	stub.invocationCalls++
	stub.caller, stub.workspaceID, stub.resourceID = caller, workspaceID, invocationID
	return stub.invocationResponse, stub.err
}

func (stub *metadataReadHandlerStub) GetTrace(_ context.Context, caller workspace.AuthenticatedCaller, workspaceID string, traceID contracts.TraceID) (*invocation.RouterResponse, error) {
	stub.traceCalls++
	stub.caller, stub.workspaceID, stub.traceID = caller, workspaceID, traceID
	return stub.traceResponse, stub.err
}

func TestInvocationReadHandlerProxiesAuthorizedMetadataAndCorrelation(t *testing.T) {
	reader := &metadataReadHandlerStub{
		invocationResponse: metadataHTTPResponse(http.StatusOK, validInvocationMetadataJSON),
		traceResponse:      metadataHTTPResponse(http.StatusOK, validTraceMetadataJSON),
	}
	handler := newInvocationReadTestHandler(t, invocationAuthenticatorStub{caller: catalog.AuthenticatedCaller{ID: "owner-a", AuthenticationKind: "development-static"}}, reader)
	invocationResponse := serveInvocationReadTestRequest(handler, "/v4/workspaces/workspace-a/invocations/inv-a")
	if invocationResponse.Code != http.StatusOK || invocationResponse.Body.String() != validInvocationMetadataJSON || invocationResponse.Header().Get(TraceHeader) == "" {
		t.Fatalf("Invocation response = %d %q trace=%q", invocationResponse.Code, invocationResponse.Body.String(), invocationResponse.Header().Get(TraceHeader))
	}
	traceRequest := httptest.NewRequest(http.MethodGet, "/v4/workspaces/workspace-a/traces/trace-a", nil)
	traceResponse := httptest.NewRecorder()
	handler.ServeHTTP(traceResponse, traceRequest)
	if traceResponse.Code != http.StatusOK || traceResponse.Body.String() != validTraceMetadataJSON || reader.traceCalls != 1 {
		t.Fatalf("Trace response = %d %q calls=%d", traceResponse.Code, traceResponse.Body.String(), reader.traceCalls)
	}
	if reader.invocationCalls != 1 || reader.caller.ID != "owner-a" || reader.workspaceID != "workspace-a" || reader.traceID != "trace-a" {
		t.Fatalf("reader arguments = %#v", reader)
	}
}

func TestInvocationReadHandlerRejectsBeforeRouterAndPreservesReadFailures(t *testing.T) {
	tests := []struct {
		name       string
		authErr    error
		readerErr  error
		status     int
		code       contracts.PlatformErrorCode
		path       string
		response   *invocation.RouterResponse
		wantInvoke int
	}{
		{name: "unauthenticated first", authErr: ErrUnauthenticated, status: http.StatusUnauthorized, code: contracts.ErrorCodeUnauthenticated, path: "/v4/workspaces/workspace-a/invocations/inv-a"},
		{name: "invalid path", status: http.StatusBadRequest, code: contracts.ErrorCodeValidationError, path: "/v4/workspaces/bad%20workspace/invocations/inv-a"},
		{name: "workspace forbidden", readerErr: workspace.ErrForbidden, status: http.StatusForbidden, code: contracts.ErrorCodeForbidden, path: "/v4/workspaces/workspace-a/invocations/inv-a", wantInvoke: 1},
		{name: "workspace missing", readerErr: workspace.ErrNotFound, status: http.StatusNotFound, code: contracts.ErrorCodeNotFound, path: "/v4/workspaces/workspace-a/invocations/inv-a", wantInvoke: 1},
		{name: "workspace dependency", readerErr: workspace.ErrDependency, status: http.StatusServiceUnavailable, code: contracts.ErrorCodeDependency, path: "/v4/workspaces/workspace-a/invocations/inv-a", wantInvoke: 1},
		{name: "read deadline", readerErr: context.DeadlineExceeded, status: http.StatusServiceUnavailable, code: contracts.ErrorCodeDependency, path: "/v4/workspaces/workspace-a/invocations/inv-a", wantInvoke: 1},
		{name: "Router not found", response: metadataHTTPResponse(http.StatusNotFound, `{"code":"NOT_FOUND"}`), status: http.StatusNotFound, code: contracts.ErrorCodeNotFound, path: "/v4/workspaces/workspace-a/invocations/inv-a", wantInvoke: 1},
		{name: "Router dependency", response: metadataHTTPResponse(http.StatusServiceUnavailable, `{"code":"DEPENDENCY_ERROR"}`), status: http.StatusServiceUnavailable, code: contracts.ErrorCodeDependency, path: "/v4/workspaces/workspace-a/invocations/inv-a", wantInvoke: 1},
		{name: "Router wrong media", response: &invocation.RouterResponse{StatusCode: http.StatusOK, ContentType: "text/plain", Body: io.NopCloser(strings.NewReader("internal"))}, status: http.StatusServiceUnavailable, code: contracts.ErrorCodeDependency, path: "/v4/workspaces/workspace-a/invocations/inv-a", wantInvoke: 1},
		{name: "Router not-found wrong media", response: &invocation.RouterResponse{StatusCode: http.StatusNotFound, ContentType: "text/plain", Body: io.NopCloser(strings.NewReader("internal"))}, status: http.StatusServiceUnavailable, code: contracts.ErrorCodeDependency, path: "/v4/workspaces/workspace-a/invocations/inv-a", wantInvoke: 1},
		{name: "Router malformed success", response: metadataHTTPResponse(http.StatusOK, `{}`), status: http.StatusServiceUnavailable, code: contracts.ErrorCodeDependency, path: "/v4/workspaces/workspace-a/invocations/inv-a", wantInvoke: 1},
		{name: "Router content-bearing success", response: metadataHTTPResponse(http.StatusOK, strings.Replace(validInvocationMetadataJSON, `"events":[`, `"input":{"secret":"value"},"events":[`, 1)), status: http.StatusServiceUnavailable, code: contracts.ErrorCodeDependency, path: "/v4/workspaces/workspace-a/invocations/inv-a", wantInvoke: 1},
		{name: "Router metadata exceeds separate limit", response: metadataHTTPResponse(http.StatusOK, validInvocationMetadataJSON+strings.Repeat(" ", 5000)), status: http.StatusServiceUnavailable, code: contracts.ErrorCodeDependency, path: "/v4/workspaces/workspace-a/invocations/inv-a", wantInvoke: 1},
		{name: "Router duplicate member", response: metadataHTTPResponse(http.StatusOK, strings.Replace(validInvocationMetadataJSON, `"invocationId":"inv-a","rootTaskId"`, `"invocationId":"inv-a","invocationId":"inv-a","rootTaskId"`, 1)), status: http.StatusServiceUnavailable, code: contracts.ErrorCodeDependency, path: "/v4/workspaces/workspace-a/invocations/inv-a", wantInvoke: 1},
		{name: "Router trailing JSON", response: metadataHTTPResponse(http.StatusOK, validInvocationMetadataJSON+`{}`), status: http.StatusServiceUnavailable, code: contracts.ErrorCodeDependency, path: "/v4/workspaces/workspace-a/invocations/inv-a", wantInvoke: 1},
		{name: "Router unknown nested event member", response: metadataHTTPResponse(http.StatusOK, strings.Replace(validInvocationMetadataJSON, `"eventId":"event-a"`, `"eventId":"event-a","secret":"value"`, 1)), status: http.StatusServiceUnavailable, code: contracts.ErrorCodeDependency, path: "/v4/workspaces/workspace-a/invocations/inv-a", wantInvoke: 1},
		{name: "Router malformed trace record", response: metadataHTTPResponse(http.StatusOK, strings.Replace(validTraceMetadataJSON, `,"createdAt":"2026-07-16T12:00:00Z"`, ``, 1)), status: http.StatusServiceUnavailable, code: contracts.ErrorCodeDependency, path: "/v4/workspaces/workspace-a/invocations/inv-a", wantInvoke: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader := &metadataReadHandlerStub{err: test.readerErr, invocationResponse: test.response}
			handler := newInvocationReadTestHandler(t, invocationAuthenticatorStub{caller: catalog.AuthenticatedCaller{ID: "owner-a"}, err: test.authErr}, reader)
			response := serveInvocationReadTestRequest(handler, test.path)
			if response.Code != test.status || reader.invocationCalls != test.wantInvoke {
				t.Fatalf("status=%d calls=%d body=%s", response.Code, reader.invocationCalls, response.Body.String())
			}
			var payload contracts.PreCorrelationPlatformErrorV4
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil || payload.Code != test.code || payload.TraceID == "" {
				t.Fatalf("payload=%#v decode=%v", payload, err)
			}
			if strings.Contains(response.Body.String(), "internal") {
				t.Fatal("raw Router dependency detail was exposed")
			}
		})
	}
}

func TestInvocationReadHandlerRequiresDependencies(t *testing.T) {
	traces, err := newTraceGenerator(bytes.NewReader(make([]byte, 16)))
	if err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	if _, err := NewInvocationReadHandler(nil, &metadataReadHandlerStub{}, traces, logger, time.Second, 1024); err == nil {
		t.Fatal("nil authenticator accepted")
	}
	if _, err := NewInvocationReadHandler(invocationAuthenticatorStub{}, nil, traces, logger, time.Second, 1024); err == nil {
		t.Fatal("nil metadata reader accepted")
	}
}

func TestInvocationReadHandlerRejectsMalformedTraceRecord(t *testing.T) {
	reader := &metadataReadHandlerStub{traceResponse: metadataHTTPResponse(http.StatusOK, strings.Replace(validTraceMetadataJSON, `,"createdAt":"2026-07-16T12:00:00Z"`, ``, 1))}
	handler := newInvocationReadTestHandler(t, invocationAuthenticatorStub{caller: catalog.AuthenticatedCaller{ID: "owner-a"}}, reader)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v4/workspaces/workspace-a/traces/trace-a", nil))
	if response.Code != http.StatusServiceUnavailable || reader.traceCalls != 1 {
		t.Fatalf("malformed Trace response = %d, calls=%d, body=%s", response.Code, reader.traceCalls, response.Body.String())
	}
}

func newInvocationReadTestHandler(t *testing.T, authenticator Authenticator, reader InvocationMetadataReader) http.Handler {
	t.Helper()
	traces, err := newTraceGenerator(bytes.NewReader(make([]byte, 16)))
	if err != nil {
		t.Fatal(err)
	}
	handler, err := NewInvocationReadHandler(authenticator, reader, traces, slog.New(slog.NewTextHandler(io.Discard, nil)), time.Second, 4096)
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	return mux
}

func serveInvocationReadTestRequest(handler http.Handler, path string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func metadataHTTPResponse(status int, body string) *invocation.RouterResponse {
	return &invocation.RouterResponse{StatusCode: status, ContentType: "application/json", Body: io.NopCloser(strings.NewReader(body))}
}

const validInvocationMetadataJSON = `{"invocation":{"invocationId":"inv-a","rootTaskId":"task-a","traceId":"trace-a","caller":{"type":"user","id":"owner-a"},"workspaceId":"workspace-a","targetAgentId":"agent-a","agentCardVersion":"1.0.0","capability":"capability.read","status":"pending","createdAt":"2026-07-16T12:00:00Z","updatedAt":"2026-07-16T12:00:00Z"},"events":[{"schemaVersion":"0.3","eventId":"event-a","sequence":0,"occurredAt":"2026-07-16T12:00:00Z","type":"created","status":"pending","invocationId":"inv-a","rootTaskId":"task-a","traceId":"trace-a","caller":{"type":"user","id":"owner-a"},"workspaceId":"workspace-a","targetAgentId":"agent-a","agentCardVersion":"1.0.0","capability":"capability.read"}]}`

const validTraceMetadataJSON = `{"traceId":"trace-a","invocations":[{"invocationId":"inv-a","rootTaskId":"task-a","traceId":"trace-a","caller":{"type":"user","id":"owner-a"},"workspaceId":"workspace-a","targetAgentId":"agent-a","agentCardVersion":"1.0.0","capability":"capability.read","status":"pending","createdAt":"2026-07-16T12:00:00Z","updatedAt":"2026-07-16T12:00:00Z"}]}`
