# NeKiro Platform Direction

- Status: Active
- Updated: 2026-07-13
- Decision: [ADR 0003](../decisions/0003-runtime-agnostic-platform-boundary.md)

## Product Thesis

NeKiro is a runtime-agnostic platform for registering, authorizing, invoking,
and auditing independently built Agents. It gives an organization one trusted
control and data path for Agents without requiring those Agents to share a
language, model provider, internal framework, or deployment model.

The platform's first value proposition is:

> Register any supported A2A Agent, grant a Workspace access to an exact
> version, invoke it through one governed endpoint, and inspect the complete
> cross-Agent call lineage.

NeKiro operates Agents from the outside. It does not implement how an Agent
reasons or executes internally.

## Target Users And Jobs

| User | Job NeKiro must make reliable |
| --- | --- |
| Agent developer | Publish a versioned capability without coupling the Agent to platform internals |
| Platform operator | Control which Agent versions are available and diagnose managed calls across teams |
| Workspace owner | Review declared permissions, install an exact Agent version, and enable or disable access |
| Application developer | Invoke installed Agents through one stable API without learning each Agent endpoint or framework |

The first product is for teams operating more than one independently owned
Agent. A single application that only needs an LLM, tools, memory, or a local
multi-Agent workflow is better served by an Agent Runtime framework.

## Platform Differentiation

NeKiro's durable responsibility is the trust boundary between callers and
independently operated Agents:

1. Registry owns immutable, versioned Agent Card publication facts.
2. Discovery provides a derived capability query without becoming a second
   source of truth.
3. Workspace Installation records accepted permissions and the exact resolved
   Agent version.
4. A2A Router mediates every managed user-to-Agent and Agent-to-Agent call.
5. Invocation Ledger records metadata-only, append-only lifecycle and lineage
   facts across runtime boundaries.
6. Language-neutral contracts keep Console, platform services, SDKs, and Agent
   implementations interoperable.

These capabilities remain valuable when an Agent changes its model, tools,
workflow engine, implementation language, or Runtime framework.

## Relationship To Agent Runtime Frameworks

Frameworks such as `trpc-agent-go` build and run Agent internals. They may own
LLM integration, tool execution, graph workflows, memory, RAG, sessions,
evaluation, and runtime telemetry. NeKiro does not duplicate those features.

The intended relationship is complementary:

```text
Agent Runtime (for example trpc-agent-go, ADK, or a custom server)
  -> exposes an A2A endpoint and Agent Card
  -> integrates through a thin NeKiro adapter when needed
  -> is registered and versioned by NeKiro Registry
  -> is authorized through Workspace Installation
  -> is invoked through NeKiro A2A Router
  -> contributes platform-level facts to Invocation Ledger
```

`trpc-agent-go` is a strong candidate for the first Go reference integration.
If adopted, it MUST remain one supported Runtime rather than a dependency that
defines NeKiro's platform semantics. Framework-specific extensions belong in
an adapter or an explicitly versioned optional profile.

## Runtime Integration Boundary

The core platform may require an Agent to:

- expose the supported A2A methods and a reachable endpoint;
- publish metadata that conforms to the active Agent Card contract;
- accept platform correlation context without trusting caller-supplied
  Workspace or authorization facts;
- route managed nested calls back through the A2A Router;
- avoid returning credentials in Cards, errors, events, or logs.

The optional Agent SDK may help with Card conformance, context propagation, and
nested Router calls. It MUST NOT provide model abstractions, prompt management,
tool execution, planning, workflow graphs, memory, RAG, or generic session
runtime behavior.

## Phase 1 Product Proof

Phase 1 is proven by one cross-runtime scenario, not by the number of schemas
or screens:

1. Run two sample Agents implemented with different Runtime stacks. One may use
   `trpc-agent-go`; the other must not depend on the same Agent Runtime.
2. Register and publish both versioned Agent Cards.
3. Discover and install both Agents into one Workspace with explicit
   permission acceptance.
4. Invoke Agent A through the Gateway and Router.
5. Have Agent A invoke Agent B through the Router rather than its endpoint.
6. Return the result through the original Gateway request.
7. Query one Ledger lineage containing the root and child Invocations with
   distinguishable success or failure semantics.

Passing this scenario proves that NeKiro adds value above either Runtime. A
same-framework-only demo does not prove runtime independence.

## Scope Filters

Every proposed platform feature must answer:

1. Which Agent developer, platform operator, Workspace owner, or application
   developer job does it improve?
2. Does it remain useful when the Agent Runtime is replaced?
3. Does it belong to Control Plane, Data Plane, an adapter, or the Agent
   Runtime itself?
4. Which module owns the behavior and data?
5. Which versioned contract and failure semantics cross the boundary?
6. How does the cross-runtime acceptance scenario verify it?

If a feature fails the runtime-replacement test, it is not NeKiro core unless
an ADR establishes a cross-runtime platform requirement.

## Explicit Non-Goals

NeKiro core does not provide:

- LLM provider or prompt abstractions;
- tool or MCP execution runtimes;
- generic planners, workflow graphs, or local multi-Agent orchestration;
- Agent memory, RAG, knowledge stores, or session execution;
- Agent evaluation, prompt optimization, or self-evolution;
- automatic Agent deployment or a Kubernetes Runtime;
- Marketplace ranking, billing, or revenue sharing before the core loop has
  real usage.

## Directional Roadmap

### Stage 1: Trusted Invocation Loop

Complete Register, Discover, Install, Invoke, and Record with cross-runtime
sample Agents and clear failure semantics.

### Stage 2: Operational Governance

Add policy, identity, quota, approval, health, and operational capabilities
only when usage evidence and a dedicated Spec define ownership and behavior.

### Stage 3: Ecosystem

Build certification, Marketplace, billing, and federation only on top of proven
Registry, Installation, Router, and Ledger behavior.

The roadmap advances because of observed operating pressure, not because a
neighboring Agent framework exposes another runtime feature.
