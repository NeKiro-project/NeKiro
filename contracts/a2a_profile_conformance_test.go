package contracts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"iter"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2aclient"
	"github.com/a2aproject/a2a-go/a2asrv"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

var (
	_ func(*a2aclient.Client, context.Context, *a2a.MessageSendParams) (a2a.SendMessageResult, error) = (*a2aclient.Client).SendMessage
	_ func(*a2aclient.Client, context.Context, *a2a.MessageSendParams) iter.Seq2[a2a.Event, error]    = (*a2aclient.Client).SendStreamingMessage
	_ func(*a2aclient.Client, context.Context, *a2a.TaskQueryParams) (*a2a.Task, error)               = (*a2aclient.Client).GetTask
	_ func(*a2aclient.Client, context.Context, *a2a.TaskIDParams) (*a2a.Task, error)                  = (*a2aclient.Client).CancelTask

	_ func(a2asrv.RequestHandler, context.Context, *a2a.MessageSendParams) (a2a.SendMessageResult, error) = a2asrv.RequestHandler.OnSendMessage
	_ func(a2asrv.RequestHandler, context.Context, *a2a.MessageSendParams) iter.Seq2[a2a.Event, error]    = a2asrv.RequestHandler.OnSendMessageStream
	_ func(a2asrv.RequestHandler, context.Context, *a2a.TaskQueryParams) (*a2a.Task, error)               = a2asrv.RequestHandler.OnGetTask
	_ func(a2asrv.RequestHandler, context.Context, *a2a.TaskIDParams) (*a2a.Task, error)                  = a2asrv.RequestHandler.OnCancelTask

	_ a2a.SendMessageResult = (*a2a.Message)(nil)
	_ a2a.SendMessageResult = (*a2a.Task)(nil)
	_ a2a.Event             = (*a2a.Message)(nil)
	_ a2a.Event             = (*a2a.Task)(nil)
	_ a2a.Event             = (*a2a.TaskStatusUpdateEvent)(nil)
	_ a2a.Event             = (*a2a.TaskArtifactUpdateEvent)(nil)
)

const conformanceFixtureRoot = "a2a-profile/v0.3.0/conformance/"

func TestA2AProfileConformanceMetadata(t *testing.T) {
	profile, err := LoadA2AProfileV02()
	if err != nil {
		t.Fatalf("load Profile v0.2: %v", err)
	}
	manifest, err := LoadA2AConformanceManifestV02()
	if err != nil {
		t.Fatalf("load conformance manifest: %v", err)
	}

	if profile.SchemaVersion != A2AProfileSchemaVersionV02 {
		t.Fatalf("profile schema version = %q, want %q", profile.SchemaVersion, A2AProfileSchemaVersionV02)
	}
	if profile.ProtocolVersion != A2AProfileProtocolVersion {
		t.Fatalf("protocol version = %q, want %q", profile.ProtocolVersion, A2AProfileProtocolVersion)
	}
	if profile.SDK.Module != A2AProfileSDKModule || profile.SDK.Version != A2AProfileSDKVersion {
		t.Fatalf("SDK pin = %s %s, want %s %s", profile.SDK.Module, profile.SDK.Version, A2AProfileSDKModule, A2AProfileSDKVersion)
	}
	if profile.Conformance.FixtureAuthority != "hand-authored" {
		t.Fatalf("fixture authority = %q, want hand-authored", profile.Conformance.FixtureAuthority)
	}

	wantMethods := map[string]bool{
		"message/send":   false,
		"message/stream": false,
		"tasks/get":      false,
		"tasks/cancel":   false,
	}
	for _, operation := range profile.Operations {
		if _, exists := wantMethods[operation.Method]; !exists {
			t.Fatalf("unexpected operation %q", operation.Method)
		}
		if wantMethods[operation.Method] {
			t.Fatalf("duplicate operation %q", operation.Method)
		}
		wantMethods[operation.Method] = true
	}
	for method, found := range wantMethods {
		if !found {
			t.Fatalf("required operation %q is missing", method)
		}
	}

	for _, states := range [][]A2AProfileTaskStateV02{
		profile.TaskStates.Transient,
		profile.TaskStates.Terminal,
		profile.TaskStates.Unsupported,
		{profile.TaskStates.Unspecified},
	} {
		for _, state := range states {
			if state.State == "timeout" {
				t.Fatal("timeout was declared as an A2A TaskState")
			}
		}
	}

	if manifest.ProfileSchemaVersion != profile.SchemaVersion || manifest.ProtocolVersion != profile.ProtocolVersion {
		t.Fatalf("manifest versions = profile %q protocol %q, want %q and %q", manifest.ProfileSchemaVersion, manifest.ProtocolVersion, profile.SchemaVersion, profile.ProtocolVersion)
	}
	caseIDs := make(map[string]struct{}, len(manifest.Cases))
	for _, testCase := range manifest.Cases {
		if _, exists := caseIDs[testCase.ID]; exists {
			t.Fatalf("duplicate manifest case id %q", testCase.ID)
		}
		caseIDs[testCase.ID] = struct{}{}
		if _, err := fs.Stat(a2aProfileV02Files, conformanceFixtureRoot+testCase.File); err != nil {
			t.Fatalf("case %s fixture %q: %v", testCase.ID, testCase.File, err)
		}
		if testCase.RequestFile != "" {
			if _, err := fs.Stat(a2aProfileV02Files, conformanceFixtureRoot+testCase.RequestFile); err != nil {
				t.Fatalf("case %s request fixture %q: %v", testCase.ID, testCase.RequestFile, err)
			}
		}
	}

	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	schemaDocument, err := readEmbeddedJSONDocument("schemas/a2a-profile.v0.2.schema.json")
	if err != nil {
		t.Fatalf("read Profile Schema v0.2: %v", err)
	}
	if err := compiler.AddResource("https://schemas.nekiro.dev/a2a-profile/v0.2", schemaDocument); err != nil {
		t.Fatalf("add Profile Schema v0.2: %v", err)
	}
	schema, err := compiler.Compile("https://schemas.nekiro.dev/a2a-profile/v0.2")
	if err != nil {
		t.Fatalf("compile Profile Schema v0.2: %v", err)
	}
	profileDocument, err := readEmbeddedJSONDocument("a2a-profile/v0.3.0/profile.v0.2.json")
	if err != nil {
		t.Fatalf("read Profile v0.2: %v", err)
	}
	if err := schema.Validate(profileDocument); err != nil {
		t.Fatalf("Profile v0.2 does not match its schema: %v", err)
	}
}

