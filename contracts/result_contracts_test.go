package contracts

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestInvocationResultAndChunksPreserveArbitraryJSONValues(t *testing.T) {
	validator := mustResultContractValidator(t)
	testCases := []struct {
		name  string
		value string
	}{
		{name: "null", value: `null`},
		{name: "false", value: `false`},
		{name: "zero", value: `0`},
		{name: "string", value: `"answer"`},
		{name: "array", value: `[1,false,null,{"nested":"value"}]`},
		{name: "object", value: `{"answer":42,"items":["a","b"]}`},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := InvocationResult{
				SchemaVersion: InvocationResultSchemaVersion,
				InvocationID:  "inv-result",
				RootTaskID:    "task-result",
				TraceID:       "trace-result",
				Status:        "succeeded",
				Result:        json.RawMessage(testCase.value),
			}
			if err := validator.ValidateInvocationResult(result); err != nil {
				t.Fatalf("validate Invocation Result: %v", err)
			}
			assertRawJSONFieldEquals(t, result, "result", testCase.value)

			sequence := mustResultStreamSequenceValidator(t, validator, "inv-stream", "task-stream", "trace-stream")
			if err := sequence.Accept(resultStreamEvent(ResultStreamEventAccepted, 0)); err != nil {
				t.Fatalf("accept stream start: %v", err)
			}
			chunkIndex := int64(0)
			chunk := resultStreamEvent(ResultStreamEventChunk, 1)
			chunk.ChunkIndex = &chunkIndex
			chunk.Chunk = json.RawMessage(testCase.value)
			if err := sequence.Accept(chunk); err != nil {
				t.Fatalf("accept result chunk: %v", err)
			}
			assertRawJSONFieldEquals(t, chunk, "chunk", testCase.value)
			if err := sequence.Accept(resultStreamEvent(ResultStreamEventCompleted, 2)); err != nil {
				t.Fatalf("accept stream completion: %v", err)
			}
			if err := sequence.Finish(); err != nil {
				t.Fatalf("finish complete stream: %v", err)
			}
		})
	}
}

func TestResultStreamFirstTerminalWins(t *testing.T) {
	validator := mustResultContractValidator(t)
	sequence := mustResultStreamSequenceValidator(t, validator, "inv-stream", "task-stream", "trace-stream")
	if err := sequence.Accept(resultStreamEvent(ResultStreamEventAccepted, 0)); err != nil {
		t.Fatalf("accept stream start: %v", err)
	}
	if err := sequence.Accept(resultStreamEvent(ResultStreamEventCompleted, 1)); err != nil {
		t.Fatalf("accept first terminal event: %v", err)
	}

	timeoutError, err := NewCorrelatedPlatformErrorV2(ErrorCodeTimeout, "trace-stream", "inv-stream", "task-stream")
	if err != nil {
		t.Fatalf("create timeout error: %v", err)
	}
	lateTimeout := resultStreamEvent(ResultStreamEventTimedOut, 2)
	lateTimeout.Error = &timeoutError
	if err := sequence.Accept(lateTimeout); !errors.Is(err, ErrResultStreamTerminated) {
		t.Fatalf("late terminal error = %v, want ErrResultStreamTerminated", err)
	}
	if sequence.TerminalType() != ResultStreamEventCompleted {
		t.Fatalf("terminal type = %q, want completed", sequence.TerminalType())
	}
	if err := sequence.Finish(); err != nil {
		t.Fatalf("finish first-terminal-wins stream: %v", err)
	}
}

func TestResultStreamInterruptedAndOutOfOrder(t *testing.T) {
	validator := mustResultContractValidator(t)
	sequence := mustResultStreamSequenceValidator(t, validator, "inv-stream", "task-stream", "trace-stream")
	if err := sequence.Accept(resultStreamEvent(ResultStreamEventAccepted, 0)); err != nil {
		t.Fatalf("accept stream start: %v", err)
	}
	chunkIndex := int64(0)
	chunk := resultStreamEvent(ResultStreamEventChunk, 1)
	chunk.ChunkIndex = &chunkIndex
	chunk.Chunk = json.RawMessage(`{"partial":true}`)
	if err := sequence.Accept(chunk); err != nil {
		t.Fatalf("accept partial chunk: %v", err)
	}
	if err := sequence.Finish(); !errors.Is(err, ErrResultStreamInterrupted) {
		t.Fatalf("interrupted finish error = %v, want ErrResultStreamInterrupted", err)
	}

	outOfOrder := resultStreamEvent(ResultStreamEventCompleted, 3)
	if err := sequence.Accept(outOfOrder); err == nil || !strings.Contains(err.Error(), "sequence must be 2") {
		t.Fatalf("out-of-order error = %v, want sequence rejection", err)
	}
}

