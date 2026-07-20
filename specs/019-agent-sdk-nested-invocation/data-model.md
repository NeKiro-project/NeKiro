# Data Model: Agent SDK Nested Invocation

| Entity | Owned by | Fields / invariants |
|---|---|---|
| PlatformContext | SDK | `invocationId`, `rootTaskId`, `traceId`, `workspaceId`, `agentId`; all required safe identifiers, no inferred values |
| AgentBinding | Router auth adapter | exact Agent ID plus opaque token digest; duplicate IDs/digests rejected |
| NestedInvocationRequestV1 | contracts | `parentInvocationId`, `targetAgentId`, `capability`, object `input`, `stream`; no trusted extras |
| Child Dispatch Request | Router nested adapter | generated child ID; parent-derived Workspace/root Task/Trace; caller `{type:agent,id}`; exact target/capability/input/mode |
| Child Invocation | Router Ledger | existing Invocation Event 0.3 lifecycle; child `parentInvocationId` points to parent and content remains excluded |

The parent projection is read-only input to derivation. The adapter never
updates Workspace, Catalog, or Ledger tables directly.
