# Research Notes: Agent SDK Nested Invocation

## Existing evidence

1. ADR 0006 freezes `router-agent.v1.yaml` as the only SDK destination and
   requires one opaque credential bound to one exact `(Workspace, Agent)` pair.
2. `contracts.NestedInvocationRequestV1` already excludes trusted caller,
   Workspace, root Task, Trace, endpoint, credential, and child identity fields.
3. `apps/a2a-router/internal/transport/a2a` already propagates platform context
   headers to target Agents and supports the active JSON/SSE result modes.
4. `DispatchHandler` already owns exact resolution, A2A transport, deadlines,
   and append-only Ledger semantics; nested handling must delegate rather than
   duplicate those rules.
5. `NestedLedgerReader.GetInvocationByParentID` and the v4 projection validator
   provide the trusted parent lookup and metadata-only boundary.

## Rejected approaches

- Reusing the Control Plane service endpoint for Agent SDK calls would permit
  caller-class confusion and would not bind the authenticated Agent identity.
- Accepting context/endpoint/caller fields from the SDK request would allow
  lineage and Workspace forgery.
- Calling the target Agent directly from the SDK would bypass Router policy,
  resolution, and Ledger facts.
- Adding a retry, redirect-following client, fallback token, or alternate route
  would contradict the accepted no-fallback policy.

## Open boundary

Deployment wiring for the agent-principal map and the second Runtime sample are
left to the parent acceptance slice; this child tests the handler with explicit
constructor dependencies.
