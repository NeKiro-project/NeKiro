# Data Model: Streaming A2A Result Delivery

This feature introduces no durable result entity. The following values exist
only during one live Router invocation or as metadata-only Ledger facts.

## Result Stream Event v2

| Field | Meaning | Validation |
| --- | --- | --- |
| `schemaVersion` | Active stream contract version | Exactly `2` |
| `sequence` | Gapless outer event sequence | Starts at `0` and increments by one |
| `type` | `accepted`, `chunk`, or terminal type | Must match `status` and terminal rules |
| `status` | Pending, running, or terminal status | Contract enum; first event pending |
| `invocationId` | Root Invocation identity | Matches request exactly |
| `rootTaskId` | Root Task identity | Matches request exactly |
| `traceId` | Trace identity | Matches request exactly |
| `chunkIndex` | Zero-based chunk sequence | Required only for `chunk`; gapless |
| `chunk` | Transient opaque Agent event value | Required only for `chunk`; never persisted |
| `error` | Correlated Platform Error v4 | Required only for failure/cancel/timeout |

## A2A Stream Event

An upstream event is one of the profile-approved A2A message, task,
task-status-update, or task-artifact-update values. The Router validates its
raw JSON-RPC `result` serialization and profile semantics before placing it
inside a Result Stream Event `chunk`; the complete upstream SSE block is also
bounded as the transport memory boundary.

## SSE Frame

The wire representation of one Result Stream Event is:

```text
data: {compact JSON object}\n
\n
```

The entire UTF-8 frame, including the `data:` prefix and delimiters, is
bounded by the required SSE event limit. Literal CR/LF in JSON strings is
escaped by JSON serialization and never becomes a physical line break.

## Invocation Event 0.3 Stream Fact

The Ledger may contain only:

```text
type: stream
status: running
chunkIndex: zero-based metadata index
chunkBytes: serialized upstream-event byte count
```

No `chunk`, A2A event, input, output, credential, or raw dependency detail is
stored. Terminal facts remain the existing `succeeded`, `failed`, `canceled`,
or `timed_out` lifecycle records.

## State Transitions

```text
accepted/pending
  -> chunk/running (zero or more)
  -> completed/succeeded | failed/failed | canceled/canceled | timed_out/timed_out
```

The first terminal event closes the live stream. EOF before a terminal event is
an interrupted delivery and does not become `completed`.
