package a2a

import (
	"context"
	"errors"
	"net/http"
	"net/url"

	"github.com/NeKiro-project/NeKiro/contracts"
	a2atransport "github.com/NeKiro-project/nekiro-a2a-transport-go"
	a2ago "github.com/a2aproject/a2a-go/a2a"
	"github.com/a2aproject/a2a-go/a2aclient"
)

const (
	HeaderTraceID            = contracts.RouterAgentTraceHeader
	HeaderInvocationID       = contracts.RouterAgentInvocationHeader
	HeaderRootTaskID         = contracts.RouterAgentRootTaskHeader
	HeaderParentInvocationID = contracts.RouterAgentParentInvocationHeader
	HeaderWorkspaceID        = contracts.RouterAgentWorkspaceHeader
)

type Client struct {
	transportClient    *a2atransport.Client
	credentialIssuer   CredentialIssuer
	targetSelector     TargetSelector
	inputLimitBytes    int64
	responseLimitBytes int64
	a2aEventLimitBytes int64
}

type TargetSelector interface {
	Select(context.Context, Target, ContextHeaders) (Target, error)
}

type CredentialIssuer interface {
	Issue(contracts.RouterInvocationCredentialContextV1) (string, error)
}

type ContextHeaders struct {
	TraceID            contracts.TraceID
	InvocationID       string
	RootTaskID         string
	ParentInvocationID string
	WorkspaceID        string
}

func NewClient(httpClient *http.Client, credentialIssuer CredentialIssuer, inputLimitBytes, responseLimitBytes, a2aEventLimitBytes, sseEventLimitBytes int64) (*Client, error) {
	return newClient(httpClient, credentialIssuer, nil, inputLimitBytes, responseLimitBytes, a2aEventLimitBytes, sseEventLimitBytes)
}

func NewClientWithTargetSelector(httpClient *http.Client, credentialIssuer CredentialIssuer, targetSelector TargetSelector, inputLimitBytes, responseLimitBytes, a2aEventLimitBytes, sseEventLimitBytes int64) (*Client, error) {
	if targetSelector == nil {
		return nil, errors.New("A2A target selector is required")
	}
	return newClient(httpClient, credentialIssuer, targetSelector, inputLimitBytes, responseLimitBytes, a2aEventLimitBytes, sseEventLimitBytes)
}

func newClient(httpClient *http.Client, credentialIssuer CredentialIssuer, targetSelector TargetSelector, inputLimitBytes, responseLimitBytes, a2aEventLimitBytes, sseEventLimitBytes int64) (*Client, error) {
	if httpClient == nil {
		return nil, errors.New("A2A transport HTTP client is required")
	}
	if credentialIssuer == nil {
		return nil, errors.New("a2a transport credential issuer is required")
	}
	if inputLimitBytes < contracts.RuntimeByteLimitMinimum || inputLimitBytes > contracts.RuntimeByteLimitMaximum {
		return nil, errors.New("A2A Agent input limit is invalid")
	}
	if responseLimitBytes < contracts.RuntimeByteLimitMinimum || responseLimitBytes > contracts.RuntimeByteLimitMaximum {
		return nil, errors.New("A2A Agent response limit is invalid")
	}
	if a2aEventLimitBytes < contracts.RuntimeByteLimitMinimum || a2aEventLimitBytes > contracts.RuntimeByteLimitMaximum {
		return nil, errors.New("A2A event limit is invalid")
	}
	if sseEventLimitBytes < contracts.RuntimeByteLimitMinimum || sseEventLimitBytes > contracts.RuntimeByteLimitMaximum {
		return nil, errors.New("SSE event limit is invalid")
	}
	transportClient, err := a2atransport.NewClient(httpClient)
	if err != nil {
		return nil, err
	}
	return &Client{transportClient: transportClient, credentialIssuer: credentialIssuer, targetSelector: targetSelector, inputLimitBytes: inputLimitBytes, responseLimitBytes: responseLimitBytes, a2aEventLimitBytes: a2aEventLimitBytes}, nil
}

