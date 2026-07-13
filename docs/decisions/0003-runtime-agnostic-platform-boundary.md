# ADR 0003: Runtime-Agnostic Platform Boundary

- Status: Accepted
- Date: 2026-07-13
- Decision owners: Platform Architecture, Control Plane, A2A Router, Agent SDK

## Context

Go Agent frameworks such as `trpc-agent-go` already provide production Agent
execution capabilities including model adapters, tools, graph workflows,
memory, knowledge retrieval, sessions, evaluation, telemetry, and A2A
client/server integration. NeKiro also uses Agent Card and A2A concepts, which
creates visible overlap at the protocol boundary.

Without an explicit product boundary, NeKiro could expand into Agent-internal
runtime features, duplicate mature frameworks, and couple its Control Plane or
Router to one implementation stack. That would weaken the reason for a
platform-level Registry, Workspace authorization model, managed routing path,
and cross-Agent Ledger.

## Decision

NeKiro is a runtime-agnostic Agent Operating Platform, not a general Agent
Runtime framework.

- NeKiro core owns registration, versioned publication, discovery, Workspace
  installation and permission acceptance, exact-version resolution, managed
  routing, and append-only invocation lineage.
- External Agent Runtimes own model calls, prompts, tools, planning, workflow
  execution, memory, RAG, sessions, evaluation, and runtime-internal telemetry.
- Control Plane and A2A Router MUST NOT depend on a full Agent Runtime
  framework. They may depend on protocol-focused libraries selected by ADR.
- Runtime-specific behavior MUST live in isolated adapters or sample Agents and
  MUST NOT redefine language-neutral platform contracts.
- The Agent SDK remains thin: Card conformance, platform context propagation,
  and nested invocation through the Router.
- Phase 1 acceptance uses at least two sample Agents backed by different
  Runtime implementations and proves a Router-mediated nested call with one
  correlated Ledger lineage.

`trpc-agent-go` is a candidate reference Runtime for a Go sample and adapter.
If adopted, it is not the implementation foundation for NeKiro Control Plane
or Data Plane. Its framework-specific A2A extensions may be supported only
through explicit, versioned optional profile or adapter decisions.

## Consequences

- NeKiro can support Go, Python, and other Agent implementations through the
  same public contracts.
- Agent Runtime innovation remains outside the platform core and can evolve
  independently.
- Platform observability records black-box lifecycle and lineage facts; it may
  correlate with Runtime telemetry but does not require internal reasoning or
  tool traces in Ledger.
- Sample Agents and conformance tests must demonstrate runtime independence,
  adding some integration work to Phase 1.
- Features that only benefit one Runtime are rejected from core or implemented
  in that Runtime's adapter.
- Existing Spec 001 contract work remains valid. It defines the common A2A and
  invocation boundary required by this decision and needs no compatibility
  migration.

## Rejected Alternatives

### Build A NeKiro Agent Runtime

Rejected because it duplicates mature model, tool, workflow, memory, and
session frameworks without strengthening the platform trust boundary.

### Adopt `trpc-agent-go` As The Platform Core

Rejected because it would couple platform contracts and service ownership to
one Runtime and make cross-runtime support secondary.

### Let Agents Call Each Other Directly By URL

Rejected for managed calls because it bypasses Workspace authorization,
exact-version resolution, policy hooks, correlation, and Ledger recording.
