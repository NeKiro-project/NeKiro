# Tasks: Invocation and Trace Metadata Reads

**Input**: Design documents from `specs/018-invocation-trace-reads/`

**Tests**: Complete approved implementation before mapped tests, then run the
independent Review and Converge gates.

## Phase 1: SDD Gate

- [X] T001 Observe AGENTS, Spec 010/T008, active v4/v3 contracts, existing
  Workspace authorization, Router Ledger Store, and current process wiring.
- [X] T002 Specify and clarify owner authorization, exact sibling paths,
  response ownership, failure mapping, and no-fallback policy in `spec.md`.
- [X] T003 Plan ownership, flow, write scope, tests, and compatibility in
  `plan.md`, `research.md`, and `data-model.md`.
- [X] T004 Analyze constitution, Spec, Plan, Tasks, contracts, and dependency
  graph before runtime changes.

## Phase 2: Foundational Read Boundary

- [X] T005 Add authenticated Router Internal v3 GET route registration around
  the existing LedgerHandler without changing Ledger storage semantics.
- [X] T006 Extend the existing Control Plane Router client with one-attempt
  Invocation and Trace GET adapters on the configured Router origin.
- [X] T007 Add a Workspace-authorized metadata read service that makes no Router
  call before owner authorization and maps dependency boundaries explicitly.
- [X] T008 Wire the public v4 Invocation/Trace read routes through the Gateway,
  preserving request Trace correlation and safe error mapping.
- [X] T008a Add the separate strict metadata response limit, bounded JSON
  decoding, duplicate/unknown/trailing member rejection, active DTO validation,
  and metadata-only disclosure checks before HTTP 200.

## Phase 3: User Stories and Tests

- [X] T009 Add Router route/auth/readiness tests for exact v3 paths, auth-first
  rejection, 404 mapping, and validated metadata responses.
- [X] T010 Add Router client and read-service tests for exact path, one request,
  owner/foreign Workspace isolation, and dependency failures.
- [X] T011 Add Gateway HTTP tests for success, restart-shaped metadata,
  invalid/auth/forbidden/not-found/dependency outcomes, and zero-call policy.

## Phase 4: Verification, Review, and Converge

- [X] T012 Run focused/full tests, vet, WSL race, Compose config, diff check,
  forbidden-content scans, and fallback audit; record evidence here.
- [X] T013 Obtain independent Spec/Standards Review against AGENTS, active
  contracts, Spec, Plan, Tasks, and write scope.
- [X] T014 Converge every finding into the Spec/Tasks, fix approved gaps, and
  repeat the independent Review before marking this feature complete.

## Dependencies and Execution Order

```text
T001 -> T002 -> T003 -> T004 -> T005/T006/T007 -> T008 -> T009/T010/T011
  -> T012 -> T013 -> T014
```

T005 and T006 are sequential with shared Router client/handler files. T007 can
run after the contract analysis and before T008; T009-T011 are post-
implementation tests with disjoint primary owners.

## Requirement Coverage

| Requirement | Tasks |
| --- | --- |
| FR-001/FR-002 | T008, T011 |
| FR-003/FR-004 | T005-T008, T010-T011 |
| FR-005/FR-009 | T005, T009-T011 |
| FR-006/FR-007 | T008, T009, T011 |
| FR-008/FR-011 | T009-T014 |
| FR-012 | T008a, T011-T014 |
| SC-001/SC-004 | T009-T011 |
| SC-005 | T012-T014 |

## Fallback Report

```text
Fallback delta: removed 0, retained 0, added 0, net 0
Added fallback evidence: none
```

## Verification Evidence

Focused implementation tests passed after strict response validation:

```text
go test -count=1 ./apps/control-plane/internal/gateway ./apps/control-plane/internal/config ./apps/control-plane/internal/invocation ./apps/control-plane/cmd/control-plane ./apps/a2a-router/internal/api ./contracts
```

Repository gates passed:

```text
go test -count=1 ./...
go vet ./...
git diff --check
wsl.exe -d Ubuntu-26.04 -- bash -lc 'cd /mnt/e/NeKiro && go test -race -count=1 ./apps/control-plane/... ./apps/a2a-router/... ./agents/runtime-b'
docker compose --file deploy/compose.yaml config --quiet
```

The read-scope scan found no Control Plane import of Router Ledger storage,
Agent endpoint, result-content persistence, retry, cache, alternate route, or
credential fallback. Workspace authorization failures do not call the Router;
Router 404 remains public `NOT_FOUND`, while wrong media, internal auth,
transport, and 5xx outcomes become safe `DEPENDENCY_ERROR`.
The production Router HTTP client rejects redirects with
`http.ErrUseLastResponse`, and the active InvocationRecord required fields and
error-code enum are checked before a metadata `200`. Compose configuration was
validated with all newly required Control Plane-to-Router variables supplied.

Independent Review (second pass) confirmed no remaining P0/P1/P2 findings. The
reviewed fixes covered Router redirect handling, record required-field/enum
validation, Router read `403` mapping and contract declaration, strict
metadata framing negative cases, and Control Plane deployment wiring. Converge
is complete; the local ignored `.env` may still need manual refresh with the
new variables, but the tracked template, Compose file, and runbook are
consistent.

Fallback delta: removed `0`, retained `0`, added `0`, net `0`. Added fallback
evidence: none.
