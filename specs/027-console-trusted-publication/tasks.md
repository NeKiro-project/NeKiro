# Tasks: Console Trusted Publication Loop

**Input**: Design documents from `specs/027-console-trusted-publication/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, and `contracts/console-gateway-mapping.md`

**Tests**: Tests are scheduled after the corresponding approved implementation and map to the acceptance scenarios, failure semantics, or compatibility requirements in Spec 027.

## Delivery protocol

The four delivery slices are strictly gated and may not be started out of
order without an explicit dependency decision:

1. Before implementing each upstream Issue, launch a fresh subagent with the
   Issue URL, Spec 027, and the current diff/base. Record its scope,
   dependencies, forbidden behavior, and expected acceptance evidence.
2. Implement only the reviewed Issue slice on its own branch/PR.
3. Run the mapped tests and checks.
4. Launch an independent review agent that did not implement the slice. It
   must review the Issue, Spec, Plan, Tasks, active contracts, constitution,
   security/secret boundary, and diff.
5. Fix all High/Medium findings, rerun relevant tests, and obtain a clean
   review before starting the next slice.

## Phase 1: Repository setup and source ownership

- [ ] T001 [US5] Record the standalone Console source relationship and platform import ownership in `E:/Progarms/NeKiro/specs/027-console-trusted-publication/quickstart.md` and the platform delivery notes without creating a second live production path.
- [ ] T002 [P] [US5] Confirm the clean standalone Console baseline in `E:/Progarms/NeKiro-Console` and preserve its production route plus all four comparison demo routes as the source for Console Issues #2/#4/#3.
- [ ] T003 [P] [US5] Confirm the root workspace/CI import target `E:/Progarms/NeKiro/apps/console` and lockfile strategy; do not modify root business code before the standalone slices are reviewed.
- [ ] T004 [P] [US5] Record the trace-body/header policy in the contract mapping: body `traceId` is authoritative, an optional header must match, and missing header remains valid under the active OpenAPI.

## Phase 2: Slice A - Trusted Publication Gateway client (Console Issue #2)

**Goal**: Give the production Console strict, Gateway-only request and response mappings for Binding, Challenge, and Release operations.

**Pre-implementation gate**:

- [ ] T005 [US1] Launch a subagent to read Console Issue #2, Spec 027 FR-001 through FR-005/FR-012/FR-014, the Trusted Publication v1 OpenAPI/schema, and the current client. Record the exact route matrix, ownership boundary, secret rules, and any ambiguity before touching `apps/console/src/api`.

### Implementation

- [ ] T006 [US1] Add `EndpointBinding`, `VerificationChallenge`, `AgentRelease`, and Trusted Publication error code types to `E:/Progarms/NeKiro-Console/src/api/nekiro.ts` or a dedicated contract module, matching `contracts/schemas/trusted-publication.v1.schema.json` with no unknown-field acceptance.
- [ ] T007 [US1] Add strict request methods in `E:/Progarms/NeKiro-Console/src/api/nekiro.ts` for binding create/read, challenge issue/complete, Release create/read, and Release verify/publish/suspend/revoke using only the public `/v4` Gateway paths from `specs/027-console-trusted-publication/contracts/console-gateway-mapping.md`.
- [ ] T008 [US1] Add response validators and relationship checks in `E:/Progarms/NeKiro-Console/src/api/nekiro.ts` for required IDs, semver, URI, enum, lowercase digest, date-time, Card/Binding/Release identity, and trace/error consistency; treat the error-body trace as authoritative and reject an optional mismatched header; keep challenge proof out of logs, storage, and later request bodies.
- [ ] T009 [US1] Keep the standalone API contract types in `E:/Progarms/NeKiro-Console/src/api/nekiro.ts` or a dedicated API contract module; defer Release-aware Installation view-model changes in `src/types.ts` to Issue #4.

### Tests

- [ ] T010 [P] [US1] Add focused API contract tests in `E:/Progarms/NeKiro-Console/src/api/nekiro.test.ts` for every Trusted Publication request path, method, body, Authorization behavior, exact success mapping, unknown-field rejection, and ID relationship mismatch (Spec US1 acceptance scenarios 2-5, FR-002/FR-008/FR-012/FR-014).
- [ ] T011 [P] [US1] Add API failure tests in `E:/Progarms/NeKiro-Console/src/api/nekiro.test.ts` for `WRONG_PROOF`, `CHALLENGE_EXPIRED`, `CHALLENGE_REUSED`, `DISALLOWED_NETWORK`, `ENDPOINT_UNAVAILABLE`, `CONFLICT`, `FORBIDDEN`, invalid response bodies, absent trace headers, mismatched trace headers, and safe transport errors while asserting no secret leakage (Spec FR-005 and Edge Cases).
- [ ] T012 [US1] Run the Slice A typecheck and focused tests, scan source/test output for bearer values, challenge proof fixtures, and direct/internal URLs, and capture the Issue acceptance evidence.

**Review gate**:

- [ ] T013 [US1] Launch an independent review agent for Console Issue #2 against the Slice A diff, Spec 027, Plan, Tasks, Trusted Publication v1, and constitution; do not proceed to Slice B until it reports no High/Medium findings.

## Phase 3: Slice B - Trusted publication and Release operations UI (Console Issue #4)

**Goal**: Make the provider workflow and trusted Release-aware Workspace workflow operable in the production Console.

**Pre-implementation gate**:

- [x] T014 [US2] Launch a subagent to read Console Issue #4, Spec 027 US1-US3, FR-002 through FR-008/FR-012/FR-016, Slice A results, and the existing Registry/Installations components. Record explicit provider/owner credential contexts, Release-ID preflight semantics, UI state transitions, destructive confirmations, stale-state handling, and non-goals before editing components.

### Implementation

- [x] T015 [US1] Add `E:/Progarms/NeKiro-Console/src/components/TrustedPublicationTab.tsx` with Binding creation/read, Challenge issue/complete, Release create/read, and server-defined lifecycle actions; show exact state, trace/error, provenance, and recovery ownership.
- [x] T016 [US1] Update `E:/Progarms/NeKiro-Console/src/App.tsx` to use explicit provider and Workspace-owner API clients, load/select trusted publication records, require an explicit Release-ID preflight before installation, keep challenge proof transient, refresh authoritative reads after mutations, and route provider operations through the API client only.
- [x] T017 [US2] Update `E:/Progarms/NeKiro-Console/src/components/RegistryTab.tsx` so the production trust path no longer presents Catalog publish as equivalent to trusted Release publication; keep registration and discovery views clear about draft, trusted Release, and disabled states.
- [x] T018 [US2] Update `E:/Progarms/NeKiro-Console/src/components/InstallationsTab.tsx` and `E:/Progarms/NeKiro-Console/src/types.ts` to require a Gateway Release-ID preflight, display published immutable provenance before install and installed Release ID after install, submit an exact version constraint, reject mismatched installation provenance, and preserve exact disabled/uninstalled/lifecycle errors.
- [x] T019 [US2] Add explicit confirmation UI for suspend/revoke/uninstall, disable actions while an authoritative operation is pending, and avoid optimistic transitions, retry, reconnect, alternate endpoint, or local-storage state.

### Tests

- [x] T020 [P] [US1] Add component/API integration tests for pending -> verified Binding, challenge expiry/reuse/error presentation, and verified -> published Release transitions in `E:/Progarms/NeKiro-Console/src/` using server-shaped fixtures only; assert provider identity and provider/owner token separation.
- [x] T021 [P] [US2] Add component tests for Release-ID preflight, published-state/provenance matching, exact version installation, installed Release-ID mismatch rejection, suspend/revoke confirmation, disabled/enable/uninstall behavior, and no fabricated success on errors (Spec US2/US3 acceptance scenarios, FR-006/FR-007/FR-012).
- [x] T022 [US1] Run Slice B typecheck/build and secret/mock-route scans, then capture a production-route manual or automated walkthrough for registration through trusted publish and install.

**Review gate**:

- [x] T023 [US2] Launch an independent review agent for Console Issue #4 against the Slice B diff, Slice A review evidence, Spec 027, active contracts, UI ownership, and fallback/secret policy; do not proceed to Slice C until it reports no High/Medium findings.

Slice B evidence: pre-implementation issue review completed; the first
post-implementation review found High/Medium issues and was followed by a
repair pass; the final independent review reported `PASS` with no actionable
High/Medium findings. Standalone checks passed: `npm run typecheck`,
`npm run lint`, `npm test` (32 tests), `npm run build`, and `git diff --check`.
Implementation PR: https://github.com/NeKiro-project/NeKiro-Console/pull/6

## Phase 4: Slice C - Canonical reverse cross-runtime acceptance (NeKiro Issue #60)

**Goal**: Prove `Runtime B -> A2A Router -> Runtime A` and the returned root/child lineage using the existing clean Compose acceptance.

**Pre-implementation gate**:

- [ ] T024 [US4] Launch a subagent to read NeKiro Issue #60, Spec 027 US4, FR-009 through FR-011/FR-016, Specs 019-021/026, and the existing `tests/e2e/invoke-record/invoke_record_test.go`. Record the exact fixture direction, expected Release provenance, direct-access boundary, and forbidden fallback behavior.

### Implementation

- [ ] T025 [US4] Extend the deterministic Runtime B sample with explicit Router URL/token, target Agent ID/capability, response/event limits, and Router Agent credential configuration; inject the existing Agent SDK and invoke only the configured target through Agent Router v1 for the reverse fixture. Extend the existing Runtime A adapter with a separately declared deterministic responder capability while preserving its `runtime.cross` caller path. Do not add a direct peer endpoint, platform storage access, a new runtime framework, retry, or fallback.
- [ ] T026 [US4] Add the reverse fixture setup and assertions in `tests/e2e/invoke-record/invoke_record_test.go` for B root -> A child `parentInvocationId`, one `rootTaskId`, one `traceId`, distinct exact Release IDs/Card digests, returned result correlation, and no direct endpoint access or exposed Agent host port.

### Tests

- [ ] T027 [US4] Run the focused clean Compose E2E and existing backend unit/contract/race/vet gates required by the modified acceptance; preserve exact failures for unavailable endpoint, timeout, cancellation, and credential boundary cases.

**Review gate**:

- [ ] T028 [US4] Launch an independent review agent for NeKiro Issue #60 against the Slice C diff, Specs 019-027, active A2A/Router/Invocation contracts, and acceptance evidence; do not proceed to Slice D until it reports no High/Medium findings.

## Phase 5: Slice D - Real frontend CI and browser acceptance (Console Issue #3 / parent #59)

**Goal**: Make CI execute the real production Console and visibly prove the complete loop in a fresh environment.

**Pre-implementation gate**:

- [ ] T029 [US5] Launch a subagent to read Console Issue #3, parent NeKiro Issue #59, Spec 027 US4-US5, FR-013 through FR-017, Slice A/B/C evidence, root workspace scripts, and existing CI. Record the browser runner, environment boundary, demo isolation, and failure conditions before editing CI or browser tests.

### Implementation

- [ ] T030 [US5] Add the repository-approved browser acceptance harness under `E:/Progarms/NeKiro-Console/e2e/` or the existing test convention, with explicit Gateway/Workspace/provider configuration and no mock production data; cover the visible Register -> Verify -> Publish -> Discover -> Install -> Invoke -> Record path.
- [ ] T031 [US5] Add browser assertions for JSON/SSE correlation, nested B -> Router -> A Trace display, exact lifecycle failures, challenge-proof non-persistence, and isolated `#/demo*` routes in `E:/Progarms/NeKiro-Console/e2e/`.
- [ ] T032 [US5] Update standalone Console CI and the platform root CI/import configuration so `pnpm typecheck`, `pnpm test`, `pnpm build`, and browser acceptance execute non-empty production scripts and fail on missing configuration or skipped project discovery; import reviewed source into `E:/Progarms/NeKiro/apps/console` without `.git`, `node_modules`, `dist`, or credentials.
- [ ] T033 [US5] Update `E:/Progarms/NeKiro-Console/README.md`, `E:/Progarms/NeKiro/docs/runbooks/local-development.md`, and Spec 027 quickstart with explicit environment requirements, provider/Workspace ownership, live workflow, and recovery boundaries.

