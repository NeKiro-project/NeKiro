# Tasks: Public Agent Share URL

**Input**: `specs/028-public-agent-share/spec.md`, `plan.md`, `research.md`, `data-model.md`, `contracts/`, and `quickstart.md`

**Tests**: Required after each mapped implementation slice. Tests must exercise the approved behavior and may not create fallback policy.

## Phase 1: Setup

**Purpose**: Lock delivery traceability and active plan context.

- [X] T001 Record upstream Issue #71, confirmed Release-selection policy, and active branch in `specs/028-public-agent-share/spec.md`
- [X] T002 Confirm repository-local Git identity and cleanly isolate Spec 028 files on `codex/028-public-agent-share`
- [X] T003 Inventory public-share fallback surfaces and preserve the zero-addition budget in `specs/028-public-agent-share/research.md`

---

## Phase 2: Foundational Contracts and Configuration

**Purpose**: Define language-neutral public identity and safe response facts before business code.

- [X] T004 [P] Define Public Agent Share v1 JSON Schema in `contracts/schemas/public-agent-share.v1.schema.json`
- [X] T005 [P] Define anonymous resolution OpenAPI and exact failure mapping in `contracts/openapi/public-agent-share.v1.yaml`
- [X] T006 Add Go Public Agent Share v1 DTOs and version constant in `contracts/public_agent_share.go` and `contracts/contracts.go`
- [X] T007 Extend Catalog v3 `CatalogEntry` with additive paired `publicAgentId` and `publicUrl` fields in `contracts/openapi/control-plane.v3.yaml` and `contracts/contracts.go`
- [X] T008 Add strict public-origin configuration with no default/inference in `apps/control-plane/internal/config/config.go`, `apps/console/src/consoleConfig.ts`, and `apps/console/src/vite-env.d.ts`
- [X] T009 Add contract and configuration tests after implementation in `contracts/public_agent_share_contracts_test.go`, `contracts/active_contracts_integration_test.go`, `apps/control-plane/internal/config/config_test.go`, and `apps/console/src/consoleConfig.test.ts`

**Checkpoint**: Contract documents validate; missing or invalid origins fail explicitly.

---

## Phase 3: User Story 1 - Provider receives stable NeKiro identity (Priority: P1)

**Goal**: Allocate, persist, and return one immutable public Agent identity and canonical URL at registration.

**Independent Test**: Register multiple versions of one Agent, restart the Catalog, and verify all reads return the same `publicAgentId` and URL while another Agent receives a distinct identity.

- [X] T010 [US1] Add Catalog schema v5 public identity migration and immutable trigger in `apps/control-plane/migrations/005_public_agent_share.sql` and `apps/control-plane/internal/catalog/postgres/005_public_agent_share.sql`
- [X] T011 [US1] Embed Catalog migration v5 and extend readiness schema checks in `apps/control-plane/internal/catalog/postgres/migrations.go`
- [X] T012 [US1] Add Public Agent ID generation and identity fields in `apps/control-plane/internal/catalog/model.go` and `apps/control-plane/internal/catalog/service.go`
- [X] T013 [US1] Persist and return the stable public identity under registration concurrency in `apps/control-plane/internal/catalog/postgres/store.go`
- [X] T014 [US1] Return paired public identity/URL facts from Catalog registration and owner reads in `apps/control-plane/internal/catalog/service.go`
- [X] T015 [US1] Wire the explicit public origin and ID generator in `apps/control-plane/cmd/control-plane/main.go` and `deploy/compose.yaml`
- [X] T016 [US1] Add migration, store, service, concurrency, restart, and contract tests after implementation in `apps/control-plane/internal/catalog/postgres/*_test.go`, `apps/control-plane/internal/catalog/service_test.go`, and `apps/control-plane/internal/gateway/catalog_handler_test.go`

**Checkpoint**: Provider registration returns a stable canonical URL without exposing or changing an endpoint.

---

## Phase 4: User Story 2 - Recipient resolves and reviews shared Agent (Priority: P1)

