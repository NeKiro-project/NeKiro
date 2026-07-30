# Data Model: Standalone A2A Transport

This feature adds no persistent data. The following are immutable or
request-scoped library values.

## Client

Represents a reusable transport client constructed from an explicit non-nil
HTTP client. It stores a cloned HTTP client whose redirect policy rejects all
redirects. A nil underlying `http.Transport` retains Go's documented use of
`http.DefaultTransport`.

## Call Options

| Field | Required | Validation | Ownership |
| --- | --- | --- | --- |
| Endpoint | Yes | Absolute `http` or `https` URL, host present, no user info | Caller; NeKiro obtains it from exact resolution |
| Maximum response bytes | Yes | Positive and safe for bounded-read accounting | Caller; NeKiro passes effective configured/Card output bound |
| Maximum event bytes | Streaming only | Positive and safe for full SSE-event accounting | Caller; NeKiro passes effective configured/Card event bound |
| Interceptors | Optional | Every supplied value is used in declared order; nil entries are invalid | Caller; NeKiro supplies one request-scoped credential interceptor |

No endpoint, byte bound, interceptor, header, or credential is inferred.

## Stream Item

| Field | Meaning |
| --- | --- |
| Event | One supported official decoded A2A event |
| Result | Immutable raw JSON-RPC `result` bytes associated with the event |

The raw result is transient caller data. The standalone module does not store
it, and NeKiro does not write it to Ledger.

## Failure

| Kind | Trigger | NeKiro adapter mapping |
| --- | --- | --- |
| `invalid_argument` | Missing/invalid endpoint, bound, request, task ID, or interceptor | `A2A_PROTOCOL_ERROR` at the adapter boundary |
| `protocol` | Invalid JSON-RPC/SSE/media/result/event wire behavior | `A2A_PROTOCOL_ERROR` |
| `remote_agent` | Official A2A error returned by the remote Agent | `AGENT_EXECUTION_FAILED` |
| `unavailable` | Network/URL transport failure, non-success HTTP transport status, or interrupted remote body | `AGENT_UNAVAILABLE` |
| `deadline_exceeded` | Caller deadline | `TIMEOUT` |
| `canceled` | Caller cancellation | `CANCELED` |
| `response_too_large` | Complete response/event exceeds explicit bound | `AGENT_RESPONSE_TOO_LARGE` |

Every Failure retains a cause for `errors.Is`/`errors.As`, but its own display
text contains only the stable kind.

## NeKiro Adapter Values

The existing Router `Target`, `ContextHeaders`, credential context, dispatch
request, resolution response, internal stream event, and classified platform
error remain downstream. Their state transitions and ownership do not change.

## Lifecycle

```text
explicit Call Options
  -> validate
  -> construct isolated official A2A client
  -> inject caller-owned metadata once per protocol operation
  -> execute bounded JSON or SSE transport
  -> return supported decoded value + raw stream result
     OR one typed Failure
  -> close body / finish iterator
```

No retry, reconnect, replay, cache, persistence, alternate source, or degraded
state exists.
