# Roadmap Draft: After Workspace Installation

- Status: Draft for review
- Created: 2026-07-15
- Basis: project main 6365a89, docs/handoffs/CURRENT.md, AGENTS.md, and GitHub issues #2, #6, #7, #8, #9

This document is a planning note, not an active Spec Kit feature. It should be
used to decide the next issue batch after the Minimal Workspace and Agent
Installation issue group is fully accepted.

## Current Read

The repository has already established the contract and backend trust boundary
needed for the first half of the Phase 1 loop:

    Register -> Discover -> Install

Completed or merged work:

- Contract foundation is complete in specs/001-complete-invocation-contracts.
- Catalog registration, publication, disablement, and discovery are complete in
  specs/002-catalog-registry-discovery.
- Minimal Workspace and Installation contracts are defined in
  specs/003-workspace-installation-contracts.
- Workspace create/read is delivered by #4.
- Install published Agent and pin exact version is delivered by #5.

The active handoff still treats Installation inspection/lifecycle, Invocation
Dispatch, A2A Router, Ledger, SDK, Sample Agents, Frontend, and the complete
E2E loop as future scope. GitHub currently leaves the parent Workspace task and
its remaining slices open:

- #6: exact installed Agent capability resolution
- #7: list and inspect Workspace Installation history
- #8: enable, disable, and uninstall Installation access
- #9: cross-slice Workspace acceptance, restart, concurrency, review, and
  convergence

Before moving to a new product area, close the issue-tracker loop for #6-#9.
The codebase already contains runtime surfaces for inspection, lifecycle, and
resolution, so the first step is review and convergence rather than assuming a
greenfield implementation.

## Immediate Closeout Plan

### 1. Reconcile #6 Exact Resolution

Review the implementation against the issue acceptance criteria:

- request correlation preservation for invocationId, rootTaskId, and traceId
- enabled, non-uninstalled, exact-version Installation requirement
- currently published Catalog Card re-read through the Catalog boundary
- capability and accepted-permission authorization
- fixed error precedence for invalid, not found, not installed, disabled,
  forbidden, capability denied, and dependency failure
- no fallback Card, endpoint probe, cache, retry, or compatibility branch

Expected output: either close #6 with evidence, or open a narrow remediation PR
that only touches the resolution slice.

### 2. Reconcile #7 Installation Inspection

Review exact Installation read and bounded history listing:

- owner-only access
- stable ordering by installedAt and installationId
- opaque cursor validation
- uninstalled facts remain visible
- genuine empty history is distinct from dependency failure
- no Catalog-derived second source of Installation truth

Expected output: either close #7 with evidence, or open a narrow remediation PR
for inspection and pagination only.

### 3. Reconcile #8 Lifecycle

Review lifecycle transitions:

    enabled <-> disabled -> uninstalled

The review should confirm:

- only the Workspace owner can mutate lifecycle state
- same-state and illegal terminal transitions return contract-defined failures
- uninstall is logical and preserves immutable Installation evidence
- Catalog disablement never rewrites Workspace state
- concurrent lifecycle operations serialize to one legal committed history

Expected output: either close #8 with evidence, or open a narrow remediation PR
for lifecycle state handling only.

### 4. Execute #9 Workspace Acceptance

After #6-#8 are accepted, run one combined acceptance pass:

- Register/Publish/Discover through Catalog
- Create Workspace
- Install exact published Agent with accepted permissions
- Inspect exact and list history across restart
- Disable, enable, uninstall, and reinstall after uninstall
- Resolve only enabled, exact, currently published, capability-authorized
  Installation
- Prove distinct failures for disabled, uninstalled, Catalog-disabled, missing
  permission, unknown capability, malformed input, forbidden, conflict, and
  dependency failure
- Run unit, contract, PostgreSQL integration, HTTP, restart, concurrency, vet,
  build, Compose config, and diff checks
- Run independent review and converge findings before marking parent #2 done

Expected output: #9 closes with a single evidence report, then parent #2 can be
closed if all child issues are done.

## Recommended Next Direction

After #2/#6/#7/#8/#9 are closed, the next development direction should be:

    Invoke -> Record

Concretely, the next feature should build the first real Invocation runtime
slice:

    Gateway Invocation endpoint
      -> Invocation Dispatch
      -> A2A Router
      -> Sample Agent
      -> metadata-only Invocation Ledger
      -> Invocation / Trace read APIs

This is the right next step because it consumes the completed Catalog and
Workspace trust boundaries instead of creating another isolated control-plane
island. It also advances the Phase 1 proof more directly than Frontend work,
Marketplace work, SDK expansion, billing, governance, or deployment automation.

Frontend should remain paused until the backend can prove at least one headless
runtime path:

    Register -> Discover -> Install -> Invoke -> Record

## Proposed Spec Sequence

Use fresh Spec Kit directories rather than reopening completed specs. Suggested
names are placeholders and can change when issues are created.

### Spec 006: Invocation Dispatch and Router Runtime Foundation

Goal: create the smallest backend path that can accept an authorized invocation
and hand it to a separate Router process.

Scope:

