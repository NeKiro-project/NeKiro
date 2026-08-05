# Architecture and ownership

NeKiro is a runtime-agnostic Agent operating platform. Its durable
responsibility is the organization-level trust boundary between callers and
independently operated Agents.

## Product boundary

Core manages the external lifecycle of an Agent:

1. Register an immutable, versioned Agent Card.
2. Discover eligible published capabilities.
3. Install an exact version into a Workspace with explicit permissions.
4. Invoke through the Gateway and A2A Router.
5. Record metadata-only lifecycle and cross-Agent lineage facts.

The Agent Runtime remains outside this boundary. It may implement reasoning,
model access, tools, workflows, memory, RAG, sessions, and runtime telemetry in
any supported language or framework.

## Deployment units

```text
Console -> Control Plane -> A2A Router -> Agent
                     \          |
                      \------ Ledger
```

| Unit | Responsibility | Ownership constraint |
| --- | --- | --- |
| Gateway | Northbound HTTP, caller and Workspace context, response shape | Does not persist Agent Cards or execute A2A transport |
| Registry | Agent Card versions and publication state | Does not run Agents or execute Invocations |
| Discovery | Derived capability query | Is never a second source of truth |
| Workspace | Installations and accepted permissions | Does not deploy Agents |
| Invocation Dispatch | Invocation identity and pre-dispatch authorization | Does not become the A2A protocol executor |
| A2A Router | Transport, context propagation, timeout/cancel, transient result forwarding, events | Does not own permanent Cards or query Registry/Workspace storage directly |
| Ledger | Append-only invocation events and query projection | Does not make routing or authorization decisions |

## Trust boundaries

- Console and external applications call the Gateway only.
- The Gateway never calls an Agent directly; managed calls go through the A2A Router.
- The Router resolves exact Card and Release facts through a controlled Control Plane API.
- Nested Agent-to-Agent calls return to the Router and preserve `root_task_id`, `parent_invocation_id`, and `trace_id`.
- Ledger entries contain metadata and lineage, never Agent payloads, credentials, or keys.
- Cross-process data uses versioned artifacts under `contracts/`, not internal implementation types.

## Canonical source

- [Phase 1 Architecture Specification](../source-docs/architecture/phase-1-spec.md)
- [NeKiro Platform Direction](../source-docs/architecture/platform-direction.md)
- [Core repository boundary ADR](../source-docs/decisions/0009-core-repository-boundary.md)
- [Repository map](../repositories.md)
