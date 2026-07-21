# Fix implementation record

## Changed areas

- `apps/a2a-router/internal/nested`: Workspace-bound principals and parent
  Workspace/Agent authorization.
- `apps/a2a-router/internal/resolution`: strict Control Plane v3
  status/code/phase/correlation validation.
- `sdks/agent-sdk`: explicit response/event limits, JSON/SSE split, incremental
  SSE framing and sequence validation, safe `RouterError`.
- `contracts` and Router API: private root parent lineage, active v3 contract
  inventory, and contract tests.
- `specs/019`, README, compatibility guide, phase-1 architecture, ADR 0002 and
  ADR 0006: policy and SDD convergence.

## Fallback audit

Fallback delta: removed 1, retained 0, added 0, net -1.

Added fallback evidence: none.

The removed fallback is the SDK's unsupported 16 MiB `NewClient` default.
Limits are now explicit and validated against `RuntimeByteLimitMaximum`.

## Rollback

Revert the convergence commit/working-tree patch as one review unit. No
database migration or external service state was changed by this fix.
