# Research: Workspace Installation Inspection

## Observation

The Issue #7 worktree starts from the Spec 005 install/pin implementation. The
active runtime already contains:

- `workspace.Service.GetInstallation` and `ListInstallations` with owner
  authorization, cursor decoding, and explicit page-size validation.
- A PostgreSQL `GetInstallation` point query and a bounded keyset list query
  ordered by `installed_at ASC, installation_id ASC`.
- The Northbound v3 GET routes, strict query parser, trace header, and v3
  Workspace error mapping.
- Workspace-owned tables retaining uninstalled rows and an order index.

The baseline lacks an independent Issue #7 specification, focused inspection
tests, restart/pagination/authorization/dependency evidence, and a contract
mapping test dedicated to the read/list operations. Existing tests cover only a
small list continuation and lifecycle workflow.

## Decisions

### Reuse the active contract

The feature consumes `GET /v3/workspaces/{workspaceId}/installations` and
`GET /v3/workspaces/{workspaceId}/installations/{installationId}` from
Northbound API v3. It does not change OpenAPI, Installation v2, or Platform
Error v3 schemas.

### Keep owner authorization before fact lookup

The service first reads the requested Workspace and applies the owner policy.
Only then does it read one Installation or list rows. The PostgreSQL query is
scoped by `workspace_id`, so a valid Installation ID from another Workspace is
not observable.

### Keep keyset pagination

The current ordering tuple is encoded in an opaque strict base64url cursor.
Continuation uses `installed_at > after` or equal timestamp plus
`installation_id > afterID`. PostgreSQL reads `limit + 1` rows to determine
`hasMore`; the service emits a cursor only when there is another page.

### Treat empty and dependency outcomes distinctly

PostgreSQL allocates a non-nil result slice before scanning. Query, scan, and
rows iteration failures return `ErrDependency` and no list payload. Tests must
prove that an empty result is only produced after Workspace lookup,
authorization, and a successful list query.

## Fallback Inventory

| Candidate behavior | Classification | Evidence |
| --- | --- | --- |
| Explicit `items: []` for a successful no-row query | Keep | Active OpenAPI InstallationList contract and Spec 003 acceptance |
| Required explicit list limit | Keep | Active Northbound v3 contract; omission is validation failure |
| Default page size | Remove | Spec 003/active v3 require explicit `limit` |
| Filtering uninstalled rows | Remove | Spec 003 history semantics require current and historical rows |
| Restarting traversal after malformed/mismatched cursor | Remove | Active v3 says cursor mismatch is validation failure |
| Empty result on a failed Workspace/list dependency | Remove | Constitution failure-safety and Issue #7 acceptance |
| Catalog query, cache, endpoint probe, retry, or alternate source | Remove | Workspace owns Installation facts; no fallback policy |

Fallback delta for this feature: removed `0`, retained `2`, added `0`, net `0`.
The retained behaviors are explicit contract semantics, not failure fallbacks.

## Verification Gaps To Close

1. Add service unit coverage for complete facts, owner-first authorization,
   cross-Workspace not-found, invalid cursor, deterministic unchanged-data
   traversal, empty results, and dependency propagation.
2. Add PostgreSQL integration coverage for current/history rows, equal-time
   ordering, bounded continuation, empty history, process restart, and query
   failures.
3. Add HTTP coverage for both GET operations, all active v3 failures, trace
   equality, strict query validation, and response secret exclusion.
4. Add contract coverage proving the active OpenAPI operations and response
   schemas accept complete and empty list values.
