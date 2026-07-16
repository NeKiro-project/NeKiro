# Tasks: Invocation Ledger

## Phase 1: Design Gate

- [x] T001 Create Spec, clarification, plan, research, data model, and quickstart in `specs/014-invocation-ledger/`
- [x] T002 Analyze Spec/Plan/Tasks consistency against contracts and constitution before implementation

## Phase 2: Implementation

- [ ] T003 [US1] Implement explicit Ledger migration and strict readiness in `apps/a2a-router/internal/ledger/001_ledger.sql` and `apps/a2a-router/internal/ledger/migrations.go`
- [ ] T004 [US1] Implement typed errors, append transaction, immutable events, and atomic projection in `apps/a2a-router/internal/ledger/errors.go` and `apps/a2a-router/internal/ledger/store.go`
- [ ] T005 [US2] Enforce locked parent-derived nested lineage in `apps/a2a-router/internal/ledger/store.go`
- [ ] T006 [US3] Implement Workspace-scoped restart-safe Invocation/Trace reads in `apps/a2a-router/internal/ledger/store.go`
- [ ] T007 [US3] Implement Router Internal v3 read adapters in `apps/a2a-router/internal/api/ledger_handler.go`

## Phase 3: Mapped Tests

- [ ] T008 [US1] Add migration/readiness and lifecycle atomicity PostgreSQL tests in `apps/a2a-router/internal/ledger/postgres_integration_test.go`
- [ ] T009 [US2] Add parent-lineage race and mismatch PostgreSQL tests in `apps/a2a-router/internal/ledger/postgres_integration_test.go`
- [ ] T010 [US3] Add restart/order/isolation/content-exclusion PostgreSQL tests in `apps/a2a-router/internal/ledger/postgres_integration_test.go`
- [ ] T011 [US3] Add HTTP mapping tests in `apps/a2a-router/internal/api/ledger_handler_test.go`
- [ ] T012 Run formatting, unit, integration, race, vet, full repository, and fallback checks

## Phase 4: Independent Delivery Gates

- [ ] T013 Independent Review by an agent that did not implement this branch
- [ ] T014 Converge Review findings and complete fresh independent Review

## Dependencies

`T001 -> T002 -> T003 -> T004 -> T005 -> T006 -> T007 -> T008/T009/T010 -> T011 -> T012 -> T013 -> T014`

Mapped tests follow implementation by project policy. T008-T010 are logically
parallel test concerns but share a file and therefore are executed serially by
one implementation owner. Review and Converge remain unchecked for root.
