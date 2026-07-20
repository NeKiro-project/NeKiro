# Implementation Plan: Invocation Ledger

**Branch**: `codex/014-invocation-ledger` | **Date**: 2026-07-16 | **Spec**: [spec.md](spec.md)

## Summary

Implement a Router-owned PostgreSQL Ledger package using `pgx/v5`: immutable
Event 0.3 rows, one current Invocation projection, transaction-serialized
append validation, explicit tern migration/readiness, restart-safe reads, and
thin Internal v3 read handlers. The package exposes typed append/read behavior
for later Router integration but does not wire the Router process.

## Technical Context

**Language**: Go 1.26

**Dependencies**: `pgx/v5`, `tern/v2`, existing `contracts` runtime validators,
Go `net/http` and `encoding/json`

**Storage**: PostgreSQL 17, schema `ledger`

**Testing**: Unit tests plus real PostgreSQL integration tests gated by the
existing explicit database test environment

**Constraints**: exact write scope from Issue #23; no content persistence; no
fallback; implementation precedes mapped tests

## Constitution Check

- Phase 1 loop: PASS, this is the durable `Record` boundary.
- Ownership: PASS, only Router/Ledger writes the new schema.
- Contracts: PASS, Event 0.3 and Router Internal v3 are consumed unchanged.
- Failure/secret safety: PASS, dependencies fail explicitly and the physical
  model has no content, credential, endpoint, or raw-error column.
- SDD/review: PASS for implementation entry; independent Review and Converge
  remain root-owned delivery gates.

## Design

### Package boundary

`apps/a2a-router/internal/ledger/` owns domain errors, Store, SQL, migration,
readiness, and tests. `apps/a2a-router/internal/api/ledger_handler.go` adapts
Store reads to Router Internal v3 JSON without owning authentication.

### Append transaction

1. Validate Event 0.3 and parse its exact RFC3339 timestamp.
2. Begin a PostgreSQL transaction.
3. For sequence zero, require `created/pending`; when nested, lock and validate
   the parent projection before inserting the child projection.
4. For later sequence, lock the Invocation projection and load its ordered
   events. Re-run the contract lifecycle validator and accept the candidate.
5. Insert the immutable event and insert/update the projection.
6. Commit. Any error rolls back both changes.

The projection row lock serializes one Invocation. Unique constraints provide
the final event-ID and sequence conflict guard. There is no retry.

### Persistence

Events store fixed scalar metadata and an optional fixed error code. Safe
Platform Error v4 messages are reconstructed on read. The projection stores
the same immutable context plus current status, optional latency/error code,
and first/latest timestamps. No JSON payload column exists.

### Reads

`GetInvocation(workspaceID, invocationID)` selects the projection and ordered
events in a repeatable-read, read-only transaction. `GetTrace(workspaceID,
traceID)` selects projections by `created_at, invocation_id`, then performs the
stable parent-before-child ordering required by the active Trace validator.
Query predicates include Workspace; absent and foreign values are both
`ErrNotFound`.

### Migration and readiness

An embedded tern migration creates schema, tables, checks, and indexes.
`Migrate(..., "up")` is explicit. `CheckSchema` verifies expected version and
owned relation/column/index/constraint shape; serving integration must call it
but is outside this branch.

## Project Structure

```text
apps/a2a-router/internal/
├── api/ledger_handler.go
└── ledger/
    ├── 001_ledger.sql
    ├── errors.go
    ├── migrations.go
    ├── store.go
    ├── store_test.go
    └── postgres_integration_test.go
specs/014-invocation-ledger/
```

## Post-Design Constitution Check

PASS. The design stays inside Ledger ownership, consumes active contracts,
keeps result/content data transient, and adds no alternate behavior.
