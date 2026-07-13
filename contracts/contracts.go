package contracts

import (
	"fmt"
	"time"
)

const (
	AgentCardSchemaVersion       = "0.1"
	InvocationEventSchemaVersion = "0.1"
	A2AProtocolVersion           = "0.3.0"
)

type JSONSchema map[string]any

type AgentOwner struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}

type AgentProtocol struct {
	Type      string `json:"type"`
	Version   string `json:"version"`
	Transport string `json:"transport"`
	Endpoint  string `json:"endpoint"`
}

type AgentSkill struct {
	ID                  string     `json:"id"`
	Name                string     `json:"name"`
	Description         string     `json:"description"`
	InputSchema         JSONSchema `json:"inputSchema"`
	OutputSchema        JSONSchema `json:"outputSchema"`
	RequiredPermissions []string   `json:"requiredPermissions"`
}

type AgentAuthentication struct {
	Type string `json:"type"`
}

type PermissionDeclaration struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

type AgentLimits struct {
	TimeoutMS      int64 `json:"timeoutMs"`
	MaxInputBytes  int64 `json:"maxInputBytes"`
	MaxOutputBytes int64 `json:"maxOutputBytes"`
	Streaming      bool  `json:"streaming"`
}

type AgentCard struct {
	SchemaVersion  string                  `json:"schemaVersion"`
	AgentID        string                  `json:"agentId"`
	Name           string                  `json:"name"`
	Description    string                  `json:"description"`
	Owner          AgentOwner              `json:"owner"`
	Version        string                  `json:"version"`
	Protocol       AgentProtocol           `json:"protocol"`
	Skills         []AgentSkill            `json:"skills"`
	Authentication AgentAuthentication     `json:"authentication"`
	Permissions    []PermissionDeclaration `json:"permissions"`
	Limits         AgentLimits             `json:"limits"`
}

type CatalogEntry struct {
	Card              AgentCard  `json:"card"`
	PublicationStatus string     `json:"publicationStatus"`
	RegisteredAt      time.Time  `json:"registeredAt"`
	PublishedAt       *time.Time `json:"publishedAt,omitempty"`
}

type RegisterAgentRequest struct {
	Card AgentCard `json:"card"`
}

type SearchAgentsQuery struct {
	Query      *string `json:"query,omitempty"`
	Capability *string `json:"capability,omitempty"`
	OwnerID    *string `json:"ownerId,omitempty"`
	Limit      *int    `json:"limit,omitempty"`
	Cursor     *string `json:"cursor,omitempty"`
}

type SearchAgentsResponse struct {
	Items      []CatalogEntry `json:"items"`
	NextCursor *string        `json:"nextCursor,omitempty"`
}

