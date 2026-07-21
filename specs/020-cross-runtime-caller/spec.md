# Feature Specification: Cross-Runtime Caller Sample

**Feature Branch**: `codex/020-cross-runtime-caller`

**Created**: 2026-07-21

**Status**: Draft

**Input**: GitHub Issue #29 / parent Spec 010 T010

## User Scenarios & Testing

### User Story 1 - Receive and compose a managed root invocation (Priority: P1)

As the platform acceptance suite, I need an independently implemented second
Runtime that receives a root A2A call and returns a deterministic combined
result, so cross-runtime value is demonstrated without adding runtime behavior
to NeKiro core.

**Why this priority**: The second sample is the missing proof that the Router
and Ledger boundary is runtime-agnostic rather than a wrapper around Runtime B.

**Independent Test**: Start Runtime A with explicit configuration, send it a
valid Profile request containing a deterministic fixture input, and verify the
response contains the fixed Runtime A marker and the nested Runtime B result.

**Acceptance Scenarios**:

1. **Given** a valid root request and complete Runtime A configuration, **when**
   the request reaches the sample, **then** Runtime A returns one valid A2A
   result whose combined payload is deterministic for the input.
2. **Given** a missing, blank, malformed, or out-of-range required setting,
   **when** Runtime A starts, **then** startup fails and no inferred endpoint,
   credential, target, limit, or identity is used.
3. **Given** an invalid root A2A message, **when** Runtime A receives it,
   **then** the adapter returns an explicit protocol/validation failure and no
   nested call is attempted.

### User Story 2 - Perform a Router-mediated nested call (Priority: P1)

As a managed Agent, I need to invoke Runtime B through the thin Agent SDK and
the Agent Router, so the child call receives platform authorization and remains
in the same invocation lineage.

**Why this priority**: Direct URL calls would bypass the Workspace-bound
credential, exact-version resolution, policy hooks, and Ledger recording.

**Independent Test**: Exercise Runtime A behind an Agent Router test server,
capture the SDK request, and verify it contains only the contract-permitted
target/capability/input fields, uses the configured bearer credential, and
produces the nested result without any direct Runtime B request.

**Acceptance Scenarios**:

1. **Given** a valid root context, **when** Runtime A invokes its configured
   callee, **then** the SDK sends exactly one request to the Router Agent v1
   destination and never constructs or contacts a target Agent URL.
2. **Given** a successful child result, **when** Runtime A composes the root
   result, **then** Workspace ID, root Task ID, and Trace ID remain exact and
   the child Invocation ID is distinct with the root Invocation as parent.
3. **Given** a Router rejection or dependency failure, **when** the nested call
   fails, **then** Runtime A returns the failure to the A2A boundary and does not
   claim a successful combined result or retry/alternate the call.

### User Story 3 - Demonstrate runtime isolation (Priority: P2)

As a platform maintainer, I need evidence that Runtime A and Runtime B share no
runtime-internal types or storage, so either implementation can evolve behind
the same platform contracts.

**Why this priority**: Isolation is a governance requirement of ADR 0003 and a
reviewable acceptance property, not an implementation convenience.

**Independent Test**: Inspect module boundaries and run Runtime A tests without
Runtime B packages or storage; the only shared dependencies are versioned
platform contracts, the thin SDK, and the A2A wire protocol.

**Acceptance Scenarios**:

1. **Given** a clean checkout, **when** Runtime A is built from its nested
   module, **then** its Runtime framework dependency is present only below
   `agents/runtime-a/` and no platform core package imports it.
2. **Given** two concurrent root calls, **when** both perform nested calls,
   **then** their contexts and deterministic results do not cross or reuse
   process-local Runtime B state.

## Edge Cases

- Missing or whitespace-padded required environment values fail startup.
- Unsupported A2A method, missing message ID, wrong role, wrong part kind, or
  extra fixture fields fails before SDK invocation.
- Invalid platform context headers fail the root request rather than being
  synthesized.
- A nested result with changed root Task/Trace correlation is rejected by the
  SDK and cannot be presented as success.
- Router and nested failures remain visible; no retry, cache, alternate route,
  stale result, empty result, or degraded success is allowed.

## Requirements

### Functional Requirements

- **FR-001**: Runtime A MUST live under `agents/runtime-a/` as an isolated
  sample boundary with its own pinned Go module.
- **FR-002**: Runtime A MUST use `trpc-agent-go` `v1.10.0` only inside that
  sample boundary; the framework MUST NOT be imported by Control Plane, Router,
  contracts, or `sdks/agent-sdk`.
