# Feature Specification: Invocation Ledger

**Feature Branch**: `codex/014-invocation-ledger`

**Created**: 2026-07-16

**Status**: Implemented in WIP branch; non-integration verified; pending PostgreSQL integration, Review, and Converge

**Input**: GitHub Issue #23 and Spec 010 T004.

## Context

The Router needs a durable, metadata-only source of Invocation lifecycle facts.
This slice owns the PostgreSQL event store, its transactionally maintained read
projection, schema migration/readiness checks, and Workspace-scoped internal
read handlers. Router process wiring, service authentication, routing, Agent
transport, and final Compose integration are owned by other slices.

## Clarifications

### Session 2026-07-16

- A successfully committed `created` event defines Invocation acceptance.
- Every append locks the Invocation projection, validates the full lifecycle,
  inserts exactly one immutable event, and advances the projection in the same
  transaction. A failed transaction changes neither table.
- Event IDs are globally unique and `(invocation_id, sequence)` is unique.
  Sequence and stream chunk index both begin at zero and are gap-free.
- The first committed terminal event is immutable. A later append is rejected.
- Context is exact and immutable. A child references an existing running parent
  in the same Workspace and inherits its root Task and Trace; its Agent caller
  must be the parent's target Agent.
- Reads require an explicit Workspace path value. Missing and foreign resources
  both return not found; storage failure remains dependency failure.
- Only fixed error codes are stored. Safe contract messages are reconstructed
  from Platform Error v4 when events are read.
- No input, output, chunk value, endpoint, credential, raw dependency text, or
  Runtime telemetry is accepted by the persistence model.

## User Scenarios & Testing

### User Story 1 - Commit Immutable Lifecycle Facts (Priority: P1)

As the Router, I need to append each accepted Invocation lifecycle fact and
advance its projection atomically so execution and audit state cannot disagree.

**Independent Test**: Append a complete lifecycle against real PostgreSQL and
verify gap-free events, matching projection, immutable terminal state, and no
partial change after invalid or conflicting appends.

**Acceptance Scenarios**:

1. A valid root lifecycle commits `created -> routing -> started -> terminal`
   with sequences `0..n` and a projection matching the last event.
2. Invalid sequence, context, status/type, time, chunk order, duplicate ID, or
   post-terminal append is rejected without changing either table.
3. Concurrent appends for one Invocation commit at most the one valid next
   event; no gap, duplicate sequence, or second terminal is created.

### User Story 2 - Preserve Nested Lineage (Priority: P1)

As the Router, I need child Invocations bound to committed parent facts so
nested calls cannot forge Workspace or lineage.

**Independent Test**: Create a running parent, append one valid child, and prove
that absent, terminal, cross-Workspace, mismatched-trace, mismatched-root, and
wrong-Agent parents are rejected without a child fact.

**Acceptance Scenarios**:

1. A valid child has a new Invocation ID, the parent's Workspace/root/Trace,
   the parent ID, and an Agent caller equal to the parent's target Agent.
2. Parent validation and child creation occur in one transaction with the
   parent projection locked, so concurrent terminalization is deterministic.

### User Story 3 - Read Invocation and Trace Metadata After Restart (Priority: P1)

As an internal Control Plane caller, I need Workspace-scoped Invocation and
Trace reads that survive Router reconstruction and never expose Agent content.

**Independent Test**: Reconstruct the Store over a new pool and read one
Invocation and one parent-child Trace in deterministic order.

**Acceptance Scenarios**:

1. Invocation detail contains the projection and all ordered Event 0.3 facts.
2. Trace results are ordered by creation time and Invocation ID and contain
   each projection once.
3. Missing or foreign-Workspace facts are not found, while database failure is
   distinguishable and never returned as empty success.

## Requirements

- **FR-001**: The Ledger MUST own a dedicated PostgreSQL schema containing an
  append-only event table and a transactionally maintained projection table.
- **FR-002**: Each event MUST have a globally unique event ID and a gap-free,
  zero-based sequence unique within its Invocation.
- **FR-003**: Append MUST validate Invocation Event 0.3, immutable context,
  lifecycle transition, stream chunk order, nondecreasing timestamp, and first
  terminal wins before committing.
- **FR-004**: Event insert and projection insert/update MUST commit atomically.
- **FR-005**: A child `created` append MUST validate and lock an existing
  running parent and enforce exact Workspace/root/Trace/Agent-caller lineage.
- **FR-006**: The persistent schema and Go model MUST have no field capable of
  storing input, output, chunk value, endpoint, credential, raw error text, or
  Runtime telemetry.
- **FR-007**: Invocation reads MUST return one Workspace-bound projection and
  ordered events; Trace reads MUST return Workspace-bound projections in
  deterministic parent-before-child order, with `created_at, invocation_id`
  ordering among otherwise eligible entries.
- **FR-008**: Missing and foreign-Workspace reads MUST return not found;
  dependency failure MUST remain distinct from not found or empty success.
- **FR-009**: Migration MUST be explicit and idempotent upward; readiness MUST
  reject missing, stale, or structurally incompatible Ledger schema.
- **FR-010**: Store reconstruction over the same database MUST preserve exact
  events, projections, lineage, ordering, and terminal state.
- **FR-011**: HTTP read handlers MUST implement only the Router Internal v3
  Invocation/Trace response shapes and leave authentication/process wiring to
  the Router owner.
- **FR-012**: No retry, cache, alternate store, reconciliation, stale read,
  degraded success, or compatibility fallback may be added.

## Non-Goals

- Router command/configuration, listener setup, internal authentication,
  resolution, Agent transport, cancellation, or Compose wiring.
- Public Gateway reads, Console UI, result persistence/replay, event mutation,
  retention, billing, analytics, or a repair/reconciliation worker.

## Success Criteria

- **SC-001**: Real PostgreSQL tests prove atomic lifecycle append/projection,
  zero-based gap-free sequence, and exactly one terminal under concurrency.
- **SC-002**: Real PostgreSQL tests prove valid nested lineage and rejection of
  every parent/context mismatch without a partial child record.
- **SC-003**: Restart tests return byte-equivalent contract metadata and stable
  Invocation/Trace ordering from a reconstructed Store.
- **SC-004**: Schema inspection and API responses contain none of the prohibited
  content categories.

## Fallback Classification

- **Remove**: none.
- **Keep**: none.
- **Needs policy**: none.

`Fallback delta: removed 0, retained 0, added 0, net 0`

`Added fallback evidence: none`
