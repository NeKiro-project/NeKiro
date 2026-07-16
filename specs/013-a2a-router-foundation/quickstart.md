# Quickstart: A2A Router Foundation

Expected verification after implementation:

```powershell
go test ./apps/a2a-router/internal/config ./apps/a2a-router/internal/auth ./apps/a2a-router/internal/resolution ./apps/a2a-router/internal/api ./apps/a2a-router/cmd/a2a-router
go test ./...
go vet ./...
git diff --check
```

A valid local Router process requires explicit configuration for every
destination, credential, limit, and deadline. No default localhost URL, weak
token, anonymous mode, retry, cache, Agent endpoint call, or Ledger write is
allowed in this feature.
