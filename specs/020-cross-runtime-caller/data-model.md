# Data Model: Cross-Runtime Caller Sample

This feature adds no durable schema. The following transient values are the only data owned by the sample adapter/runtime; the framework's process-local Session is discarded with the sample process and is not shared with Runtime B.

## RuntimeAConfig

| Field | Source | Validation | Use |
| --- | --- | --- | --- |
| ListenAddress | required environment | exact host:port, port 1..65535, no surrounding whitespace | process listener |
| RouterURL | required environment | absolute http/https URL with host, no surrounding whitespace | passed to SDK client |
| RouterToken | required environment | non-empty exact credential, no surrounding whitespace or logging | SDK bearer auth |
| TargetAgentID | required environment | safe identifier grammar | SDK nested target |
| Capability | required environment | safe identifier grammar | SDK nested capability |
| ResponseLimitBytes | required environment | strict decimal 1..2147483647 | SDK JSON/error body bound |
| EventLimitBytes | required environment | strict decimal 1..2147483647 | SDK SSE bound |

## PlatformContext

Transient trusted values read from authenticated A2A request metadata:

```text
InvocationID
RootTaskID
TraceID
WorkspaceID
AgentID (Runtime A configured identity)
```

Every field is required. The adapter never derives, substitutes, or accepts these values from the A2A message body.

## CombinedResult

Transient JSON data returned in one A2A agent message:

```json
{
  "agent": "runtime-a",
  "childInvocationId": "<validated child id>",
  "childResult": <validated child result JSON>
}
```

The child result is preserved as JSON tokens. `childInvocationId` is lineage metadata and is excluded when comparing deterministic business payloads. No input, credential, raw Router error, or Runtime-internal event data is copied into platform facts or logs.