func TestA2AProfileConformanceTaskStateMapping(t *testing.T) {
	tests := []struct {
		state          a2a.TaskState
		classification A2ATaskStateClassification
		status         string
		errorCode      PlatformErrorCode
	}{
		{a2a.TaskStateSubmitted, A2ATaskStateTransient, "running", ""},
		{a2a.TaskStateWorking, A2ATaskStateTransient, "running", ""},
		{a2a.TaskStateCompleted, A2ATaskStateTerminal, "succeeded", ""},
		{a2a.TaskStateFailed, A2ATaskStateTerminal, "failed", ErrorCodeAgentExecutionFailed},
		{a2a.TaskStateCanceled, A2ATaskStateTerminal, "canceled", ErrorCodeCanceled},
		{a2a.TaskStateRejected, A2ATaskStateTerminal, "failed", ErrorCodeAgentExecutionFailed},
	}
	for _, testCase := range tests {
		t.Run(string(testCase.state), func(t *testing.T) {
			mapping, err := MapA2ATaskState(testCase.state)
			if err != nil {
				t.Fatalf("MapA2ATaskState(%q): %v", testCase.state, err)
			}
			if mapping.Classification != testCase.classification || mapping.InvocationStatus != testCase.status || mapping.ErrorCode != testCase.errorCode {
				t.Fatalf("mapping for %q = %+v", testCase.state, mapping)
			}
		})
	}

	for _, state := range []a2a.TaskState{
		a2a.TaskStateAuthRequired,
		a2a.TaskStateInputRequired,
		a2a.TaskStateUnknown,
		a2a.TaskStateUnspecified,
		a2a.TaskState("paused-by-provider"),
	} {
		t.Run("reject-"+string(state), func(t *testing.T) {
			_, err := MapA2ATaskState(state)
			var stateError *A2AProfileStateError
			if !errors.As(err, &stateError) {
				t.Fatalf("MapA2ATaskState(%q) error = %v, want A2AProfileStateError", state, err)
			}
			if stateError.ErrorCode != ErrorCodeA2AProtocol {
				t.Fatalf("MapA2ATaskState(%q) error code = %q, want %q", state, stateError.ErrorCode, ErrorCodeA2AProtocol)
			}
		})
	}

	for name, task := range map[string]*a2a.Task{
		"nil":             nil,
		"zero":            {},
		"missing context": {ID: "task-1", Status: a2a.TaskStatus{State: a2a.TaskStateWorking}},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := ValidateA2ATask(task)
			var taskError *A2AProfileTaskError
			if !errors.As(err, &taskError) {
				t.Fatalf("ValidateA2ATask() error = %v, want A2AProfileTaskError", err)
			}
			if taskError.ErrorCode != ErrorCodeA2AProtocol {
				t.Fatalf("ValidateA2ATask() error code = %q, want %q", taskError.ErrorCode, ErrorCodeA2AProtocol)
			}
		})
	}
}

func TestA2AProfileConformanceFixtures(t *testing.T) {
	manifest, err := LoadA2AConformanceManifestV02()
	if err != nil {
		t.Fatalf("load conformance manifest: %v", err)
	}
	for _, testCase := range manifest.Cases {
		t.Run(testCase.ID, func(t *testing.T) {
			err := validateManifestCase(t, testCase)
			if testCase.ExpectedValid && err != nil {
				t.Fatalf("valid fixture rejected: %v", err)
			}
			if !testCase.ExpectedValid && err == nil {
				t.Fatal("incompatible fixture was accepted")
			}
		})
	}
}