func (client *Client) selectTarget(ctx context.Context, target Target, headers ContextHeaders) (Target, error) {
	if client.targetSelector == nil {
		return target, nil
	}
	selected, err := client.targetSelector.Select(ctx, target, headers)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return Target{}, ctxErr
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return Target{}, err
		}
		return Target{}, classify(contracts.ErrorCodeDependency, errors.New("instance directory selection failed"))
	}
	if err := validateEndpointDestinationChange(target.Endpoint, selected.Endpoint); err != nil {
		return Target{}, classify(contracts.ErrorCodeA2AProtocol, err)
	}
	selectedEndpoint := selected.Endpoint
	selected.Endpoint = target.Endpoint
	if selected != target {
		return Target{}, classify(contracts.ErrorCodeA2AProtocol, errors.New("instance selector changed a field other than the network endpoint"))
	}
	selected.Endpoint = selectedEndpoint
	return selected, nil
}

func validateEndpointDestinationChange(original, selected string) error {
	originalURL, err := url.Parse(original)
	if err != nil {
		return errors.New("original target endpoint is invalid")
	}
	selectedURL, err := url.Parse(selected)
	if err != nil || selectedURL.Scheme != originalURL.Scheme || selectedURL.Opaque != originalURL.Opaque ||
		(selectedURL.User == nil) != (originalURL.User == nil) || selectedURL.User.String() != originalURL.User.String() ||
		selectedURL.Path != originalURL.Path ||
		selectedURL.RawPath != originalURL.RawPath || selectedURL.ForceQuery != originalURL.ForceQuery ||
		selectedURL.RawQuery != originalURL.RawQuery || selectedURL.Fragment != originalURL.Fragment ||
		selectedURL.RawFragment != originalURL.RawFragment || selectedURL.OmitHost != originalURL.OmitHost ||
		selectedURL.Host == "" {
		return errors.New("instance selector may change only the target endpoint authority")
	}
	return nil
}

func (client *Client) SendMessage(ctx context.Context, target Target, headers ContextHeaders, params *a2ago.MessageSendParams) (a2ago.SendMessageResult, error) {
	if target.Endpoint == "" {
		return nil, classify(contracts.ErrorCodeA2AProtocol, errors.New("A2A target endpoint is required"))
	}
	if params == nil || params.Message == nil {
		return nil, classify(contracts.ErrorCodeA2AProtocol, errors.New("A2A message/send params are required"))
	}
	if headers.TraceID == "" || headers.InvocationID == "" || headers.RootTaskID == "" || headers.WorkspaceID == "" {
		return nil, classify(contracts.ErrorCodeA2AProtocol, errors.New("platform context headers are required"))
	}
	responseLimit := client.responseLimitBytes
	if target.MaxOutputBytes < responseLimit {
		responseLimit = target.MaxOutputBytes
	}
	result, err := client.transportClient.SendMessage(ctx, a2atransport.CallOptions{
		Endpoint: target.Endpoint, MaxResponseBytes: responseLimit,
		Interceptors: []a2aclient.CallInterceptor{newCredentialInterceptor(client.credentialIssuer, credentialContext(target, headers))},
	}, params)
	if err != nil {
		return nil, classifyTransportError(err)
	}
	return result, nil
}

type credentialInterceptor struct {
	a2aclient.PassthroughInterceptor
	issuer  CredentialIssuer
	context contracts.RouterInvocationCredentialContextV1
}

func newCredentialInterceptor(issuer CredentialIssuer, context contracts.RouterInvocationCredentialContextV1) a2aclient.CallInterceptor {
	return &credentialInterceptor{issuer: issuer, context: context}
}

func (interceptor *credentialInterceptor) Before(ctx context.Context, request *a2aclient.Request) (context.Context, error) {
	token, err := interceptor.issuer.Issue(interceptor.context)
	if err != nil {
		return ctx, classify(contracts.ErrorCodeInternal, err)
	}
	request.Meta.Append(contracts.RouterAgentAuthorizationHeader, "Bearer "+token)
	for name, value := range contracts.RouterAgentContextHeadersV1(interceptor.context) {
		request.Meta.Append(name, value)
	}
	return ctx, nil
}

func credentialContext(target Target, headers ContextHeaders) contracts.RouterInvocationCredentialContextV1 {
	return contracts.RouterInvocationCredentialContextV1{
		Audience: target.Audience, WorkspaceID: headers.WorkspaceID, AgentID: target.AgentID, AgentVersion: target.Version,
		ReleaseID: target.ReleaseID, CardDigest: target.CardDigest, Capability: target.Capability, InvocationID: headers.InvocationID,
		RootTaskID: headers.RootTaskID, ParentInvocationID: headers.ParentInvocationID, TraceID: headers.TraceID,
	}
}
