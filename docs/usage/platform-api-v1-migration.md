# Platform API v1 migration

NeKiro `v0.1.0` uses one URL version at each owned HTTP boundary. This is a
pre-release cutover with no compatibility aliases.

## Route mapping

| Pre-release route family | v0.1.0 route family |
| --- | --- |
| `/v3/agents...` | `/v1/agents...` |
| `/v3/workspaces...` | `/v1/workspaces...` |
| `/v4/providers...` | `/v1/providers...` |
| `/v4/releases...` | `/v1/releases...` |
| `/v4/public/agents...` | `/v1/public/agents...` |
| `/v4/workspaces/{workspaceId}/invocations...` | `/v1/workspaces/{workspaceId}/invocations...` |
| `/v4/workspaces/{workspaceId}/traces...` | `/v1/workspaces/{workspaceId}/traces...` |
| `/internal/v2/resolve-agent` | `/internal/v1/resolve-agent` |
| `/internal/v3/resolve-installed-version` | `/internal/v1/resolve-installed-version` |
| `/internal/v4/invocations` | `/internal/v1/invocations` |
| `/internal/v3/workspaces/...` | `/internal/v1/workspaces/...` |
| `/agent/v1/invocations` | unchanged |

Resource representations, status codes, media negotiation, error phase
semantics, exact Release provenance, and metadata-only Ledger rules are the
latest pre-release behavior; only the owning HTTP API identity is reset.

## Required configuration changes

- Set `NEKIRO_ROUTER_INTERNAL_URL` to the exact
  `/internal/v1/invocations` URL.
- Set `NEKIRO_CONTROL_PLANE_RESOLVE_URL` to the exact
  `/internal/v1/resolve-agent` URL.
- Set `NEKIRO_CONTROL_PLANE_VERSION_URL` to the exact
  `/internal/v1/resolve-installed-version` URL.
- Update Gateway clients to construct only `/v1` public routes.
- Keep Agent SDK nested calls on `/agent/v1/invocations`.

Configuration containing a retired path fails validation. Clients must not
probe an old path after a failure or reinterpret `404` as an empty result.

## Payload versions

URL API v1 does not rename independently versioned payloads. Consumers must
continue to honor the active Agent Card, Installation, Platform Error,
Invocation Event, result stream, A2A Profile, and Router credential schema
identities declared by the v1 OpenAPI documents.

## Verification

After migration:

1. Run Core contract, unit, PostgreSQL integration, race, and vet checks.
2. Run Console and Go SDK client tests and confirm no active `/v2`, `/v3`, or
   `/v4` request remains.
3. Run Stack backend and browser acceptance against exact component commits.
4. Confirm retired public and internal paths return `404` without invoking an
   owner service.
5. Confirm one root and child Invocation preserve `root_task_id`,
   `parent_invocation_id`, and `trace_id` in Ledger.

There is no downgrade or mixed-version operating mode.