func TestA2AProfileConformanceClientPaths(t *testing.T) {
	messageParams := mustMessageParamsFixture(t, "message-send-request.json")
	for _, testCase := range []struct {
		name     string
		fixture  string
		wantType any
	}{
		{"message result", "message-send-message-response.json", (*a2a.Message)(nil)},
		{"task result", "message-send-task-response.json", (*a2a.Task)(nil)},
	} {
		t.Run("message-send-"+testCase.name, func(t *testing.T) {
			server := newA2AFixtureServer(t, testCase.fixture, "message/send", nil)
			defer server.Close()
			transport := a2aclient.NewJSONRPCTransport(server.URL, server.Client())
			result, err := transport.SendMessage(t.Context(), messageParams)
			if err != nil {
				t.Fatalf("SendMessage: %v", err)
			}
			if reflect.TypeOf(result) != reflect.TypeOf(testCase.wantType) {
				t.Fatalf("SendMessage result type = %T, want %T", result, testCase.wantType)
			}
		})
	}

	t.Run("message-stream-four-event-kinds", func(t *testing.T) {
		server := newA2AFixtureServer(t, "message-stream-valid.sse", "message/stream", nil)
		defer server.Close()
		transport := a2aclient.NewJSONRPCTransport(server.URL, server.Client())
		params := mustMessageParamsFixture(t, "message-stream-request.json")
		var events []a2a.Event
		for event, err := range transport.SendStreamingMessage(t.Context(), params) {
			if err != nil {
				t.Fatalf("SendStreamingMessage: %v", err)
			}
			events = append(events, event)
		}
		assertFourStreamingEventKinds(t, events)
	})

	t.Run("tasks-get", func(t *testing.T) {
		server := newA2AFixtureServer(t, "tasks-get-response.json", "tasks/get", nil)
		defer server.Close()
		transport := a2aclient.NewJSONRPCTransport(server.URL, server.Client())
		task, err := transport.GetTask(t.Context(), mustTaskQueryFixture(t, "tasks-get-request.json"))
		if err != nil {
			t.Fatalf("GetTask: %v", err)
		}
		if _, err := ValidateA2ATask(task); err != nil {
			t.Fatalf("GetTask result violates Profile v0.2: %v", err)
		}
		if len(task.History) != 1 {
			t.Fatalf("GetTask history length = %d, want 1", len(task.History))
		}
	})

	t.Run("tasks-cancel", func(t *testing.T) {
		server := newA2AFixtureServer(t, "tasks-cancel-response.json", "tasks/cancel", nil)
		defer server.Close()
		transport := a2aclient.NewJSONRPCTransport(server.URL, server.Client())
		params := mustTaskIDFixture(t, "tasks-cancel-request.json")
		task, err := transport.CancelTask(t.Context(), params)
		if err != nil {
			t.Fatalf("CancelTask: %v", err)
		}
		mapping, err := ValidateA2ATask(task)
		if err != nil {
			t.Fatalf("CancelTask result violates Profile v0.2: %v", err)
		}
		if task.ID != params.ID || mapping.State != a2a.TaskStateCanceled {
			t.Fatalf("CancelTask result = task %q state %q, want task %q canceled", task.ID, mapping.State, params.ID)
		}
	})

	for _, testCase := range []struct {
		name      string
		operation string
		fixture   string
		want      error
	}{
		{"get-not-found", "tasks/get", "tasks-get-not-found-response.json", a2a.ErrTaskNotFound},
		{"cancel-not-found", "tasks/cancel", "tasks-cancel-not-found-response.json", a2a.ErrTaskNotFound},
		{"cancel-not-cancelable", "tasks/cancel", "tasks-cancel-not-cancelable-response.json", a2a.ErrTaskNotCancelable},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			server := newA2AFixtureServer(t, testCase.fixture, testCase.operation, nil)
			defer server.Close()
			transport := a2aclient.NewJSONRPCTransport(server.URL, server.Client())
			var err error
			if testCase.operation == "tasks/get" {
				_, err = transport.GetTask(t.Context(), mustTaskQueryFixture(t, "tasks-get-request.json"))
			} else {
				_, err = transport.CancelTask(t.Context(), mustTaskIDFixture(t, "tasks-cancel-request.json"))
			}
			if !errors.Is(err, testCase.want) {
				t.Fatalf("operation error = %v, want %v", err, testCase.want)
			}
		})
	}

	t.Run("five-context-headers", func(t *testing.T) {
		headers := mustContextHeadersFixture(t)
		server := newA2AFixtureServer(t, "tasks-get-response.json", "tasks/get", headers)
		defer server.Close()
		meta := make(a2aclient.CallMeta, len(headers))
		for name, value := range headers {
			meta[name] = []string{value}
		}
		client, err := a2aclient.NewFromEndpoints(t.Context(), []a2a.AgentInterface{
			{URL: server.URL, Transport: a2a.TransportProtocolJSONRPC},
		}, a2aclient.WithInterceptors(a2aclient.NewStaticCallMetaInjector(meta)))
		if err != nil {
			t.Fatalf("create A2A client: %v", err)
		}
		defer func() {
			if err := client.Destroy(); err != nil {
				t.Errorf("destroy A2A client: %v", err)
			}
		}()
		if _, err := client.GetTask(t.Context(), mustTaskQueryFixture(t, "tasks-get-request.json")); err != nil {
			t.Fatalf("GetTask with context headers: %v", err)
		}
	})
}

