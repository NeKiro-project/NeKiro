# Feature Specification: Streaming A2A Result Delivery

**Feature Branch**: `017-streaming-a2a-events`

**Created**: 2026-07-16

**Status**: Implemented and converged through T021

**Input**: Continue the approved Spec 016 follow-up T017: implement bounded
streaming A2A result delivery with strict SSE event framing and lifecycle
semantics.

## Context

Spec 016 delivers exact non-streaming `message/send` calls and transient JSON
results. The Router Internal v3 contract already reserves `stream=true` with
`Accept: text/event-stream`, Result Stream Event v2, A2A `message/stream`, and
required Agent/A2A/SSE byte limits. This feature closes that streaming leg
without changing Catalog, Workspace, Registry, Agent Card, or Ledger ownership.

The accepted policy is recorded in ADR 0006: the Router must fail closed on
oversized A2A or SSE events, must not buffer an entire stream, must emit one
compact JSON value per `data:` line followed by one blank line and an immediate
flush, and must never report a clean terminal result before its Ledger fact is
committed.

## User Scenarios & Testing

### User Story 1 - Stream an Agent Result (Priority: P1)

As the Control Plane Invocation Dispatch service, I need a valid streaming
invocation to receive ordered transient result events from the resolved Agent
so callers can consume progress without polling or result persistence.

**Why this priority**: Streaming is the remaining invocation mode required by
the active Router Internal v3 contract and Phase 1 Invoke path.

**Independent Test**: A Runtime B A2A endpoint emits its known stream; a Router
dispatch request with `stream=true` receives `accepted`, ordered `chunk`
events, and exactly one terminal event with matching invocation, root task, and
trace identifiers.

**Acceptance Scenarios**:

1. **Given** a valid resolved streaming Agent Card and `Accept:
   text/event-stream`, **when** the Agent emits a valid A2A stream, **then**
   the Router returns HTTP 200 and forwards ordered Result Stream Event v2
   values beginning with `accepted` and ending with one terminal event.
2. **Given** a stream containing a valid message, task, status update, or
   artifact update, **when** the Router forwards the event, **then** the value
   is delivered as one JSON `chunk` without concatenating or persisting Agent
   content.
3. **Given** a valid stream, **when** the caller disconnects, **then** the
   Router stops delivery and does not fabricate a successful terminal event.

### User Story 2 - Enforce Bounded SSE Delivery (Priority: P1)

As a platform operator, I need every upstream A2A event and downstream SSE
event to obey explicit byte limits and framing rules so one Agent cannot cause
unbounded buffering or malformed protocol output.

**Why this priority**: Size and framing are trust-boundary controls; accepting
an unbounded event would invalidate the required Router safety policy.

**Independent Test**: Run a stream with events at, below, and above the
configured A2A and SSE limits and inspect raw response bytes and error events.

**Acceptance Scenarios**:

1. **Given** an A2A event whose serialized JSON-RPC result and complete
   upstream SSE block are within the effective configured/Card bound, **when**
   it is received, **then** the Router forwards it without full-stream
   buffering.
2. **Given** an A2A event larger than the required A2A event limit, **when** it
   is received after acceptance, **then** the Router emits one correlated
   `failed` Result Stream Event v2 with `AGENT_RESPONSE_TOO_LARGE` if the SSE
   response is still writable and stops reading further events.
3. **Given** a serialized Result Stream Event whose compact UTF-8 JSON data
   exceeds the required SSE event limit, **when** it is about to be written,
   **then** the Router emits a bounded correlated failure where possible and
   never writes a truncated or multi-line JSON value.
4. **Given** any valid output event, **when** its raw bytes are inspected,
   **then** it contains exactly one `data:` line, one blank delimiter, escaped
   CR/LF inside JSON strings, and an immediate flush.

### User Story 3 - Preserve Streaming Lifecycle and Failure Facts (Priority: P1)

As a platform operator, I need streaming failures, timeouts, cancellations,
and Ledger interruptions to remain explicit and correlated so incomplete output
cannot be mistaken for success.

**Why this priority**: A stream without a first-terminal and audit boundary
cannot safely close an invocation.

**Independent Test**: Exercise endpoint failure, invalid A2A event, timeout,
caller disconnect, interrupted EOF, and Ledger append failure before and after
SSE commitment; validate the Result Stream sequence and metadata-only Ledger
facts.

**Acceptance Scenarios**:

1. **Given** a valid accepted stream, **when** it ends before a terminal A2A
   event, **then** the Router reports interrupted delivery and does not emit
   `completed` or a successful Ledger terminal fact.
2. **Given** an Agent or protocol failure after SSE commitment, **when** the
   stream remains writable, **then** the Router emits exactly one correlated
   `failed` terminal event with the classified Platform Error and closes.
3. **Given** a deadline or caller cancellation during an accepted stream,
   **when** the first terminal outcome is committed, **then** it is represented
   as `timed_out`/`TIMEOUT` or `canceled`/`CANCELED`, and later Agent events do
   not overwrite it.
4. **Given** a Ledger append fails after an Agent side effect, **when** the
   stream is still writable, **then** the Router emits a correlated delivery
   failure and does not fabricate a durable terminal Ledger event or success.

## Edge Cases

- Request mode and `Accept` mismatch remains a pre-acceptance `406` with no
  Ledger fact or Agent request.
- The resolved Card disables streaming or declares an unsupported profile,
  endpoint, capability, or authentication type; the Router fails from routing
  without opening an Agent stream.
- A2A returns an event with changed task/context identity, an unsupported event
  kind, duplicate ordering, a terminal event followed by more data, malformed
  JSON, or an SSE event with multiple data lines.
- A single JSON result is exactly at the configured limit while its complete
  SSE/JSON-RPC envelope is larger than the limit; the complete upstream block
  boundary is authoritative and is never truncated.
