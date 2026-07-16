# Research: A2A Router Foundation

## Process Boundary

**Decision**: Add `apps/a2a-router` as a standalone Go process with its own config, auth, handlers, and Dockerfile.
**Rationale**: The constitution requires Router to be an independent Data Plane process and not a Control Plane package.
**Rejected**: Reusing Control Plane server assembly or importing Control Plane internals; both break ownership.

## Resolution Direction

**Decision**: Resolve exact Agent facts only through Control Plane Internal v2 `/internal/v2/resolve-agent`.
**Rationale**: Catalog and Workspace facts remain Control Plane-owned; Router must not read their tables or keep permanent Card copies.
**Rejected**: Direct database reads, Catalog store imports, cached Card snapshots, or endpoint probes.

## Readiness

**Decision**: Readiness proves local config and handler assembly only.
**Rationale**: Dependency probing would conflate readiness with Control Plane/Agent/Ledger health and risks hidden fallback behavior.
**Rejected**: Startup calls to Control Plane, database migrations, Agent health checks, or Ledger checks.

## Post-Resolution Placeholder

**Decision**: Return correlated `ROUTE_NOT_FOUND` after successful resolution until T006 owns Agent transport.
**Rationale**: This truthfully states routing is not implemented without fabricating success or side effects.
**Rejected**: Mock Agent result, direct endpoint call, silent 204, or fake Ledger fact.