func TestA2AProfileConformanceServerPaths(t *testing.T) {
	handler := &profileFixtureHandler{
		sendResult: mustSendResultFixture(t, "message-send-message-response.json"),
		stream:     mustStreamEventsFixture(t, "message-stream-valid.sse", "message-stream-request.json"),
		getTask:    mustTaskResultFixture(t, "tasks-get-response.json"),
		cancelTask: mustTaskResultFixture(t, "tasks-cancel-response.json"),
	}
	server := httptest.NewServer(a2asrv.NewJSONRPCHandler(handler))
	defer server.Close()

	t.Run("message-send", func(t *testing.T) {
		body, _ := callA2AServerFixture(t, server.URL, "message-send-request.json", "")
		request := mustFixtureBytes(t, "message-send-request.json")
		envelope, err := validateResponseEnvelope(body, request)
		if err != nil {
			t.Fatalf("a2asrv message/send response: %v", err)
		}
		event, err := a2a.UnmarshalEventJSON(envelope.Result)
		if err != nil {
			t.Fatalf("decode a2asrv message/send result: %v", err)
		}
		if _, ok := event.(*a2a.Message); !ok {
			t.Fatalf("a2asrv message/send result type = %T, want *a2a.Message", event)
		}
		if handler.lastMessage == nil || handler.lastMessage.Message == nil || handler.lastMessage.Message.ID != "message-user-1" {
			t.Fatalf("a2asrv OnSendMessage params = %+v", handler.lastMessage)
		}
	})

	t.Run("message-stream", func(t *testing.T) {
		body, mediaType := callA2AServerFixture(t, server.URL, "message-stream-request.json", "text/event-stream")
		if mediaType != "text/event-stream" {
			t.Fatalf("message/stream Content-Type = %q, want text/event-stream", mediaType)
		}
		events, err := validateSSEStream(body, mustFixtureBytes(t, "message-stream-request.json"))
		if err != nil {
			t.Fatalf("a2asrv message/stream response: %v", err)
		}
		assertFourStreamingEventKinds(t, events)
		if handler.lastStreamMessage == nil || handler.lastStreamMessage.Message == nil || handler.lastStreamMessage.Message.ID != "message-user-stream-1" {
			t.Fatalf("a2asrv OnSendMessageStream params = %+v", handler.lastStreamMessage)
		}
	})

	t.Run("tasks-get", func(t *testing.T) {
		body, _ := callA2AServerFixture(t, server.URL, "tasks-get-request.json", "")
		envelope, err := validateResponseEnvelope(body, mustFixtureBytes(t, "tasks-get-request.json"))
		if err != nil {
			t.Fatalf("a2asrv tasks/get response: %v", err)
		}
		var task a2a.Task
		if err := json.Unmarshal(envelope.Result, &task); err != nil {
			t.Fatalf("decode a2asrv tasks/get result: %v", err)
		}
		if _, err := ValidateA2ATask(&task); err != nil {
			t.Fatalf("a2asrv tasks/get result violates Profile v0.2: %v", err)
		}
		if handler.lastQuery == nil || handler.lastQuery.ID != "task-1" || handler.lastQuery.HistoryLength == nil || *handler.lastQuery.HistoryLength != 1 {
			t.Fatalf("a2asrv OnGetTask params = %+v", handler.lastQuery)
		}
	})

	t.Run("tasks-cancel", func(t *testing.T) {
		body, _ := callA2AServerFixture(t, server.URL, "tasks-cancel-request.json", "")
		envelope, err := validateResponseEnvelope(body, mustFixtureBytes(t, "tasks-cancel-request.json"))
		if err != nil {
			t.Fatalf("a2asrv tasks/cancel response: %v", err)
		}
		var task a2a.Task
		if err := json.Unmarshal(envelope.Result, &task); err != nil {
			t.Fatalf("decode a2asrv tasks/cancel result: %v", err)
		}
		mapping, err := ValidateA2ATask(&task)
		if err != nil || mapping.State != a2a.TaskStateCanceled {
			t.Fatalf("a2asrv tasks/cancel result = %+v, mapping %+v, error %v", task, mapping, err)
		}
		if handler.lastTaskID == nil || handler.lastTaskID.ID != "task-1" {
			t.Fatalf("a2asrv OnCancelTask params = %+v", handler.lastTaskID)
		}
	})

	handler.getErr = a2a.ErrTaskNotFound
	t.Run("tasks-get-not-found", func(t *testing.T) {
		body, _ := callA2AServerFixture(t, server.URL, "tasks-get-request.json", "")
		assertJSONRPCErrorCode(t, body, -32001)
	})
	handler.getErr = nil

	for _, testCase := range []struct {
		name string
		err  error
		code int
	}{
		{"tasks-cancel-not-found", a2a.ErrTaskNotFound, -32001},
		{"tasks-cancel-not-cancelable", a2a.ErrTaskNotCancelable, -32002},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			handler.cancelErr = testCase.err
			body, _ := callA2AServerFixture(t, server.URL, "tasks-cancel-request.json", "")
			assertJSONRPCErrorCode(t, body, testCase.code)
		})
	}
}

type wireEnvelope struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
	Result  json.RawMessage `json:"result"`
	Error   json.RawMessage `json:"error"`
}

type wireError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func validateManifestCase(t *testing.T, testCase A2AConformanceCaseV02) error {
	t.Helper()
	fixture := mustFixtureBytes(t, testCase.File)
	var request []byte
	if testCase.RequestFile != "" {
		request = mustFixtureBytes(t, testCase.RequestFile)
	}

	switch testCase.FixtureKind {
	case "request":
		return validateRequestEnvelope(fixture, testCase.Operation)
	case "response", "error":
		envelope, err := validateResponseEnvelope(fixture, request)
		if err != nil {
			return err
		}
		if testCase.FixtureKind == "error" {
			return validateExpectedWireError(envelope.Error, testCase.ProtocolError)
		}
		return validateOperationResult(testCase.Operation, envelope.Result, request)
	case "stream":
		if testCase.MediaType != "text/event-stream" {
			return fmt.Errorf("stream media type is %q", testCase.MediaType)
		}
		_, err := validateSSEStream(fixture, request)
		return err
	case "headers":
		return validateContextHeaderFixture(fixture)
	default:
		return fmt.Errorf("unknown fixture kind %q", testCase.FixtureKind)
	}
}

