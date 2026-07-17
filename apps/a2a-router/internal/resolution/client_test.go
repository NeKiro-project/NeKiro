package resolution

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Nene7ko/NeKiro/contracts"
)

func TestClientResolveSendsExactInternalV2Request(t *testing.T) {
	requestValue := contracts.ResolveAgentRequest{InvocationID: "inv-a", RootTaskID: "task-a", TraceID: "trace-a", WorkspaceID: "workspace-a", AgentID: "agent-a", Version: "1.0.0", Capability: "capability-a"}
	var received contracts.ResolveAgentRequest
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/internal/v2/resolve-agent" || request.Method != http.MethodPost || request.Header.Get("Authorization") != "Bearer control-token" || request.Header.Get("Content-Type") != "application/json" || request.Header.Get("Accept") != "application/json" {
			t.Errorf("unexpected request: %s %s %#v", request.Method, request.URL.Path, request.Header)
		}
		if err := json.NewDecoder(request.Body).Decode(&received); err != nil {
			t.Error(err)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(writer, `{"card":{"schemaVersion":"0.2","agentId":"agent-a","name":"Agent","description":"Agent","owner":{"id":"team","displayName":"Team"},"version":"1.0.0","protocol":{"type":"a2a","version":"0.3.0","transport":"jsonrpc-http","endpoint":"https://agent.example/a2a"},"skills":[{"id":"capability-a","name":"Capability","description":"Capability","inputSchema":{},"outputSchema":{},"requiredPermissions":[]}],"authentication":{"type":"none"},"permissions":[],"limits":{"timeoutMs":1000,"maxInputBytes":1024,"maxOutputBytes":1024,"streaming":true}},"installation":{"installationId":"inst-a","workspaceId":"workspace-a","agentId":"agent-a","installedVersion":"1.0.0","acceptedPermissions":[],"status":"enabled"}}`)
	}))
	defer server.Close()
	client, err := NewClient(server.Client(), server.URL+"/internal/v2/resolve-agent", "control-token", 4096)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := client.Resolve(context.Background(), requestValue)
	if err != nil {
		t.Fatal(err)
	}
	if received != requestValue || resolved.Card.AgentID != requestValue.AgentID || resolved.Installation.InstalledVersion != requestValue.Version {
		t.Fatalf("received=%#v resolved=%#v", received, resolved)
	}
}

func TestClientResolveMapsTypedFailuresAndDependenciesWithoutRetry(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		calls++
		writer.Header().Set("Content-Type", "application/json")
		writer.Header().Set("x-nek-trace-id", "trace-control")
		writer.WriteHeader(http.StatusForbidden)
		_, _ = io.WriteString(writer, `{"code":"CAPABILITY_NOT_ALLOWED","message":"The requested capability is not allowed.","traceId":"trace-a","invocationId":"inv-a","rootTaskId":"task-a"}`)
	}))
	defer server.Close()
	client, _ := NewClient(server.Client(), server.URL, "control-token", 1024)
	_, err := client.Resolve(context.Background(), contracts.ResolveAgentRequest{})
	var failure *Failure
	if !errors.As(err, &failure) || failure.Code != contracts.ErrorCodeCapabilityNotAllowed || failure.StatusCode != http.StatusForbidden || failure.TraceID != "trace-control" || calls != 1 || string(failure.Body) == "" {
		t.Fatalf("failure=%#v err=%v calls=%d", failure, err, calls)
	}
}

func TestClientResolveRejectsBadMediaAndOversize(t *testing.T) {
	for _, test := range []struct {
		name        string
		contentType string
		body        string
		status      int
	}{
		{name: "bad media", contentType: "text/plain", body: "{}", status: http.StatusOK},
		{name: "oversize", contentType: "application/json", body: `{"code":"DEPENDENCY_ERROR"}`, status: http.StatusOK},
		{name: "missing trace header on error", contentType: "application/json", body: `{"code":"DEPENDENCY_ERROR"}`, status: http.StatusServiceUnavailable},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.Header().Set("Content-Type", test.contentType)
				writer.WriteHeader(test.status)
				_, _ = io.WriteString(writer, test.body)
			}))
			defer server.Close()
			client, _ := NewClient(server.Client(), server.URL, "control-token", 4)
			if _, err := client.Resolve(context.Background(), contracts.ResolveAgentRequest{}); err == nil {
				t.Fatal("invalid Control Plane response accepted")
			}
		})
	}
}
