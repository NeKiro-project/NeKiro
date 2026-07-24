---

description: "Dependency-ordered implementation tasks for Workspace Client SDK"
---

# Tasks: Workspace Client SDK

**Input**: Design documents from `specs/025-workspace-client-sdk/`

**Prerequisites**: `plan.md`, `spec.md`, `clarify.md`, `research.md`,
`data-model.md`, `contracts/client-sdk-api.md`, `quickstart.md`

**Tests**: Tests are required and follow the corresponding approved
implementation. Every test task maps to a Spec scenario, failure semantic, or
compatibility requirement.

**Organization**: Tasks are grouped by blocking contract foundation and the
three prioritized user stories. Issue #52, not this task list, owns the later
clean Compose trusted-publication acceptance expansion.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: May run in parallel because files and incomplete dependencies are disjoint
- **[Story]**: Maps to US1, US2, or US3 from `spec.md`
- Every task names its exact write path

## Phase 1: Setup

**Purpose**: Establish the standalone application SDK package without adding a
dependency or mixing it with the Agent SDK.

- [ ] T001 Create the documented `clientsdk` package boundary and package comment in `sdks/client-sdk/doc.go`, importing no Agent SDK or `apps/**/internal` package.

---

## Phase 2: Foundational Contract and Trace Corrections

**Purpose**: Make active v4 Trace and `INTERNAL_ERROR` semantics trustworthy
before the SDK consumes them.

**CRITICAL**: No user-story implementation begins until this phase passes.

### Implementation

- [ ] T002 [P] Add one required `x-nek-trace-id` response header and the phase-appropriate HTTP 500 `INTERNAL_ERROR` response to `contracts/openapi/control-plane-invocation.v4.yaml` per FR-020/FR-021.
- [ ] T003 [P] Add the matching Trace-header and correlated HTTP 500 `INTERNAL_ERROR` facts to `contracts/openapi/router-internal.v4.yaml` without changing the v4 route or request/result schemas.
- [ ] T004 Map `contracts.ErrorCodeInternal` explicitly to HTTP 500 in `apps/a2a-router/internal/api/dispatch_handler.go`, removing its default-to-503 semantic fallback.
- [ ] T005 Require exactly one Router `x-nek-trace-id` equal to `DispatchInvocationRequestV4.TraceID`, reject nil/bodyless or drifted responses, and close rejected bodies in `apps/control-plane/internal/invocation/router_client.go`.
- [ ] T006 Keep the Gateway-created Trace as the northbound response fact and remove conditional Router-header replacement in `apps/control-plane/internal/gateway/invocation_handler.go`.

### Tests

- [ ] T007 [P] Add OpenAPI contract assertions for required Trace headers, Northbound 500 phase shape, Router Internal 500 correlated shape, and exact `INTERNAL_ERROR` code lists in `contracts/result_api_contracts_test.go`.
- [ ] T008 [P] Add Router HTTP 500 `INTERNAL_ERROR` mapping and regression cases for unchanged existing error mappings in `apps/a2a-router/internal/api/dispatch_handler_test.go`.
- [ ] T009 Add missing, duplicate, malformed, and mismatched Router Trace rejection, unchanged Gateway Trace assertions, and pre/correlated Gateway HTTP 500 `INTERNAL_ERROR` runtime cases in `apps/control-plane/internal/invocation/router_client_test.go`, `apps/control-plane/internal/gateway/invocation_handler_test.go`, and `apps/control-plane/internal/invocation/service_test.go` per FR-021/SC-007.

**Checkpoint**: Both active v4 contracts, Router, Router client, and Gateway
agree on one Trace and exact 500 semantics; focused foundation tests pass.

---

## Phase 3: User Story 1 - Invoke an installed Agent from application code (Priority: P1) MVP

**Goal**: Configure one Workspace Client and perform a strict non-streaming
Gateway invocation using only Agent ID, capability, and input per call.

