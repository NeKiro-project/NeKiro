package contracts

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"

	"github.com/a2aproject/a2a-go/a2a"
)

const (
	A2AProfileSchemaVersionV02 = "0.2"
	A2AProfileProtocolVersion  = "0.3.0"
	A2AProfileSDKModule        = "github.com/a2aproject/a2a-go"
	A2AProfileSDKVersion       = "v0.3.15"
)

//go:embed schemas/a2a-profile.v0.2.schema.json a2a-profile/v0.3.0/profile.v0.2.json a2a-profile/v0.3.0/conformance/*.json a2a-profile/v0.3.0/conformance/*.sse
var a2aProfileV02Files embed.FS

type A2AProfileV02 struct {
	SchemaVersion   string                   `json:"schemaVersion"`
	ProtocolVersion string                   `json:"protocolVersion"`
	SDK             A2ASDK                   `json:"sdk"`
	Transport       string                   `json:"transport"`
	AgentCardPath   string                   `json:"agentCardPath"`
	Operations      []A2AProfileOperationV02 `json:"operations"`
	TaskStates      A2AProfileTaskStatesV02  `json:"taskStates"`
	ContextHeaders  A2AContextHeaders        `json:"contextHeaders"`
	Conformance     A2AProfileConformanceV02 `json:"conformance"`
}

type A2AProfileOperationV02 struct {
	Method              string   `json:"method"`
	ClientMethod        string   `json:"clientMethod"`
	ServerMethod        string   `json:"serverMethod"`
	Interaction         string   `json:"interaction"`
	RequestType         string   `json:"requestType"`
	AcceptedResultKinds []string `json:"acceptedResultKinds,omitempty"`
	AcceptedEventKinds  []string `json:"acceptedEventKinds,omitempty"`
	ExpectedErrors      []string `json:"expectedErrors,omitempty"`
}

type A2AProfileTaskStatesV02 struct {
	Transient   []A2AProfileTaskStateV02 `json:"transient"`
	Terminal    []A2AProfileTaskStateV02 `json:"terminal"`
	Unsupported []A2AProfileTaskStateV02 `json:"unsupported"`
	Unspecified A2AProfileTaskStateV02   `json:"unspecified"`
}

type A2AProfileTaskStateV02 struct {
	State            string            `json:"state"`
	InvocationStatus string            `json:"invocationStatus"`
	ErrorCode        PlatformErrorCode `json:"errorCode,omitempty"`
}

type A2AProfileConformanceV02 struct {
	Manifest         string `json:"manifest"`
	FixtureAuthority string `json:"fixtureAuthority"`
	JSONRPCVersion   string `json:"jsonrpcVersion"`
	SSEMediaType     string `json:"sseMediaType"`
}

type A2AConformanceManifestV02 struct {
	SchemaVersion        string                  `json:"schemaVersion"`
	ProfileSchemaVersion string                  `json:"profileSchemaVersion"`
	ProtocolVersion      string                  `json:"protocolVersion"`
	Cases                []A2AConformanceCaseV02 `json:"cases"`
}

type A2AConformanceCaseV02 struct {
	ID             string   `json:"id"`
	File           string   `json:"file"`
	RequestFile    string   `json:"requestFile,omitempty"`
	Operation      string   `json:"operation"`
	FixtureKind    string   `json:"fixtureKind"`
	ExpectedValid  bool     `json:"expectedValid"`
	WireResultKind string   `json:"wireResultKind,omitempty"`
	GoConcreteType string   `json:"goConcreteType,omitempty"`
	ProtocolError  string   `json:"protocolError,omitempty"`
	MediaType      string   `json:"mediaType"`
	Rules          []string `json:"rules"`
}

type A2ATaskStateClassification string

const (
	A2ATaskStateTransient A2ATaskStateClassification = "transient"
	A2ATaskStateTerminal  A2ATaskStateClassification = "terminal"
)

type A2ATaskStateMapping struct {
	State            a2a.TaskState
	Classification   A2ATaskStateClassification
	InvocationStatus string
	ErrorCode        PlatformErrorCode
}

