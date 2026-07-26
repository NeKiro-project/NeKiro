# Implementation Plan: Console Trusted Publication Loop

**Branch**: `codex/027-console-trusted-publication` | **Date**: 2026-07-26 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/027-console-trusted-publication/spec.md`

## Summary

Advance the existing production Glass 2.0 Console in its standalone upstream
repository, make its public Gateway client consume the active Trusted
Publication v1 contracts, add provider Release operations and release-aware
Workspace controls, then import the reviewed Console into the platform
repository as `apps/console` for root CI and a fresh-environment browser
acceptance. Extend the backend acceptance separately to prove the canonical
Runtime B -> Router -> Runtime A direction.

The Console remains a thin northbound client. It does not own Agent Cards,
trust state, Release state, Workspace authorization, routing, or Ledger facts.
The browser calls only Gateway routes and keeps all configured credentials in
memory supplied through explicit environment configuration. Provider and
Workspace-owner credentials are separate client contexts, and provider identity
is supplied explicitly rather than inferred from an Agent Card.

## Technical Context

**Language/Version**: TypeScript with the existing React/Vite frontend; Go for the existing backend acceptance harness

**Primary Dependencies**: Existing Console React, Vite, TailwindCSS, lucide-react, Motion, Node test runner, and the active NeKiro JSON/OpenAPI contracts; browser acceptance may use a repository-approved Playwright runner only if required by the existing CI environment

**Storage**: No new storage. Console state is transient browser memory. Existing PostgreSQL Catalog, Workspace, and Router Ledger schemas remain authoritative.

**Testing**: TypeScript typecheck, focused Node contract tests, production build, fresh Compose/PostgreSQL backend acceptance, and browser-level production-route acceptance

**Target Platform**: Browser and Linux CI for the frontend; portable Go tests with Linux Compose for backend acceptance

**Project Type**: React web application plus cross-service acceptance coverage

**Performance Goals**: Existing Console interaction responsiveness; no new polling, retry, reconnect, background health loop, or result persistence is permitted

**Constraints**: Gateway-only browser traffic; active contract versions only; strict response validation; no direct Agent, Router-internal, database, mock-production, localhost, token-trimming, legacy-publish, or alternate-endpoint fallback

**Scale/Scope**: One Owner-only Workspace, explicit development-static principals, two independently implemented sample Agents, one production Console route, and four independently reviewable implementation slices

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **Phase 1 loop**: PASS. The Console makes Register -> Discover -> Install -> Invoke -> Record operational, while trusted publication supplies the required trust gate.
- **Ownership**: PASS. Console owns only presentation and request mapping; Catalog owns Card/Binding/Release facts, Workspace owns Installations, Router owns invocation transport and Ledger facts.
- **Runtime independence**: PASS. The UI and acceptance use protocol/contract fields only; Runtime A and Runtime B remain independent sample implementations.
- **Contracts**: PASS. No public contract is changed. The plan maps Trusted Publication v1, Card 0.2, Catalog/Northbound v3, Workspace/Installation v2, Invocation v4, Router metadata v3, Event 0.3, and stream result v2. Because Catalog search has no Release ID and Installation remains version-constraint based, Slice B uses an explicit Release-ID read preflight and validates the returned installed Release ID after installation.
- **Invocation lineage**: PASS. The UI validates and displays Gateway/Router-provided identifiers and the backend acceptance asserts one root Task/Trace with a parent child relationship.
- **Failure safety**: PASS. Invalid, missing, forbidden, dependency, lifecycle, timeout, cancellation, and stream protocol failures remain distinct. No retry, alternate endpoint, or silent success is added.
- **SDD traceability**: PASS. Every implementation and test task maps to a Spec story or requirement; tests are scheduled after the corresponding implementation.
- **Cross-runtime proof**: PASS. The reverse B -> Router -> A fixture is a separate task and acceptance gate, and the Console is runtime-neutral.

**Post-design gate**: PASS. The design does not introduce a new contract, data owner, backend dependency, or fallback policy.

## Research Decisions

See [research.md](research.md) for the evidence and decisions. The important
decisions are:

1. Implement Console Issues #2/#4/#3 in the standalone `NeKiro-Console` repository, which is the upstream frontend source.
2. Import the reviewed standalone Console source into `apps/console` in the platform repository for root pnpm discovery and deployment ownership; do not use a submodule or maintain a second hand-edited production implementation.
3. Add strict handwritten TypeScript mappings for Trusted Publication v1 rather than generating runtime objects or copying Go internals.
4. Use server reads after mutations. Do not optimistically advance Binding, Release, Installation, or Invocation state.
5. Treat the one-time challenge proof as transient response data only. Do not put it in local storage, URLs, logs, or later request bodies.
6. Extend the existing backend acceptance fixture for B -> Router -> A rather than adding a second Compose stack or a runtime framework dependency.
7. Require explicit browser configuration names: `VITE_NEKIRO_PROVIDER_ID`, `VITE_NEKIRO_PROVIDER_TOKEN`, `VITE_NEKIRO_OWNER_TOKEN`, and `VITE_NEKIRO_DEFAULT_WORKSPACE_ID`; the previous single-token configuration is not a compatibility input.
8. Use `GET /v4/releases/{releaseId}` as the Workspace installation handoff. The UI must receive or ask for the Release ID, preflight its published state and exact Card identity, submit an exact version constraint, and reject an installation response with a different Release ID.

## Data and Contract Design

The data ownership and frontend view mapping are documented in
[data-model.md](data-model.md). The active endpoint and response mapping matrix
is documented in [contracts/console-gateway-mapping.md](contracts/console-gateway-mapping.md).

No migration or contract version bump is planned. The active database uniqueness
rule permits one immutable Release per Agent/Card version, so the Release-ID
preflight plus post-install equality check supplies provenance without changing
the Installation request. If implementation discovers that an active Gateway
route cannot support this flow, work must stop and return to the Spec/ADR
process rather than adding a browser fallback.

## Project Structure

### Documentation

```text
specs/027-console-trusted-publication/
|-- spec.md
|-- plan.md
|-- research.md
|-- data-model.md
|-- quickstart.md
|-- contracts/console-gateway-mapping.md
|-- checklists/requirements.md
`-- tasks.md
```

