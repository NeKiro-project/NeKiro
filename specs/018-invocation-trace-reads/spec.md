# Feature Specification: Invocation and Trace Metadata Reads

**Feature Branch**: `018-invocation-trace-reads`

**Created**: 2026-07-16

**Status**: Implemented, integration-verified, independently reviewed, and converged

**Input**: GitHub #27 / Spec 010 T008, active Northbound Invocation API v4,
Router Internal API v3, Invocation Event 0.3, and the Router-owned Ledger
implementation.

## Clarifications

### Session 2026-07-16

- Q: How does Gateway prevent a malformed or content-bearing Router 200 body
  from being exposed? A: It reads at most the required
  `NEKIRO_GATEWAY_METADATA_RESPONSE_MAX_BYTES`, rejects duplicate/unknown JSON
  members and trailing values, validates the active InvocationDetail/Trace
  contract, and maps every failure to `503 DEPENDENCY_ERROR`.
- Q: Is the metadata response limit inferred from the invocation request or
  Router limit? A: No. It is a separate required strict byte configuration;
  no other limit is reused or defaulted.

## Context

Spec 012 provides the public Invocation create boundary and Specs 014, 016,
and 017 create Router-owned metadata facts. This feature makes the `Record`
side observable through two Workspace-scoped Northbound reads without exposing
Agent input, output, chunks, credentials, or dependency internals. The Gateway
remains the only public entry point; the Router remains the only Ledger reader.

## User Scenarios & Testing

### User Story 1 - Inspect one Invocation (Priority: P1)

As a Workspace owner, I need to inspect one managed Invocation and its ordered
metadata-only lifecycle facts after the Router has recorded it, including after
the Router process is reconstructed.

**Independent Test**: An authorized owner requests one exact Workspace and
Invocation pair through Gateway, the Router read proxy returns the persisted
projection and ordered Event 0.3 facts, and the same result is returned after a
fresh Router Ledger store is constructed.

**Acceptance Scenarios**:

1. **Given** an enabled or terminal Invocation in the caller's Workspace,
   **when** `GET /v4/workspaces/{workspaceId}/invocations/{invocationId}` is
   requested, **then** Gateway authorizes the Workspace owner before making
   one authenticated Router Internal v3 read and returns the exact metadata
   response.
2. **Given** a committed non-terminal history after a persistence interruption,
   **when** it is read, **then** the last committed status and ordered facts
   are returned unchanged; no success or terminal event is fabricated.

### User Story 2 - Inspect one Trace lineage (Priority: P1)

As a Workspace owner, I need to inspect all managed Invocations in one Trace so
parent and child calls can be audited without retrieving Agent result content.

**Independent Test**: A fixture Ledger contains root and child projections;
the owner requests the Trace and receives deterministic root-before-child
ordering with exact Workspace and Trace correlation.

**Acceptance Scenarios**:

1. **Given** a Trace with one or more managed Invocations, **when**
   `GET /v4/workspaces/{workspaceId}/traces/{traceId}` is requested, **then**
   Gateway returns the Router's metadata-only lineage in stable parent-before-
   child order.
2. **Given** a Trace from another Workspace, **when** a caller requests it,
   **then** Workspace authorization fails before Router access and no foreign
   projection is exposed.

### User Story 3 - Preserve explicit read failures (Priority: P1)

As a platform operator, I need not-found, forbidden, invalid, and dependency
failures to remain distinguishable so a read outage cannot look like an empty
Trace or a successful response.

**Independent Test**: Exercise unknown Workspace, non-owner, unknown
Invocation/Trace, invalid identifiers, Router authentication failure, Router
unavailability, malformed media, and persistence failure; compare status,
Platform Error code, correlation, and downstream call count.

**Acceptance Scenarios**:

1. **Given** an unknown resource, **when** an authorized read is made, **then**
   Gateway returns `404 NOT_FOUND` and does not return an empty success DTO.
2. **Given** a non-owner caller, **when** a read is made, **then** Gateway
   returns `403 FORBIDDEN` without contacting Router.
3. **Given** Router or Ledger dependency failure, **when** a read is made,
   **then** Gateway returns `503 DEPENDENCY_ERROR` with a safe public error and
   no raw dependency message, credential, or empty successful result.

## Functional Requirements

- **FR-001**: Gateway MUST authenticate the caller and create a request Trace
  before any Workspace or Router operation; authentication and path failures
  MUST not contact Router.
- **FR-002**: Gateway MUST strictly validate Workspace, Invocation, and Trace
  identifiers against the active contract grammar before authorization.
- **FR-003**: Workspace MUST authorize the authenticated owner before a read;
  the read path MUST use the existing Workspace-owned authorization boundary
  and MUST NOT read Workspace or Ledger tables from Gateway.