**Independent Test**: An `httptest` Gateway receives one exact v4 request with
one credential header and returns a validated correlated Result; forbidden
routing fields do not exist in the public request type.

### Implementation for User Story 1

- [ ] T010 [P] [US1] Implement immutable `Config`, canonical Gateway-origin validation, explicit HTTP client cloning/redirect rejection, Workspace/credential validation, byte-limit validation, and `NewClient` in `sdks/client-sdk/config.go` per FR-002 through FR-004 and FR-012.
- [ ] T011 [P] [US1] Implement safe-identifier, duplicate-free JSON-object, strict EOF/member, bounded-read, exact media/header, and request encoding helpers in `sdks/client-sdk/json.go` without importing service internals.
- [ ] T012 [US1] Implement the exact three-field `InvokeRequest`, private v4 wire request, one-request transport, strict non-streaming Result v1 decode/header correlation, and public `Result` in `sdks/client-sdk/client.go` per FR-005 through FR-009 and FR-011/FR-013.

### Tests for User Story 1

- [ ] T013 [P] [US1] Add configuration tests for nil client, accepted documented nil Transport, every invalid origin component, missing/blank/whitespace/control credential, invalid Workspace, all limit boundaries, no trimming, clone isolation, and redirect rejection in `sdks/client-sdk/config_test.go`.
- [ ] T014 [P] [US1] Add request/success tests for exact path/body/headers, only three public business fields, raw-number preservation, invalid/duplicate/non-object input, encoded request N/N+1 bounds, strict JSON/media/member/Trace validation, pre-canceled context, body closure, no retry, credential secrecy, and concurrent Client use in `sdks/client-sdk/client_test.go`.
- [ ] T015 [US1] Add reflection/import-direction contract guards for the frozen public Config/InvokeRequest/Result surface and forbidden routing fields in `sdks/client-sdk/contract_test.go` per FR-006/FR-016/FR-017/SC-001/SC-002.

**Checkpoint**: US1 is independently usable for authenticated non-streaming
calls and does not expose endpoint, Router, version, Release, or Agent secrets.

---

## Phase 4: User Story 2 - Consume a live streaming result (Priority: P2)

**Goal**: Deliver strictly validated incremental SSE through a one-consumer
Stream with unambiguous terminal, EOF, interruption, cancellation, and Close
semantics.

**Independent Test**: A live test Gateway yields accepted, ordered chunks,
terminal, and EOF; the Client exposes events incrementally and rejects every
malformed, truncated, reordered, post-terminal, or correlation-changing stream.

### Implementation for User Story 2

- [ ] T016 [US2] Implement `InvokeStream`, exact SSE request negotiation, `Stream`, single-data-line compact JSON framing, strict Result Stream Event v2 decoding, header/first-event correlation, sequence validation, real-EOF completion, recorded early-Close interruption, and body ownership in `sdks/client-sdk/stream.go` per FR-008/FR-010/FR-011.

### Tests for User Story 2

- [ ] T017 [P] [US2] Add incremental accepted/chunk/completed and failed/canceled/timed-out terminal cases plus exact event/header correlation and body closure tests in `sdks/client-sdk/stream_test.go`.
- [ ] T018 [US2] Add malformed/CRLF/multi-field/oversized/duplicate/unknown/trailing SSE, wrong first event, sequence/chunk drift, early EOF, post-terminal event, context cancellation, Close-before-accepted/active/terminal/EOF, repeat Close, and Recv-after-Close tests in `sdks/client-sdk/stream_test.go` per SC-004.

**Checkpoint**: US2 works independently on the same Client configuration and
never treats partial streaming output as success.

---

## Phase 5: User Story 3 - Handle platform failures with safe typed context (Priority: P3)

**Goal**: Return only validated Gateway Platform Error v4 status/code/Trace and
optional correlation while rejecting malformed or semantically impossible
error responses.

**Independent Test**: Every row of the frozen HTTP status/code/phase matrix is
accepted into `PlatformError`; every invalid pairing, shape, message, member,
media, Trace, size, or raw-secret case remains a local validation error.

