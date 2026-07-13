# Contract Design: Directional Internal APIs

## Control Plane Internal API v1

**File**: `contracts/openapi/control-plane-internal.v1.yaml`

**Served by**: Control Plane  
**Called by**: A2A Router

Contains only:

- `POST /internal/v1/resolve-agent`

The response contains the exact authorized Agent Card version and Installation
facts. Errors distinguish forbidden, disabled, not found, and dependency
failure. The Router never reads Registry or Workspace tables directly.

## Router Internal API v2

**File**: `contracts/openapi/router-internal.v2.yaml`

**Served by**: A2A Router  
**Called by**: Control Plane

Contains:

- `POST /internal/v2/invocations` for direct JSON/SSE result delivery.
- `GET /internal/v2/invocations/{invocationId}` for Router-owned Ledger facts.
- `GET /internal/v2/invocations/{invocationId}/events` for metadata-only Ledger
  event streaming when an internal consumer needs it.
- `GET /internal/v2/traces/{traceId}` for Router-owned lineage facts.

## Ownership Rules

- Each document has exactly one service owner and one server destination.
- Generated clients must configure the destination explicitly; production
  endpoints have no localhost fallback.
- Resolve data flows Router → Control Plane. Dispatch/result and Ledger queries
  flow Control Plane → Router.
- DTOs come from contracts; neither service imports the other's internal Go
  packages.

## Historical Contract

`contracts/openapi/router-internal.v1.yaml` remains unchanged as migration
evidence. It mixes destinations and returns dispatch acceptance only, so it is
not an active runtime contract.
