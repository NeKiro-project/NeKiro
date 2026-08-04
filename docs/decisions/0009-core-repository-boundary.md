# ADR 0009: Core Repository and Satellite Ownership

- Status: Accepted
- Date: 2026-08-04
- Decision owner: NeKiro project owner
- Migration issue: [#80](https://github.com/NeKiro-project/NeKiro/issues/80)

## Context

Before the repository split, NeKiro contained core services, canonical
contracts, service-owned SQL migrations, a copied production Console, public
SDKs, sample Agent Runtimes, full-stack Compose assembly, cross-product
acceptance, Spec Kit artifacts, and delivery-history documents. That made the
core checkout a second source for independently maintained components and
coupled unrelated release and CI lifecycles.

SQL is not an independent product in this repository. Catalog, Workspace, and
Ledger migrations express the persistent-state contract owned by the services
that execute and validate those schemas. Moving them to a database repository
would reverse data ownership and create another release dependency.

## Decision

The project adopts these canonical owners:

| Owner | Canonical responsibility |
|---|---|
| `NeKiro-project/NeKiro` | Control Plane, A2A Router, contracts, owned migrations, core tests, architecture, and core usage |
| `NeKiro-project/NeKiro-Console` | Production Console and browser behavior |
| `NeKiro-project/nekiro-sdk-go` | Public Go Agent and application SDKs |
| `NeKiro-project/NeKiro-Samples` | Runtime A, Runtime B, sample Cards, and sample-specific tests |
| `NeKiro-project/NeKiro-Stack` | Multi-component development stack, immutable release manifest, and product acceptance |
| `NeKiro-project/nekiro-a2a-transport-go` | Reusable A2A HTTP, JSON-RPC, and SSE transport mechanics |

The core repository retains Catalog, Workspace, and Ledger migrations beside
their owning modules. Each migration version has one canonical SQL source;
manually synchronized SQL or embedded string copies are removed.

Cross-repository production dependencies use immutable reviewed versions or
digests. Missing or incompatible releases fail explicitly. The core repository
does not retain source mirrors, vendored fallbacks, local production
replacements, floating branches, or alternate component paths.

Core pull-request CI remains source-independent from every satellite. A
separate workflow triggered after each merge to Core `main` may call reusable
workflows owned by the SDK, Samples, and Stack repositories. Every reusable
workflow reference is pinned to a full commit SHA, receives the exact merged
Core SHA explicitly, and keeps its checkout, commands, and success criteria in
the owning satellite repository. This post-merge orchestration is compatibility
evidence, not a source mirror or a fallback Core build path.

The existing Spec Kit process is retired after the final repository-extraction
feature. Future durable evidence is GitHub Issues, ADRs, pull requests, CI,
independent review, and releases. The core documentation tree retains only
architecture, decisions, contract usage, and operator or developer usage.
Roadmaps, handoffs, delivery diaries, and narrative history belong in GitHub or
the Wiki.

## Release Ordering

1. Core contracts remain canonical and publish an explicit compatibility
   identity.
2. SDK, Console, and sample releases verify against that identity.
3. NeKiro-Stack pins immutable compatible component releases and executes the
   product acceptance.
4. Core source mirrors are removed only after the target owner is canonical and
   independently verified.
5. No released state contains two writable production authorities.

## Compatibility

This decision changes repository ownership, CI placement, and the published Go
source identity. The core module moves from
`github.com/Nene7ko/NeKiro` to `github.com/NeKiro-project/NeKiro`; Go consumers
must update their `go.mod` and imports. This is an intentional source-level
breaking change. No compatibility shim or legacy module is provided because the
core had no SemVer tag or Release to preserve and a second module identity would
be an unsupported fallback authority.

Wire APIs, language-neutral contracts, schema semantics, authentication,
authorization, routing, streaming, cancellation, errors, and Ledger lineage do
not change. Historical migrations stay available for every supported upgrade
path.

## Consequences

- The core repository can build and test without Console, SDK, sample, or
  full-stack source.
- Satellites have independent releases and focused CI.
- Product acceptance becomes an explicit cross-repository release gate.
- Every Core main merge produces visible SDK, sample, backend, and browser
  compatibility evidence for the exact merged commit.
- Cross-repository changes require upstream-before-downstream release ordering.
- Git history and GitHub records replace tracked Spec Kit delivery directories.

## Rejected Alternatives

### Extract SQL to a database repository

Rejected because migrations are part of each service's owned state contract
and must release with that service.

### Split Control Plane domains into separate repositories

Rejected because logical ownership does not require premature repository or
deployment fragmentation.

### Retain copied source as a fallback

Rejected because dual authorities drift and hide dependency or compatibility
failures.

## Fallback Delta

Fallback delta: removed 3, retained 0, added 0, net -3.

The removed paths were Runtime A's local core-module `replace` and the Runtime
A/Runtime B monorepo-source Docker builds. The satellite repositories now use
published module identities and source-owned image builds.

Added fallback evidence: none.
