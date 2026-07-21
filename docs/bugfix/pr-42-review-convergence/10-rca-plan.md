# Root cause and repair plan

## Root causes

1. The first nested trust model treated Agent identity as sufficient and did
   not make Workspace part of the credential principal.
2. Control Plane Internal v3 resolution validated Platform Error shape but not
   its operation-specific status/code and correlation phase.
3. The SDK convenience constructor invented a response limit and represented
   SSE as a bounded aggregate body instead of a validated stream.
4. The shared Go dispatch DTO was used for both root wire input and trusted
   child lineage without excluding the private field from JSON.
5. Tests and SDD artifacts described an earlier design and did not assert full
   stream lifecycle or active v3 inventory.

## Repair boundary

Bind credentials to `(workspaceId, agentId)`, check both fields against the
committed parent, enforce the v3 error table, require explicit SDK response and
event limits, expose `InvokeStream`/`Recv`, validate v2 result-stream events,
retain only safe Router error fields, mark parent lineage as in-process-only,
and converge tests/docs/ADR/active-contract inventory.

## Risk controls

No new retry, alternate destination, default limit, raw-detail propagation, or
historical compatibility fallback is introduced. Existing v2 exact Card
resolution remains separate from the additive v3 installed-version operation.
