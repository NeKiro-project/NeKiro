# Data Model: Invocation and Trace Metadata Reads

This feature adds no durable tables. It exposes existing Router-owned
projections through versioned read contracts.

## InvocationDetailResponseV4

| Part | Source | Rule |
| --- | --- | --- |
| `invocation` | Router Ledger projection | One Workspace-scoped metadata record |
| `events` | Router Ledger immutable events | Ordered, gap-free Event 0.3 facts |
| Agent result content | none | Never present in the response |

The latest projection status must equal the final committed event status. A
non-terminal final event remains visible as-is; the read path does not infer a
terminal result from a missing event.

## TraceResponseV4

| Field | Rule |
| --- | --- |
| `traceId` | Exactly the requested Trace ID |
| `invocations` | Non-empty, same Workspace/Trace/root Task, parent before child |

Trace ordering is owned by the Router Ledger Store. The Gateway does not sort,
merge, or filter the response.

## Read Boundary Values

```text
Public caller + Workspace ID + resource ID
  -> Gateway Trace
  -> Workspace owner authorization
  -> Router Internal v3 GET
  -> InvocationDetailResponseV4 | TraceResponseV4
```

No read value contains input, output, chunks, credentials, raw dependency
messages, endpoint secrets, or a replay cursor.

## Error Mapping

| Boundary outcome | Public status/code | Router call |
| --- | --- | --- |
| Invalid auth/path | 401/400 | none |
| Unknown Workspace | 404 `NOT_FOUND` | none |
| Non-owner | 403 `FORBIDDEN` | none |
| Authorized missing Invocation/Trace | 404 `NOT_FOUND` | one |
| Router auth/media/transport/5xx | 503 `DEPENDENCY_ERROR` | one |
