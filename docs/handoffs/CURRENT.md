# Current Handoff: Invocation Stack Integration

**Updated**: 2026-07-18 (Asia/Hong_Kong)

**State**: T001 and T002 are on `main`. The T003-T008 implementation stack is
combined with current `main` on
`codex/010-invocation-stack-integration` and is being delivered through PR
#41. T009 Agent SDK, T010 Runtime A nested caller, and T011 full backend
acceptance remain open under parent Issue #19.

## Repository State

- Upstream: `https://github.com/NeKiro-project/NeKiro.git`
- Fork: `https://github.com/XnLemon/NeKiro.git`
- Integration branch: `codex/010-invocation-stack-integration`
- Base: upstream `main` at PR #34 merge commit `3577ffc`
- Integration PR: `https://github.com/NeKiro-project/NeKiro/pull/41`
- Active parent artifacts: `specs/010-invocation-routing-ledger/`
- Required Git identity: `Nene7ko_ <1604009816@qq.com>`

The previous stacked PRs #35, #36, #39, #40, #38, and #37 merged into
dependency branches, not `main`. PR #41 is their single `main`-based integration
point and includes the latest `main` changes.

## Delivered Scope

- **T001 / Spec 011**: Invocation, Router, Ledger, SDK-facing, size, deadline,
  cancellation, error, and compatibility contracts are frozen.
- **T002 / Spec 012**: Gateway v4 Invocation Dispatch authorizes the exact
  Workspace installation and forwards live JSON or SSE only through Router
  Internal v3.
- **T003 / Spec 013**: The independently deployed A2A Router has strict required
  configuration, service authentication, readiness, and controlled Control
  Plane exact resolution.
- **T004 / Spec 014**: The Router owns an append-only metadata-only Invocation
  Ledger, transactional projection, PostgreSQL migrations/readiness, and
  Workspace-scoped Invocation/Trace reads.
- **T005 / Spec 015**: Runtime B is a deterministic direct A2A callee supporting
  `message/send`, `message/stream`, `tasks/get`, and `tasks/cancel` without
  platform database access.
- **T006 / Spec 016**: Non-streaming exact dispatch returns transient results
  only after terminal Ledger persistence and preserves explicit transport
  failures.
- **T007 / Spec 017**: Streaming enforces separate A2A/SSE bounds, ordered
  accepted/chunk/terminal frames, deadline/disconnect cancellation, and
  first-terminal-wins behavior.
- **T008 / Spec 018**: Authorized Northbound Invocation and Trace reads preserve
  Workspace isolation and expose metadata only.

## Verification

Local integration verification passed:

```powershell
go test -count=1 ./...
go vet ./...
go build ./...
go mod tidy -diff
gofmt -l apps agents contracts
git diff --check
```

PR #41 GitHub Actions run `29614928695` passed:

- `workspace-integration`, including the PostgreSQL 17 Ledger integration suite
- `go-quality`
- `frontend`
- `compose-config`

The local Docker daemon is not running, the configured `Ubuntu-26.04` WSL
distribution is absent, and Corepack timed out downloading pnpm. These local
limitations are not reported as passing checks; GitHub CI supplies the required
PostgreSQL and frontend evidence. The root `.dockerignore` now explicitly
includes `apps/a2a-router/**` for the Router image build context.

## Remaining Delivery Gates

- Complete the fresh independent Review and Spec 014 Converge record for PR
  #41 before marking the integration ready.
- Merge PR #41 into `main`, then close Issues #22-#27 and link their stacked PRs
  to the integration PR.
- Keep #19 and Issues #28-#30 open.

After PR #41, the next implementation entry is T009 / Issue #28 under
`specs/019-agent-sdk-nested-invocation/`. Do not start T010 or T011 before their
declared blockers are complete.

## Runtime and Fallback Boundaries

The implemented managed root path is:

```text
Gateway -> Invocation Dispatch -> A2A Router -> Runtime B
                              \-> metadata-only Invocation Ledger
```

The Agent SDK, Runtime A, Router-mediated nested invocation, complete
`Register -> Discover -> Install -> Invoke -> Record` clean-environment E2E,
and Frontend runtime remain unimplemented. Results and stream chunks remain
transient; Ledger stores metadata only.

```text
Fallback delta: removed 2, retained 0, added 0, net -2
Added fallback evidence: none
```

No retry, cache, alternate route/store, stale Card, compatibility runtime
branch, default credential, inferred endpoint, or degraded success was added.

## Recovery

```powershell
git clone https://github.com/XnLemon/NeKiro.git
Set-Location NeKiro
git remote add upstream https://github.com/NeKiro-project/NeKiro.git
git fetch origin --prune
git fetch upstream --prune
git switch --track origin/codex/010-invocation-stack-integration
git config --local user.name Nene7ko_
git config --local user.email 1604009816@qq.com
git status --short --branch
```

Before changing behavior, read `AGENTS.md`, the active child Spec artifacts,
the language-neutral contracts, and the relevant ADRs. Public behavior,
contract, data ownership, or failure-policy changes must return to SDD before
code changes.
