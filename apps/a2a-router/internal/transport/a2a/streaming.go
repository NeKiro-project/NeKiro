package a2a

import (
	"context"
	"encoding/json"
	"errors"
	"iter"
	"time"

	streammodel "github.com/NeKiro-project/NeKiro/apps/a2a-router/internal/stream"
	"github.com/NeKiro-project/NeKiro/contracts"
	a2atransport "github.com/NeKiro-project/nekiro-a2a-transport-go"
	a2ago "github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2aclient"
)

const streamCancelAttemptTimeout = time.Second

func (client *Client) SendStreaming(ctx context.Context, dispatch contracts.DispatchInvocationRequestV4, resolved contracts.ResolveAgentResponse) iter.Seq2[streammodel.Event, error] {
	return func(yield func(streammodel.Event, error) bool) {
		target, err := NewTarget(resolved, dispatch.Capability)
		if err != nil {
			yield(streammodel.Event{}, err)
			return
		}
		contextHeaders := ContextHeaders{TraceID: dispatch.TraceID, InvocationID: dispatch.InvocationID, RootTaskID: dispatch.RootTaskID, ParentInvocationID: dispatch.ParentInvocationID, WorkspaceID: dispatch.WorkspaceID}
		target, err = client.selectTarget(ctx, target, contextHeaders)
		if err != nil {
			yield(streammodel.Event{}, err)
			return
		}
		if !target.Streaming {
			yield(streammodel.Event{}, classify(contracts.ErrorCodeRouteNotFound, errors.New("resolved Agent Card does not enable streaming")))
			return
		}
		inputLimit := client.inputLimitBytes
		if target.MaxInputBytes < inputLimit {
			inputLimit = target.MaxInputBytes
		}
		if int64(len(dispatch.Input)) > inputLimit {
			yield(streammodel.Event{}, classify(contracts.ErrorCodePayloadTooLarge, errors.New("dispatch input exceeds the resolved Agent limit")))
			return
		}
		params, err := messageSendParams(dispatch)
		if err != nil {
			yield(streammodel.Event{}, classify(contracts.ErrorCodeA2AProtocol, err))
			return
		}

		eventLimit := client.a2aEventLimitBytes
		if target.MaxOutputBytes < eventLimit {
			eventLimit = target.MaxOutputBytes
		}
		responseLimit := client.responseLimitBytes
		if target.MaxOutputBytes < responseLimit {
			responseLimit = target.MaxOutputBytes
		}
		invocationContext := credentialContext(target, contextHeaders)
		interceptors := []a2aclient.CallInterceptor{newCredentialInterceptor(client.credentialIssuer, invocationContext)}
		callOptions := a2atransport.CallOptions{Endpoint: target.Endpoint, MaxResponseBytes: responseLimit, MaxEventBytes: eventLimit, Interceptors: interceptors}

		var taskID, contextID string
		artifactLast := make(map[string]bool)
		artifactSeen := make(map[string]bool)
		terminalSeen := false
		var pendingTerminal *streammodel.Event
		defer func() {
			if taskID == "" || ctx.Err() == nil {
				return
			}
			// A local deadline/disconnect may leave a known remote task running.
			// Make one bounded cancellation attempt; never retry or reroute.
			cancelCtx, cancel := context.WithTimeout(context.Background(), streamCancelAttemptTimeout)
			defer cancel()
			// Cancellation failure cannot replace the already committed local
			// timeout/cancel outcome and is intentionally not retried or promoted.
			_, _ = client.transportClient.CancelTask(cancelCtx, a2atransport.CallOptions{
				Endpoint: target.Endpoint, MaxResponseBytes: responseLimit, Interceptors: interceptors,
			}, a2ago.TaskID(taskID))
		}()
		for item, eventErr := range client.transportClient.SendStreamingMessage(ctx, callOptions, params) {
			if eventErr != nil {
				yield(streammodel.Event{}, classifyTransportError(eventErr))
				return
			}
			mapped, mapErr := mapA2AStreamEvent(item.Event)
			if mapErr != nil {
				yield(streammodel.Event{}, classify(contracts.ErrorCodeA2AProtocol, mapErr))
				return
			}
			if terminalSeen {
				yield(streammodel.Event{}, classify(contracts.ErrorCodeA2AProtocol, errors.New("A2A stream emitted an event after terminal")))
				return
			}
			if mapped.Kind == "artifact-update" {
				if mapped.ArtifactID == "" {
					yield(streammodel.Event{}, classify(contracts.ErrorCodeA2AProtocol, errors.New("A2A artifact identity is required")))
					return
				}
				if mapped.ArtifactAppend && !artifactSeen[mapped.ArtifactID] {
					yield(streammodel.Event{}, classify(contracts.ErrorCodeA2AProtocol, errors.New("A2A artifact append has no base chunk")))
					return
				}
				if artifactLast[mapped.ArtifactID] {
					yield(streammodel.Event{}, classify(contracts.ErrorCodeA2AProtocol, errors.New("A2A artifact emitted after its last chunk")))
					return
				}
				if !mapped.ArtifactAppend && artifactSeen[mapped.ArtifactID] {
					yield(streammodel.Event{}, classify(contracts.ErrorCodeA2AProtocol, errors.New("A2A artifact base chunk was repeated")))
					return
				}
				artifactSeen[mapped.ArtifactID] = true
				if mapped.ArtifactLast {
					artifactLast[mapped.ArtifactID] = true
				}
			}
			payload := append(json.RawMessage(nil), item.Result...)
			if len(payload) == 0 {
				yield(streammodel.Event{}, classify(contracts.ErrorCodeA2AProtocol, errors.New("A2A stream result payload is unavailable")))
				return
			}
			if int64(len(payload)) > eventLimit {
				yield(streammodel.Event{}, classify(contracts.ErrorCodeAgentResponseTooLarge, errors.New("A2A streaming event exceeds the configured limit")))
				return
			}
			if taskID == "" {
				taskID, contextID = mapped.TaskID, mapped.ContextID
			} else if mapped.TaskID != taskID || mapped.ContextID != contextID {
				yield(streammodel.Event{}, classify(contracts.ErrorCodeA2AProtocol, errors.New("A2A stream task or context identity changed")))
				return
			}
			if mapped.TerminalType != "" {
				for artifactID := range artifactSeen {
					if !artifactLast[artifactID] {
						yield(streammodel.Event{}, classify(contracts.ErrorCodeA2AProtocol, errors.New("A2A artifact stream ended before last chunk")))
						return
					}
				}
				terminalSeen = true
				pending := mapped
				pending.Payload = payload
				pendingTerminal = &pending
				continue
			}
			mapped.Payload = payload
			if !yield(mapped, nil) {
				return
			}
		}
		for artifactID := range artifactSeen {
			if !artifactLast[artifactID] {
				yield(streammodel.Event{}, classify(contracts.ErrorCodeA2AProtocol, errors.New("A2A artifact stream ended before lastChunk")))
				return
			}
		}
		if pendingTerminal != nil {
			if ctx.Err() != nil {
				yield(streammodel.Event{}, ctx.Err())
				return
			}
			if !yield(*pendingTerminal, nil) {
				return
			}
			return
		}
		if ctx.Err() != nil {
			yield(streammodel.Event{}, ctx.Err())
			return
		}
		yield(streammodel.Event{}, streammodel.ErrInterrupted)
	}
}

