# Tasks: Extract Reusable A2A Transport

**Input**: Design documents from `specs/029-extract-a2a-transport/`

**Tests**: Required after corresponding implementation tasks and mapped to the
Spec acceptance/failure matrix.

**Organization**: Tasks are grouped by user story. Upstream and downstream
write scopes are explicit because two repositories and an immutable release
ordering are involved.

## Phase 1: Setup

**Purpose**: Establish the two-repository delivery boundary without touching
the user's separate Spec 028 worktree.

- [X] T001 Confirm repository-local Git identity, clean isolated branches, Apache-2.0 compatibility, and ADMIN/write access in `E:/nekiro-a2a-transport-go/` and `E:/NeKiro-a2a-transport-integration/`
- [X] T002 Initialize the standalone Go module, root `a2atransport` package documentation, source-origin `NOTICE`, README skeleton, and `.gitignore` in `E:/nekiro-a2a-transport-go/`
- [X] T003 [P] Add standalone Linux CI for formatting, test, race, vet, and module tidiness in `E:/nekiro-a2a-transport-go/.github/workflows/ci.yml`
- [X] T004 [P] Record the accepted dependency decision and current Spec Kit plan reference in `E:/NeKiro-a2a-transport-integration/docs/decisions/0008-standalone-a2a-transport-module.md` and `E:/NeKiro-a2a-transport-integration/AGENTS.md`

---

## Phase 2: Foundational Public Contract

**Purpose**: Create the dependency-free public seam required by all stories.

**Critical**: No upstream file in this phase may import a NeKiro application,
contract, SDK, Runtime, or storage package.

- [X] T005 Define compatibility constants, `CallOptions`, `StreamItem`, `FailureKind`, and immutable `Failure` APIs in `E:/nekiro-a2a-transport-go/doc.go`, `E:/nekiro-a2a-transport-go/options.go`, and `E:/nekiro-a2a-transport-go/errors.go`
- [X] T006 Implement explicit constructor/call-option validation, HTTP client cloning, redirect rejection, interceptor copying, and documented nil-Transport preservation in `E:/nekiro-a2a-transport-go/client.go`
- [X] T007 Add the generic official-client factory and exact per-operation interceptor wiring for `message/send`, `message/stream`, and `tasks/cancel` in `E:/nekiro-a2a-transport-go/client.go`
- [X] T008 Add post-implementation public-contract tests for nil/invalid dependencies, endpoint parsing, positive bounds, nil interceptors, caller-client isolation, redirects, compatibility constants, and safe typed failures in `E:/nekiro-a2a-transport-go/client_test.go` and `E:/nekiro-a2a-transport-go/errors_test.go`

**Checkpoint**: A standalone consumer can construct the package and observe
explicit invalid-argument failures without any NeKiro dependency or default.

---

## Phase 3: User Story 1 - Maintain One Reusable Transport Source (Priority: P1)

**Goal**: Make standalone upstream the only production source for reusable
JSON-RPC/SSE transport mechanics.

**Independent Test**: Upstream completes request-response and streaming calls
against an official test server, while NeKiro contains no duplicate low-level
implementation after adapter integration.

- [X] T009 [US1] Implement bounded strict JSON-RPC response transport with exact media, request/response ID, XOR result/error, duplicate-member, unknown-field, trailing-data, body-close, and overflow behavior in `E:/nekiro-a2a-transport-go/jsonrpc.go`
- [X] T010 [US1] Implement bounded strict SSE transport with exact media, one unique ID line, one data line, blank delimiter, matching JSON-RPC ID, raw result capture, and interrupted/overflow behavior in `E:/nekiro-a2a-transport-go/stream.go`
- [X] T011 [US1] Implement public `SendMessage`, `SendStreamingMessage`, and `CancelTask` operations with supported official result/event kinds and typed transport failure classification in `E:/nekiro-a2a-transport-go/client.go`, `E:/nekiro-a2a-transport-go/jsonrpc.go`, and `E:/nekiro-a2a-transport-go/stream.go`
- [X] T012 [P] [US1] Add the post-implementation strict JSON-RPC positive/negative matrix and exact bound/body-close cases in `E:/nekiro-a2a-transport-go/jsonrpc_test.go`
- [X] T013 [P] [US1] Add the post-implementation SSE framing, unique ID, raw result, overflow, EOF, and official-server integration matrix in `E:/nekiro-a2a-transport-go/stream_test.go`
- [X] T014 [US1] Run upstream format, unit, race, vet, tidy, and diff checks and record passing commands in `E:/nekiro-a2a-transport-go/README.md`

**Checkpoint**: The standalone module independently implements and verifies the
complete extracted wire mechanics.

---

## Phase 4: User Story 2 - Reuse Without Platform Coupling (Priority: P1)

**Goal**: Prove external consumers need no NeKiro module, contract, local
replacement, or invented metadata.

**Independent Test**: A clean temporary module imports the reviewed public tag
and compiles non-streaming and streaming examples with only explicit options.