- **FR-004**: For an authorized read, Control Plane MUST make exactly one
  authenticated GET to the corresponding Router Internal v3 path on the same
  explicitly configured Router destination; it MUST NOT use an alternate
  endpoint, cache, retry, or direct database path.
- **FR-005**: A successful Invocation response MUST be the active
  `InvocationDetailResponseV4`; a successful Trace response MUST be the active
  `TraceResponseV4`. Router-owned validation remains authoritative for stored
  Event 0.3 and lineage semantics.
- **FR-006**: The public boundary MUST preserve the distinction between
  `VALIDATION_ERROR`, `UNAUTHENTICATED`, `FORBIDDEN`, `NOT_FOUND`, and
  `DEPENDENCY_ERROR`, with the active Northbound v4 status mapping.
- **FR-007**: Router HTTP 404 MUST remain a public `404 NOT_FOUND`; internal
  authentication failures, malformed responses, wrong media, transport
  failures, and 5xx responses MUST become safe `503 DEPENDENCY_ERROR` results.
- **FR-008**: No successful read, error body, log, or Ledger query response
  MAY contain Agent input, output, result chunks, credentials, tokens, or raw
  dependency details.
- **FR-009**: Reads MUST be restart-safe and deterministic: a fresh Router
  Ledger store returns the same committed projection/event order, and a
  missing resource is not represented as `[]` or an empty success object.
- **FR-010**: The feature MUST consume the active versioned contracts without
  adding historical dual-read, compatibility, fallback, retry, or result
  persistence behavior.
- **FR-011**: Workspace authorization and Router metadata reads MUST use the
  existing required Gateway invocation deadline; deadline or cancellation MUST
  remain a safe `503 DEPENDENCY_ERROR` and MUST NOT become an empty success.
- **FR-012**: Gateway MUST enforce the separate required metadata response byte
  limit, strict JSON framing, active DTO validation, Workspace/Trace
  correlation, and metadata-only field set before committing a 200 response.

## Key Entities

- **InvocationDetailResponseV4**: One metadata-only Invocation projection and
  its ordered immutable Event 0.3 facts.
- **TraceResponseV4**: One Trace identity and deterministic parent-before-child
  Invocation projections for one Workspace.
- **Authorized Read**: A read that passed public authentication, identifier
  validation, and Workspace owner authorization before Router access.
- **Router Read Proxy**: The authenticated Control Plane-to-Router GET adapter;
  it does not own Ledger data or make routing decisions.

## Failure and Security Boundary

- Public authentication, invalid identifiers, unknown Workspace, and
  non-owner failures happen in Gateway/Workspace before Router access.
- Router `404` means the authorized Workspace has no matching metadata fact.
- Router `401`, `403`, malformed response, wrong media, transport error, and
  `5xx` are dependency failures at the public boundary; internal status or raw
  body is not exposed.
- A valid 200 response is metadata-only and is returned without adding a
  second public source of Ledger truth.

## Success Criteria

- **SC-001**: Every authorized read makes one Router request with the exact
  Workspace and resource path; every pre-authorization failure makes zero.
- **SC-002**: Invocation and Trace reads remain identical across a fresh
  Router Ledger store reconstruction for 100 repeated reads.
- **SC-003**: Foreign Workspace reads expose no projection, event, result,
  credential, or raw dependency detail in 100 isolation cases.
- **SC-004**: Not-found, forbidden, invalid, and dependency matrices return
  distinct active v4 codes/statuses with the Gateway Trace correlation.
- **SC-005**: Static and HTTP inspection finds no retries, caches, alternate
  routes, direct database imports, result content, or secret material in the
  read path.

## Assumptions

- Active Northbound Invocation API v4 and Router Internal API v3 are the sole
  runtime contracts; no older `/v2` read route is served by this feature.
- The existing `NEKIRO_ROUTER_INTERNAL_URL` identifies the Router's v3
  Invocation operation. Read paths use the same scheme/host/port and the
  contract-defined v3 sibling path; this is one destination, not a fallback.
- The existing Router Ledger Store and `LedgerHandler` validate durable
  metadata and lineage before returning a 200 response.
- Request context cancellation is propagated to the Router read request; no
  polling, replay, pagination, or result retrieval API is added. The existing
  required Gateway invocation deadline bounds the read dependency as well.

## Non-Goals

- Invocation creation, A2A transport, streaming, cancellation, or Ledger write
  semantics already delivered by sibling Specs.
- Console UI, SDK, nested Agent calls, sample Runtime A, billing, analytics,
  result replay, input/output retrieval, pagination, or search indexing.
- Direct Control Plane access to Router Ledger tables or Agent endpoints.
- Retries, caches, alternate Router destinations, compatibility reads,
  speculative response limits, or inferred/default credentials.

## Fallback Report

```text
Fallback delta: removed 0, retained 0, added 0, net 0
Added fallback evidence: none
```
