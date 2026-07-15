# Tasks: Workspace Installation Inspection

**Input**: Design documents from `/specs/007-installation-inspection/`.

**Prerequisites**: `spec.md`, `plan.md`, `research.md`, `data-model.md`,
`contracts/installation-inspection-api.md`, and `quickstart.md`.

**Scope**: Issue #7 exact Installation read and bounded current/history list
evidence. Do not change lifecycle, Catalog, Router, Ledger, SDK, or Console.

## Phase 1: Observe And Design Gate

- [X] T001 Read AGENTS.md, constitution, architecture/contract docs, Spec
  003/004/005, active v3/Installation v2 contracts, and current GET/list/store
  implementation; record baseline in `research.md`.
- [X] T002 Freeze owner-first authorization, historical row inclusion,
  deterministic cursor semantics, explicit empty/dependency distinction, and
  zero-added-fallback policy in `spec.md`.
- [X] T003 Create `plan.md`, `data-model.md`, contract guide, quickstart, and
  requirements checklist for Issue #7.
- [X] T004 Run cross-artifact analysis before runtime edits and record the
  result in this task file.

## Phase 2: Existing Runtime Audit And Narrow Corrections

- [X] T005 Verify `workspace.Service.GetInstallation` and
  `ListInstallations` preserve owner-first authorization, cross-Workspace
  not-found, current/history inclusion, invalid cursor failure, and no empty
  fallback; modify only if mapped tests prove a gap (FR-001 through FR-011).
- [X] T006 Verify PostgreSQL point/keyset queries preserve all committed fields,
  use strict `(installed_at, installation_id)` continuation, retain uninstalled
  rows, and return dependency failures distinctly (FR-001 through FR-010).
- [X] T007 Verify the Gateway GET adapters require explicit limit, authenticate
  before service calls, map v3 errors, preserve trace, and emit no fact on
  failure (FR-001, FR-003, FR-008 through FR-013).

## Phase 3: Post-Implementation Tests

- [X] T008 [P] [US1] Add service unit tests in
  `apps/control-plane/internal/workspace/service_test.go` for complete current
  and uninstalled facts, owner authorization ordering, unknown Workspace,
  unknown Installation, cross-Workspace mismatch, invalid cursor, and
  dependency propagation (US1/US3; FR-001, FR-002, FR-008, FR-009, FR-010).
- [X] T009 [P] [US2] Add service/cursor unit tests in
  `apps/control-plane/internal/workspace/service_test.go` and
  `apps/control-plane/internal/workspace/cursor_test.go` for equal timestamp
  tie ordering, bounded continuation without duplicates/omissions, cursor
  Workspace/limit mismatch, empty items, and no next cursor (US2; FR-003 through
  FR-007).
- [X] T010 [P] [US1] Add contract tests in
  `contracts/workspace_api_contracts_test.go` proving both active GET routes,
  Installation v2 complete/historical objects, InstallationList empty/object
  shapes, required limit, and v3 error response mappings (FR-001 through FR-013).
- [X] T011 [US2] Add real PostgreSQL integration tests in
  `apps/control-plane/internal/workspace/postgres/inspection_integration_test.go`
  for current/history ordering, equal-time tie-breaks, bounded keyset pages,
  empty history, injected query/scan failure, and direct store field equality
  (US1/US2; SC-001 through SC-003).
- [X] T012 [US1] Add restart and history integration coverage in
  `apps/control-plane/internal/workspace/integration/workspace_test.go` by
  reconstructing the pool/store/service and comparing exact current and
  uninstalled facts and full pagination traversal (US1/US2; SC-001, SC-002).
- [X] T013 [US3] Add HTTP tests in
  `apps/control-plane/internal/gateway/workspace_handler_test.go` for read/list
  success, explicit empty array, pagination query/cursor, unauthenticated,
  non-owner, unknown Workspace/Installation, cross-Workspace, invalid query,
  dependency failures, trace equality, and absence of Installation facts on
  errors (US3; FR-003, FR-007 through FR-013).

## Phase 4: Verification, Review, And Converge

- [X] T014 Run focused unit/contract/HTTP tests, then broad `go test`, race,
  vet, build, integration-if-configured, `go mod tidy -diff`, and
  `git diff --check`; record exact outcomes in this file and quickstart.
- [X] T015 Run review against Issue #7 acceptance, Spec, Plan,
  Tasks, active contracts, and constitution; fix all valid High/Medium findings
  through tasks/code/tests and rerun verification.
- [X] T016 Run Converge audit, mark every completed task `[X]`, revalidate the
  requirements checklist, inspect the final diff, configure local Git identity,
  and commit the complete Issue #7 branch.

## Dependency And Write Scope

Phase 2 is serial because service, store, and Gateway behavior share one read
contract. Unit, contract, and HTTP tests can be developed in parallel after the
runtime audit; PostgreSQL schema-resetting packages run serially. Only the
listed Workspace/Gateway/contract test files and Issue #7 docs may be changed.

## Cross-Artifact Analysis Evidence

The read-only analysis after task generation found no unresolved constitution
conflict, public-contract change, ownership violation, or unsupported fallback.
All Issue acceptance criteria map to implementation audit tasks and post-code
tests:

| Acceptance area | Tasks | Evidence |
| --- | --- | --- |
| Exact current/history read | T005, T006, T008, T011, T012, T013 | Complete committed fields, terminal timestamps, restart equality |
| Bounded deterministic history | T005, T006, T009, T011, T012, T013 | Keyset order, equal-time tie, no duplicate/omission traversal |
| Historical rows retained | T006, T008, T011, T012 | Uninstalled rows returned and restart-readable |
| Empty vs dependency failure | T005, T006, T008, T011, T013 | `items: []` only after successful query; dependency maps to 503 |
| Failure/authorization semantics | T005, T007, T008, T010, T013 | 400/401/403/404/503 and no fact leakage |
| Workspace-only reads | T005, T006, T007, T008, T011 | No Catalog port/query/mutation |
| Zero fallback | T005-T016 | Inventory and final report |

**Analysis result**: PASS. Implementation may proceed within the approved
scope.

## Verification Evidence

- Focused: `go test -count=1 ./contracts`,
  `go test -count=1 ./apps/control-plane/internal/workspace/...`, and
  `go test -count=1 ./apps/control-plane/internal/gateway` passed.
- Broad: `go test ./...`, `go vet ./...`, `go build ./...`,
  `go mod tidy -diff`, and `git diff --check` passed.
- Race: `go test -race -count=1 ./apps/control-plane/internal/workspace/... ./apps/control-plane/internal/gateway ./contracts` passed.
- PostgreSQL: integration-tag packages compiled with `go test -tags=integration -run '^$' ...`; runtime integration was not run because `NEKIRO_TEST_DATABASE_URL` was unavailable.
- Review: final diff checked against Issue #7 scope, active contracts, ownership boundaries, error semantics, and zero-fallback policy; no High/Medium findings remained.

## Fallback Delta

Fallback delta target remains `removed 0, retained 2, added 0, net 0`. The
retained behaviors are explicit empty-array and required-limit contract
semantics; no new fallback may be added.