- **FR-003**: Runtime A MUST accept the active A2A Profile root request and
  validate the message shape before running its Runtime agent.
- **FR-004**: Runtime A MUST require explicit, validated configuration for its
  own Agent ID, listen address, Router URL, Router bearer credential, target
  Agent ID, capability, response limit, and SSE event limit. No value may be
  inferred or defaulted.
- **FR-005**: Runtime A MUST derive platform context only from the authenticated
  managed transport headers and MUST pass exact Invocation ID, Workspace ID,
  root Task ID, Trace ID, and Agent ID into the SDK.
- **FR-006**: Runtime A MUST invoke Runtime B exactly once through
  `sdks/agent-sdk` and the Router Agent v1 endpoint; direct target endpoint
  calls are forbidden.
- **FR-007**: Runtime A MUST return a deterministic combined JSON result that
  identifies Runtime A and embeds the validated child result without leaking
  credentials, raw dependency errors, or alternate content.
- **FR-008**: Runtime A MUST preserve root/child lineage: the SDK request uses
  the root Invocation as parent, and the child result MUST retain the root Task
  and Trace identifiers.
- **FR-009**: Runtime A MUST propagate validation, Router, protocol, and
  dependency failures explicitly; it MUST NOT add retries, caches, alternate
  sources/routes, compatibility branches, or empty-success fallbacks.
- **FR-010**: Runtime A tests MUST prove deterministic JSON behavior, required
  configuration failures, invalid input rejection, exact SDK request shape,
  correlation rejection, direct-URL absence, concurrency isolation, and
  content/secret exclusion.

### Key Entities

- **Runtime A Configuration**: Required deployment values for the isolated
  sample process; includes listener, Router destination/credential, callee
  identity, capability, and explicit byte limits.
- **Platform Context**: Trusted root Invocation, Workspace, root Task, Trace,
  and Runtime A identity propagated from managed transport headers.
- **Combined Result**: Deterministic Runtime A response containing the child
  result while preserving the outer A2A message contract.

### Runtime/Platform Boundary

- **Platform-owned behavior**: Agent Card resolution, Workspace authorization,
  Router-mediated nested invocation, lineage creation, Ledger facts, and the
  SDK contract.
- **Runtime-owned behavior**: Runtime A's Agent/Runner/Event execution and its
  deterministic result composition; no model, tool, workflow, memory, or
  runtime state is promoted into NeKiro core.
- **Cross-runtime proof**: Runtime A uses `trpc-agent-go` while Runtime B uses
  the direct `a2a-go` implementation; they share only the A2A wire protocol,
  platform contracts, and thin SDK, and one Router-mediated parent-child call
  is verifiable in Ledger lineage.

## Success Criteria

### Measurable Outcomes

- **SC-001**: 100% of valid Runtime A root requests in the focused test suite
  return the same byte-equivalent combined JSON for the same input.
- **SC-002**: 100% of focused nested-call tests show exactly one SDK request to
  the Router and zero requests to a Runtime B target URL.
- **SC-003**: 100% of accepted nested results preserve the exact root Task ID,
  Trace ID, Workspace ID, and distinct parent/child Invocation IDs.
- **SC-004**: Every required-setting missing/blank/invalid case fails before a
  listener accepts traffic, with zero inferred configuration values.
- **SC-005**: A race-enabled test run with at least 100 concurrent Runtime A
  calls reports no context leakage, duplicate child identity, or cross-call
  result contamination.
- **SC-006**: Static and runtime inspection finds zero Runtime framework imports
  outside `agents/runtime-a/` and zero credentials, raw dependency details, or
  Agent input/output content in the sample's logs or platform facts.

## Assumptions

- Runtime B and the Agent SDK contracts already merged from Specs 015 and 019
  are the only callee and nested-call surfaces used by this feature.
- The active A2A Profile accepts a JSON message result for the root sample call;
  streaming Runtime A behavior is owned by the final acceptance feature, not
  this child slice.
- The final Compose process supplies all required Runtime A settings explicitly;
  this feature does not add deployment defaults.

## Non-Goals

- No Console UI, Control Plane, Router core, contracts, or SDK feature expansion.
- No direct Agent URL, result replay/persistence, retry/cache, alternate route,
  compatibility runtime branch, deployment/orchestration, or model/tool/
  workflow/memory behavior.
- No replacement of Runtime B or sharing of its in-memory task state.
