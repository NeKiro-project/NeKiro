package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Nene7ko/NeKiro/apps/a2a-router/internal/ledger"
	"github.com/Nene7ko/NeKiro/contracts"
)

type fakeLedgerReader struct {
	detail contracts.InvocationDetailResponseV4
	trace  contracts.TraceResponseV4
	err    error
}

func (reader fakeLedgerReader) GetInvocation(context.Context, string, string) (contracts.InvocationDetailResponseV4, error) {
	return reader.detail, reader.err
}

func (reader fakeLedgerReader) GetTrace(context.Context, string, contracts.TraceID) (contracts.TraceResponseV4, error) {
	return reader.trace, reader.err
}

func TestLedgerHandlerMapsContractReadsAndFailures(t *testing.T) {
	detail := handlerDetail(t)
	handler, err := NewLedgerHandler(fakeLedgerReader{detail: detail, trace: contracts.TraceResponseV4{
		TraceID: detail.Invocation.TraceID, Invocations: []contracts.InvocationRecordV4{detail.Invocation},
	}})
	if err != nil {
		t.Fatalf("construct Ledger handler: %v", err)
	}
	request := httptest.NewRequest(http.MethodGet, "/internal/v3/read", nil)
	response := httptest.NewRecorder()
	if err := handler.ServeInvocationRead(response, request, "workspace-a", "inv-handler", "trace-request"); err != nil {
		t.Fatalf("serve Invocation read: %v", err)
	}
	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("Invocation response status/header = %d/%q", response.Code, response.Header().Get("Content-Type"))
	}
	var decoded contracts.InvocationDetailResponseV4
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil || decoded.Invocation.InvocationID != "inv-handler" {
		t.Fatalf("decode Invocation response = %#v, %v", decoded, err)
	}

	for _, test := range []struct {
		name   string
		err    error
		status int
		code   contracts.PlatformErrorCode
	}{
		{name: "not found", err: ledger.ErrNotFound, status: http.StatusNotFound, code: contracts.ErrorCodeNotFound},
		{name: "dependency", err: ledger.ErrDependency, status: http.StatusServiceUnavailable, code: contracts.ErrorCodeDependency},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler, err := NewLedgerHandler(fakeLedgerReader{err: test.err})
			if err != nil {
				t.Fatalf("construct failure handler: %v", err)
			}
			response := httptest.NewRecorder()
			if err := handler.ServeTraceRead(response, request, "workspace-a", "trace-request"); err != nil {
				t.Fatalf("serve failure response: %v", err)
			}
			var platformError contracts.PreCorrelationPlatformErrorV4
			if err := json.Unmarshal(response.Body.Bytes(), &platformError); err != nil {
				t.Fatalf("decode failure response: %v", err)
			}
			if response.Code != test.status || platformError.Code != test.code || platformError.TraceID != "trace-request" {
				t.Fatalf("failure response = %d %#v", response.Code, platformError)
			}
		})
	}
}

func TestLedgerHandlerRejectsInvalidStoredContractAsDependencyFailure(t *testing.T) {
	invalid := handlerDetail(t)
	invalid.Invocation.WorkspaceID = "workspace-other"
	handler, err := NewLedgerHandler(fakeLedgerReader{detail: invalid})
	if err != nil {
		t.Fatalf("construct invalid-contract handler: %v", err)
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/internal/v3/read", nil)
	if err := handler.ServeInvocationRead(response, request, "workspace-a", "inv-handler", "trace-request"); err != nil {
		t.Fatalf("serve invalid stored contract: %v", err)
	}
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("invalid stored contract status = %d", response.Code)
	}
}

func TestNewLedgerHandlerRequiresReader(t *testing.T) {
	if _, err := NewLedgerHandler(nil); err == nil {
		t.Fatal("nil Ledger reader was accepted")
	}
	if !errors.Is(ledger.ErrDependency, ledger.ErrDependency) {
		t.Fatal("dependency sentinel is not stable")
	}
}

func handlerDetail(t *testing.T) contracts.InvocationDetailResponseV4 {
	t.Helper()
	at := time.Date(2026, 7, 16, 12, 0, 0, 0, time.UTC)
	event := contracts.InvocationEventV03{
		SchemaVersion: contracts.RuntimeInvocationEventSchemaVersion,
		EventID:       "event-handler", Sequence: 0, OccurredAt: at.Format(time.RFC3339Nano),
		Type: "created", Status: "pending", InvocationID: "inv-handler", RootTaskID: "task-handler",
		TraceID: "trace-handler", Caller: contracts.Caller{Type: "user", ID: "user-a"}, WorkspaceID: "workspace-a",
		TargetAgentID: "agent-a", AgentCardVersion: "1.0.0", Capability: "document.read",
	}
	return contracts.InvocationDetailResponseV4{
		Invocation: contracts.InvocationRecordV4{
			InvocationID: event.InvocationID, RootTaskID: event.RootTaskID, TraceID: event.TraceID,
			Caller: event.Caller, WorkspaceID: event.WorkspaceID, TargetAgentID: event.TargetAgentID,
			AgentCardVersion: event.AgentCardVersion, Capability: event.Capability, Status: event.Status,
			CreatedAt: at, UpdatedAt: at,
		},
		Events: []contracts.InvocationEventV03{event},
	}
}