- The response writer cannot flush or the caller disconnects immediately after
  headers; no clean terminal success is inferred.
- The A2A client knows a task ID when deadline/cancellation occurs; at most one
  explicit `tasks/cancel` attempt is made, with no retry or alternate route.

## Requirements

### Functional Requirements

- **FR-001**: The Router MUST accept `stream=true` only with the exact
  `text/event-stream` media mode and MUST preserve all existing pre-acceptance
  validation and resolution semantics.
- **FR-002**: The Router MUST call the exact resolved Agent using A2A
  `message/stream` and propagate trusted platform context headers.
- **FR-003**: The Router MUST emit Result Stream Event v2 values beginning with
  `accepted`, preserving zero-based event and chunk order, and ending with
  exactly one `completed`, `failed`, `canceled`, or `timed_out` terminal event.
- **FR-004**: Every streamed chunk MUST retain the outer invocation, root task,
  and trace correlation and MUST remain transient caller data.
- **FR-005**: The Router MUST enforce the required configured A2A event limit
  on both the complete upstream SSE block (the memory boundary) and its raw
  serialized JSON-RPC result before forwarding each upstream event and MUST
  map overflow to `AGENT_RESPONSE_TOO_LARGE`.
- **FR-006**: The Router MUST serialize each downstream event as compact UTF-8
  JSON on exactly one SSE `data:` line, followed by one blank line and an
  immediate flush; it MUST reject malformed, multi-line, truncated, or
  oversized output.
- **FR-007**: The Router MUST enforce the required SSE event limit on the full
  serialized event/frame boundary without buffering the entire stream.
- **FR-008**: The Router MUST validate A2A stream event kind, task/context
  identity, terminal ordering, and protocol errors against the active A2A
  Profile before forwarding data.
- **FR-009**: The Router MUST commit the accepted/routing/running Ledger facts
  in order and MUST commit the terminal fact before emitting a clean terminal
  success event.
- **FR-010**: Timeout, cancellation, endpoint failure, protocol failure, Agent
  execution failure, response overflow, interrupted EOF, and Ledger failure
  MUST remain distinct and correlated at their owning boundary.
- **FR-011**: If an A2A task ID is known when local timeout or cancellation
  occurs, the Router MUST make at most one `tasks/cancel` attempt and MUST NOT
  retry, switch endpoints, or fabricate success.
- **FR-012**: The feature MUST not persist Agent input, output, chunks,
  credentials, raw dependency messages, or a replay cursor in Ledger or query
  responses.
- **FR-013**: The implementation MUST remain inside Router-owned packages and
  active shared contracts and MUST NOT import Control Plane internals or add a
  full Agent Runtime dependency.

### Key Entities

- **Result Stream Event v2**: One transient accepted, chunk, or terminal value
  with stable sequence and correlation identifiers.
- **A2A Stream Event**: One validated message, task, status update, or artifact
  event received from the resolved Agent.
- **SSE Frame**: One compact JSON result event serialized as a single `data:`
  line plus blank delimiter and flushed immediately.
- **Effective Event Bound**: The explicit configured limit applied to the full
  serialized A2A or SSE event; no inferred default is allowed.

### Runtime/Platform Boundary

- **Platform-owned behavior**: media negotiation, exact route resolution,
  context propagation, A2A event validation, byte limits, SSE framing,
  lifecycle sequencing, terminal classification, and metadata-only Ledger
  facts.
- **Runtime-owned behavior**: model/tool/workflow execution and the business
  meaning of Agent chunks; the Router forwards opaque JSON values only.
- **Cross-runtime proof**: Runtime B's independent A2A stream must pass through
  the same Router stream path; no Runtime-internal type or storage is shared.

## Success Criteria

### Measurable Outcomes

- **SC-001**: A valid Runtime B stream is delivered with one accepted event,
  zero-based ordered chunks, and exactly one terminal event in 100 consecutive
  test invocations.
- **SC-002**: No emitted SSE `data:` JSON value exceeds the configured SSE
  event limit, and no A2A event above the configured A2A limit reaches the
  caller in 100 boundary tests.
- **SC-003**: Raw stream inspection finds exactly one data line, one blank
  delimiter, and an immediate flush for every emitted event; no CR/LF inside a
  JSON string becomes a physical line break.
- **SC-004**: Every invalid stream scenario produces a distinct correlated
  failure or interrupted-delivery outcome and never a successful terminal
  event.
- **SC-005**: Ledger inspection shows only metadata and preserves the first
  committed terminal outcome under timeout, cancellation, and delivery races.

## Assumptions

- The active Router Internal v3, Result Stream Event v2, Invocation Event 0.3,
  A2A Profile 0.2/protocol 0.3.0, and ADR 0006 are authoritative and require
  no contract version change for this feature.
- Runtime B remains the deterministic cross-runtime streaming fixture; no new
  model provider or Agent Runtime is introduced.
- The existing strict Router configuration already supplies the A2A event and
  Agent response limits; this feature adds a separate required
  `NEKIRO_ROUTER_SSE_EVENT_LIMIT_BYTES` setting for SSE serialization and never
  infers it from another limit.
- The existing request context is the source of caller disconnect and deadline
  cancellation; no result replay or reconnect API is added.

## Non-Goals

- Non-streaming `message/send`, Catalog, Registry, Workspace, Installation,
  Frontend, SDK expansion, result persistence, polling, replay, or reconnect
  cursors.
- General-purpose Agent Runtime behavior such as models, tools, planners,
  workflows, memory, or RAG.
- Supporting Card authentication types other than `none`.
- Introducing retries, caches, alternate endpoints, fallback credentials,
  default limits, or compatibility reads.