func validateRequestEnvelope(data []byte, operation string) error {
	envelope, err := decodeWireEnvelope(data)
	if err != nil {
		return err
	}
	if envelope.JSONRPC != "2.0" {
		return fmt.Errorf("JSON-RPC version = %q", envelope.JSONRPC)
	}
	if len(envelope.ID) == 0 {
		return errors.New("request id is missing")
	}
	if envelope.Method != operation {
		return fmt.Errorf("request method = %q, want %q", envelope.Method, operation)
	}
	if len(envelope.Params) == 0 {
		return errors.New("request params are missing")
	}

	switch operation {
	case "message/send", "message/stream":
		var params a2a.MessageSendParams
		if err := json.Unmarshal(envelope.Params, &params); err != nil {
			return fmt.Errorf("decode MessageSendParams: %w", err)
		}
		if params.Message == nil || params.Message.ID == "" || params.Message.Role != a2a.MessageRoleUser || len(params.Message.Parts) == 0 {
			return errors.New("message/send params do not contain a concrete user message")
		}
	case "tasks/get":
		var params a2a.TaskQueryParams
		if err := json.Unmarshal(envelope.Params, &params); err != nil {
			return fmt.Errorf("decode TaskQueryParams: %w", err)
		}
		if params.ID == "" || params.HistoryLength == nil || *params.HistoryLength != 1 {
			return errors.New("tasks/get params do not contain task id and historyLength=1")
		}
	case "tasks/cancel":
		var params a2a.TaskIDParams
		if err := json.Unmarshal(envelope.Params, &params); err != nil {
			return fmt.Errorf("decode TaskIDParams: %w", err)
		}
		if params.ID == "" {
			return errors.New("tasks/cancel task id is empty")
		}
	default:
		return fmt.Errorf("operation %q is outside Profile v0.2", operation)
	}
	return nil
}

func validateResponseEnvelope(data, requestData []byte) (wireEnvelope, error) {
	envelope, err := decodeWireEnvelope(data)
	if err != nil {
		return wireEnvelope{}, err
	}
	if envelope.JSONRPC != "2.0" {
		return wireEnvelope{}, fmt.Errorf("JSON-RPC version = %q", envelope.JSONRPC)
	}
	if len(envelope.ID) == 0 {
		return wireEnvelope{}, errors.New("response id is missing")
	}
	if len(requestData) > 0 {
		request, err := decodeWireEnvelope(requestData)
		if err != nil {
			return wireEnvelope{}, fmt.Errorf("decode request for response: %w", err)
		}
		equal, err := equalJSON(envelope.ID, request.ID)
		if err != nil {
			return wireEnvelope{}, fmt.Errorf("compare request and response ids: %w", err)
		}
		if !equal {
			return wireEnvelope{}, errors.New("response id does not match request id")
		}
	}
	hasResult := len(envelope.Result) > 0
	hasError := len(envelope.Error) > 0
	if hasResult == hasError {
		return wireEnvelope{}, errors.New("response must contain exactly one of result or error")
	}
	return envelope, nil
}

func validateOperationResult(operation string, result, requestData []byte) error {
	var typedKind struct {
		Kind string `json:"kind"`
	}
	if err := json.Unmarshal(result, &typedKind); err != nil {
		return fmt.Errorf("decode result kind: %w", err)
	}

	switch operation {
	case "message/send":
		if typedKind.Kind != "message" && typedKind.Kind != "task" {
			return fmt.Errorf("message/send result kind = %q", typedKind.Kind)
		}
		event, err := a2a.UnmarshalEventJSON(result)
		if err != nil {
			return err
		}
		if task, ok := event.(*a2a.Task); ok {
			_, err = ValidateA2ATask(task)
		}
		return err
	case "tasks/get", "tasks/cancel":
		if typedKind.Kind != "task" {
			return fmt.Errorf("%s result kind = %q", operation, typedKind.Kind)
		}
		var task a2a.Task
		if err := json.Unmarshal(result, &task); err != nil {
			return fmt.Errorf("decode task result: %w", err)
		}
		mapping, err := ValidateA2ATask(&task)
		if err != nil {
			return err
		}
		request, err := decodeWireEnvelope(requestData)
		if err != nil {
			return err
		}
		if operation == "tasks/get" {
			var params a2a.TaskQueryParams
			if err := json.Unmarshal(request.Params, &params); err != nil {
				return err
			}
			if task.ID != params.ID || params.HistoryLength == nil || len(task.History) != *params.HistoryLength {
				return errors.New("tasks/get result does not preserve task id and requested history length")
			}
			return nil
		}
		var params a2a.TaskIDParams
		if err := json.Unmarshal(request.Params, &params); err != nil {
			return err
		}
		if task.ID != params.ID || mapping.State != a2a.TaskStateCanceled {
			return errors.New("tasks/cancel result is not the same task in canceled state")
		}
		return nil
	default:
		return fmt.Errorf("operation %q has no result profile", operation)
	}
}

