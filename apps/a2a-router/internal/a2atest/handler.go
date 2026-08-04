package a2atest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"iter"

	"github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2asrv"
)

// Handler supplies the minimal successful A2A responses needed by Router tests.
type Handler struct{}

var _ a2asrv.RequestHandler = (*Handler)(nil)

func NewHandler() *Handler {
	return &Handler{}
}

func (*Handler) OnSendMessage(_ context.Context, params *a2a.MessageSendParams) (a2a.SendMessageResult, error) {
	message, fixture, value, err := parseFixture(params)
	if err != nil {
		return nil, err
	}
	if fixture != "success" {
		return nil, invalidParams("fixture requires message/stream")
	}
	contextID := message.ContextID
	if contextID == "" {
		contextID = derivedID("context", message.ID)
	}
	return &a2a.Message{
		ID:        derivedID("message", message.ID),
		ContextID: contextID,
		Role:      a2a.MessageRoleAgent,
		Parts: []a2a.Part{a2a.DataPart{Data: map[string]any{
			"fixture": fixture,
			"value":   value,
		}}},
	}, nil
}

func (*Handler) OnSendMessageStream(_ context.Context, params *a2a.MessageSendParams) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		message, fixture, value, err := parseFixture(params)
		if err != nil {
			yield(nil, err)
			return
		}
		if fixture != "stream-success" {
			yield(nil, invalidParams("fixture requires message/send"))
			return
		}

		taskID := a2a.TaskID(derivedID("task", message.ID))
		contextID := message.ContextID
		if contextID == "" {
			contextID = derivedID("context", message.ID)
		}
		task := &a2a.Task{
			ID:        taskID,
			ContextID: contextID,
			Status:    a2a.TaskStatus{State: a2a.TaskStateWorking},
			History:   []*a2a.Message{message},
		}
		if !yield(task, nil) {
			return
		}
		if !yield(&a2a.Message{
			ID:        derivedID("stream-message", message.ID),
			TaskID:    taskID,
			ContextID: contextID,
			Role:      a2a.MessageRoleAgent,
			Parts: []a2a.Part{a2a.DataPart{Data: map[string]any{
				"fixture": fixture,
				"phase":   "working",
				"value":   value,
			}}},
		}, nil) {
			return
		}

		artifactID := a2a.ArtifactID(derivedID("artifact", message.ID))
		for sequence, chunk := range []struct {
			appendPart bool
			last       bool
		}{
			{},
			{appendPart: true, last: true},
		} {
			if !yield(&a2a.TaskArtifactUpdateEvent{
				TaskID:    taskID,
				ContextID: contextID,
				Append:    chunk.appendPart,
				LastChunk: chunk.last,
				Artifact: &a2a.Artifact{
					ID: artifactID,
					Parts: []a2a.Part{a2a.DataPart{Data: map[string]any{
						"fixture":  fixture,
						"sequence": sequence,
						"value":    value,
					}}},
				},
			}, nil) {
				return
			}
		}
		yield(&a2a.TaskStatusUpdateEvent{
			TaskID:    taskID,
			ContextID: contextID,
			Status:    a2a.TaskStatus{State: a2a.TaskStateCompleted},
			Final:     true,
		}, nil)
	}
}

func parseFixture(params *a2a.MessageSendParams) (*a2a.Message, string, any, error) {
	if params == nil || params.Message == nil || params.Message.ID == "" {
		return nil, "", nil, invalidParams("message and messageId are required")
	}
	if params.Message.Role != a2a.MessageRoleUser || len(params.Message.Parts) != 1 {
		return nil, "", nil, invalidParams("one user data part is required")
	}
	part, ok := params.Message.Parts[0].(a2a.DataPart)
	if !ok || len(part.Data) != 2 {
		return nil, "", nil, invalidParams("fixture and value are required")
	}
	fixture, ok := part.Data["fixture"].(string)
	if !ok || fixture == "" {
		return nil, "", nil, invalidParams("fixture must be a non-empty string")
	}
	value, exists := part.Data["value"]
	if !exists {
		return nil, "", nil, invalidParams("value is required")
	}
	return params.Message, fixture, value, nil
}

func derivedID(prefix, value string) string {
	digest := sha256.Sum256([]byte(prefix + ":" + value))
	return prefix + "-" + hex.EncodeToString(digest[:16])
}

func invalidParams(message string) error {
	return fmt.Errorf("%s: %w", message, a2a.ErrInvalidParams)
}

func (*Handler) OnGetTask(context.Context, *a2a.TaskQueryParams) (*a2a.Task, error) {
	return nil, a2a.ErrUnsupportedOperation
}

func (*Handler) OnListTasks(context.Context, *a2a.ListTasksRequest) (*a2a.ListTasksResponse, error) {
	return nil, a2a.ErrUnsupportedOperation
}

func (*Handler) OnCancelTask(context.Context, *a2a.TaskIDParams) (*a2a.Task, error) {
	return nil, a2a.ErrUnsupportedOperation
}

func (*Handler) OnResubscribeToTask(context.Context, *a2a.TaskIDParams) iter.Seq2[a2a.Event, error] {
	return func(yield func(a2a.Event, error) bool) {
		yield(nil, a2a.ErrUnsupportedOperation)
	}
}

func (*Handler) OnGetTaskPushConfig(context.Context, *a2a.GetTaskPushConfigParams) (*a2a.TaskPushConfig, error) {
	return nil, a2a.ErrUnsupportedOperation
}

func (*Handler) OnListTaskPushConfig(context.Context, *a2a.ListTaskPushConfigParams) ([]*a2a.TaskPushConfig, error) {
	return nil, a2a.ErrUnsupportedOperation
}

func (*Handler) OnSetTaskPushConfig(context.Context, *a2a.TaskPushConfig) (*a2a.TaskPushConfig, error) {
	return nil, a2a.ErrUnsupportedOperation
}

func (*Handler) OnDeleteTaskPushConfig(context.Context, *a2a.DeleteTaskPushConfigParams) error {
	return a2a.ErrUnsupportedOperation
}

func (*Handler) OnGetExtendedAgentCard(context.Context) (*a2a.AgentCard, error) {
	return nil, a2a.ErrUnsupportedOperation
}
