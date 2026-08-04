# Implementation Plan: Public Agent Share URL

**Branch**: `codex/028-public-agent-share` | **Date**: 2026-07-30 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/028-public-agent-share/spec.md`; upstream Issue [#71](https://github.com/NeKiro-project/NeKiro/issues/71).

## Summary

Give every Catalog Agent identity one immutable opaque `publicAgentId` and an operator-origin canonical URL. Add an anonymous Gateway read that resolves that identity to the public identity state and all currently eligible published trusted Releases. All Card-derived display, owner, capability, and permission fields live inside eligible Release items, so an identity with no published trusted Release leaks no draft Card data. The Console accepts the canonical URL either as `/a/{publicAgentId}` navigation or pasted input, requires B to explicitly select one exact Release and permissions, and delegates the write to the existing authenticated Workspace Installation boundary. Invocation and Ledger behavior remain unchanged.

## Technical Context

**Language/Version**: Go 1.26 for Control Plane and contracts; TypeScript 5.8 with React 19/Vite 6 for Console

**Primary Dependencies**: Go standard `net/http`, `crypto/rand`, existing `pgx/v5` Catalog store, existing React/Vite Console, existing versioned NeKiro contracts

**Storage**: PostgreSQL Catalog schema v5 adds one immutable `public_agent_id` to `catalog.agent_identities`; Workspace storage is unchanged

**Testing**: Go unit/contract/integration tests, Console typecheck and Node tests, production build, clean Compose backend E2E, production-route Playwright acceptance

**Target Platform**: Linux server/CI and modern browsers; Windows local development remains supported

**Project Type**: Control Plane web service plus React web application

**Performance Goals**: Resolve one public identity and at most 100 eligible Releases in one bounded Catalog read; no polling or background reconciliation

**Constraints**: Gateway-only frontend traffic; anonymous read-only resolution; exact Release selection; no endpoint/proof/credential exposure; no implicit latest, retry, alternate origin, alias, redirect, stale data, or Release substitution

**Scale/Scope**: One public identity per Agent identity, up to 100 visible published Releases per response, existing single-owner Workspace model, two cross-runtime Sample Agents

## Constitution Check

*GATE: Passed before research and re-checked after design.*

- **Phase 1 loop**: PASS. The URL closes the handoff from trusted publication/discovery to exact Workspace installation and managed invocation.
- **Ownership**: PASS. Catalog owns public identity and the public Release projection; Workspace continues to own Installation; Router/Ledger are unchanged.
- **Runtime independence**: PASS. Public identity and trusted Release metadata do not depend on language, model, framework, or runtime internals.
- **Contracts**: PASS. Public Agent Share v1 is a language-neutral OpenAPI/JSON Schema contract. Existing Catalog v3 gains only optional response fields; Installation v2 is reused unchanged.
- **Invocation lineage**: PASS. The public URL never invokes. Installed Agents continue through Gateway/Router with exact Release provenance and existing Task/Trace lineage.
- **Failure safety**: PASS. Malformed, unknown, not-installable, unauthorized, conflict, lifecycle, and dependency states remain distinct. No fallback is introduced.
- **SDD traceability**: PASS. Issue #71, Spec FRs, tasks, contracts, and acceptance tests map directly.
- **Cross-runtime proof**: PASS. Backend E2E installs a Release discovered through the public view and retains Runtime B -> Router -> Runtime A lineage.

**Post-design gate**: PASS. The design adds no Runtime behavior, alternate data owner, direct Agent path, or parallel Installation state machine.

## Research Decisions

See [research.md](research.md). The binding decisions are:

1. Use a generated `agt_` plus 128-bit lowercase-hex public identifier, separate from Agent Card `agentId` and Release ID.
2. Backfill existing identities once during Catalog migration v5 and enforce uniqueness plus update immutability in PostgreSQL.
3. Require explicit `NEKIRO_PUBLIC_AGENT_ORIGIN` and `VITE_NEKIRO_PUBLIC_AGENT_ORIGIN`; missing, blank, credential-bearing, path-bearing, or otherwise invalid origins fail at their owning boundary.
4. Add anonymous `GET /v4/public/agents/{publicAgentId}`. The identity envelope contains no Card-derived fields; each eligible Release contains its own safe published Card projection and immutable Release provenance. Agent protocol endpoint, binding IDs, evidence digests, credentials, drafts, and Ledger data are excluded.
5. Return an existing registered identity with `availability=not_installable` and an empty Release list; an unknown identity is `NOT_FOUND`, malformed input is `VALIDATION_ERROR`, and dependency failure remains `DEPENDENCY_ERROR`.
6. Present eligible Releases in deterministic published-time order for display only. The client must require explicit selection and must not preselect the first entry.
7. Reuse the exact-version Installation request and verify `installedReleaseId` equals the selected public Release ID.

## Data and Contract Design

The owned data and invariants are in [data-model.md](data-model.md). The public/Gateway/Installation mapping is in [contracts/public-agent-share-mapping.md](contracts/public-agent-share-mapping.md).

The language-neutral facts are:

- `contracts/schemas/public-agent-share.v1.schema.json`
- `contracts/openapi/public-agent-share.v1.yaml`
- additive optional `publicAgentId` and `publicUrl` properties on Catalog v3 `CatalogEntry`

The public contract intentionally omits the A2A endpoint and endpoint-binding provenance. Exact trust is represented by Release ID, Card version, Card digest, state, and publication time; installation revalidates the authoritative Release.

## Project Structure

### Documentation

```text
specs/028-public-agent-share/
|-- spec.md
|-- plan.md
|-- research.md
|-- data-model.md
|-- quickstart.md
|-- contracts/public-agent-share-mapping.md
|-- checklists/requirements.md
`-- tasks.md
```

### Source Code

```text
contracts/
|-- contracts.go
|-- public_agent_share.go
|-- schemas/public-agent-share.v1.schema.json
`-- openapi/public-agent-share.v1.yaml

apps/control-plane/
|-- migrations/005_public_agent_share.sql
|-- internal/catalog/
|   |-- public_share.go
|   |-- model.go
|   `-- postgres/
|       |-- 005_public_agent_share.sql
|       |-- migrations.go
|       `-- store.go
|-- internal/config/config.go
|-- internal/gateway/public_share_handler.go
`-- cmd/control-plane/main.go

apps/console/src/
|-- api/nekiro.ts
|-- publicAgentUrl.ts
|-- components/PublicAgentPage.tsx
|-- components/PublicAgentInstallPanel.tsx
|-- components/InstallationsTab.tsx
|-- main.tsx
`-- types.ts

tests/e2e/invoke-record/invoke_record_test.go
deploy/compose.yaml
.github/workflows/ci.yml
```

**Structure Decision**: Extend the existing Catalog deployment and production Console. Public resolution is a Catalog-owned read model behind Gateway; no new service, database, Router dependency, or Marketplace module is introduced.

## Complexity Tracking

No constitution violation requires justification.

## Verification Commands

```text
go test ./...
go vet ./...
pnpm typecheck
pnpm test
pnpm build
go test -tags=e2e -count=1 ./tests/e2e/invoke-record
pnpm --dir apps/console run test:e2e
docker compose --file deploy/compose.yaml config --quiet
git diff --check
```
