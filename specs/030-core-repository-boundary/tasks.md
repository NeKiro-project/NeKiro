# Tasks: Core Repository Boundary

**Input**: Design documents from `specs/030-core-repository-boundary/`

**Prerequisites**: `plan.md`, `spec.md`, `research.md`, `data-model.md`, `contracts/`, `quickstart.md`

**Tests**: Implementation tasks precede their mapped verification tasks. No
task may add copied-source, local-replacement, floating-ref, alternate-component,
or stale-release fallback behavior.

## Phase 1: Provenance and GitHub Foundation

- [x] T001 Create the durable repository-extraction Issue in `NeKiro-project/NeKiro` and link ADR 0009, the target repository map, compatibility impact, and merge order
- [x] T002 Create and push annotated tag `pre-repository-split-2026-08-04` at commit `fbe075751ed77f328e24361822a00d6cf99df367`
- [x] T003 Create empty public repositories `NeKiro-project/nekiro-sdk-go`, `NeKiro-project/NeKiro-Samples`, and `NeKiro-project/NeKiro-Stack` without generated initial commits
**Checkpoint**: The durable Issue, source tag, and empty target repositories exist before the preliminary dependency-identity change.

---

## Phase 2: Official Dependency Identity

- [x] T006 [US1] Change the core module declaration and all Go imports from `github.com/Nene7ko/NeKiro` to `github.com/NeKiro-project/NeKiro` in `go.mod`, `apps/**`, `contracts/**`, `tests/**`, `sdks/**`, and `agents/**`
- [x] T007 [US1] Remove assumptions about the personal namespace from retained documentation and build definitions in `README.md`, `docs/**`, and `apps/*/Dockerfile`
- [x] T008 [US1] Run core formatting, tidy, build, unit, race, and vet checks after the module identity change
- [ ] T009 [US1] Commit, push, review, and merge the preliminary ADR/governance/official-module PR from `codex/core-repository-split`

**Checkpoint**: The organization repository is a valid immutable Go dependency source.

### History export after official identity

- [ ] T004 Export `sdks/`, `agents/`, and the `deploy/` plus `tests/e2e/` union from a clean checkout of the new official-identity `main`, and push only the explicit export commits to target `main`
- [ ] T005 Verify export tree identities, run `git fsck --full` in source and targets, and record provenance in each target README or pull request

**Checkpoint**: Historical source and attribution exist in canonical target repositories before any core deletion.

---

## Phase 3: User Story 2 - One Migration Source Per Owner (Priority: P1)

**Goal**: Preserve every supported schema upgrade with one canonical SQL source beside its owning service.

**Independent Test**: Fresh and historical Catalog, Workspace, and Ledger databases migrate to the expected version and pass readiness from the embedded canonical files.

### Implementation

- [ ] T010 [P] [US2] Create canonical Catalog migrations under `apps/control-plane/internal/catalog/postgres/migrations/` and move schema versions 001-005 there
- [ ] T011 [P] [US2] Create canonical Workspace migrations under `apps/control-plane/internal/workspace/postgres/migrations/` and move schema versions 001-002 there
- [ ] T012 [P] [US2] Move Ledger migrations under `apps/a2a-router/internal/ledger/migrations/`
- [ ] T013 [US2] Replace Catalog raw SQL constants and duplicate files with direct embedded-FS loading in `apps/control-plane/internal/catalog/postgres/migrations.go`
- [ ] T014 [US2] Replace Workspace raw SQL constants and duplicate files with direct embedded-FS loading in `apps/control-plane/internal/workspace/postgres/migrations.go`
- [ ] T015 [US2] Simplify Ledger embedding to direct canonical migration-FS loading in `apps/a2a-router/internal/ledger/migrations.go`
- [ ] T016 [US2] Delete `apps/control-plane/migrations/` and any internal duplicate SQL or tests that only compare maintained copies

### Tests

- [ ] T017 [P] [US2] Update Catalog migration unit/integration tests for canonical embedded files, fresh install, ordered upgrade, version count, repeat execution, and readiness
- [ ] T018 [P] [US2] Update Workspace migration unit/integration tests for canonical embedded files, fresh install, ordered upgrade, version count, repeat execution, and readiness
- [ ] T019 [P] [US2] Update Ledger migration unit/integration tests for canonical embedded files, fresh install, v1-to-v2 upgrade, version count, repeat execution, and readiness
- [ ] T020 [US2] Run all PostgreSQL tagged suites, including `./apps/a2a-router/internal/ledger`, and compare supported schema behavior before and after consolidation

**Checkpoint**: Each owner/version pair has exactly one SQL file and all supported upgrade paths pass.

---

## Phase 4: User Story 3 - Independent Satellite Repositories (Priority: P1)

**Goal**: SDK, samples, and Console build and release without source copies or sibling checkout replacements.

### SDK implementation