### Production Console in the platform repository

```text
E:/Progarms/NeKiro-Console/
|-- package.json
|-- src/
|   |-- App.tsx
|   |-- api/nekiro.ts
|   |-- api/nekiro.test.ts
|   |-- components/
|   |   |-- RegistryTab.tsx
|   |   |-- TrustedPublicationTab.tsx
|   |   |-- InstallationsTab.tsx
|   |   |-- InvocationsTab.tsx
|   |   `-- LedgerTab.tsx
|   |-- types.ts
|   `-- demos/
`-- e2e/

E:/Progarms/NeKiro/apps/console/   # reviewed source imported for platform CI
```

### Backend acceptance

```text
tests/e2e/invoke-record/invoke_record_test.go
specs/027-console-trusted-publication/docs/reverse-lineage.md (if needed)
```

**Structure Decision**: Keep the platform backend and Router boundaries
unchanged. Console Issues #2/#4/#3 write to the standalone Console repository;
after each reviewed slice, the resulting source is imported into the platform
repository as a normal `apps/console` workspace package so existing root CI
commands discover real frontend scripts. The imported copy is generated from
the reviewed upstream source and is not a second hand-edited production path.
Add only the deterministic Runtime B caller behavior and backend acceptance
fixture required to prove the reverse nested direction. Runtime B may reuse the
existing Agent SDK and Router Agent v1 boundary, but it must not gain a direct
Runtime A endpoint, platform storage access, a new runtime framework, or a
second public contract.

## PR and Review Slices

| Slice | Upstream Issue | Owner boundary | Independent completion |
| --- | --- | --- | --- |
| A | Console #2 | standalone `NeKiro-Console/src/api`, API tests, trusted transport validation | All active Trusted Publication requests and strict response/error mappings pass focused tests |
| B | Console #4 | standalone `NeKiro-Console/src/App.tsx`, API client configuration, and production components | Provider can complete Binding/Challenge/Release lifecycle; Workspace owner preflights an explicit published Release and installation validates the exact returned Release ID |
| C | NeKiro #60 | Runtime B deterministic caller fixture and backend E2E acceptance | Runtime B -> Router -> Runtime A returns with one correlated root/child lineage and provenance |
| D | Console #3 / parent #59 | standalone Console CI/browser acceptance plus reviewed import to `apps/console` and root CI | CI executes real typecheck/tests/build/browser acceptance; live production route proves the loop |

Each slice is implemented on its own branch/PR. Before coding a slice, a
separate subagent must read the Issue and this Spec and report scope,
dependencies, and forbidden fallback behavior. After implementation, an
independent review agent must review the diff against Issue, Spec, Plan, Tasks,
contracts, and constitution. A failing review returns the slice to Tasks and
requires a fresh review after fixes.

## Complexity Tracking

No constitution violation requires justification. The additional repository
package is required because root CI currently uses a workspace-wide no-op when
no frontend package is present; importing the existing production Console into
`apps/console` removes that false green without changing platform service
ownership.

## Verification Commands

```text
pnpm install --frozen-lockfile
pnpm typecheck
pnpm test
pnpm build
go test ./...
go test -tags=e2e -count=1 ./tests/e2e/invoke-record
go vet ./...
docker compose --file deploy/compose.yaml config --quiet
git diff --check
```

The browser acceptance additionally requires the explicit environment values
documented in [quickstart.md](quickstart.md). No command may substitute a
localhost, mock, or alternate endpoint when those values are missing.
