# Research: Streaming A2A Result Delivery

## Decision 1: Use the pinned A2A JSON-RPC streaming operation

- **Decision**: Call A2A `message/stream` through the existing
  `a2aproject/a2a-go` JSON-RPC client and accept only the active profile's
  message, task, status-update, and artifact-update event kinds.
- **Rationale**: The pinned A2A Profile 0.2 explicitly declares
  `message/stream`, `text/event-stream`, and those four event kinds. Runtime B
  already exposes a deterministic stream fixture using the same library.
- **Alternatives considered**: Reusing `message/send` and polling tasks would
  violate the server-stream contract and introduce a result lifecycle; adding
  a custom A2A parser would duplicate the pinned protocol implementation.

## Decision 2: Validate and bound upstream events before forwarding

- **Decision**: Measure the serialized A2A event body against the required
  A2A event limit, validate profile semantics and stable task/context identity,
  then map one event to one transient Result Stream Event v2 chunk.
- The complete upstream SSE block is bounded before parsing as the transport
  memory guard; the validated raw JSON-RPC `result` bytes are then forwarded
  and checked against the same effective A2A event limit.
- **Rationale**: ADR 0006 requires explicit A2A and SSE limits and forbids
  truncation or whole-stream buffering. Shared runtime validators already
  enforce gapless sequence/chunk indexes and first-terminal semantics.
- **Alternatives considered**: Buffering the full stream or truncating an
  oversized event would create unbounded memory or false success.

## Decision 3: Emit strict one-line SSE frames

- **Decision**: Marshal each Result Stream Event v2 compactly, reject any frame
  whose complete UTF-8 `data:` line plus delimiter exceeds the separately
  configured `NEKIRO_ROUTER_SSE_EVENT_LIMIT_BYTES`, write exactly one `data:`
  line and one blank line, and flush after each frame.
- **Rationale**: ADR 0006 and Router Internal v3 define the framing as an
  executable boundary; multiline data, extra SSE fields, malformed JSON, and
  missing delimiters are invalid.
- **Alternatives considered**: Generic SSE helpers that split data across
  multiple lines are incompatible with the active profile's single-line rule.

## Decision 4: Keep streaming output transient and Ledger metadata-only

- **Decision**: Append only lifecycle/chunk byte metadata to Invocation Event
  0.3. Never put A2A event or Result Stream `chunk` values in Ledger rows,
  logs, or read responses.
- **Rationale**: Result delivery and audit facts have different retention and
  secrecy semantics; the active event schema provides `chunkIndex` and
  `chunkBytes` specifically for metadata-only facts.
- **Alternatives considered**: Persisting chunks for replay would violate
  ADR 0002/0006 and require a new retention and authorization policy.

## Decision 5: Resolve the first terminal outcome once

- **Decision**: Use the existing Result Stream v2 validator and request context
  cancellation. Emit at most one terminal event; after a terminal or an
  interrupted EOF, later source events are discarded and cannot overwrite the
  result.
- **Rationale**: The shared validator and ADR 0006 define first-terminal
  immutability and distinguish timeout, cancellation, protocol failure, and
  interrupted delivery.
- **Alternatives considered**: Letting the source stream continue after a
  terminal event would expose contradictory outcomes and make Ledger history
  ambiguous.

The one-shot remote `tasks/cancel` request and the post-cancellation terminal
Ledger append each use a bounded one-second operational grace. These bounds
are explicit completion policies, not retries, alternate routes, or fallback
results; a failure still leaves the local timeout/cancel winner visible.

## Evidence Sources

- `docs/decisions/0006-invocation-runtime-trust-and-failure-policy.md`
- `docs/decisions/0002-invocation-result-transport-and-internal-api-direction.md`
- `contracts/openapi/router-internal.v3.yaml`
- `contracts/schemas/invocation-result-stream-event.v2.schema.json`
- `contracts/a2a-profile/v0.3.0/profile.v0.2.json`
- `contracts/invocation-runtime/v1/semantic-rules.md`
- `agents/runtime-b/handler.go` and its streaming tests
