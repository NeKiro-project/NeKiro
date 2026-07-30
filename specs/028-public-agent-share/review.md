# Independent Review: Public Agent Share URL

**Review date**: 2026-07-30
**Scope**: Issue #71, `spec.md`, `plan.md`, `tasks.md`, contract artifacts, project constitution in `AGENTS.md`, and the implemented Spec 028 code and acceptance evidence.

## Result

**Approved for convergence. No blocking findings.**

The implementation preserves the existing `Register -> Discover -> Install -> Invoke -> Record` boundary and adds a Catalog-owned immutable public identity with a Gateway-only anonymous projection. The public URL remains discovery metadata and is never used as an Agent endpoint, credential, or installation authorization.

## Review Checks

| Area | Evidence reviewed | Result |
| --- | --- | --- |
| Stable identity | Catalog v5 migration, immutable trigger, unique index, registration and read paths | Pass |
| Canonical origin | `NEKIRO_PUBLIC_AGENT_ORIGIN`, `VITE_NEKIRO_PUBLIC_AGENT_ORIGIN`, exact URL validation | Pass |
| Contract ownership | Public Agent Share v1 JSON Schema/OpenAPI and Catalog v3 additive fields | Pass |
| Anonymous resolution | `GET /v4/public/agents/{publicAgentId}`, no authentication header, explicit validation/not-found/dependency errors | Pass |
| Projection safety | Only published trusted Releases are projected; draft Card, endpoint, binding, evidence, credential, Workspace, and Ledger fields are excluded | Pass |
| Exact Release semantics | No preselection; explicit UI choice; exact Card version and post-install `installedReleaseId` validation | Pass |
| Workspace boundary | Share flow reuses authenticated existing Installation API and Workspace owner policy | Pass |
| Invocation boundary | Share URL does not enter invocation transport; existing Gateway -> Router -> Agent path remains unchanged | Pass |
| Lifecycle behavior | Published/suspended/revoked/unpublished states are resolved from authoritative Release facts without substitution or retry | Pass |
| Migration/readiness | Embedded and checked-in migration match; schema version/checks cover public identity column, index, and immutability trigger | Pass |
| Secret and endpoint leakage | Unit, integration, E2E, and secrecy scans cover response, URL, logs, browser requests, and fixtures | Pass |
| Failure policy | No new retry, alternate origin, Host inference, legacy alias, latest selection, fallback source, or silent success | Pass |

## Verification Evidence

The following commands passed on the review worktree:

```text
go test ./...
go vet ./...
pnpm --dir apps/console typecheck
pnpm --dir apps/console test -- --run
pnpm --dir apps/console build
git diff --check
```

The recorded clean-volume backend acceptance and production-route Playwright acceptance also passed, including public resolution, exact installation, lifecycle transitions, cross-runtime invocation lineage, secrecy checks, direct `/a/{publicAgentId}` navigation, pasted canonical URL resolution, anonymous browser access, JSON/SSE invocation, and Ledger/Trace reads.

## Findings

No CRITICAL, HIGH, MEDIUM, or LOW findings were identified that require a code or contract change before delivery. The existing project-level deferred governance items remain outside this feature's approved scope.

## Convergence Decision

Spec 028 is converged against its approved specification, plan, contracts, tasks T001-T039, and constitution constraints. T040 review and T042 convergence evidence are complete; no additional implementation task is required.

Fallback delta: removed 0, retained 0, added 0, net 0.
Added fallback evidence: none.