### Implementation for User Story 3

- [ ] T019 [US3] Implement `PlatformError`, `Correlated`, bounded strict pre/correlated Platform Error v4 decoding, exact one-value Trace matching, and the complete 400/401/403/404/406/409/413/500/502/503/504 status-code-phase matrix in `sdks/client-sdk/errors.go` per FR-014/FR-015/FR-021.
- [ ] T020 [US3] Route every non-200 `Invoke` and `InvokeStream` response through the typed safe error adapter, closing bodies and preserving local transport/context errors with `errors.Is`, in `sdks/client-sdk/client.go` and `sdks/client-sdk/stream.go`.

### Tests for User Story 3

- [ ] T021 [US3] Add every valid error-matrix row and invalid status/code/phase/message/media/header/member/correlation/oversize case, application-credential and raw-body secrecy sentinels, `errors.As`, `Correlated`, and non-stream/stream integration cases in `sdks/client-sdk/errors_test.go` per all US3 scenarios and SC-003/SC-005.

**Checkpoint**: All three stories are functional; applications can distinguish
every approved platform state without parsing or retaining raw error data.

---

## Phase 6: Documentation, Verification, Review, and Delivery

**Purpose**: Complete the application example, project status, fallback audit,
independent Review, convergence, PR, CI, merge, and Issue closure.

- [ ] T022 [P] Add a compiled installation-then-invocation example and SDK usage/error/stream/credential guidance in `sdks/client-sdk/example_test.go` and `sdks/client-sdk/README.md` per FR-018.
- [ ] T023 [P] Document the new SDK entry point and v4 Trace/500 compatibility correction in `README.md` and `docs/contracts/compatibility.md`.
- [ ] T024 Update trusted-publication Slice D T015/T016, active repository state, handoff, and managed plan status in `specs/023-trusted-agent-publication/tasks.md`, `docs/handoffs/CURRENT.md`, `AGENTS.md`, and `specs/025-workspace-client-sdk/tasks.md` only after their evidence is complete.
- [ ] T025 Run `gofmt`, focused contract/Router/Control Plane/SDK tests, SDK race, root tests, root vet, `golangci-lint run`, and `git diff --check` using `specs/025-workspace-client-sdk/quickstart.md`; record exact evidence in `specs/025-workspace-client-sdk/tasks.md`.
- [ ] T026 Audit forbidden imports, exported fields, route/version strings, credential-like values, retry/redirect/default branches, and result/error logs across `sdks/client-sdk`, `apps/control-plane/internal`, `apps/a2a-router/internal`, and `contracts`; record `Fallback delta: removed 2, retained 1, added 0, net -2` and `Added fallback evidence: none` in `specs/025-workspace-client-sdk/tasks.md`.
- [ ] T027 Run a fresh independent Review Agent against Issue #51, AGENTS.md, Spec/Clarify/Plan/Tasks/contracts, complete diff, tests, secret/fallback policy, and both Runtime boundaries; append every High/Medium finding as a new unchecked task in `specs/025-workspace-client-sdk/tasks.md` before fixes.
- [ ] T028 Resolve all Review tasks in the exact owning paths recorded under `specs/025-workspace-client-sdk/tasks.md`, rerun their mapped verification, and preserve tests-after-approved-implementation ordering.
- [ ] T029 Run a fresh final independent Review and `speckit-converge`; require zero High/Medium findings, no unchecked implementation task, and no unresolved Spec/code divergence, recording the final verdict in `specs/025-workspace-client-sdk/tasks.md`.
- [ ] T030 In `E:/NeKiro`, confirm repository-local `Nene7ko_ <1604009816@qq.com>` identity, commit the complete Issue #51 branch, push `codex/workspace-client-sdk`, open a ready PR referencing #51 and parent #47, and wait for all required CI checks.
- [ ] T031 In `E:/NeKiro`, resolve any CI or inline Review findings through the same Tasks/Review loop, merge only after green CI and fresh PASS, close #51, sync local/upstream/fork `main`, and leave #47 open for dependent Issue #52.