- [X] T015 [US2] Document public construction, explicit call options, interceptor ownership, typed failures, compatibility identity, and forbidden fallback behavior in `E:/nekiro-a2a-transport-go/README.md`
- [X] T016 [P] [US2] Add compile-checked request-response and streaming examples with explicit endpoints, limits, and interceptors in `E:/nekiro-a2a-transport-go/example_test.go`
- [X] T017 [P] [US2] Add an architecture/dependency test proving the upstream import graph contains no NeKiro platform module in `E:/nekiro-a2a-transport-go/architecture_test.go`
- [X] T018 [US2] Scan upstream code, fixtures, docs, and test output for credentials, platform imports, retries, alternate endpoints, default limits, legacy decoders, truncation, and degraded success; record the fallback result in `E:/nekiro-a2a-transport-go/README.md`

**Checkpoint**: Source and tests prove standalone ownership before any release
or downstream dependency update.

---

## Phase 5: User Story 3 - Preserve NeKiro Semantics (Priority: P1)

**Goal**: Replace the private transport mechanics with the standalone module
while retaining all platform-owned behavior.

**Independent Test**: Existing Router/contract/cross-runtime/acceptance suites
produce the same results and Ledger facts through a temporary local module
link, followed by a tagged dependency.

- [X] T019 [US3] Add a temporary uncommitted local module replacement only for migration verification in `E:/NeKiro-a2a-transport-integration/go.mod`
- [X] T020 [US3] Adapt Router client construction, per-operation call options, and request-scoped credential interceptors to the standalone API in `E:/NeKiro-a2a-transport-integration/apps/a2a-router/internal/transport/a2a/client.go`
- [X] T021 [US3] Replace private envelope/network classification with standalone `FailureKind` to Platform Error v4 mapping in `E:/NeKiro-a2a-transport-integration/apps/a2a-router/internal/transport/a2a/errors.go`
- [X] T022 [US3] Consume standalone stream items and raw results while retaining Profile validation, identity/artifact/terminal policy, one-shot cancellation, and platform event mapping in `E:/NeKiro-a2a-transport-integration/apps/a2a-router/internal/transport/a2a/streaming.go`
- [X] T023 [US3] Delete only the migrated JSON-RPC/SSE/client factory implementation and its duplicate private tests from `E:/NeKiro-a2a-transport-integration/apps/a2a-router/internal/transport/a2a/`
- [X] T024 [P] [US3] Update post-implementation Router adapter tests for credentials, redirects, safe failure mapping, raw result preservation, limit boundaries, and cancellation in `E:/NeKiro-a2a-transport-integration/apps/a2a-router/internal/transport/a2a/*_test.go`
- [X] T025 [P] [US3] Run focused post-implementation contracts, Router API, Runtime B, and root tests against the temporary local module in `E:/NeKiro-a2a-transport-integration/`
- [X] T026 [US3] Run Runtime A and invoke-to-record cross-runtime acceptance against the temporary local module and record exact lineage/result/Ledger equivalence in `E:/NeKiro-a2a-transport-integration/specs/029-extract-a2a-transport/tasks.md`

**Checkpoint**: All platform behavior passes while the standalone module is the
only low-level transport implementation.

---

## Phase 6: User Story 4 - Explicit Compatibility and Release (Priority: P2)

**Goal**: Review and publish an immutable upstream release, then consume it
without a local replacement.

**Independent Test**: A clean external module and NeKiro both resolve the
public tag, and changing it requires an explicit dependency diff.

- [X] T027 [US4] Run an independent upstream review against Spec 029, ADR 0008, the public API contract, fallback inventory, source provenance, and all upstream diffs in `E:/nekiro-a2a-transport-go/`
- [X] T028 [US4] Resolve every blocking upstream review finding, rerun upstream tests, and obtain a fresh independent PASS in `E:/nekiro-a2a-transport-go/`
- [X] T029 [US4] Commit and push the upstream branch, create and merge its reviewed PR with green CI, then create and push immutable tags `v0.1.0` and conformance-complete `v0.1.1` in `E:/nekiro-a2a-transport-go/`
- [X] T030 [US4] Prove a clean temporary external module resolves and compiles `github.com/NeKiro-project/nekiro-a2a-transport-go@v0.1.1` without `replace` or NeKiro imports
- [X] T031 [US4] Remove the temporary replacement and require upstream `v0.1.1`, update sums, and verify the committed module graph in `E:/NeKiro-a2a-transport-integration/go.mod` and `E:/NeKiro-a2a-transport-integration/go.sum`
- [ ] T032 [US4] Rerun focused, full, race, vet, module, diff, fallback, secrecy, Runtime A, and invoke-to-record checks using only the public tag in `E:/NeKiro-a2a-transport-integration/`

**Checkpoint**: Upstream `v0.1.1` is public and NeKiro no longer depends on a
local or duplicate implementation.

---

## Phase 7: Review and Convergence

**Purpose**: Close downstream delivery with independent evidence.

