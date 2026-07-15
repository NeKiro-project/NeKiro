# Implementation Plan: Workspace Installation Inspection

**Branch**: `codex/issue-7-installation-inspection` | **Date**: 2026-07-15 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `/specs/007-installation-inspection/spec.md`

## Summary

Close the Issue #7 evidence gap for the existing Northbound v3 Installation
read/list operations. Reuse the Workspace-owned PostgreSQL facts, owner policy,
keyset cursor, and error mappings already present from Spec 003/005. Add only
the focused contract, service, PostgreSQL, restart, pagination, authorization,
dependency, and HTTP tests required by the Issue. If testing exposes a
behavioral gap, correct it at the owning boundary without changing the active
contract.

## Technical Context

**Language/Version**: Go 1.26

**Primary Dependencies**: `net/http`, `httptest`, `testing`, pgx/v5, PostgreSQL,
existing contract validator and OpenAPI test helpers

**Storage**: Existing PostgreSQL `workspace.workspaces` and
`workspace.installations` tables from Workspace migration 001

**Testing**: Go unit, contract, HTTP, race, and `integration`-tagged PostgreSQL
tests; restart tests reconstruct a pool/store/service against the same database

**Target Platform**: Linux container/server; local Go development environment

**Project Type**: Go Control Plane Workspace module and Gateway adapter

**Performance Goals**: Bounded list reads return at most 100 rows per page and
use the existing Workspace order index; pagination must be deterministic on
unchanged data.

**Constraints**: Gateway-only Northbound access; owner-first authorization;
Workspace-only data ownership; active Installation v2/Northbound v3 contracts;
no Catalog access, mutation, retry, cache, endpoint probe, alternate source,
default limit, or compatibility fallback.

**Scale/Scope**: Two existing GET operations, one Installation entity, one
opaque cursor format, and the Issue #7 test/documentation slice.

## Constitution Check

- **Phase 1 loop**: PASS. Inspection validates the durable authorization fact
  between Install and later Invoke.
- **Ownership**: PASS. Workspace owns all read/list facts; Gateway only adapts
  HTTP; no Catalog table or service probing is added.
- **Runtime independence**: PASS. No Agent Runtime is imported or invoked.
- **Contracts**: PASS. Active Installation v2 and Northbound v3 are consumed
  unchanged; no new public contract is needed.
- **Invocation lineage**: N/A. Inspection creates no Invocation, Task, Trace
  lineage, or Ledger row; it propagates only the Gateway trace header.
- **Failure safety**: PASS. Unknown, forbidden, invalid, and dependency
  outcomes remain distinct; an empty list is only a successful no-row query.
- **SDD traceability**: PASS. Each FR and acceptance scenario maps to tasks and
  post-implementation tests.
- **Cross-runtime proof**: N/A. This feature reads platform facts only.

## Existing Boundary Audit

1. `workspace.Service.GetInstallation` loads Workspace, authorizes owner, then
   calls the Workspace store using both IDs.
2. `workspace.Service.ListInstallations` validates the explicit limit/cursor,
   loads Workspace, authorizes owner, and calls the Workspace store.
3. PostgreSQL `ListInstallations` uses `limit + 1`, ordered keyset pagination,
   and includes all statuses because no status predicate is present.
4. `WorkspaceHandler` registers both GET routes, parses strict query values,
   authenticates before service calls, writes trace headers, and maps v3
   Workspace errors.
5. Existing tests are insufficient for the Issue's complete acceptance matrix;
   the implementation phase therefore focuses on evidence and boundary gaps.

## File Structure

### Feature Documentation

```text
specs/007-installation-inspection/
├── spec.md
├── checklists/requirements.md
├── plan.md
├── research.md
├── data-model.md
├── contracts/installation-inspection-api.md
├── quickstart.md
└── tasks.md
```

### Runtime And Tests

```text
apps/control-plane/internal/workspace/service.go             # only if audit finds a runtime gap
apps/control-plane/internal/workspace/cursor.go              # only if cursor gap is found
apps/control-plane/internal/workspace/service_test.go         # mapped unit evidence
apps/control-plane/internal/workspace/cursor_test.go          # cursor edge evidence
apps/control-plane/internal/workspace/postgres/store.go       # only if query gap is found
apps/control-plane/internal/workspace/postgres/inspection_integration_test.go
apps/control-plane/internal/workspace/integration/workspace_test.go
apps/control-plane/internal/gateway/workspace_handler.go     # only if adapter gap is found
apps/control-plane/internal/gateway/workspace_handler_test.go # mapped HTTP evidence
contracts/workspace_api_contracts_test.go                    # active route/schema evidence
```

## Implementation Order

1. Freeze this feature's SDD artifacts and run prerequisite/cross-artifact
   analysis.
2. Add mapped unit and contract tests against existing boundaries.
3. Add real PostgreSQL current/history, empty, pagination, and restart tests;
   keep schema-resetting packages serial.
4. Add HTTP authorization, failure, trace, query, and secret-exclusion tests.
5. Correct only verified runtime gaps, then rerun all mapped tests.
6. Run broad static, race, build, diff, and available PostgreSQL checks; update
   task evidence, checklist, and quickstart.
7. Perform independent review/converge audit and commit with the required local
   Git identity.

## Complexity Tracking

No constitution violations require justification.