---

## Dependencies & Execution Order

### Phase dependencies

- **Setup**: T001 starts immediately.
- **Foundation**: T002-T006 follow T001. T007-T009 follow their corresponding
  implementation and block every user story.
- **US1**: T010/T011 follow Foundation and may run in parallel; T012 follows
  both; T013/T014 follow their implementation; T015 follows the complete US1
  export surface.
- **US2**: T016 follows T010-T012; T017/T018 follow T016.
- **US3**: T019 may start after Foundation and Config helpers, but T020 follows
  T012/T016/T019; T021 follows T020.
- **Delivery**: T022/T023 follow stable public APIs and may run in parallel;
  T024-T031 are sequential evidence/review/delivery gates.

### User story dependencies

```text
Foundation
  -> US1 non-streaming MVP
       -> US2 streaming
       -> US3 integration into non-streaming
  -> US3 error decoder (may begin beside US1 after shared config/helpers)
```

US2 and the standalone US3 decoder are independently testable after Foundation;
their final method integration depends on US1's Client/request transport.

### Parallel opportunities

- T002 and T003 write disjoint OpenAPI files.
- T004, T005, and T006 write distinct Router, Router-client, and Gateway files.
- T007 and T008 write disjoint contract/Router tests; T009 owns Control Plane tests.
- T010 and T011 write disjoint SDK configuration and JSON helper files.
- T013 and T014 write disjoint tests after implementation.
- T022 and T023 write SDK versus project documentation.

## Parallel Examples

### Foundation

```text
Task: T002 Northbound v4 Trace/500 contract correction
Task: T003 Router Internal v4 Trace/500 contract correction

Task: T004 Router INTERNAL_ERROR status implementation
Task: T005 Control Plane Router Trace enforcement
Task: T006 Gateway Trace ownership enforcement
```

### User Story 1

```text
Task: T010 Client configuration and construction
Task: T011 strict JSON/wire helpers

After T012:
Task: T013 configuration tests
Task: T014 request/result tests
```

## Implementation Strategy

### MVP first

1. Complete T001-T009 so v4 is a trustworthy SDK target.
2. Complete T010-T015 for one strict application non-streaming call.
3. Run the US1 checkpoint before streaming/error expansion.

### Incremental delivery

1. Foundation -> exact Trace and 500 contract.
2. US1 -> application JSON invocation MVP.
3. US2 -> live validated streaming.
4. US3 -> complete typed failure matrix.
5. Documentation/full gates -> independent Review -> convergence -> PR/CI/merge.

## Traceability Matrix

| Requirements | Tasks | Evidence |
| --- | --- | --- |
| FR-001-FR-009, SC-001-SC-002 | T001, T010-T015 | Package/import guard, exact request, JSON success, concurrency |
| FR-010-FR-013, SC-004 | T011-T012, T016-T018 | Explicit limits, context, strict incremental stream and EOF |
| FR-014-FR-015, SC-003/SC-005 | T019-T021, T026 | Complete typed matrix, raw-error and credential secrecy |
| FR-016-FR-018 | T001, T015, T022-T023 | Dependency direction, distinct SDK, compiled example/docs |
| FR-019 | T007-T009, T013-T015, T017-T018, T021, T025 | Contract/unit/race/vet/lint evidence |
| FR-020-FR-021, SC-007 | T002-T009 | One Gateway Trace and explicit 500 Internal semantics |
| Review/converge/DoD | T024-T031 | Status docs, fallback report, independent Reviews, CI and merge |

## Notes

- Tests follow the approved implementation within every phase.
- The Client SDK never imports the Agent SDK or a service-internal package.
- No task may add credential issuance, service accounts, role delegation,
  retry, redirect, alternate route, result persistence, or v3 compatibility.
- New ambiguity or public behavior returns to Spec/Plan/Tasks and re-analysis
  before code changes.