### Tests

- [ ] T034 [US5] Run root frontend typecheck, focused tests, production build, browser acceptance, `git diff --check`, and secret scans; verify comparison demos remain available and separate from production evidence.
- [ ] T035 [US5] Run the full mapped backend/Console verification set and record fresh-environment acceptance IDs, review evidence, and any non-blocking residual policy items as `Needs policy` rather than fallback.

**Review gate**:

- [ ] T036 [US5] Launch an independent review agent for Console Issue #3/parent #59 against the complete Slice D diff, Spec 027, CI behavior, browser evidence, secret boundary, and all prior review records; do not merge until it reports no High/Medium findings.

## Phase 6: Convergence and delivery records

- [ ] T037 [P] Update `specs/027-console-trusted-publication/quickstart.md`, the parent Issue, and each child Issue with exact verification commands, accepted limitations, and links to the reviewed PR evidence.
- [ ] T038 [P] Run a final read-only Spec/Plan/Tasks/contract/constitution consistency analysis and append any remaining approved work to this `tasks.md`; do not implement unapproved behavior.
- [ ] T039 Confirm each slice has its own branch, commit identity `Nene7ko_ <1604009816@qq.com>`, upstream PR, passing CI, pre-implementation subagent report, independent review result, and no unresolved High/Medium finding.

## Dependencies & Execution Order

```text
T001-T004
  -> T005-T013 (Slice A)
  -> T014-T023 (Slice B)
  -> T024-T028 (Slice C)
  -> T029-T036 (Slice D)
  -> T037-T039
```

Slice A is the only prerequisite for Slice B. Slice C is backend-only and can
be developed in parallel with Slice B after its own pre-implementation review,
but the delivery order remains A -> B -> C -> D for this execution so each
review gate is explicit and the final browser acceptance consumes all prior
behavior. Tasks marked `[P]` have disjoint write scopes and may run together
within a slice.

## Implementation Strategy

1. Import the existing production Console so root CI has a real package.
2. Complete Slice A and stop for independent review.
3. Complete Slice B and stop for independent review.
4. Complete the reverse backend acceptance and stop for independent review.
5. Complete CI/browser acceptance and stop for independent review.
6. Run convergence and publish delivery records only after all slices pass.
