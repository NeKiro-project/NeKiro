# Implementation Plan: Invocation and Trace Metadata Reads

**Branch**: `018-invocation-trace-reads` | **Date**: 2026-07-16 | **Spec**:
[spec.md](spec.md)

**Input**: GitHub #27 / Spec 010 T008, active Northbound Invocation API v4,
Router Internal API v3, and existing Router Ledger read adapters.

## Summary

Expose Workspace-authorized Invocation and Trace metadata through Gateway.
Extend the existing Router client with exact v3 GET paths, add a Control Plane
read service that authorizes through Workspace, wire the already-owned Router
Ledger handler to authenticated routes, and preserve explicit public failure
semantics. No contract version or durable schema changes are required.

## Technical Context

**Language/Version**: Go 1.26

**Primary Dependencies**: Go `net/http`, `encoding/json`, existing `pgx/v5`
Ledger Store, active runtime contract validators.

**Storage**: Existing Router-owned PostgreSQL Ledger only; no new tables and no
Control Plane read copy.

**Testing**: Gateway unit/HTTP tests, Router route/auth tests, Router client
direction tests, existing Ledger integration/read tests, full Go tests, vet,
race, Compose config, and diff checks.

**Deadline**: Reuse the required `NEKIRO_GATEWAY_INVOCATION_DEADLINE_MS` for
the Workspace authorization and Router metadata read request; no new default
or unbounded dependency call is introduced.

**Metadata response bound**: Require the independent strict
`NEKIRO_GATEWAY_METADATA_RESPONSE_MAX_BYTES` setting. Gateway reads at most one
bounded Router body, validates it against the active runtime DTO, and only then
commits 200.

**Project Type**: Control Plane Gateway plus Router internal read boundary.

## Constitution Check

- **Phase 1 loop**: PASS. This completes the observable `Record` read side.
- **Ownership**: PASS. Workspace authorizes; Router Ledger reads its own store;
  Gateway never imports Ledger storage.
- **Contracts**: PASS. Consumes Northbound v4 and Router Internal v3 unchanged.
- **Runtime independence**: PASS. No Agent Runtime behavior is introduced.
- **Failure safety**: PASS. Missing, forbidden, and dependency outcomes stay
  distinct; no empty-success or raw dependency fallback exists.
- **Traceability**: PASS. Every requirement maps to an explicit task and
  post-implementation test.

## Architecture and Flow

```text
Caller
  -> Gateway auth + path validation
  -> Workspace.GetWorkspace owner authorization
  -> RouterClient GET /internal/v3/workspaces/{workspaceId}/...
  -> Router auth + LedgerHandler + Ledger Store validation
  -> Gateway JSON response or safe public error
```

The Router client derives only the contract-defined sibling path on the same
configured Router origin. It never accepts a target endpoint from the caller.

## Error Mapping

| Source | Public result |
| --- | --- |
| Gateway auth/path | v4 `UNAUTHENTICATED` / `VALIDATION_ERROR` |
| Workspace not found/forbidden/invalid | v4 `NOT_FOUND` / `FORBIDDEN` / `VALIDATION_ERROR` |
| Router HTTP 404 | v4 `NOT_FOUND` |
| Router auth, malformed JSON/media, transport, or 5xx | v4 `DEPENDENCY_ERROR` |
| Read deadline/cancellation | v4 `DEPENDENCY_ERROR` |
| Router HTTP 200 with invalid/oversized body | v4 `DEPENDENCY_ERROR` |
| Router HTTP 200 | exact bounded, validated metadata JSON |

The public handler never forwards internal error bodies or dependency details.

## Write Scope

Owned:

- `specs/018-invocation-trace-reads/**`
- `apps/control-plane/internal/invocation/router_client.go` and read service
- `apps/control-plane/internal/gateway/invocation_read_handler.go` and tests
- `apps/control-plane/cmd/control-plane/main.go` wiring/tests
- `apps/a2a-router/internal/api/ledger_handler.go` route/auth tests
- `apps/a2a-router/cmd/a2a-router/main.go` and assembly tests

Referenced but not re-owned: `contracts/**`, `apps/a2a-router/internal/ledger/**`,
Workspace/Catalog persistence, Frontend, SDK, and Agent samples. The required
metadata limit must be surfaced in local/deployment environment documentation;
final Compose/CI acceptance remains parent-owned.

## Complexity Tracking

No constitution violations require justification. The read service and route
adapter are the minimum seams needed to preserve module ownership.

## Fallback Report

```text
Fallback delta: removed 0, retained 0, added 0, net 0
Added fallback evidence: none
```
