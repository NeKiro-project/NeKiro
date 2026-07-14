# Implementation Plan: Install a Published Agent and Pin an Exact Version

**Branch**: `codex/005-install-agent-pin`
**Date**: 2026-07-15
**Spec**: [spec.md](spec.md)

**Input**: Issue #5 and the active Workspace/Installation contracts frozen by
issues #3 and #4.

## Summary

Complete the `Workspace owner -> Catalog selection -> durable Installation`
slice in the existing Control Plane. The Gateway strictly decodes the
Installation request, including required-versus-empty permission presence. The
Workspace service authenticates and authorizes the owner, calls Catalog through
the existing `CatalogReader` port, validates the exact permission subset, and
constructs one enabled pin. The Workspace PostgreSQL adapter serializes the
current-install uniqueness check under the Workspace row lock and returns the
committed database fact. No new public contract version or runtime process is
introduced.

## Constitution Check

*GATE: Passed before implementation and must be re-checked after tests and
independent Review.*

- **Phase 1 loop - PASS**: This is the approved `Discover -> Install` step.
- **Ownership - PASS**: Workspace owns Installation rows; Catalog owns version
  selection and Card facts; Gateway is the only northbound adapter.
- **Runtime independence - PASS**: Installation never invokes, probes, or
  deploys an Agent Runtime.
- **Contracts - PASS**: Active Installation v2 and Northbound v3 are reused;
  no historical route or dual-read fallback is added.
- **Failure safety - PASS**: Missing required fields, no match, owner denial,
  conflict, and dependency failure stay distinct. Empty permissions are an
  explicit product value, not a failure fallback.
- **SDD and review - PASS**: This feature has independent Spec, Plan, Tasks,
  analysis, mapped tests, Review, remediation, and Converge gates.

## Architecture and Ownership

```text
HTTP POST /v3/workspaces/{workspaceId}/installations
  -> Gateway authentication and strict request presence checks
  -> workspace.Service.Install
  -> Workspace Store: owner/current preflight
  -> CatalogReader: published SemVer selection
  -> exact Card permission validation
  -> Workspace Store: locked uniqueness recheck + insert
```

- Gateway owns JSON shape, required-array presence, authentication ordering,
  trace, and public error mapping.
- Workspace owns owner authorization, request domain rules, permission subset
  validation, Installation facts, and persistence transaction ordering.
- Catalog owns published candidate visibility and SemVer selection. Workspace
  never imports or queries Catalog storage.
- PostgreSQL owns the one-current partial unique index and the transaction that
  locks the Workspace before rechecking/inserting.

## Transaction and Race Semantics

1. Validate authentication, path/request shape, constraint, and permission
   array presence before any persistence or Catalog call.
2. Read the Workspace and invoke the owner policy.
3. Preflight current Installation conflict.
4. Call `CatalogReader.SelectPublished` outside a held Workspace transaction.
   The returned exact Card is the selection linearization point.
5. Validate permissions against that exact Card.
6. Start the Workspace-owned insert transaction, lock the Workspace row,
   recheck current uniqueness, and insert the enabled Installation.
7. Return database `RETURNING` values so timestamps and empty arrays match the
   committed representation.

A later Catalog disable does not rewrite a committed Installation. Any
Catalog/store failure maps explicitly and no stale or alternate source is used.

## Contract Surface

Issue #5 consumes, without modifying, the active artifacts from #3:

- `contracts/schemas/installation.v2.schema.json`
- `contracts/openapi/control-plane.v3.yaml`
- `contracts/schemas/platform-error.v3.schema.json`
- `contracts/installation_contracts.go`

The operation is `POST /v3/workspaces/{workspaceId}/installations`, with `201`
success and the fixed `400/401/403/404/409/503` error set. The request must
contain `agentId`, `versionConstraint`, and `acceptedPermissions`; an empty
array is valid while omitted or null is not.

## Baseline Findings and Required Corrections

- The dependent branch already implements the broad Install service, store,
  and route, but its focused evidence is sparse.
- The permission subset helper must preserve a non-nil empty slice so PostgreSQL
  stores `{}` and JSON returns `[]`, not SQL `NULL` or JSON `null`.
- The HTTP adapter must distinguish a missing required permission array from an
  explicit empty array before constructing the domain request.
- The PostgreSQL insert must return its committed row values to avoid a
  pre-storage timestamp becoming a second fact.

## Tests and Files

Runtime files to verify or adjust:

- `apps/control-plane/internal/workspace/service.go`
- `apps/control-plane/internal/workspace/postgres/store.go`
- `apps/control-plane/internal/gateway/workspace_handler.go`

Focused evidence:

- Service unit tests for owner, selection, pre-release/build tie, permissions,
  empty set, duplicate conflict, dependency ordering, and no Catalog fallback.
- Gateway HTTP tests for required/empty/null permission arrays, exact errors,
  trace, owner denial, and no service call on invalid input.
- PostgreSQL integration tests for exact persisted fields, restart, empty array,
  newer publication immutability, 100-way race, and dependency failures.

## Implementation Order

1. Freeze this #5 Spec, Plan, research, data model, contract guide,
   checklist, and Tasks; update the active plan pointer.
2. Run cross-artifact analysis before code edits.
3. Correct the domain/adapter/HTTP gaps above without changing active public
   contracts.
4. Add mapped unit, HTTP, and PostgreSQL tests after implementation.
5. Run full static, race, integration, Compose, and fallback verification.
6. Run independent Review, resolve findings through Spec/Tasks, run Converge,
   update handoff/Issue #5, and create the target-main PR.

## Complexity Tracking

No constitution violations require justification. Presence-aware HTTP decoding
is a boundary adapter detail required to honor the existing schema's required
array without changing the shared domain DTO or inventing a fallback.
