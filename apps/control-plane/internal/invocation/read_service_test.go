package invocation

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Nene7ko/NeKiro/apps/control-plane/internal/workspace"
	"github.com/Nene7ko/NeKiro/contracts"
)

type metadataWorkspaceStub struct {
	value contracts.Workspace
	err   error
	calls int
}

func (stub *metadataWorkspaceStub) GetWorkspace(context.Context, workspace.AuthenticatedCaller, string) (contracts.Workspace, error) {
	stub.calls++
	return stub.value, stub.err
}

type metadataRouterStub struct {
	response       *RouterResponse
	err            error
	invocationCall int
	traceCall      int
	workspaceID    string
	resourceID     string
	traceID        contracts.TraceID
}

func (stub *metadataRouterStub) GetInvocation(_ context.Context, workspaceID, invocationID string) (*RouterResponse, error) {
	stub.invocationCall++
	stub.workspaceID, stub.resourceID = workspaceID, invocationID
	return stub.response, stub.err
}

func (stub *metadataRouterStub) GetTrace(_ context.Context, workspaceID string, traceID contracts.TraceID) (*RouterResponse, error) {
	stub.traceCall++
	stub.workspaceID, stub.traceID = workspaceID, traceID
	return stub.response, stub.err
}

func TestMetadataReadServiceAuthorizesWorkspaceBeforeRouter(t *testing.T) {
	workspaceReader := &metadataWorkspaceStub{value: contracts.Workspace{WorkspaceID: "workspace-a", OwnerID: "owner-a"}}
	router := &metadataRouterStub{response: metadataReadResponse()}
	service, err := NewMetadataReadService(workspaceReader, router)
	if err != nil {
		t.Fatal(err)
	}
	response, err := service.GetInvocation(context.Background(), workspace.AuthenticatedCaller{ID: "owner-a"}, "workspace-a", "inv-a")
	if err != nil {
		t.Fatalf("authorized Invocation read: %v", err)
	}
	if response != router.response || workspaceReader.calls != 1 || router.invocationCall != 1 || router.workspaceID != "workspace-a" || router.resourceID != "inv-a" {
		t.Fatalf("workspace/router calls = %d/%d response=%p", workspaceReader.calls, router.invocationCall, response)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestMetadataReadServiceDoesNotContactRouterOnWorkspaceFailure(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
	}{
		{name: "not found", err: workspace.ErrNotFound},
		{name: "forbidden", err: workspace.ErrForbidden},
		{name: "dependency", err: workspace.ErrDependency},
	} {
		t.Run(test.name, func(t *testing.T) {
			workspaceReader := &metadataWorkspaceStub{err: test.err}
			router := &metadataRouterStub{response: metadataReadResponse()}
			service, err := NewMetadataReadService(workspaceReader, router)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := service.GetTrace(context.Background(), workspace.AuthenticatedCaller{ID: "caller-a"}, "workspace-a", "trace-a"); !errors.Is(err, test.err) {
				t.Fatalf("error=%v want=%v", err, test.err)
			}
			if router.traceCall != 0 {
				t.Fatalf("Router calls=%d, want zero", router.traceCall)
			}
		})
	}
}

func TestMetadataReadServiceRequiresBothPorts(t *testing.T) {
	if _, err := NewMetadataReadService(nil, &metadataRouterStub{}); err == nil {
		t.Fatal("nil Workspace reader accepted")
	}
	if _, err := NewMetadataReadService(&metadataWorkspaceStub{}, nil); err == nil {
		t.Fatal("nil metadata Router accepted")
	}
}

func metadataReadResponse() *RouterResponse {
	return &RouterResponse{StatusCode: http.StatusOK, ContentType: "application/json", Body: io.NopCloser(strings.NewReader("{}"))}
}
