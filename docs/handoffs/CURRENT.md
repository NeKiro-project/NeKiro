# Current Handoff: Invoke-to-Record Backend

**Updated**: 2026-07-21 (Asia/Shanghai)

**State**: The backend Invoke-to-Record slice is implemented on
`codex/021-invoke-record-acceptance` at `d9c1ed9`. PR #44 was merged upstream,
Issue #30 was closed, and the parent Issue #19 was closed after the clean
Compose/PostgreSQL acceptance passed.

## Repository State

- Upstream: `https://github.com/NeKiro-project/NeKiro.git`
- Fork: `https://github.com/XnLemon/NeKiro.git`
- Current branch: `codex/021-invoke-record-acceptance`
- Active backend acceptance artifacts: `specs/021-invoke-record-acceptance/`
- Parent invocation artifacts: `specs/010-invocation-routing-ledger/`
- Required Git identity: `Nene7ko_ <1604009816@qq.com>`

## Delivered Scope

- Catalog, Discovery, Workspace, Installation, and exact-version authorization.
- Gateway v4 Invocation Dispatch through Router Internal v3.
- Independent A2A Router with JSON/SSE delivery and strict endpoint resolution.
- Router-owned append-only metadata-only Invocation Ledger and scoped reads.
- Runtime B direct A2A sample.
- Thin Agent SDK and authenticated Router-mediated nested invocation.
- Isolated Runtime A caller using `trpc-agent-go` only inside its own module.
- Runtime A -> Router -> Runtime B parent-child lineage.
- Compose/PostgreSQL deployment and real Invoke-to-Record E2E acceptance.

## Verification

GitHub Actions run `29810057739` passed `go-quality`,
`runtime-samples-quality`, `workspace-integration`, `compose-config`,
`frontend`, and `backend-acceptance`. The acceptance covers JSON, SSE, nested
lineage, restart durability, Workspace isolation, failure semantics, secrecy,
and 100-concurrent outcomes. Local Docker unavailability is recorded in
`specs/021-invoke-record-acceptance/quickstart.md`; it is not treated as a local
pass.

## Remaining Scope

- `apps/console` is not present; frontend work remains intentionally paused.
- `sdks/client-sdk` and production identity/governance are not implemented.
- Spec 010 T020/T021 remain `Needs policy` for task retention/capacity,
  timeout ownership, graceful shutdown, and in-flight SSE/Ledger semantics.
- Do not add retry, cache, stale-card compatibility, silent task eviction, or
  degraded success without a new approved Spec/ADR.

Before changing public behavior, read `AGENTS.md`, the active child Spec
artifacts, the language-neutral contracts, and the relevant ADRs. Contract,
data-ownership, trace, or failure-policy changes must return to SDD before code
changes.
