# Quickstart: Install a Published Agent and Pin an Exact Version

This guide validates issue #5 on the dependent Workspace create/read branch.
It does not claim Installation inspection/lifecycle or Router completion.

## Checks

```powershell
go test -count=1 ./contracts
go test -count=1 ./apps/control-plane/internal/workspace/...
go test -count=1 ./apps/control-plane/internal/gateway
```
Run the two schema-resetting PostgreSQL packages serially against a dedicated
`_test` database:

```powershell
go test -tags=integration -count=1 ./apps/control-plane/internal/workspace/postgres
go test -tags=integration -count=1 ./apps/control-plane/internal/workspace/integration
```

## Owner Workflow

With an explicit owner bearer token, an existing Workspace, and a published
Catalog Card:

```powershell
$headers = @{ Authorization = "Bearer $ownerToken" }
$body = '{"agentId":"runtime-a","versionConstraint":"^1.0.0","acceptedPermissions":[]}'
$installation = Invoke-RestMethod -Method Post `
  -Uri "$base/v3/workspaces/workspace-alpha/installations" `
  -Headers $headers -ContentType 'application/json' -Body $body
if ($installation.status -ne 'enabled' -or $null -eq $installation.acceptedPermissions -or $installation.acceptedPermissions.Count -ne 0) {
  throw 'empty permission Installation was not preserved'
}
```

Verify that omitted or `null` `acceptedPermissions` returns `400`, a
non-owner returns `403`, no matching published version returns `404`, a
duplicate or concurrent loser returns `409`, and Catalog/Workspace failures
return `503`. Publish a newer matching version and verify the existing exact
pin and permission snapshot do not change. Reconstruct the service against the
same database and compare all Installation fields.

## Static and Fallback Verification

```powershell
go test -race -count=1 ./...
go vet ./...
go build ./...
go mod tidy -diff
git diff --check
```

Fallback delta must remain:

```text
Fallback delta: removed 0, retained 1, added 0, net 0
Added fallback evidence: explicit empty accepted-permission set is product policy; no dependency fallback
```
