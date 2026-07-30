# Public API Contract: nekiro-a2a-transport-go v0

This document fixes the initial review surface for the standalone Go module.
It is an implementation contract, not a replacement for NeKiro's
language-neutral A2A Profile.

## Package Identity

```go
import "github.com/NeKiro-project/nekiro-a2a-transport-go"
```

The root package name is `a2atransport`.

## Compatibility Identity

```go
const ProtocolVersion = "0.3.0"
const A2AGoVersion = "v0.3.15"
```

Changing either constant requires explicit compatibility review and a module
release. These constants do not dynamically select a fallback implementation.

## Public Types

```go
type CallOptions struct {
    Endpoint         string
    MaxResponseBytes int64
    MaxEventBytes    int64
    Interceptors     []a2aclient.CallInterceptor
}

type StreamItem struct {
    Event  a2a.Event
    Result json.RawMessage
}

type FailureKind string

const (
    FailureInvalidArgument  FailureKind = "invalid_argument"
    FailureProtocol         FailureKind = "protocol"
    FailureRemoteAgent      FailureKind = "remote_agent"
    FailureUnavailable      FailureKind = "unavailable"
    FailureDeadlineExceeded FailureKind = "deadline_exceeded"
    FailureCanceled         FailureKind = "canceled"
    FailureResponseTooLarge FailureKind = "response_too_large"
)

type Failure struct { /* immutable kind and cause */ }

func (f *Failure) Error() string
func (f *Failure) Unwrap() error
func (f *Failure) Kind() FailureKind
func FailureKindOf(err error) (FailureKind, bool)
```

`Failure.Error()` returns stable non-secret category text and does not include
the wrapped cause. `Unwrap` preserves programmatic cause matching.

## Client Operations

```go
func NewClient(httpClient *http.Client) (*Client, error)

func (c *Client) SendMessage(
    ctx context.Context,
    options CallOptions,
    params *a2a.MessageSendParams,
) (a2a.SendMessageResult, error)

func (c *Client) SendStreamingMessage(
    ctx context.Context,
    options CallOptions,
    params *a2a.MessageSendParams,
) iter.Seq2[StreamItem, error]

func (c *Client) CancelTask(
    ctx context.Context,
    options CallOptions,
    taskID a2a.TaskID,
) (*a2a.Task, error)
```

## Operation Semantics

- `NewClient` rejects nil and clones caller state. Calls never mutate the
  caller's HTTP client.
- Every operation rejects redirects.
- `SendMessage` requires endpoint, response bound, params, and message.
- `SendStreamingMessage` additionally requires an event bound and returns one
  `StreamItem` for each accepted upstream event.
- `CancelTask` requires a non-empty Task ID and a response bound.
- Interceptors execute through the official A2A client for each operation.
  Nil entries are invalid; no default interceptor exists.
- Successful JSON responses require exact `application/json`; successful
  streaming responses require exact `text/event-stream` after media parsing.
- JSON-RPC responses require version `2.0`, matching supported request ID,
  exactly one non-null result or error, no duplicate members, no unknown
  envelope fields, and no trailing data.
- Each SSE event requires one unique non-empty `id` line, one `data` line, a
  blank delimiter, valid JSON, a matching JSON-RPC ID, and a non-null result.
- A complete response or event exceeding its explicit bound fails. No partial
  value is returned as success.
- The library validates that decoded stream events use one of the active
  official event kinds. NeKiro's active Profile remains responsible for deeper
  Message/Task states, identity sequences, artifact order, and platform mapping.

## Forbidden Dependencies and Behavior

- No import from any NeKiro application, contracts, SDK, Runtime, or storage
  package.
- No endpoint discovery, Agent Card resolution, authorization policy,
  credential generation, Workspace context, Release provenance, Ledger, or
  platform error/result DTO.
- No retry, reconnect, replay, cache, alternate endpoint, alternate credential,
  protocol downgrade, legacy decoder, default limit, or response truncation.