func validateExpectedWireError(data json.RawMessage, expected string) error {
	var rpcError wireError
	if err := json.Unmarshal(data, &rpcError); err != nil {
		return fmt.Errorf("decode JSON-RPC error: %w", err)
	}
	wantCode := map[string]int{
		"task-not-found":      -32001,
		"task-not-cancelable": -32002,
	}[expected]
	if wantCode == 0 {
		return fmt.Errorf("unknown expected protocol error %q", expected)
	}
	if rpcError.Code != wantCode || rpcError.Message == "" {
		return fmt.Errorf("JSON-RPC error = %d %q, want code %d", rpcError.Code, rpcError.Message, wantCode)
	}
	return nil
}

func validateSSEStream(data, requestData []byte) ([]a2a.Event, error) {
	request, err := decodeWireEnvelope(requestData)
	if err != nil {
		return nil, fmt.Errorf("decode stream request: %w", err)
	}
	blocks, err := parseSSEBlocks(data)
	if err != nil {
		return nil, err
	}

	var events []a2a.Event
	var taskID a2a.TaskID
	var contextID string
	terminal := false
	type artifactState struct {
		finished bool
	}
	artifacts := make(map[a2a.ArtifactID]artifactState)

	for index, block := range blocks {
		if terminal {
			return nil, fmt.Errorf("event %d arrived after terminal", index)
		}
		envelope, err := validateResponseEnvelope(block, requestData)
		if err != nil {
			return nil, fmt.Errorf("stream event %d envelope: %w", index, err)
		}
		equal, err := equalJSON(envelope.ID, request.ID)
		if err != nil || !equal {
			return nil, fmt.Errorf("stream event %d response id mismatch", index)
		}
		if len(envelope.Error) > 0 {
			return nil, fmt.Errorf("stream event %d is an error", index)
		}
		event, err := a2a.UnmarshalEventJSON(envelope.Result)
		if err != nil {
			return nil, fmt.Errorf("stream event %d: %w", index, err)
		}
		info := event.TaskInfo()
		if info.TaskID == "" || info.ContextID == "" {
			return nil, fmt.Errorf("stream event %d has empty task or context id", index)
		}
		if index == 0 {
			taskID = info.TaskID
			contextID = info.ContextID
		} else if info.TaskID != taskID || info.ContextID != contextID {
			return nil, fmt.Errorf("stream event %d changed task/context identity", index)
		}

		switch typed := event.(type) {
		case *a2a.Task:
			mapping, err := ValidateA2ATask(typed)
			if err != nil {
				return nil, fmt.Errorf("stream event %d task: %w", index, err)
			}
			terminal = mapping.Classification == A2ATaskStateTerminal
		case *a2a.Message:
			if typed.ID == "" || typed.Role != a2a.MessageRoleAgent || len(typed.Parts) == 0 {
				return nil, fmt.Errorf("stream event %d is not a concrete Agent message", index)
			}
		case *a2a.TaskStatusUpdateEvent:
			mapping, err := MapA2ATaskState(typed.Status.State)
			if err != nil {
				return nil, fmt.Errorf("stream event %d status: %w", index, err)
			}
			isTerminalState := mapping.Classification == A2ATaskStateTerminal
			if typed.Final != isTerminalState {
				return nil, fmt.Errorf("stream event %d final flag contradicts state %q", index, typed.Status.State)
			}
			terminal = typed.Final
		case *a2a.TaskArtifactUpdateEvent:
			if typed.Artifact == nil || typed.Artifact.ID == "" || len(typed.Artifact.Parts) == 0 {
				return nil, fmt.Errorf("stream event %d has an incomplete artifact", index)
			}
			state, seen := artifacts[typed.Artifact.ID]
			if state.finished {
				return nil, fmt.Errorf("stream event %d updated artifact %q after lastChunk", index, typed.Artifact.ID)
			}
			if typed.Append && !seen {
				return nil, fmt.Errorf("stream event %d appends artifact %q before its base", index, typed.Artifact.ID)
			}
			if !typed.Append && seen {
				return nil, fmt.Errorf("stream event %d replaces existing artifact %q", index, typed.Artifact.ID)
			}
			artifacts[typed.Artifact.ID] = artifactState{finished: typed.LastChunk}
		default:
			return nil, fmt.Errorf("stream event %d type %T is outside Profile v0.2", index, event)
		}
		events = append(events, event)
	}

	if !terminal {
		return nil, errors.New("stream reached EOF without terminal event")
	}
	for artifactID, state := range artifacts {
		if !state.finished {
			return nil, fmt.Errorf("artifact %q did not receive lastChunk", artifactID)
		}
	}
	return events, nil
}

func parseSSEBlocks(data []byte) ([][]byte, error) {
	normalized := strings.ReplaceAll(string(data), "\r\n", "\n")
	normalized = strings.TrimSuffix(normalized, ": end-of-stream\n")
	if !strings.HasSuffix(normalized, "\n\n") {
		return nil, errors.New("SSE stream does not end with a blank line")
	}
	normalized = strings.TrimSuffix(normalized, "\n\n")
	if normalized == "" {
		return nil, errors.New("SSE stream is empty")
	}
	rawBlocks := strings.Split(normalized, "\n\n")
	blocks := make([][]byte, 0, len(rawBlocks))
	eventIDs := make(map[string]struct{}, len(rawBlocks))
	for index, rawBlock := range rawBlocks {
		lines := strings.Split(rawBlock, "\n")
		if len(lines) != 2 || !strings.HasPrefix(lines[0], "id: ") || !strings.HasPrefix(lines[1], "data: ") {
			return nil, fmt.Errorf("SSE block %d is not one id line and one data line", index)
		}
		eventID := strings.TrimPrefix(lines[0], "id: ")
		if eventID == "" {
			return nil, fmt.Errorf("SSE block %d has empty id", index)
		}
		if _, exists := eventIDs[eventID]; exists {
			return nil, fmt.Errorf("SSE block %d repeats id %q", index, eventID)
		}
		eventIDs[eventID] = struct{}{}
		blocks = append(blocks, []byte(strings.TrimPrefix(lines[1], "data: ")))
	}
	return blocks, nil
}