**Goal**: Resolve the public identity anonymously to a secrecy-safe view of explicitly selectable published trusted Releases.

**Independent Test**: Resolve a registered unpublished Agent as `not_installable`, publish two exact Releases for the identity, resolve both without authentication, and verify forbidden fields and secrets are absent.

- [X] T017 [US2] Implement Catalog public projection and exact lifecycle filtering in `apps/control-plane/internal/catalog/public_share.go`
- [X] T018 [US2] Implement bounded PostgreSQL public identity and Release reads in `apps/control-plane/internal/catalog/postgres/store.go`
- [X] T019 [US2] Implement anonymous Gateway resolution with trace-correlated errors in `apps/control-plane/internal/gateway/public_share_handler.go`
- [X] T020 [US2] Register the public route without authentication in `apps/control-plane/cmd/control-plane/main.go`
- [X] T021 [US2] Add strict public response mapping and anonymous API client in `apps/console/src/api/nekiro.ts`
- [X] T022 [US2] Add exact canonical URL parsing/formatting with no trim, alias, redirect, or alternate origin in `apps/console/src/publicAgentUrl.ts`
- [X] T023 [US2] Add direct `/a/{publicAgentId}` public review page and routing in `apps/console/src/components/PublicAgentPage.tsx` and `apps/console/src/main.tsx`
- [X] T024 [US2] Add service, store, Gateway, client, URL parser, component, and secrecy tests after implementation in `apps/control-plane/internal/catalog/public_share_test.go`, `apps/control-plane/internal/catalog/postgres/public_share_integration_test.go`, `apps/control-plane/internal/gateway/public_share_handler_test.go`, `apps/console/src/api/nekiro.test.ts`, `apps/console/src/publicAgentUrl.test.ts`, and `apps/console/src/components/consoleSurface.test.tsx`

**Checkpoint**: Anonymous users can review only public facts and cannot install or invoke.

---

## Phase 5: User Story 3 - Recipient installs exact trusted Release (Priority: P1)

**Goal**: Open or paste the canonical URL, explicitly select an exact published Release, accept permissions, and create a matching Workspace Installation.

**Independent Test**: Starting only from a URL, select one Release, accept its permissions, install into an authorized Workspace, and verify exact version/Release equality before invoking through Router.

- [X] T025 [US3] Implement reusable public Release selection and permission confirmation UI in `apps/console/src/components/PublicAgentInstallPanel.tsx`
- [X] T026 [US3] Add paste-URL resolution and explicit Release selection to `apps/console/src/components/InstallationsTab.tsx`
- [X] T027 [US3] Wire share-originated exact Installation and post-response provenance validation in `apps/console/src/App.tsx` and `apps/console/src/api/nekiro.ts`
- [X] T028 [US3] Keep direct public pages view-only without complete owner configuration and enable installation only with an explicit valid Workspace-owner context in `apps/console/src/main.tsx` and `apps/console/src/components/PublicAgentPage.tsx`
- [X] T029 [US3] Add Console policy/component/API tests after implementation for no preselection, permission consent, stale lifecycle, unauthorized Workspace, conflict, and mismatched Release responses in `apps/console/src/consolePolicy.test.ts`, `apps/console/src/components/consoleSurface.test.tsx`, and `apps/console/src/api/nekiro.test.ts`
- [X] T030 [US3] Extend clean backend acceptance from public URL resolution through exact Installation, Router invocation, and Ledger lineage in `tests/e2e/invoke-record/invoke_record_test.go`
- [X] T031 [US3] Extend production-route Playwright acceptance for open and paste entry paths in `apps/console/e2e/console.spec.ts`

**Checkpoint**: URL-originated installation is a normal exact trusted Installation; public URL never reaches Agent transport.

---

## Phase 6: User Story 4 - Provider/operator lifecycle visibility (Priority: P2)

**Goal**: Public installability follows authoritative trusted Release lifecycle without an independent state or recovery path.