- [ ] T021 [P] [US3] Move extracted SDK packages to `agent/`, `agent/routerauth/`, and `client/` in `nekiro-sdk-go`
- [ ] T022 [US3] Add `go.mod`, `go.sum`, `README.md`, `LICENSE`, `.gitignore`, and package documentation for module `github.com/NeKiro-project/nekiro-sdk-go`
- [ ] T023 [US3] Point SDK contract imports at the immutable official core module commit and remove all local `replace` directives
- [ ] T024 [US3] Add `.github/workflows/ci.yml` with full-SHA Actions, format, tidy, build, test, race, vet, dependency-boundary, license, secrecy, and aggregate `required` jobs
- [ ] T025 [US3] Run SDK local checks and push a `codex/repository-bootstrap` pull request

### Samples implementation

- [ ] T026 [P] [US3] Normalize extracted Samples into one standalone module with `internal/challengeproof/`, `runtime-a/`, and `runtime-b/`
- [ ] T027 [US3] Update Runtime A and Runtime B imports to official core and SDK module identities and remove `replace ../..`
- [ ] T028 [US3] Rewrite both sample Dockerfiles to build from the standalone Samples context and downloaded immutable Go modules rather than copied core source
- [ ] T029 [US3] Add `README.md`, `LICENSE`, module files, explicit sample configuration, and `.github/workflows/ci.yml` with full-SHA Actions, per-runtime quality, image builds, dependency-boundary, license, secrecy, and aggregate `required` jobs
- [ ] T030 [US3] Run Samples local checks and push a `codex/repository-bootstrap` pull request

### Console implementation

- [ ] T031 [P] [US3] Fetch `NeKiro-Console/main` and sync accepted core Console differences, including public Agent share components, URL parsing, API/config/types, and related tests
- [ ] T032 [US3] Remove any Console workflow checkout of core and full-stack Compose acceptance; retain only Console-owned typecheck, unit, build, and isolated browser behavior
- [ ] T033 [US3] Align Console `.github/workflows/ci.yml` to workflow `CI`, least privilege, concurrency, full-SHA Actions, explicit versions/timeouts, license/secrecy checks, and aggregate `required`
- [ ] T034 [US3] Run frozen install, typecheck, unit tests, production build, and isolated browser tests; push a `codex/core-split-sync` pull request

### Satellite verification

- [ ] T035 [P] [US3] Verify SDK has no core internal imports, local replacements, floating dependencies, or copied contract source
- [ ] T036 [P] [US3] Verify Samples has no core internal imports, local replacements, database access, or copied SDK/contract source
- [ ] T037 [P] [US3] Verify Console contains the accepted production UI and no full-stack source dependency

**Checkpoint**: All three satellite pull requests pass independently before core removes their source.

---

## Phase 5: User Story 4 - Immutable Product Assembly (Priority: P2)

**Goal**: Stack assembles exact core and satellite revisions and owns the complete product acceptance.

**Independent Test**: A clean Stack run validates its manifest, builds or pulls only exact components, and passes the existing backend and browser acceptance.

### Implementation

- [ ] T038 [US4] Create `components.json` and a validator in `NeKiro-Stack/scripts/` that rejects missing components, non-full SHAs, branches, `latest`, local paths, unverified tags, and digest mismatches
- [ ] T039 [US4] Rewrite extracted `compose.yaml` to reference exact externally prepared image tags/digests and remove all production `build:` source contexts
- [ ] T040 [US4] Add a preparation script that checks out exact manifest SHAs into verified temporary directories and builds deterministic component images without copying source into Stack
- [ ] T041 [US4] Move backend acceptance to `tests/backend/` and update imports/configuration for the official core module and Stack-owned Compose path
- [ ] T042 [US4] Move the production browser acceptance and sanitized log capture from core/Console CI into Stack-owned tests and scripts
- [ ] T043 [US4] Add Stack `README.md`, `LICENSE`, explicit `.env.example`, local usage, cleanup behavior, and `.github/workflows/ci.yml` with full-SHA Actions, manifest, Compose, backend, browser, license, secrecy, and aggregate `required` jobs

### Tests

- [ ] T044 [P] [US4] Add manifest negative tests for missing, malformed, floating, retagged, and digest-mismatched components
- [ ] T045 [P] [US4] Validate Compose with explicit test-only configuration and no local production build contexts
- [ ] T046 [US4] Run trusted publication, public share, exact install, JSON, SSE, nested Runtime B -> Router -> Runtime A, Ledger, negative-path, cancellation, and secrecy acceptance from Stack
- [ ] T047 [US4] Push a `codex/repository-bootstrap` pull request and record the exact accepted component manifest

**Checkpoint**: Product proof passes across repository boundaries before the core cutover merges.

---

## Phase 6: Core Cutover (User Stories 1, 3, and 5)

**Goal**: Make the core checkout independently releasable and retain only current architecture, contracts, and usage documentation.

### Implementation