- Control Plane Gateway implements the active POST
  /v3/workspaces/{workspaceId}/invocations boundary enough to create
  invocation/root task/trace context.
- Invocation Dispatch validates Workspace installation through the existing
  exact-resolution boundary.
- apps/a2a-router is introduced as an independent Go process with health,
  configuration, internal auth, and a Router Internal v2 HTTP boundary.
- Router calls Control Plane Internal v2 /internal/v2/resolve-agent; it never
  reads Catalog or Workspace storage directly.
- Non-streaming result transport works against a deterministic local sample A2A
  server.

Non-goals:

- Console UI
- multi-runtime nested Agent proof
- billing, Marketplace, deployment, K8s, queues, or cache
- result persistence or replay

Definition of done:

- one installed Agent can be invoked through Gateway -> Dispatch -> Router
- resolution and dispatch failures preserve correlation identifiers
- no Agent input/output is written to Ledger or logs
- unit, contract, internal HTTP, integration, restart, and failure-path tests pass
- fallback delta reports added 0

### Spec 007: Metadata-Only Invocation Ledger and Trace Reads

Goal: make Record real with append-only invocation lifecycle facts.

Scope:

- Router-owned Ledger event store and migration
- lifecycle events for accepted, routing, running, succeeded, failed, canceled,
  and timed out
- Northbound GET /v3/invocations/{invocationId} and
  GET /v3/traces/{traceId} through Gateway
- Router Internal v2 metadata read endpoints
- failure classification for timeout, cancellation, route failure, protocol
  failure, Agent failure, authorization failure, and dependency failure

Non-goals:

- Agent result storage
- replay or polling result API
- billing or analytics
- cross-runtime nested call proof

Definition of done:

- every managed invocation creates queryable metadata-only events
- terminal status and error code combinations match the active contracts
- trace and root task lineage are queryable without Agent payload content
- dependency failures never become empty, stale, or successful responses
- restart and database outage paths are covered

### Spec 008: Cross-Runtime Sample Agents and Nested A2A Proof

Goal: prove NeKiro is a runtime-agnostic platform, not one Agent framework's
wrapper.

Scope:

- two sample Agents implemented with different runtime stacks
- both expose active Agent Cards and A2A Profile behavior
- both can be registered, published, discovered, and installed
- Agent A invokes Agent B only through the Router
- child invocation records preserve rootTaskId, parentInvocationId, and traceId

Non-goals:

- generic Agent SDK runtime framework
- model/tool/prompt/memory abstractions
- Marketplace ranking or certification
- production deployment automation

Definition of done:

    Register both Agents
    -> Discover by capability
    -> Install both into one Workspace
    -> Invoke Agent A through Gateway
    -> Agent A invokes Agent B through Router
    -> Query one complete parent/child Ledger lineage

- the two Agents do not share runtime-internal types or storage
- the Agent SDK remains thin: Card consistency, platform context propagation,
  and nested Router call helper only

## Issue Batch Sketch

The next issue numbers will depend on GitHub's shared issue/PR sequence, but
the recommended batch is:

1. [Task] Deliver Invocation Dispatch, Router, and Ledger
2. [Spec] Define Invocation runtime implementation gate
3. [Dispatch] Authorize and dispatch an installed Agent invocation
4. [Router] Execute A2A non-streaming and streaming calls
5. [Ledger] Record and query metadata-only invocation lineage
6. [Acceptance] Prove Invoke -> Record with failure and restart coverage
7. [Sample Agents] Prove cross-runtime nested invocation

Keep the first task backend-only. Console work should be a later batch after the
headless flow is independently proven.

## Open Decisions Before Spec 006

These should be answered during clarify / plan, not guessed in code:

- Which minimal sample Agent implementation is used for the first Router
  foundation slice before the cross-runtime proof?
- Does Spec 006 include streaming immediately, or does it deliver non-streaming
  first and reserve streaming for the Ledger/acceptance batch?
- What exact timeout and cancellation ownership belongs to Dispatch vs Router?
- Which PostgreSQL ownership boundary stores Ledger events: Router schema only,
  or a separate logical ledger schema within the same database?
- What internal auth material format is used between Control Plane and Router
  without introducing development-only weak defaults?

## Guardrails

- Do not reopen completed specs unless a concrete defect requires it.
- Do not start Frontend before the backend headless loop works.
- Do not let Router import Control Plane internals or query Workspace/Catalog
  tables directly.
- Do not persist Agent input, output, chunks, credentials, or endpoint secrets
  in Ledger.
- Do not add fallback Cards, fallback Workspaces, localhost production endpoints,
  retries, caches, alternate data sources, or compatibility branches without
  explicit policy evidence.
- Treat [] only as valid where the contract explicitly says empty is a real
  product state; dependency failure must stay visible.

## Recommended First Action

Start by reviewing #6, #7, and #8 against the current implementation. If they
are already complete, close them with evidence and run #9. If gaps remain,
repair each gap in the smallest issue-owned PR. Only after #9 and parent #2 are
closed should the repo open Spec 006 for Invocation Dispatch and Router runtime.
