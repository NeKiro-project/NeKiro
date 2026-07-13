# Current Handoff: Spec 001 Contract Closure

**Updated**: 2026-07-13 (Asia/Shanghai)  
**Pause reason**: The user requested a repository-local checkpoint before
switching devices. Do not continue implementation automatically from this
checkpoint.

**Direction amendment**: After this implementation checkpoint, the project
adopted the runtime-agnostic platform boundary in Constitution `1.1.0` and ADR
0003. This amendment does not resume Spec 001 implementation; all module gates
and the resume order below still apply.

## Resume Point

- Repository: `E:\NeKiro`
- Branch: `main`
- HEAD: `46ca271` (`fix(contracts): tighten A2A conformance checks`)
- Worktree: clean when this handoff was written
- Push status: no push was performed
- Local Git identity: `Nene7ko_ <1604009816@qq.com>`
- `origin` and `upstream` currently both point to
  `https://github.com/XnLemon/NeKiro.git`

The branch was locally rebased while parallel module commits were reconciled.
The user explicitly accepted that rebase. It does not replace the required
post-integration fetch/rebase onto the latest remote HEAD.

## Authoritative Sources

Read these before changing code:

- `AGENTS.md`
- `.specify/memory/constitution.md`
- `docs/architecture/platform-direction.md`
- `docs/decisions/0003-runtime-agnostic-platform-boundary.md`
- `specs/001-complete-invocation-contracts/spec.md`
- `specs/001-complete-invocation-contracts/plan.md`
- `specs/001-complete-invocation-contracts/data-model.md`
- `specs/001-complete-invocation-contracts/tasks.md`
- `specs/001-complete-invocation-contracts/contracts/`
- `specs/001-complete-invocation-contracts/quickstart.md`

The task checkboxes have not yet been converged to the implementation state.
Use commits, tests, and fresh Reviews as evidence; do not mark tasks complete
solely from this handoff.

## Required Workflow

- Continue Spec Kit SDD: observe/specify/clarify/plan/tasks/analyze before code.
- Keep full Agent Runtime frameworks out of Control Plane and Router core;
  framework-specific behavior belongs in adapters or sample Agents.
- Features touching Agent integration must preserve the Phase 1 cross-runtime
  acceptance proof defined by the platform direction.
- The project intentionally implements approved behavior before mapped tests;
  do not introduce TDD as the required workflow.
- Every implementation Agent reads
  `C:\Users\16040\.codex\skills\implement\SKILL.md`.
- Every fresh Reviewer reads
  `C:\Users\16040\.codex\skills\code-review\SKILL.md`.
- A failed Review returns findings to Spec/Tasks before code fixes and requires
  a new independent Reviewer after the fix.
- Fallback-sensitive work follows
  `C:\Users\16040\.codex\skills\ai-fallback-disable\SKILL.md`.
- Frontend work remains paused.

## Module Status

### Module B: Agent Card Semantics - PASS

Independent Review passed. The implemented contract covers Agent Card `0.2`,
semantic rule IDs, credential-free endpoints, strict manifests, portable
corpus-confined paths, and cross-version fixtures.

Relevant branch commits by subject:

- `1827f7f` `feat(contracts): add agent card semantic conformance`
- `7dfa71d` `fix(contracts): tighten agent card conformance manifest`
- `5ee03d3` `fix(contracts): harden agent card conformance inputs`
- `62c8f56` `fix(contracts): reject nonportable fixture names`

Do not reopen Module B without concrete evidence from the remaining shared
scanner issue below or a new Review finding.

### Module A: Result and API Contracts - REVIEW FAIL

Latest implementation commit: `debeef1` (`fix(contracts): bind invocation
responses`). The latest fresh Review confirmed all previous findings were fixed,
including correlated post-creation errors, request-bound non-streaming results,
recursive duplicate-member rejection, directional error precision,
`INV-CORR-001`, stream interruption, explicit JSON null, and Ledger/result
separation.