func validateContextHeaderFixture(data []byte) error {
	var headers map[string]string
	if err := json.Unmarshal(data, &headers); err != nil {
		return fmt.Errorf("decode context headers: %w", err)
	}
	profile, err := LoadA2AProfileV02()
	if err != nil {
		return err
	}
	wantNames := []string{
		profile.ContextHeaders.TraceID,
		profile.ContextHeaders.InvocationID,
		profile.ContextHeaders.RootTaskID,
		profile.ContextHeaders.ParentInvocationID,
		profile.ContextHeaders.WorkspaceID,
	}
	if len(headers) != len(wantNames) {
		return fmt.Errorf("context header count = %d, want %d", len(headers), len(wantNames))
	}
	for _, name := range wantNames {
		if headers[name] == "" {
			return fmt.Errorf("context header %q is missing or empty", name)
		}
	}
	return nil
}

func decodeWireEnvelope(data []byte) (wireEnvelope, error) {
	var envelope wireEnvelope
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&envelope); err != nil {
		return wireEnvelope{}, fmt.Errorf("decode JSON-RPC envelope: %w", err)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return wireEnvelope{}, fmt.Errorf("decode JSON-RPC envelope: %w", err)
	}
	return envelope, nil
}

func equalJSON(left, right []byte) (bool, error) {
	var leftValue any
	if err := json.Unmarshal(left, &leftValue); err != nil {
		return false, err
	}
	var rightValue any
	if err := json.Unmarshal(right, &rightValue); err != nil {
		return false, err
	}
	return reflect.DeepEqual(leftValue, rightValue), nil
}

func readEmbeddedJSONDocument(path string) (any, error) {
	data, err := a2aProfileV02Files.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return jsonschema.UnmarshalJSON(bytes.NewReader(data))
}

func mustFixtureBytes(t *testing.T, file string) []byte {
	t.Helper()
	data, err := a2aProfileV02Files.ReadFile(conformanceFixtureRoot + file)
	if err != nil {
		t.Fatalf("read fixture %s: %v", file, err)
	}
	return data
}

func mustMessageParamsFixture(t *testing.T, file string) *a2a.MessageSendParams {
	t.Helper()
	envelope, err := decodeWireEnvelope(mustFixtureBytes(t, file))
	if err != nil {
		t.Fatalf("decode %s: %v", file, err)
	}
	var params a2a.MessageSendParams
	if err := json.Unmarshal(envelope.Params, &params); err != nil {
		t.Fatalf("decode MessageSendParams from %s: %v", file, err)
	}
	return &params
}

func mustTaskQueryFixture(t *testing.T, file string) *a2a.TaskQueryParams {
	t.Helper()
	envelope, err := decodeWireEnvelope(mustFixtureBytes(t, file))
	if err != nil {
		t.Fatalf("decode %s: %v", file, err)
	}
	var params a2a.TaskQueryParams
	if err := json.Unmarshal(envelope.Params, &params); err != nil {
		t.Fatalf("decode TaskQueryParams from %s: %v", file, err)
	}
	return &params
}

func mustTaskIDFixture(t *testing.T, file string) *a2a.TaskIDParams {
	t.Helper()
	envelope, err := decodeWireEnvelope(mustFixtureBytes(t, file))
	if err != nil {
		t.Fatalf("decode %s: %v", file, err)
	}
	var params a2a.TaskIDParams
	if err := json.Unmarshal(envelope.Params, &params); err != nil {
		t.Fatalf("decode TaskIDParams from %s: %v", file, err)
	}
	return &params
}

func mustTaskResultFixture(t *testing.T, file string) *a2a.Task {
	t.Helper()
	envelope, err := decodeWireEnvelope(mustFixtureBytes(t, file))
	if err != nil {
		t.Fatalf("decode %s: %v", file, err)
	}
	var task a2a.Task
	if err := json.Unmarshal(envelope.Result, &task); err != nil {
		t.Fatalf("decode task result from %s: %v", file, err)
	}
	return &task
}

func mustSendResultFixture(t *testing.T, file string) a2a.SendMessageResult {
	t.Helper()
	envelope, err := decodeWireEnvelope(mustFixtureBytes(t, file))
	if err != nil {
		t.Fatalf("decode %s: %v", file, err)
	}
	event, err := a2a.UnmarshalEventJSON(envelope.Result)
	if err != nil {
		t.Fatalf("decode send result from %s: %v", file, err)
	}
	result, ok := event.(a2a.SendMessageResult)
	if !ok {
		t.Fatalf("fixture %s type %T is not SendMessageResult", file, event)
	}
	return result
}

func mustStreamEventsFixture(t *testing.T, streamFile, requestFile string) []a2a.Event {
	t.Helper()
	events, err := validateSSEStream(mustFixtureBytes(t, streamFile), mustFixtureBytes(t, requestFile))
	if err != nil {
		t.Fatalf("decode stream fixture %s: %v", streamFile, err)
	}
	return events
}