func mapA2AStreamEvent(event a2ago.Event) (streammodel.Event, error) {
	switch value := event.(type) {
	case *a2ago.Message:
		if err := contracts.ValidateA2AMessageResult(value); err != nil {
			return streammodel.Event{}, err
		}
		if value.TaskID == "" || value.ContextID == "" {
			return streammodel.Event{}, errors.New("A2A stream message task and context IDs are required")
		}
		return streammodel.Event{Kind: "message", TaskID: string(value.TaskID), ContextID: value.ContextID}, nil
	case *a2ago.Task:
		mapping, err := contracts.ValidateA2ATask(value)
		if err != nil {
			return streammodel.Event{}, err
		}
		mapped := streammodel.Event{Kind: "task", TaskID: string(value.ID), ContextID: value.ContextID}
		if mapping.Classification == contracts.A2ATaskStateTerminal {
			mapped.TerminalStatus = mapping.InvocationStatus
			mapped.ErrorCode = mapping.ErrorCode
			switch mapping.InvocationStatus {
			case "succeeded":
				mapped.TerminalType = contracts.ResultStreamEventCompleted
			case "canceled":
				mapped.TerminalType = contracts.ResultStreamEventCanceled
			default:
				mapped.TerminalType = contracts.ResultStreamEventFailed
			}
		}
		return mapped, nil
	case *a2ago.TaskStatusUpdateEvent:
		if value.TaskID == "" || value.ContextID == "" {
			return streammodel.Event{}, errors.New("A2A status event task and context IDs are required")
		}
		mapping, err := contracts.MapA2ATaskState(value.Status.State)
		if err != nil {
			return streammodel.Event{}, err
		}
		if mapping.Classification == contracts.A2ATaskStateTerminal && !value.Final || mapping.Classification == contracts.A2ATaskStateTransient && value.Final {
			return streammodel.Event{}, errors.New("A2A status event final flag contradicts task state")
		}
		mapped := streammodel.Event{Kind: "status-update", TaskID: string(value.TaskID), ContextID: value.ContextID}
		if mapping.Classification == contracts.A2ATaskStateTerminal {
			mapped.TerminalStatus = mapping.InvocationStatus
			mapped.ErrorCode = mapping.ErrorCode
			switch mapping.InvocationStatus {
			case "succeeded":
				mapped.TerminalType = contracts.ResultStreamEventCompleted
			case "canceled":
				mapped.TerminalType = contracts.ResultStreamEventCanceled
			default:
				mapped.TerminalType = contracts.ResultStreamEventFailed
			}
		}
		return mapped, nil
	case *a2ago.TaskArtifactUpdateEvent:
		if value.TaskID == "" || value.ContextID == "" || value.Artifact == nil || value.Artifact.ID == "" || len(value.Artifact.Parts) == 0 {
			return streammodel.Event{}, errors.New("A2A artifact event is incomplete")
		}
		return streammodel.Event{Kind: "artifact-update", TaskID: string(value.TaskID), ContextID: value.ContextID, ArtifactID: string(value.Artifact.ID), ArtifactAppend: value.Append, ArtifactLast: value.LastChunk}, nil
	default:
		return streammodel.Event{}, errors.New("A2A stream event kind is unsupported")
	}
}