func TestPlatformErrorV2UsesFixedSecretSafeMessages(t *testing.T) {
	validator := mustResultContractValidator(t)
	for code, message := range platformErrorV2Messages {
		platformError, err := NewPlatformErrorV2(code, "trace-error")
		if err != nil {
			t.Fatalf("create %s error: %v", code, err)
		}
		if platformError.Message != message {
			t.Fatalf("message for %s = %q, want %q", code, platformError.Message, message)
		}
		if err := validator.ValidatePlatformError(platformError); err != nil {
			t.Fatalf("validate %s error: %v", code, err)
		}
	}

	correlated, err := NewCorrelatedPlatformErrorV2(
		ErrorCodeAgentExecutionFailed,
		"trace-error",
		"inv-error",
		"task-error",
	)
	if err != nil {
		t.Fatalf("create correlated error: %v", err)
	}
	if err := validator.ValidatePlatformError(correlated); err != nil {
		t.Fatalf("validate correlated error: %v", err)
	}
	encoded, err := json.Marshal(correlated)
	if err != nil {
		t.Fatalf("marshal correlated error: %v", err)
	}
	for _, forbidden := range []string{"secret", "token", "endpoint", "details", "stack", "result", "input"} {
		if strings.Contains(strings.ToLower(string(encoded)), forbidden) {
			t.Fatalf("public error contains forbidden content %q: %s", forbidden, encoded)
		}
	}

	if _, err := NewPlatformErrorV2(PlatformErrorCode("UNKNOWN"), "trace-error"); err == nil {
		t.Fatal("unknown Platform Error v2 code was accepted")
	}
	if _, err := NewCorrelatedPlatformErrorV2(ErrorCodeInternal, "trace-error", "token=secret", "task-error"); err == nil {
		t.Fatal("unsafe invocation correlation was accepted")
	}

	invalidDocuments := []string{
		`{"code":"INTERNAL_ERROR","message":"raw database failure","traceId":"trace-error"}`,
		`{"code":"INTERNAL_ERROR","message":"The platform could not complete the request.","traceId":"trace-error","details":"token=secret"}`,
		`{"code":"INTERNAL_ERROR","message":"The platform could not complete the request.","traceId":"trace-error","invocationId":"inv-error"}`,
	}
	for _, document := range invalidDocuments {
		if err := validateResultJSONBytes(validator.platformError, []byte(document)); err == nil {
			t.Fatalf("invalid public error was accepted: %s", document)
		}
	}
}

func TestInvocationEventV02ContainsNoResultContent(t *testing.T) {
	validator := mustResultContractValidator(t)
	event := validInvocationEventV02("started", "running", nil)
	if err := validator.ValidateInvocationEvent(event); err != nil {
		t.Fatalf("validate metadata-only event: %v", err)
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal event: %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("decode event: %v", err)
	}
	for _, forbidden := range []string{"input", "result", "chunk", "output"} {
		if _, exists := document[forbidden]; exists {
			t.Fatalf("Invocation Event contains forbidden field %q", forbidden)
		}
	}

	document["result"] = map[string]any{"secret": true}
	withResult, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("marshal event with result: %v", err)
	}
	if err := validateResultJSONBytes(validator.invocationEvent, withResult); err == nil {
		t.Fatal("Invocation Event v0.2 accepted persisted result content")
	}
}

func assertRawJSONFieldEquals(t *testing.T, value any, field string, expected string) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal mapped value: %v", err)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("decode mapped value: %v", err)
	}
	var actualValue any
	if err := json.Unmarshal(document[field], &actualValue); err != nil {
		t.Fatalf("decode %s: %v", field, err)
	}
	var expectedValue any
	if err := json.Unmarshal([]byte(expected), &expectedValue); err != nil {
		t.Fatalf("decode expected JSON: %v", err)
	}
	if !reflect.DeepEqual(actualValue, expectedValue) {
		t.Fatalf("%s = %#v, want %#v", field, actualValue, expectedValue)
	}
}

func validateResultJSONBytes(schema interface{ Validate(any) error }, data []byte) error {
	var document any
	if err := json.Unmarshal(data, &document); err != nil {
		return err
	}
	return schema.Validate(document)
}

func mustResultContractValidator(t *testing.T) *ResultContractValidator {
	t.Helper()
	validator, err := NewResultContractValidator()
	if err != nil {
		t.Fatalf("create result contract validator: %v", err)
	}
	return validator
}

func mustResultStreamSequenceValidator(
	t *testing.T,
	validator *ResultContractValidator,
	invocationID string,
	rootTaskID string,
	traceID TraceID,
) *ResultStreamSequenceValidator {
	t.Helper()
	sequence, err := NewResultStreamSequenceValidator(validator, invocationID, rootTaskID, traceID)
	if err != nil {
		t.Fatalf("create result stream sequence validator: %v", err)
	}
	return sequence
}

func resultStreamEvent(eventType ResultStreamEventType, sequence int64) InvocationResultStreamEvent {
	return InvocationResultStreamEvent{
		SchemaVersion: InvocationResultStreamEventSchemaVersion,
		Sequence:      sequence,
		Type:          eventType,
		Status:        resultStreamStatus(eventType),
		InvocationID:  "inv-stream",
		RootTaskID:    "task-stream",
		TraceID:       "trace-stream",
	}
}

func resultStreamStatus(eventType ResultStreamEventType) string {
	switch eventType {
	case ResultStreamEventAccepted:
		return "pending"
	case ResultStreamEventChunk:
		return "running"
	case ResultStreamEventCompleted:
		return "succeeded"
	case ResultStreamEventFailed:
		return "failed"
	case ResultStreamEventCanceled:
		return "canceled"
	case ResultStreamEventTimedOut:
		return "timed_out"
	default:
		panic("unsupported result stream event type in test")
	}
}

func validInvocationEventV02(eventType string, status string, platformError *PlatformErrorV2) InvocationEventV02 {
	latency := int64(12)
	event := InvocationEventV02{
		SchemaVersion:    InvocationEventV02SchemaVersion,
		EventID:          "event-v02",
		Sequence:         3,
		OccurredAt:       time.Date(2026, time.July, 13, 8, 0, 0, 0, time.UTC),
		Type:             eventType,
		Status:           status,
		InvocationID:     "inv-event",
		RootTaskID:       "task-event",
		TraceID:          "trace-event",
		Caller:           Caller{Type: "user", ID: "user-event"},
		WorkspaceID:      "workspace-event",
		TargetAgentID:    "agent-event",
		AgentCardVersion: "1.0.0",
		Capability:       "answer",
		Error:            platformError,
	}
	if eventType == "succeeded" || eventType == "failed" || eventType == "canceled" || eventType == "timed_out" {
		event.LatencyMS = &latency
	}
	return event
}
