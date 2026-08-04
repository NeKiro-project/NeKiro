<!--
Sync Impact Report
- Version change: 1.1.0 -> 2.0.0
- Modified principles: Delivery Is Spec-Driven and Independently Reviewed ->
  Repository-Owned Delivery and Independent Review; contract, ownership, and
  runtime principles updated for cross-repository releases.
- Added sections: Repository Topology; Final Spec Kit Transition.
- Removed sections: mandatory permanent Spec Kit workflow.
- Templates: plan-template.md, spec-template.md, and tasks-template.md marked as
  final-transition artifacts for Spec 030; all are removed from the tracked
  core tree when Spec 030 completes.
- Runtime guidance: AGENTS.md amended; README.md and retained docs are updated
  during Spec 030 implementation.
- Deferred items: none.
-->

# NeKiro Agent Operating Platform Constitution

## Core Principles

### I. Core Loop First

NeKiro MUST preserve the demonstrable loop
`Register -> Discover -> Install -> Invoke -> Record`. Platform work outside
that loop MUST cite an architecture decision or an observed operational need.
Repository extraction MUST NOT weaken the complete product acceptance.

### II. Runtime-Agnostic Platform, Not an Agent Framework

NeKiro MUST manage independently implemented Agents through versioned platform
contracts and supported interoperability protocols. Core services MUST NOT
become an LLM, prompt, tool, workflow, memory, RAG, session, or Agent execution
framework. Framework-specific behavior belongs in an external Runtime or
adapter.

### III. Agent Card and Versioned Contracts Are the Common Language

Agent Card, JSON Schema, OpenAPI, events, credentials, results, and the A2A
Profile are canonical cross-boundary facts. Breaking changes MUST increment the
relevant contract version and document migration impact. Generated or
language-specific mappings are consumers, never competing authorities.

### IV. Control Plane and Data Plane Own Their Boundaries

Frontend traffic MUST enter through Gateway. Managed Agent calls MUST traverse
the A2A Router. Registry, Workspace, Dispatch, Router, and Ledger MUST write
only data they own. Repository separation MUST preserve these boundaries and
MUST NOT create reverse dependencies from core services to satellites.

### V. Persistent State Evolves With Its Owner

Catalog, Workspace, and Ledger schema migrations MUST be released with their
owning core service. Each schema version MUST have one canonical migration
source. Serving processes MUST continue to fail explicit readiness when the
expected schema is absent or incompatible, and MUST NOT auto-migrate or use a
separate database repository as fallback authority.

### VI. Every Managed Invocation Traverses the Router

User-to-Agent and Agent-to-Agent calls managed by the platform MUST preserve
Invocation, Task, parent, root, and Trace lineage through the Router. Ledger
facts remain append-only and distinguish success, failure, timeout,
cancellation, routing failure, and protocol failure.

### VII. Failures Are Explicit and Secrets Stay Out

The fallback addition budget is zero unless an existing contract, ADR,
Runbook, SLO, or caller policy proves otherwise. Missing or incompatible
satellite releases MUST fail product assembly; no copied source, local
replacement, older version, floating branch, or alternate component may mask
the failure. Cards, logs, errors, events, repositories, and CI output MUST NOT
contain credentials or signing material.

### VIII. Repository-Owned Delivery and Independent Review

Each change MUST have one owning repository. Public behavior, contract,
persistent-state, or repository-boundary changes MUST start from a GitHub Issue
and an ADR when architectural ownership or compatibility changes. Implementation
MUST be followed by mapped tests and review from a reviewer that did not author
the change. Pull requests, checks, ADRs, and releases are the durable delivery
record; passing tests do not replace independent review.

## Repository Topology

- `NeKiro-project/NeKiro` owns Control Plane, A2A Router, canonical contracts,
  service-owned migrations, core verification, architecture decisions, and
  core usage documentation.
- `NeKiro-project/NeKiro-Console` owns production Console source and browser
  behavior.
- `NeKiro-project/nekiro-sdk-go` owns public Go Agent and application SDKs.
- `NeKiro-project/NeKiro-Samples` owns cross-runtime sample Agents and their
  sample-specific fixtures.
- `NeKiro-project/NeKiro-Stack` owns multi-component development assembly,
  release manifests, and full product acceptance.
- `NeKiro-project/nekiro-a2a-transport-go` owns reusable A2A wire transport;
  NeKiro core retains platform policy and contract adaptation.

Cross-repository production dependencies MUST use immutable reviewed versions.
No repository may maintain a writable copy of another repository's production
source.

## Product Boundary

NeKiro core owns registration, trusted publication, discovery, Workspace
installation and permission acceptance, exact resolution, managed invocation,
and Ledger lineage. External Agent Runtimes own model calls, prompts, tools,
planning, workflows, memory, retrieval, sessions, and response generation.
Product acceptance MUST continue to prove two independently implemented
Runtimes and one correlated nested invocation lineage.

## Platform Constraints

- Control Plane and A2A Router remain Go services.
- PostgreSQL schemas remain explicitly owned by core modules.
- Cross-boundary facts remain language-neutral contracts.
- The A2A Router remains a separately deployed data-plane process.
- Core services MUST NOT depend on Console, SDK, sample, Stack, or full Agent
  Runtime source.
- Speculative queues, search clusters, schedulers, deployment runtimes,
  billing, rating, and federation remain out of scope without explicit need.

## Delivery Workflow

1. Record the requested outcome and scope in a GitHub Issue.
2. Add or amend an ADR before changing architecture, repository ownership,
   public contracts, persistent-state semantics, or compatibility policy.
3. Implement only the approved scope in the owning repository.
4. Add tests mapped to success, failure, cancellation, compatibility, and
   secrecy requirements after implementation.
5. Run repository-local CI and, where relevant, the immutable cross-repository
   product acceptance in NeKiro-Stack.
6. Obtain independent review and resolve blocking findings before merge.
7. Publish an immutable release and update downstream pins explicitly.

## Final Spec Kit Transition

Spec 030 is the final tracked Spec Kit feature and supplies the migration plan
for this governance change. After its implementation and review, the core
repository MUST stop tracking `specs/`, `.specify/`, and repository-local Spec
Kit skills. Historical delivery evidence remains available through Git history,
Issues, pull requests, releases, and the Wiki. This constitution is removed
with `.specify/`; `AGENTS.md` and ADR 0009 become the permanent authority.

## Governance

`AGENTS.md` is the repository charter. Architecture amendments MUST state
reason, impact, compatibility, migration, repository ownership, and release
ordering. This transitional constitution uses semantic versioning: MAJOR for
incompatible governance changes, MINOR for new principles, and PATCH for
clarifications. The final transition is complete only when all repository and
product acceptance gates pass and an independent reviewer approves it.

**Version**: 2.0.0 | **Ratified**: 2026-07-13 | **Last Amended**: 2026-08-04
