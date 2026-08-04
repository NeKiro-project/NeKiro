# Implementation Plan: Core Repository Boundary

**Branch**: `codex/core-repository-split` | **Date**: 2026-08-04 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/030-core-repository-boundary/spec.md`, migration Issue [#80](https://github.com/NeKiro-project/NeKiro/issues/80), and owner authorization to create repositories, push branches, open and merge pull requests, align CI, and remove tracked Spec Kit artifacts.

**Note**: This plan is the final tracked Spec Kit transition artifact. It is
preserved by the annotated pre-split tag and Git history, then removed from the
post-cutover core tree with `specs/`, `.specify/`, and repository-local Spec Kit
skills.

## Summary

Turn `NeKiro-project/NeKiro` into a core-only Go repository while preserving
history and product behavior. Console, Go SDKs, cross-runtime samples, and the
multi-component stack become canonical satellite repositories with independent
CI and immutable release boundaries. Core-owned Catalog, Workspace, and Ledger
SQL remains beside its owner and is consolidated to one embedded file per
schema version. The official Go module path becomes
`github.com/NeKiro-project/NeKiro`. Product acceptance moves to NeKiro-Stack,
which pins exact component revisions and is the only repository that assembles
the full platform.

## Technical Context

**Language/Version**: Go 1.26 for core, SDKs, samples, and backend acceptance; TypeScript 5.8/React 19 with Node 24.16 and pnpm 11.3 for Console

**Primary Dependencies**: PostgreSQL 17, `pgx/v5`, `tern/v2`, `a2a-go`, `nekiro-a2a-transport-go`, Docker Compose, GitHub Actions, Playwright

**Storage**: PostgreSQL with independently owned Catalog, Workspace, and Ledger schemas; no data move or schema-semantic change

**Testing**: Go unit/contract/race/vet/lint, tagged PostgreSQL integration tests, migration/readiness tests, Console typecheck/unit/build, sample module tests, Compose validation, backend E2E, Playwright product acceptance

**Target Platform**: Linux containers and GitHub-hosted Ubuntu CI; local Windows and Unix development supported by repository-specific instructions

**Project Type**: Multi-repository platform consisting of two core Go services, a React Console, Go libraries, sample Agent services, and a release-assembly repository

**Performance Goals**: No runtime performance change; repository-local CI avoids unrelated toolchains and cancels superseded runs

**Constraints**: Preserve public contracts, database upgrade paths, authorization, errors, cancellation, and Ledger lineage; no source mirrors, local production replacements, floating dependencies, alternate components, or fallback paths

**Scale/Scope**: Approximately 690 tracked files split across six canonical repositories; three new repositories plus updates to three existing repositories

## Constitution Check

*GATE: Passed before research and re-checked after design.*

- **Core loop**: PASS. ADR 0009 identifies repository extraction as a blocking
  ownership prerequisite and Stack retains the full Register -> Discover ->
  Install -> Invoke -> Record acceptance.
- **Ownership**: PASS. Control Plane, Router, contracts, and schema migrations
  remain core; each extracted family has one satellite owner.
- **Runtime independence**: PASS. Core tests replace Runtime B imports with
  protocol fixtures; real Runtime A/B proof moves to Stack.
- **Contracts**: PASS. Language-neutral contracts remain canonical in core.
  Repository and module paths change, but wire and schema semantics do not.
- **Invocation lineage**: PASS. Stack retains direct, streaming, nested, and
  Ledger lineage acceptance with immutable component pins.
- **Failure safety**: PASS. Missing or incompatible component releases fail
  before assembly. No fallback is added.
- **Delivery traceability**: PASS. Issue, ADR 0009, extraction history, pull
  requests, checks, releases, and the pre-split tag provide durable evidence.
- **Cross-runtime proof**: PASS. NeKiro-Stack owns the two-runtime acceptance;
  core does not import sample source.

**Post-design gate**: PASS. The design preserves service and data ownership,
adds no protocol or product behavior, and makes every cross-repository
dependency explicit and immutable.

## Research Decisions

See [research.md](research.md). Binding decisions:

1. Keep SQL with its owning core module and embed one canonical file set.
2. Export existing source history before deleting any core copy.
3. Change the core module path to the official organization identity before
   publishing downstream Go modules.
4. Make `NeKiro-Stack` the only full-product assembler and acceptance owner.
5. Use workflow name `CI`, least privilege, shared toolchain versions, explicit
   timeouts, and an aggregate gate in every repository, with owner-specific
   integration jobs.
6. Preserve historical Spec Kit content by an annotated pre-split tag and Git
   history, then remove it from the tracked core tree.

## Data and Contract Design

Repository ownership records, component releases, integration manifests, and
cutover state are defined in [data-model.md](data-model.md). Exact path ownership
is captured in [contracts/artifact-ownership.md](contracts/artifact-ownership.md),
and Stack manifest invariants are defined in
[contracts/release-manifest.md](contracts/release-manifest.md).

## Project Structure

### Transitional Documentation

```text
specs/030-core-repository-boundary/
|-- spec.md
|-- plan.md
|-- research.md
|-- data-model.md
|-- quickstart.md
|-- contracts/
|   |-- artifact-ownership.md
|   `-- release-manifest.md
|-- checklists/requirements.md
`-- tasks.md
```

### Target Repositories

```text
NeKiro/
|-- apps/
|   |-- control-plane/
|   |   `-- internal/{catalog,workspace}/postgres/migrations/
|   `-- a2a-router/
|       `-- internal/ledger/migrations/
|-- contracts/
|-- tests/{contract,integration}/
|-- docs/{architecture,contracts,decisions,usage}/
|-- .github/workflows/ci.yml
|-- AGENTS.md
|-- go.mod
`-- go.sum

