# Contract Guide: Workspace Installation Inspection

Issue #7 consumes the active language-neutral Northbound v3 contract. The
source of truth remains:

- `contracts/openapi/control-plane.v3.yaml`
- `contracts/schemas/installation.v2.schema.json`
- `contracts/schemas/platform-error.v3.schema.json`
- `contracts/contracts.go`

## Read One Installation

`GET /v3/workspaces/{workspaceId}/installations/{installationId}`

Success: `200 OK` with one complete Installation v2 object. The response
includes the submitted `versionConstraint`, exact `installedVersion`, sorted
`acceptedPermissions`, `status`, `installedAt`, `updatedAt`, and
`uninstalledAt` only for a terminal row.

Failures:

| HTTP | Code | Meaning |
| --- | --- | --- |
| 400 | `VALIDATION_ERROR` | Workspace or Installation identifier is invalid |
| 401 | `UNAUTHENTICATED` | No trusted Gateway bearer identity |
| 403 | `FORBIDDEN` | Caller does not own the Workspace |
| 404 | `NOT_FOUND` | Workspace or Installation under it is absent |
| 503 | `DEPENDENCY_ERROR` | Workspace persistence read failed |

## List Installations

`GET /v3/workspaces/{workspaceId}/installations?limit=25&cursor=<opaque>`

`limit` is required and must be 1-100. `cursor` is optional, opaque, bound to
the Workspace and requested limit, and represents the last
`installedAt`/`installationId` tuple. Results contain current and historical
rows in ascending tuple order.

Success: `200 OK` with:

```json
{"items":[{"installationId":"installation-a","workspaceId":"workspace-a","agentId":"runtime-a","versionConstraint":"^1.0.0","installedVersion":"1.0.0","acceptedPermissions":[],"status":"uninstalled","installedAt":"2026-07-15T10:00:00Z","updatedAt":"2026-07-15T10:02:00Z","uninstalledAt":"2026-07-15T10:02:00Z"}]}
```

An empty history is `{"items":[]}` without `nextCursor`. A cursor is present
only when the response has another page.

Failures use the same `400/401/403/404/503` Workspace v3 responses. Malformed
or mismatched cursors are `400 VALIDATION_ERROR`; dependency failures are
`503 DEPENDENCY_ERROR` and never `200 items: []`.

Every response sets `x-nek-trace-id`. Error bodies use fixed Platform Error v3
messages and contain no owner, Installation, Catalog, endpoint, token, or
credential detail.

## Compatibility

No schema, route, method, status, field, or error-code change is introduced by
Issue #7. The implementation adds evidence for existing operations only.
