package invocation

import (
	"context"
	"errors"

	"github.com/Nene7ko/NeKiro/apps/control-plane/internal/workspace"
	"github.com/Nene7ko/NeKiro/contracts"
)

// MetadataRouter is the only downstream port used by the Control Plane read
// service. It returns Router-owned metadata responses and never exposes a
// Ledger store or Agent endpoint to Gateway.
type MetadataRouter interface {
	GetInvocation(context.Context, string, string) (*RouterResponse, error)
	GetTrace(context.Context, string, contracts.TraceID) (*RouterResponse, error)
}

// WorkspaceReader is the existing Workspace owner boundary. The read service
// deliberately consumes the public Workspace read port instead of duplicating
// owner policy or querying Workspace storage.
type WorkspaceReader interface {
	GetWorkspace(context.Context, workspace.AuthenticatedCaller, string) (contracts.Workspace, error)
}

type MetadataReadService struct {
	workspace WorkspaceReader
	router    MetadataRouter
}

func NewMetadataReadService(workspaceReader WorkspaceReader, router MetadataRouter) (*MetadataReadService, error) {
	if workspaceReader == nil || router == nil {
		return nil, errors.New("metadata read dependencies are required")
	}
	return &MetadataReadService{workspace: workspaceReader, router: router}, nil
}

func (service *MetadataReadService) GetInvocation(
	ctx context.Context,
	caller workspace.AuthenticatedCaller,
	workspaceID, invocationID string,
) (*RouterResponse, error) {
	if _, err := service.workspace.GetWorkspace(ctx, caller, workspaceID); err != nil {
		return nil, err
	}
	return service.router.GetInvocation(ctx, workspaceID, invocationID)
}

func (service *MetadataReadService) GetTrace(
	ctx context.Context,
	caller workspace.AuthenticatedCaller,
	workspaceID string,
	traceID contracts.TraceID,
) (*RouterResponse, error) {
	if _, err := service.workspace.GetWorkspace(ctx, caller, workspaceID); err != nil {
		return nil, err
	}
	return service.router.GetTrace(ctx, workspaceID, traceID)
}