NeKiro-Console/
|-- src/
|-- e2e/
|-- .github/workflows/ci.yml
|-- package.json
`-- pnpm-lock.yaml

nekiro-sdk-go/
|-- agent/
|   `-- routerauth/
|-- client/
|-- .github/workflows/ci.yml
|-- go.mod
`-- go.sum

NeKiro-Samples/
|-- internal/challengeproof/
|-- runtime-a/
|-- runtime-b/
|-- .github/workflows/ci.yml
|-- go.mod
`-- go.sum

NeKiro-Stack/
|-- compose.yaml
|-- components.json
|-- tests/backend/
|-- tests/browser/
|-- scripts/
`-- .github/workflows/ci.yml
```

**Structure Decision**: Keep Control Plane and Router in one core repository
because their logical boundaries do not require more service repositories.
Extract only components with independent product ownership and releases. The
existing transport repository remains independent and is aligned to the same
CI contract without absorbing NeKiro-specific policy.

## Implementation Sequence

1. Create Issue, empty target repositories, and the annotated provenance tag.
2. Merge a preliminary core PR containing ADR 0009, the official Go module
   identity, and the transitional Spec 030 governance artifacts.
3. Export SDK, Samples, and Stack history from that official-identity baseline.
4. Bootstrap SDK and then Samples against immutable core/SDK revisions.
5. Synchronize the latest Console source and reduce Console CI to frontend
   ownership.
6. Bootstrap Stack with exact component revisions and move product E2E.
7. On a new core cutover branch, consolidate migrations, replace sample imports
   in core tests, and reduce core CI, documentation, source, and root tooling.
8. Align the transport workflow and repository rules.
9. Run repository-local and cross-repository verification, independent review,
   and merge in dependency order.
10. Remove transitional Spec Kit artifacts in the final core commit and prove
    the post-cutover tracked tree has no satellite source.

## Complexity Tracking

No constitution violation requires justification.

## Verification

Repository-specific commands are listed in [quickstart.md](quickstart.md). The
final acceptance requires every repository's `CI / required` check plus a clean
NeKiro-Stack run against an explicit immutable component manifest.