- [X] T033 Run an independent NeKiro review against AGENTS.md, Spec, Clarify, Plan, Tasks, ADR 0008, active contracts, fallback policy, and both repository diffs in `E:/NeKiro-a2a-transport-integration/`
- [X] T034 Resolve every blocking downstream review finding, return any behavior/scope issue to Spec or Tasks first, rerun affected verification, and obtain a fresh independent PASS in `E:/NeKiro-a2a-transport-integration/`
- [X] T035 Update Spec 029 status, task evidence, delivery docs, and fallback delta after final verification in `E:/NeKiro-a2a-transport-integration/specs/029-extract-a2a-transport/` and `E:/NeKiro-a2a-transport-integration/docs/`
- [ ] T036 Commit and push the NeKiro branch and create a downstream PR that depends only on public upstream `v0.1.1` in `E:/NeKiro-a2a-transport-integration/`

---

## Dependencies and Execution Order

```text
Setup
  -> Foundational public contract
     -> US1 transport implementation
        -> US2 standalone reuse proof
           -> US3 downstream adapter using temporary local module
              -> US4 upstream review/release and tagged downstream dependency
                 -> downstream review/convergence
```

- US1 depends on the foundational API.
- US2 depends on the completed upstream implementation.
- US3 may use a temporary local replacement only after upstream behavior passes.
- US4 must merge/tag upstream before the downstream dependency is committed.
- Downstream review must inspect the exact tagged dependency and upstream commit.

## Verification Evidence (2026-07-30)

- Upstream review initially found dependency-graph root matching, strict
  response-envelope member handling, caller interceptor error identity, and
  `tasks/cancel` result-kind validation gaps. These were fixed in upstream
  commit `aa25abd`; the follow-up upstream CI run `30515320980` passed format,
  unit, Linux race, vet, and module-tidiness checks.
- Upstream PR [#1](https://github.com/NeKiro-project/nekiro-a2a-transport-go/pull/1)
  merged as `51ff886`, and immutable tag `v0.1.0` was pushed from that merge.
- The official-server conformance gap was closed by upstream PR
  [#2](https://github.com/NeKiro-project/nekiro-a2a-transport-go/pull/2),
  merged as `ca9255a`; immutable tag `v0.1.1` is the downstream release and
  adds `a2asrv` request-response, SSE, interceptor, and cancellation-error
  interoperability coverage.
- A temporary external module resolved and compiled `v0.1.0` and the final
  `v0.1.1` using only the public module and `a2a-go v0.3.15`; both checks had
  no `replace` directive or NeKiro platform import. The final `v0.1.1`
  consumer, created outside both repositories, completed one local
  `message/send` and one `message/stream` call, then passed `go test`,
  `go vet`, and `go mod verify`; the resolved checksum was
  `h1:NTe20xpkGtt27me7XX/8CmbMJM7IXThiYDZDJSUjrTk=`.
- NeKiro now resolves the public tag with no local replacement. Focused
  Router/API/contracts/Runtime B tests, root `go test ./...`, root `go vet
  ./...`, Runtime A tests/vet, and `git diff --check` passed.
- A clean Docker Compose/PostgreSQL stack built the public-tag images and
  `go test -tags=e2e -count=1 ./tests/e2e/invoke-record` passed, including
  JSON/SSE, negative, cancellation, credential, nested lineage, restart,
  isolation, and concurrent acceptance cases. The stack was torn down with
  volumes after the run.
- The exact lineage/result/Ledger assertions are exercised by the acceptance
  checks at `invoke_record_test.go:163-230` (SSE result and terminal Record,
  nested and reverse child Invocation lineage, trace-after-restart),
  `invoke_record_test.go:964-990` (single terminal Ledger event), and
  `invoke_record_test.go:1030-1065` (reverse Runtime B -> Router -> Runtime A
  root/child trace relationship).
- Local Windows `go test -race ./...` cannot run because the environment has
  no CGO compiler (`gcc`); downstream Linux CI remains required for T032.

## Parallel Opportunities

- T003 and T004 edit separate repositories/files after T002.
- T012 and T013 cover separate JSON-RPC and SSE files after T011.
- T016 and T017 cover examples and dependency architecture after T015.
- T024 and T025 can begin after T020-T023 because their write scopes differ;
  T026 remains ordered after focused checks.
- No release, tag, or downstream final dependency task is parallelized because
  immutable upstream ordering is part of the acceptance contract.

## Implementation Strategy

1. Establish the smallest public API and typed failure seam.
2. Move and independently verify wire mechanics upstream.
3. Prove external source isolation.
4. Replace NeKiro internals through a thin adapter using a temporary local link.
5. Review, merge, and tag upstream.
6. Remove the local link and verify NeKiro against the public immutable tag.
7. Independently review and converge the downstream PR.

## Format Validation

- Total tasks: 36.
- Setup/foundational tasks: 8.
- US1 tasks: 6.
- US2 tasks: 4.
- US3 tasks: 8.
- US4 tasks: 6.
- Review/convergence tasks: 4.
- Every task uses the required checkbox, sequential ID, optional `[P]`, required
  story label in story phases, and an explicit file or repository path.
