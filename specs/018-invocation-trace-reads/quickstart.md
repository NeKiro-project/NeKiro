# Quickstart: Invocation and Trace Metadata Reads

## Focused validation

```powershell
go test -count=1 ./apps/control-plane/internal/invocation ./apps/control-plane/internal/gateway ./apps/a2a-router/internal/api ./apps/a2a-router/cmd/a2a-router
```

The focused tests must prove auth and Workspace authorization happen before
Router access, exact v4/v3 paths and Bearer credentials, stable metadata-only
responses, parent-before-child Trace ordering, and distinct 400/401/403/404/
503 failures.

## Full validation

```powershell
go test -count=1 ./...
go vet ./...
git diff --check
wsl.exe -d Ubuntu-26.04 -- bash -lc 'cd /mnt/e/NeKiro && go test -race -count=1 ./apps/control-plane/... ./apps/a2a-router/... ./agents/runtime-b'
docker compose --file deploy/compose.yaml config --quiet
```

## Read behavior

```text
GET /v4/workspaces/{workspaceId}/invocations/{invocationId}
GET /v4/workspaces/{workspaceId}/traces/{traceId}
```

Only the Workspace owner can read these resources. A missing resource is a
typed `404 NOT_FOUND`; Router or Ledger failure is a typed `503
DEPENDENCY_ERROR`. Successful bodies contain only Invocation/Trace metadata
and immutable Event 0.3 facts.
