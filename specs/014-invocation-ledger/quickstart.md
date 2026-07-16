# Quickstart: Invocation Ledger

## Prerequisites

- Go 1.26
- A reachable PostgreSQL 17 database in an explicit integration-test variable
- No automatic localhost or embedded database fallback

## Verify

```powershell
go test ./apps/a2a-router/internal/ledger ./apps/a2a-router/internal/api
go test -tags=integration ./apps/a2a-router/internal/ledger
go test ./...
go vet ./...
```

The integration suite migrates an isolated `ledger` schema, verifies readiness,
append/projection atomicity, concurrency, lineage, restart reads, Workspace
isolation, and prohibited-content absence.
