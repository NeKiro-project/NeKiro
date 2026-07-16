# Tasks: A2A Router Foundation

**Input**: Design documents from `specs/013-a2a-router-foundation/`

**Tests**: Mapped tests follow implementation per project policy.

## Phase 1: SDD Gate

- [x] T001 Observe AGENTS, constitution, Specs 010/011, Router Internal v3, Control Plane Internal v2, current branch state, and zero-fallback policy in `specs/013-a2a-router-foundation/research.md`
- [x] T002 Specify Router process, auth, config, readiness, resolution, failure, and non-goal boundaries in `specs/013-a2a-router-foundation/spec.md`
- [x] T003 Plan ownership, write scope, contracts, verification, and fallback policy in `specs/013-a2a-router-foundation/plan.md` and supporting artifacts
- [x] T004 Analyze Spec/Plan/Tasks/constitution consistency before implementation in `specs/013-a2a-router-foundation/tasks.md`

## Phase 2: Foundational Process and Config

- [X] T005 Add standalone Router process assembly and readiness wiring in `apps/a2a-router/cmd/a2a-router/main.go`
- [X] T006 Add strict no-default Router config parsing and tests in `apps/a2a-router/internal/config/config.go` and `apps/a2a-router/internal/config/config_test.go`
- [X] T007 Add Router service bearer authentication and tests in `apps/a2a-router/internal/auth/auth.go` and `apps/a2a-router/internal/auth/auth_test.go`
- [X] T008 Add `apps/a2a-router/Dockerfile` without Compose/CI orchestration changes

## Phase 3: User Story 1 - Start a Strict Router Process (Priority: P1)

**Goal**: Valid config assembles handlers and readiness; invalid config fails before serving.

**Independent Test**: Config and readiness tests prove no default destination, token, limit, deadline, or dependency probe.

- [X] T009 [US1] Implement readiness handler and local assembly checks in `apps/a2a-router/internal/api/readiness_handler.go` and `apps/a2a-router/internal/api/readiness_handler_test.go`
- [X] T010 [US1] Add config table tests for missing/blank/malformed/unsafe/out-of-range values in `apps/a2a-router/internal/config/config_test.go`

## Phase 4: User Story 2 - Authenticate Internal Dispatch Requests (Priority: P1)

**Goal**: Only valid Router service callers can reach resolution.

**Independent Test**: HTTP tests prove invalid auth/media/body makes zero resolution calls.

- [X] T011 [US2] Implement Router Internal v3 dispatch HTTP validation in `apps/a2a-router/internal/api/dispatch_handler.go`
- [X] T012 [P] [US2] Add auth/media/body/shape/identifier/zero-resolution tests in `apps/a2a-router/internal/api/dispatch_handler_test.go`

## Phase 5: User Story 3 - Resolve Through Control Plane Only (Priority: P1)

**Goal**: Router resolves exact Agent facts only through Control Plane Internal v2.

**Independent Test**: Fake Control Plane receives exactly one correlated resolve request; typed failures are preserved.

- [X] T013 [US3] Implement Control Plane Internal v2 resolution client in `apps/a2a-router/internal/resolution/client.go`
- [X] T014 [P] [US3] Add resolution client success, typed failure, media/body, dependency, no-retry, and trace/correlation tests in `apps/a2a-router/internal/resolution/client_test.go`
- [X] T015 [US3] Wire dispatch handler to resolution client and map typed errors in `apps/a2a-router/internal/api/dispatch_handler.go`

## Phase 6: User Story 4 - Stop Before Agent Transport (Priority: P1)

**Goal**: Successful resolution returns truthful correlated `ROUTE_NOT_FOUND` placeholder.

**Independent Test**: HTTP tests prove no success, no Agent request, no Ledger write, no retry/cache, and exact correlation.

- [X] T016 [US4] Add post-resolution `ROUTE_NOT_FOUND` placeholder in `apps/a2a-router/internal/api/dispatch_handler.go`
- [X] T017 [P] [US4] Add post-resolution placeholder and no-Agent/no-Ledger tests in `apps/a2a-router/internal/api/dispatch_handler_test.go`

## Phase 7: Verification and Review

- [X] T018 Run formatting, focused/full tests, vet, fallback audit, and `git diff --check`
- [X] T019 Complete independent Review by a non-implementing agent
- [X] T020 Converge findings, update docs/tasks, repeat Review, and commit

## Dependencies and Parallelism

- T001-T004 precede implementation.
- T005-T008 establish process/config/auth before story handlers.
- US1 readiness can run after config assembly.
- US2 dispatch validation precedes US3 resolution wiring.
- US4 depends on successful resolution wiring.
- T012, T014, and T017 are parallel only after their corresponding implementation anchors exist.

## Implementation Strategy

MVP is process/config/auth/readiness plus a dispatch handler that resolves through Control Plane and returns `ROUTE_NOT_FOUND` after success. Transport, Ledger, SDK, samples, and Compose remain later Spec 010 tasks.

## Cross-Artifact Analysis

No Critical/High issues found in the generated Spec, Plan, and Tasks. Every FR-001 through FR-014 maps to at least one task: config/readiness (T005-T010), auth/validation (T007/T011/T012), Control Plane resolution (T013-T015), post-resolution placeholder and no side effects (T016/T017), and verification/review (T018-T020).

## Fallback Report

Fallback delta: removed 0, retained 0, added 0, net 0. Added fallback evidence: none.

## Verification Evidence

2026-07-16 local T018 gate passed:

- `gofmt -w` over `apps/a2a-router/**/*.go`.
- `go test ./apps/a2a-router/internal/auth ./apps/a2a-router/internal/config ./apps/a2a-router/internal/resolution ./apps/a2a-router/internal/api ./apps/a2a-router/cmd/a2a-router` passed.
- `go test ./...` passed.
- `go vet ./...` passed.
- `git diff --check` passed.
- Fallback/boundary scan found no Control Plane internal imports, no default
  destination or credential, no retry/cache/alternate source, no Agent
  endpoint call, and no Ledger write in runtime code. Test-only localhost and
  panic values are scoped to readiness/config assertions.

Independent Review-R1 found three blockers:

- Duplicate `Authorization` headers could authenticate if one value was valid.
- Control Plane resolution errors were reconstructed instead of preserving the
  exact typed status/body/trace semantics.
- Entropy failure fabricated a hard-coded pre-correlation trace identifier.

Converge fixes now require exactly one `Authorization` header, carry exact
Control Plane failure status/body/trace through `resolution.Failure`, write
that raw JSON response from the dispatch handler, and fail closed if
pre-correlation trace generation cannot obtain entropy. Focused Router tests,
`go test ./...`, `go vet ./...`, and `git diff --check` passed after the
fixes. Follow-up independent Review returned PASS with no remaining P0-P2
blocker.