type Installation struct {
	InstallationID      string    `json:"installationId"`
	WorkspaceID         string    `json:"workspaceId"`
	AgentID             string    `json:"agentId"`
	VersionConstraint   string    `json:"versionConstraint"`
	InstalledVersion    string    `json:"installedVersion"`
	AcceptedPermissions []string  `json:"acceptedPermissions"`
	Status              string    `json:"status"`
	InstalledAt         time.Time `json:"installedAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

type InstallAgentRequest struct {
	AgentID             string   `json:"agentId"`
	VersionConstraint   string   `json:"versionConstraint"`
	AcceptedPermissions []string `json:"acceptedPermissions"`
}

type UpdateInstallationRequest struct {
	Status string `json:"status"`
}

type Caller struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type PlatformErrorCode string

const (
	ErrorCodeValidationError      PlatformErrorCode = "VALIDATION_ERROR"
	ErrorCodeUnauthenticated      PlatformErrorCode = "UNAUTHENTICATED"
	ErrorCodeForbidden            PlatformErrorCode = "FORBIDDEN"
	ErrorCodeNotFound             PlatformErrorCode = "NOT_FOUND"
	ErrorCodeConflict             PlatformErrorCode = "CONFLICT"
	ErrorCodeAgentNotInstalled    PlatformErrorCode = "AGENT_NOT_INSTALLED"
	ErrorCodeAgentDisabled        PlatformErrorCode = "AGENT_DISABLED"
	ErrorCodeCapabilityNotAllowed PlatformErrorCode = "CAPABILITY_NOT_ALLOWED"
	ErrorCodeRouteNotFound        PlatformErrorCode = "ROUTE_NOT_FOUND"
	ErrorCodeA2AProtocol          PlatformErrorCode = "A2A_PROTOCOL_ERROR"
	ErrorCodeAgentUnavailable     PlatformErrorCode = "AGENT_UNAVAILABLE"
	ErrorCodeAgentExecutionFailed PlatformErrorCode = "AGENT_EXECUTION_FAILED"
	ErrorCodeDependency           PlatformErrorCode = "DEPENDENCY_ERROR"
	ErrorCodeTimeout              PlatformErrorCode = "TIMEOUT"
	ErrorCodeCanceled             PlatformErrorCode = "CANCELED"
	ErrorCodeInternal             PlatformErrorCode = "INTERNAL_ERROR"
)

var publicErrorMessages = map[PlatformErrorCode]string{
	ErrorCodeValidationError:      "The request is invalid.",
	ErrorCodeUnauthenticated:      "Authentication is required.",
	ErrorCodeForbidden:            "The requested operation is not allowed.",
	ErrorCodeNotFound:             "The requested resource was not found.",
	ErrorCodeConflict:             "The requested operation conflicts with current state.",
	ErrorCodeAgentNotInstalled:    "The Agent is not installed in this Workspace.",
	ErrorCodeAgentDisabled:        "The Agent version is disabled.",
	ErrorCodeCapabilityNotAllowed: "The requested capability is not allowed.",
	ErrorCodeRouteNotFound:        "No route is available for the Agent.",
	ErrorCodeA2AProtocol:          "The Agent returned an invalid A2A response.",
	ErrorCodeAgentUnavailable:     "The Agent is unavailable.",
	ErrorCodeAgentExecutionFailed: "The Agent failed to complete the invocation.",
	ErrorCodeDependency:           "A required platform dependency failed.",
	ErrorCodeTimeout:              "The invocation timed out.",
	ErrorCodeCanceled:             "The invocation was canceled.",
	ErrorCodeInternal:             "The platform could not complete the request.",
}

type PlatformError struct {
	Code    PlatformErrorCode `json:"code"`
	Message string            `json:"message"`
	TraceID string            `json:"traceId,omitempty"`
}

func NewPlatformError(code PlatformErrorCode, traceID string) (PlatformError, error) {
	message, exists := publicErrorMessages[code]
	if !exists {
		return PlatformError{}, fmt.Errorf("unknown platform error code %q", code)
	}
	return PlatformError{Code: code, Message: message, TraceID: traceID}, nil
}

type InvocationEvent struct {
	SchemaVersion      string         `json:"schemaVersion"`
	EventID            string         `json:"eventId"`
	Sequence           int64          `json:"sequence"`
	OccurredAt         time.Time      `json:"occurredAt"`
	Type               string         `json:"type"`
	Status             string         `json:"status"`
	InvocationID       string         `json:"invocationId"`
	RootTaskID         string         `json:"rootTaskId"`
	ParentInvocationID string         `json:"parentInvocationId,omitempty"`
	TraceID            string         `json:"traceId"`
	Caller             Caller         `json:"caller"`
	WorkspaceID        string         `json:"workspaceId"`
	TargetAgentID      string         `json:"targetAgentId"`
	AgentCardVersion   string         `json:"agentCardVersion"`
	Capability         string         `json:"capability"`
	ChunkIndex         *int64         `json:"chunkIndex,omitempty"`
	ChunkBytes         *int64         `json:"chunkBytes,omitempty"`
	LatencyMS          *int64         `json:"latencyMs,omitempty"`
	Error              *PlatformError `json:"error,omitempty"`
}

type InvokeAgentRequest struct {
	AgentID    string         `json:"agentId"`
	Capability string         `json:"capability"`
	Input      map[string]any `json:"input"`
	Stream     bool           `json:"stream"`
}

type InvocationAccepted struct {
	InvocationID string `json:"invocationId"`
	RootTaskID   string `json:"rootTaskId"`
	TraceID      string `json:"traceId"`
	Status       string `json:"status"`
}

type InvocationRecord struct {
	InvocationID       string            `json:"invocationId"`
	RootTaskID         string            `json:"rootTaskId"`
	ParentInvocationID string            `json:"parentInvocationId,omitempty"`
	TraceID            string            `json:"traceId"`
	Caller             Caller            `json:"caller"`
	WorkspaceID        string            `json:"workspaceId"`
	TargetAgentID      string            `json:"targetAgentId"`
	AgentCardVersion   string            `json:"agentCardVersion"`
	Capability         string            `json:"capability"`
	Status             string            `json:"status"`
	LatencyMS          *int64            `json:"latencyMs,omitempty"`
	ErrorCode          PlatformErrorCode `json:"errorCode,omitempty"`
	CreatedAt          time.Time         `json:"createdAt"`
	UpdatedAt          time.Time         `json:"updatedAt"`
}

type InvocationDetailResponse struct {
	Invocation InvocationRecord  `json:"invocation"`
	Events     []InvocationEvent `json:"events"`
}

type TraceResponse struct {
	TraceID     string             `json:"traceId"`
	Invocations []InvocationRecord `json:"invocations"`
}

type RouterEventEnvelope struct {
	Event InvocationEvent `json:"event"`
}

type A2ASDK struct {
	Module  string `json:"module"`
	Version string `json:"version"`
}

type A2AContextHeaders struct {
	TraceID            string `json:"traceId"`
	InvocationID       string `json:"invocationId"`
	RootTaskID         string `json:"rootTaskId"`
	ParentInvocationID string `json:"parentInvocationId"`
	WorkspaceID        string `json:"workspaceId"`
}

type A2AProfile struct {
	SchemaVersion   string            `json:"schemaVersion"`
	ProtocolVersion string            `json:"protocolVersion"`
	SDK             A2ASDK            `json:"sdk"`
	Transport       string            `json:"transport"`
	AgentCardPath   string            `json:"agentCardPath"`
	RequiredMethods []string          `json:"requiredMethods"`
	ContextHeaders  A2AContextHeaders `json:"contextHeaders"`
}

type ResolveAgentRequest struct {
	WorkspaceID string `json:"workspaceId"`
	AgentID     string `json:"agentId"`
	Version     string `json:"version"`
	Capability  string `json:"capability"`
}

type ResolvedInstallation struct {
	InstallationID      string   `json:"installationId"`
	WorkspaceID         string   `json:"workspaceId"`
	AgentID             string   `json:"agentId"`
	InstalledVersion    string   `json:"installedVersion"`
	AcceptedPermissions []string `json:"acceptedPermissions"`
	Status              string   `json:"status"`
}

type ResolveAgentResponse struct {
	Card         AgentCard            `json:"card"`
	Installation ResolvedInstallation `json:"installation"`
}

type DispatchInvocationRequest struct {
	InvocationID       string         `json:"invocationId"`
	RootTaskID         string         `json:"rootTaskId"`
	ParentInvocationID string         `json:"parentInvocationId,omitempty"`
	TraceID            string         `json:"traceId"`
	Caller             Caller         `json:"caller"`
	WorkspaceID        string         `json:"workspaceId"`
	TargetAgentID      string         `json:"targetAgentId"`
	AgentCardVersion   string         `json:"agentCardVersion"`
	Capability         string         `json:"capability"`
	Input              map[string]any `json:"input"`
	Stream             bool           `json:"stream"`
}

type DispatchInvocationAccepted struct {
	InvocationID string `json:"invocationId"`
	Accepted     bool   `json:"accepted"`
}