**Independent Test**: Publish, suspend, and revoke an exact Release while resolving the stable URL and attempting installation; verify the URL remains stable and no alternate Release is chosen.

- [X] T032 [US4] Preserve deterministic public lifecycle projection and dependency errors in `apps/control-plane/internal/catalog/public_share.go` and `apps/control-plane/internal/catalog/postgres/store.go`
- [X] T033 [US4] Add lifecycle race, suspended/revoked exclusion, dependency failure, and no-substitution tests after implementation in `apps/control-plane/internal/catalog/public_share_test.go`, `apps/control-plane/internal/catalog/postgres/public_share_integration_test.go`, and `tests/e2e/invoke-record/invoke_record_test.go`
- [X] T034 [US4] Surface explicit not-installable/lifecycle states without retry or stale success in `apps/console/src/components/PublicAgentPage.tsx` and `apps/console/src/components/PublicAgentInstallPanel.tsx`

**Checkpoint**: Stable identity survives lifecycle changes; only authoritative exact Releases are installable.

---

## Phase 7: Polish and Cross-Cutting Verification

- [X] T035 [P] Update active contract/version and architecture documentation in `AGENTS.md`, `README.md`, `docs/contracts/compatibility.md`, and `docs/architecture/phase-1-spec.md`
- [X] T036 [P] Update explicit local/CI public-origin configuration in `docs/runbooks/local-development.md`, `.github/workflows/ci.yml`, and `deploy/compose.yaml`
- [X] T037 Run `gofmt`, frontend formatting-compatible checks, `go test ./...`, `go vet ./...`, `pnpm typecheck`, `pnpm test`, `pnpm build`, Compose config, and `git diff --check`
- [X] T038 Run clean-volume `go test -tags=e2e -count=1 ./tests/e2e/invoke-record` and production Console Playwright acceptance using `specs/028-public-agent-share/quickstart.md`
- [X] T039 Run secrecy scan across public URLs, responses, logs, Console state, fixtures, and artifacts and record fallback delta in `specs/028-public-agent-share/quickstart.md`
- [X] T040 Obtain independent Review Agent assessment against Issue #71, Spec, Plan, Tasks, contracts, constitution, and fallback policy; record findings in `specs/028-public-agent-share/review.md`
- [X] T041 Resolve review findings by updating approved Spec/Tasks first where behavior changes, rerun tests, and obtain fresh independent review in `specs/028-public-agent-share/review.md` (no actionable findings)
- [X] T042 Run `speckit-converge`, update completion evidence in `specs/028-public-agent-share/tasks.md`, and prepare commits with repository-local Git identity

---

## Dependencies and Execution Order

```text
Setup
  -> Foundational contracts/config
  -> US1 stable identity
  -> US2 anonymous public projection
  -> US3 exact Workspace installation
  -> US4 lifecycle operations
  -> full verification/review/convergence
```

- US1 blocks public resolution because the stable ID must exist first.
- US2 blocks the share-originated installation UI because it owns the safe Release list.
- US3 reuses existing Workspace/Router/Ledger boundaries and does not change them.
- US4 depends on US2 projection semantics and US3 revalidation evidence.

## Parallel Opportunities

- T004 and T005 can proceed in parallel because they write separate contract files.
- Backend contract DTO work and Console origin configuration can proceed in parallel after the schemas are fixed.
- Within US2, backend service/store and Console URL parser files are disjoint, but endpoint/UI integration waits for both.
- Documentation T035 and CI/runbook T036 are disjoint after behavior is stable.

## Implementation Strategy

1. Land language-neutral facts and required config.
2. Make registration produce a durable stable identity.
3. Add anonymous safe resolution without installation UI.
4. Add explicit Release/permission installation handoff.
5. Prove lifecycle and cross-runtime invocation behavior in clean E2E.
6. Complete independent review and convergence before delivery.

## Format Validation

All 42 tasks use the required checkbox, sequential task ID, optional `[P]`, required user-story label within story phases, and explicit file paths.