One Medium finding remains:

- `contracts/agent_card_semantics.go` uses `json.Decoder.Token` in the shared
  duplicate-member scanner without `UseNumber`.
- Module A calls this scanner from `contracts/result_contracts.go`.
- A valid unconstrained result/chunk number such as `1e400` is rejected during
  duplicate scanning because Go attempts `float64` conversion.
- This violates arbitrary JSON preservation and creates cross-language drift.

Do not patch this directly. First amend Spec 001, design artifacts, and Tasks to
require valid large JSON number preservation at strict DTO boundaries and assign
ownership of the shared scanner change. Then implement `UseNumber` at the shared
scanner boundary, add result and chunk decode cases for `1e400`, run both Agent
Card and result regressions, commit, and create a fresh Module A Reviewer.

### Module C: A2A Profile - AWAITING FRESH REVIEW

Latest implementation commit: `46ca271` (`fix(contracts): tighten A2A
conformance checks`). It addresses the previous Review findings:

- mandatory JSON-RPC response baseline, result/error exclusivity, and supported
  string/number/null ID types;
- exact stable failure classification matching manifest `protocolError`;
- closed per-method A2A Profile operation variants;
- mutation fixtures for boolean, object, and array response IDs.

The implementation Agent reported these passing on the latest combined tree:

```text
go test -count=1 ./contracts -run TestA2AProfileConformance
go test -count=1 ./...
go vet ./...
git diff --check
```

Fallback delta was `removed 0, retained 0, added 0, net +0`. There is no fresh
independent Review after `46ca271`, so Module C is not yet PASS. The next action
for Module C is review only; do not modify it before a fresh Reviewer reports a
finding.

## Shared Integration - NOT STARTED

Do not begin shared integration until Module A and Module C both explicitly
PASS fresh independent Review.

Remaining integration work is T025-T032 in
`specs/001-complete-invocation-contracts/tasks.md`:

- update active aliases/constants in `contracts/contracts.go`;
- update active schemas/resources/validators in `contracts/validate.go`;
- preserve historical parse checks in `contracts/contracts_test.go`;
- add `contracts/active_contracts_integration_test.go`;
- update current status references in `AGENTS.md` and `README.md`;
- run every command in the Spec quickstart and commit the integration;
- with a clean worktree, fetch `origin`, resolve its HEAD, rebase onto the latest
  remote HEAD, and rerun all verification;
- run a fresh integrated Review against that rebased remote base.

No remote push is authorized unless the user explicitly requests it.

## Resume Order

1. Verify `git status --short --branch` and confirm HEAD includes `46ca271`.
2. Amend/analyze Spec 001 for Module A's large-number finding, then commit docs.
3. Fix Module A through its implementation Agent, run A+B regressions, and
   obtain a fresh independent Module A PASS.
4. Independently review Module C commit `46ca271`; fix and re-review only if it
   fails.
5. Start shared integration only after both gates pass.
6. Fetch/rebase onto the latest `origin/HEAD`, rerun the full quickstart, and run
   the final integrated Review.
7. Run Spec Kit convergence, update task/status artifacts, and produce the final
   Spec 001 report.

## Verification Notes

- Checksum verification stays enabled. Recommended environment on this machine:
  `GOPROXY=https://goproxy.cn,direct` and
  `GOSUMDB=sum.golang.google.cn`.
- `go test -race` has not been claimed because Windows currently has
  `CGO_ENABLED=0` and no GCC toolchain.
- Legacy Node artifacts and all `node_modules` directories were removed earlier;
  package manifests and lockfiles were intentionally retained.

## Suggested Skills

- `speckit-specify` for the Module A requirement amendment
- `speckit-analyze` after updating Spec/Plan/Tasks
- `implement` for approved code changes
- `code-review` for every fresh module and integrated Review
- `ai-fallback-disable` for all failure/default/compatibility decisions
- `speckit-converge` only after the final integrated Review passes
