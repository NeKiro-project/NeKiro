package a2a

import (
	"context"
	"encoding/json"

	"github.com/Nene7ko/NeKiro/contracts"
	a2ago "github.com/a2aproject/a2a-go/a2a"
)

func (client *Client) SendNonStreaming(ctx context.Context, dispatch contracts.DispatchInvocationRequestV3, resolved contracts.ResolveAgentResponse) (json.RawMessage, error) {
	target, err := NewTarget(resolved, dispatch.Capability)
	if err != nil {
		return nil, err
	}
	params, err := messageSendParams(dispatch)
	if err != nil {
		return nil, err
	}
	result, err := client.SendMessage(ctx, target, ContextHeaders{
		TraceID:      dispatch.TraceID,
		InvocationID: dispatch.InvocationID,
		RootTaskID:   dispatch.RootTaskID,
		WorkspaceID:  dispatch.WorkspaceID,
	}, params)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(encoded), nil
}

func messageSendParams(dispatch contracts.DispatchInvocationRequestV3) (*a2ago.MessageSendParams, error) {
	var input map[string]json.RawMessage
	if err := json.Unmarshal(dispatch.Input, &input); err != nil {
		return nil, err
	}
	data := make(map[string]any, len(input))
	for key, value := range input {
		data[key] = value
	}
	return &a2ago.MessageSendParams{Message: &a2ago.Message{
		ID:    dispatch.InvocationID,
		Role:  a2ago.MessageRoleUser,
		Parts: []a2ago.Part{a2ago.DataPart{Data: data}},
	}}, nil
}