func mustContextHeadersFixture(t *testing.T) map[string]string {
	t.Helper()
	data := mustFixtureBytes(t, "context-headers.json")
	if err := validateContextHeaderFixture(data); err != nil {
		t.Fatalf("validate context headers fixture: %v", err)
	}
	var headers map[string]string
	if err := json.Unmarshal(data, &headers); err != nil {
		t.Fatalf("decode context headers fixture: %v", err)
	}
	return headers
}

func newA2AFixtureServer(t *testing.T, fixture, operation string, expectedHeaders map[string]string) *httptest.Server {
	t.Helper()
	body := mustFixtureBytes(t, fixture)
	return httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		requestBody, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("read %s request: %v", operation, err)
			return
		}
		if err := validateRequestEnvelope(requestBody, operation); err != nil {
			t.Errorf("%s client request violates Profile v0.2: %v", operation, err)
			return
		}
		for name, value := range expectedHeaders {
			if request.Header.Get(name) != value {
				t.Errorf("%s header %s = %q, want %q", operation, name, request.Header.Get(name), value)
			}
		}
		if strings.HasSuffix(fixture, ".sse") {
			if request.Header.Get("Accept") != "text/event-stream" {
				t.Errorf("%s Accept = %q, want text/event-stream", operation, request.Header.Get("Accept"))
			}
			response.Header().Set("Content-Type", "text/event-stream")
		} else {
			response.Header().Set("Content-Type", "application/json")
		}
		_, _ = response.Write(body)
	}))
}

func assertFourStreamingEventKinds(t *testing.T, events []a2a.Event) {
	t.Helper()
	kinds := map[string]int{
		"message":         0,
		"task":            0,
		"status-update":   0,
		"artifact-update": 0,
	}
	for _, event := range events {
		switch event.(type) {
		case *a2a.Message:
			kinds["message"]++
		case *a2a.Task:
			kinds["task"]++
		case *a2a.TaskStatusUpdateEvent:
			kinds["status-update"]++
		case *a2a.TaskArtifactUpdateEvent:
			kinds["artifact-update"]++
		default:
			t.Fatalf("unexpected stream event type %T", event)
		}
	}
	for kind, count := range kinds {
		if count == 0 {
			t.Fatalf("stream contains no %s event", kind)
		}
	}
}

type profileFixtureHandler struct {
	a2asrv.RequestHandler

	sendResult a2a.SendMessageResult
	stream     []a2a.Event
	getTask    *a2a.Task
	cancelTask *a2a.Task
	getErr     error
	cancelErr  error

	lastMessage       *a2a.MessageSendParams
	lastStreamMessage *a2a.MessageSendParams
	lastQuery         *a2a.TaskQueryParams
	lastTaskID        *a2a.TaskIDParams
}

var _ a2asrv.RequestHandler = (*profileFixtureHandler)(nil)

func (h *profileFixtureHandler) OnSendMessage(_ context.Context, params *a2a.MessageSendParams) (a2a.SendMessageResult, error) {
	h.lastMessage = params
	return h.sendResult, nil
}

func (h *profileFixtureHandler) OnSendMessageStream(_ context.Context, params *a2a.MessageSendParams) iter.Seq2[a2a.Event, error] {
	h.lastStreamMessage = params
	return func(yield func(a2a.Event, error) bool) {
		for _, event := range h.stream {
			if !yield(event, nil) {
				return
			}
		}
	}
}

func (h *profileFixtureHandler) OnGetTask(_ context.Context, params *a2a.TaskQueryParams) (*a2a.Task, error) {
	h.lastQuery = params
	if h.getErr != nil {
		return nil, h.getErr
	}
	return h.getTask, nil
}

func (h *profileFixtureHandler) OnCancelTask(_ context.Context, params *a2a.TaskIDParams) (*a2a.Task, error) {
	h.lastTaskID = params
	if h.cancelErr != nil {
		return nil, h.cancelErr
	}
	return h.cancelTask, nil
}

func callA2AServerFixture(t *testing.T, serverURL, fixture, accept string) ([]byte, string) {
	t.Helper()
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, serverURL, bytes.NewReader(mustFixtureBytes(t, fixture)))
	if err != nil {
		t.Fatalf("create A2A server request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	if accept != "" {
		request.Header.Set("Accept", accept)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("call A2A server: %v", err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close A2A server response: %v", err)
		}
	}()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read A2A server response: %v", err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("A2A server status = %s, body = %s", response.Status, body)
	}
	mediaType := response.Header.Get("Content-Type")
	if separator := strings.IndexByte(mediaType, ';'); separator >= 0 {
		mediaType = mediaType[:separator]
	}
	return body, mediaType
}

func assertJSONRPCErrorCode(t *testing.T, data []byte, want int) {
	t.Helper()
	envelope, err := decodeWireEnvelope(data)
	if err != nil {
		t.Fatalf("decode JSON-RPC error response: %v", err)
	}
	if len(envelope.Result) > 0 || len(envelope.Error) == 0 {
		t.Fatalf("JSON-RPC error response has result=%s error=%s", envelope.Result, envelope.Error)
	}
	var rpcError wireError
	if err := json.Unmarshal(envelope.Error, &rpcError); err != nil {
		t.Fatalf("decode JSON-RPC error: %v", err)
	}
	if rpcError.Code != want {
		t.Fatalf("JSON-RPC error code = %d, want %d", rpcError.Code, want)
	}
}