type A2AProfileStateError struct {
	State     a2a.TaskState
	Reason    string
	ErrorCode PlatformErrorCode
}

func (e *A2AProfileStateError) Error() string {
	return fmt.Sprintf("A2A task state %q is %s in Profile v0.2", e.State, e.Reason)
}

type A2AProfileTaskError struct {
	Reason    string
	ErrorCode PlatformErrorCode
}

func (e *A2AProfileTaskError) Error() string {
	return fmt.Sprintf("A2A task violates Profile v0.2: %s", e.Reason)
}

func LoadA2AProfileV02() (A2AProfileV02, error) {
	return decodeA2AProfileV02[A2AProfileV02]("a2a-profile/v0.3.0/profile.v0.2.json")
}

func LoadA2AConformanceManifestV02() (A2AConformanceManifestV02, error) {
	return decodeA2AProfileV02[A2AConformanceManifestV02]("a2a-profile/v0.3.0/conformance/manifest.json")
}

func A2AProfileV02Files() fs.FS {
	return a2aProfileV02Files
}

func MapA2ATaskState(state a2a.TaskState) (A2ATaskStateMapping, error) {
	switch state {
	case a2a.TaskStateSubmitted, a2a.TaskStateWorking:
		return A2ATaskStateMapping{
			State:            state,
			Classification:   A2ATaskStateTransient,
			InvocationStatus: "running",
		}, nil
	case a2a.TaskStateCompleted:
		return A2ATaskStateMapping{
			State:            state,
			Classification:   A2ATaskStateTerminal,
			InvocationStatus: "succeeded",
		}, nil
	case a2a.TaskStateFailed, a2a.TaskStateRejected:
		return A2ATaskStateMapping{
			State:            state,
			Classification:   A2ATaskStateTerminal,
			InvocationStatus: "failed",
			ErrorCode:        ErrorCodeAgentExecutionFailed,
		}, nil
	case a2a.TaskStateCanceled:
		return A2ATaskStateMapping{
			State:            state,
			Classification:   A2ATaskStateTerminal,
			InvocationStatus: "canceled",
			ErrorCode:        ErrorCodeCanceled,
		}, nil
	case a2a.TaskStateAuthRequired, a2a.TaskStateInputRequired, a2a.TaskStateUnknown:
		return A2ATaskStateMapping{}, &A2AProfileStateError{
			State:     state,
			Reason:    "recognized but unsupported",
			ErrorCode: ErrorCodeA2AProtocol,
		}
	case a2a.TaskStateUnspecified:
		return A2ATaskStateMapping{}, &A2AProfileStateError{
			State:     state,
			Reason:    "unspecified",
			ErrorCode: ErrorCodeA2AProtocol,
		}
	default:
		return A2ATaskStateMapping{}, &A2AProfileStateError{
			State:     state,
			Reason:    "not defined by A2A protocol 0.3.0",
			ErrorCode: ErrorCodeA2AProtocol,
		}
	}
}

func ValidateA2ATask(task *a2a.Task) (A2ATaskStateMapping, error) {
	if task == nil {
		return A2ATaskStateMapping{}, &A2AProfileTaskError{
			Reason:    "task is missing",
			ErrorCode: ErrorCodeA2AProtocol,
		}
	}
	if task.ID == "" {
		return A2ATaskStateMapping{}, &A2AProfileTaskError{
			Reason:    "task id is empty",
			ErrorCode: ErrorCodeA2AProtocol,
		}
	}
	if task.ContextID == "" {
		return A2ATaskStateMapping{}, &A2AProfileTaskError{
			Reason:    "context id is empty",
			ErrorCode: ErrorCodeA2AProtocol,
		}
	}
	return MapA2ATaskState(task.Status.State)
}

func decodeA2AProfileV02[T any](path string) (T, error) {
	var value T
	data, err := a2aProfileV02Files.ReadFile(path)
	if err != nil {
		return value, fmt.Errorf("read %s: %w", path, err)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, fmt.Errorf("decode %s: %w", path, err)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return value, fmt.Errorf("decode %s: %w", path, err)
	}
	return value, nil
}
