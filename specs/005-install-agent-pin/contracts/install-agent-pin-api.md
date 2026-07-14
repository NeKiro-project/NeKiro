# Install Agent and Exact Pin Contract Guide

Issue #5 consumes the active Northbound v3 contract from issue #3. The
language-neutral OpenAPI and Installation v2 schema remain the source of truth.

## Install Agent

`POST /v3/workspaces/{workspaceId}/installations`

Request:

```json
{
  "agentId": "runtime-a",
  "versionConstraint": ">=1.0.0 <2.0.0",
  "acceptedPermissions": ["document.read"]
}
```

`acceptedPermissions` is required. `[]` is valid; missing and `null` are not.
IDs are exact and case-sensitive.

Success: `201 Created`

```json
{
  "installationId": "installation-alpha",
  "workspaceId": "workspace-alpha",
  "agentId": "runtime-a",
  "versionConstraint": ">=1.0.0 <2.0.0",
  "installedVersion": "1.4.3",
  "acceptedPermissions": ["document.read"],
  "status": "enabled",
  "installedAt": "2026-07-15T10:01:00.000000Z",
  "updatedAt": "2026-07-15T10:01:00.000000Z"
}
```

Failures:

| Status | Code | Meaning |
| --- | --- | --- |
| 400 | `VALIDATION_ERROR` | Invalid ID/range/array, missing/null/duplicate/unknown permission |
| 401 | `UNAUTHENTICATED` | No trusted Gateway identity |
| 403 | `FORBIDDEN` | Caller is not Workspace owner |
| 404 | `NOT_FOUND` | Workspace or matching published Agent version absent |
| 409 | `CONFLICT` | Current Installation already exists, including a race loser |
| 503 | `DEPENDENCY_ERROR` | Catalog or Workspace persistence failed |

Every result includes `x-nek-trace-id`. Error bodies use fixed Platform Error
v3 messages and contain no Card, endpoint, credentials, or dependency detail.

## Selection and Persistence Rules

- Catalog selects the highest currently published eligible version using the
  active SemVer/pre-release/build tie policy.
- Workspace validates permissions against that exact Card.
- The exact selected version and canonical permission snapshot are persisted in
  one enabled Installation and are never auto-upgraded.
- Workspace code calls the Catalog reader port; it does not query Catalog SQL.
