# Quickstart: Validate Workspace Installation Inspection

This guide validates Issue #7 only. It does not claim Installation lifecycle,
Catalog resolution, Router, Ledger, SDK, or Console completion.

## Static And Focused Checks

```sh
go test -count=1 ./contracts
go test -count=1 ./apps/control-plane/internal/workspace/...
go test -count=1 ./apps/control-plane/internal/gateway
go vet ./...
go build ./...
go test -race -count=1 ./apps/control-plane/internal/workspace/... ./apps/control-plane/internal/gateway ./contracts
git diff --check
```

## PostgreSQL Integration Prerequisite

Real PostgreSQL tests require `NEKIRO_TEST_DATABASE_URL` and reject any database
whose name does not end in `_test`. Run schema-resetting packages serially:

```sh
go test -tags=integration -count=1 ./apps/control-plane/internal/workspace/postgres
go test -tags=integration -count=1 ./apps/control-plane/internal/workspace/integration
```

When the dedicated `_test` database is unavailable, report the limitation and
do not treat skipped integration tests as passing evidence.

Verification on 2026-07-15 passed the focused and broad commands above. The
integration-tag packages compiled, but the real PostgreSQL tests were not run
because `NEKIRO_TEST_DATABASE_URL` was unavailable in the environment.

## Owner Read/List Workflow

With an authenticated owner, an existing Workspace, and Installation rows:

```text
GET /v3/workspaces/workspace-a/installations?limit=25
GET /v3/workspaces/workspace-a/installations/{installationId}
```

Concatenate `items` from each response while following `nextCursor` with the
same `limit`. Verify the ordered Installation IDs contain every current and
uninstalled row exactly once. Verify an empty Workspace returns `items: []`
without a cursor.

Verify missing or invalid `limit`/cursor is `400 VALIDATION_ERROR`, missing
bearer is `401 UNAUTHENTICATED`, non-owner is `403 FORBIDDEN`, unknown
Workspace/Installation is `404 NOT_FOUND`, and injected persistence failure is
`503 DEPENDENCY_ERROR` rather than `200 items: []`.

## Fallback Delta

```text
Fallback delta: removed 0, retained 2, added 0, net 0
Added fallback evidence: none
```

The two retained behaviors are the explicit empty array and required page-size
contract semantics; neither is a degraded dependency fallback.