- [ ] T048 [P] [US3] Replace Runtime B imports in `apps/a2a-router/internal/api/dispatch_handler_test.go` with a core-owned protocol fixture
- [ ] T049 [P] [US3] Replace Runtime B imports in `apps/a2a-router/internal/transport/a2a/client_test.go` with a core-owned JSON-RPC/SSE fixture
- [ ] T050 [US1] Create `codex/core-only-cutover` from the post-satellite baseline and rewrite `.github/workflows/ci.yml` with full-SHA Actions for core-only quality, contract, PostgreSQL Catalog/Workspace/Ledger integration, image build, dependency-boundary, license, secrecy, and aggregate `required` jobs
- [ ] T051 [US1] Rewrite `README.md`, `AGENTS.md`, and retained docs to describe core ownership, official module usage, satellite repositories, release order, and Stack acceptance
- [ ] T052 [US5] Move retained runbooks into `docs/usage/` and remove `docs/bugfix/`, `docs/handoffs/`, `docs/roadmap/`, Console delivery reports, and obsolete Spec/path links
- [ ] T053 [US3] Delete `apps/console/`, `sdks/`, `agents/`, `deploy/`, and `tests/e2e/` after their target pull requests pass
- [ ] T054 [US1] Delete frontend-only root manifests and configuration (`package.json`, `pnpm-lock.yaml`, `pnpm-workspace.yaml`, `tsconfig.base.json`, `vitest.config.ts`) and replace `.env.example` with core-only usage documentation
- [ ] T055 [US5] Remove `.specify/`, `specs/`, and `.agents/skills/speckit-*`; remove the managed Spec Kit block and transitional amendment wording while retaining permanent ADR 0009 governance in `AGENTS.md`

### Tests

- [ ] T056 [P] [US3] Run core unit/race/vet/build tests and prove Router tests no longer import sample source
- [ ] T057 [P] [US2] Run all core PostgreSQL migration/readiness suites after satellite deletion
- [ ] T058 [P] [US1] Scan tracked paths and imports to prove zero Console, SDK, Samples, Stack, Spec Kit, Node-only tooling, copied source, local replacements, or obsolete path references remain
- [ ] T059 [US5] Run documentation link/scope review and confirm architecture, decisions, contracts, and usage are the only retained docs families

**Checkpoint**: A fresh core checkout builds and verifies without Node, browser, sample, SDK, or Stack source.

---

## Phase 7: Cross-Repository CI and Governance

- [ ] T060 [P] Align `nekiro-a2a-transport-go/.github/workflows/ci.yml` to full-SHA Actions and the shared workflow, toolchain, timeout, quality, license, security, and aggregate-gate conventions
- [ ] T061 [P] Configure public-repository main/tag rulesets for pull request review, resolved conversations, `CI / required`, and force-push/deletion protection where GitHub permits
- [ ] T062 Record the private Console ruleset limitation and apply all available CI/review protections without changing repository visibility without explicit owner direction
- [ ] T063 Verify every repository reports a successful workflow named `CI` and stable aggregate `required` job

---

## Phase 8: Review, Convergence, and Merge

- [ ] T064 Run the complete quickstart verification and `git diff --check` in every changed repository
- [ ] T065 Obtain independent review of core, SDK, Samples, Console, Stack, and transport changes against ADR 0009 and the acceptance matrix
- [ ] T066 Resolve every blocking review finding and rerun affected repository-local and Stack acceptance checks
- [ ] T067 Merge in dependency order: core identity prerequisite, SDK, Samples, Console, Stack, core cutover, and transport alignment as applicable
- [ ] T068 Confirm final default branches, immutable provenance tag, repository descriptions/links, CI runs, and no open cutover blocker
- [ ] T069 Report fallback delta and exact added-fallback evidence in the final core pull request and completion record

## Dependencies and Execution Order

- Phase 1 blocks all deletion and establishes provenance.
- Phase 2 blocks SDK and Samples dependency publication.
- Phase 3 SDK blocks Samples; Console can proceed in parallel after Phase 1.
- Phase 5 depends on buildable core, SDK, Samples, and Console revisions.
- Phase 6 deletion depends on passing satellite and Stack pull requests.
- Phase 7 can proceed per repository after each workflow exists.
- Phase 8 merges only after all repository-local checks and immutable Stack
  acceptance pass.

## Parallel Opportunities

- Migration work for Catalog, Workspace, and Ledger uses disjoint files.
- SDK and Console bootstrap can proceed in parallel after provenance.
- Core Router fixture replacements can proceed in parallel with migration work.
- Transport CI alignment is independent of core implementation.
- Final repository scans and PostgreSQL suites can run in parallel after cleanup.

## Completion Evidence

- One annotated pre-split source tag.
- Three history-preserving satellite repositories plus updated Console and
  transport repositories.
- Successful `CI / required` in every repository.
- Successful immutable NeKiro-Stack product acceptance.
- Core tracked tree contains only core services, contracts, owned migrations,
  core verification, architecture/contract/usage docs, and governance.
- Fallback delta reports `added 0`; added fallback evidence is `none`.
