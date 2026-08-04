# Core development

This guide covers the two Go services and their owned PostgreSQL schemas. The
Console, sample Agents, Compose assembly, and product E2E are maintained in
[NeKiro-Stack](https://github.com/NeKiro-project/NeKiro-Stack) and the other
satellite repositories listed in the root README.

## Requirements

- Go 1.26 or newer
- PostgreSQL 17 for integration tests and local service execution
- Docker only when building Core images

Run commands from the repository root.

## Local quality checks

```powershell
go mod download
gofmt -l apps contracts tests
go mod tidy
git diff --exit-code -- go.mod go.sum
go build ./...
go test ./...
go test -race ./...
go vet ./...
```

## PostgreSQL integration

Use a dedicated database whose name ends in `_test`. The suites own and reset
their schemas, so never point them at a shared, staging, or production database.

```powershell
$env:NEKIRO_TEST_DATABASE_URL = 'postgresql://user:password@127.0.0.1:5432/nekiro_core_test?sslmode=disable'
go test -tags=integration -count=1 ./apps/control-plane/internal/catalog/postgres
go test -tags=integration -count=1 ./apps/control-plane/internal/workspace/postgres
go test -tags=integration -count=1 ./apps/control-plane/internal/workspace/integration
go test -tags=integration -count=1 ./apps/a2a-router/internal/ledger
go test -tags=integration -count=1 ./tests/integration/catalog
```

Catalog migrations are embedded from
`apps/control-plane/internal/catalog/postgres/migrations`, Workspace migrations
from `apps/control-plane/internal/workspace/postgres/migrations`, and Ledger
migrations from `apps/a2a-router/internal/ledger/migrations`. Service startup
checks schema readiness but never performs an implicit migration.

## Service commands

Both services require explicit validated configuration. Missing database,
identity, endpoint, size, deadline, or credential values fail at startup; no
localhost, weak-key, no-op Ledger, or alternate endpoint defaults exist.

Apply migrations before serving:

```powershell
$env:NEKIRO_DATABASE_URL = 'postgresql://user:password@127.0.0.1:5432/nekiro?sslmode=disable'
go run ./apps/control-plane/cmd/control-plane migrate up
go run ./apps/a2a-router/cmd/a2a-router migrate up
```

After setting the full Control Plane configuration, start it with:

```powershell
go run ./apps/control-plane/cmd/control-plane serve
```

After setting the full Router configuration, start it with:

```powershell
go run ./apps/a2a-router/cmd/a2a-router serve
```

The owning configuration packages are the canonical variable inventory:

- `apps/control-plane/internal/config`
- `apps/a2a-router/internal/config`

Do not commit local bearer tokens, token digests, database URLs, Ed25519 keys,
or production origins. The Router private key remains only on the Router; Agent
verifiers consume the corresponding public key from their own deployment.

## Images

```powershell
docker build --file apps/control-plane/Dockerfile --tag nekiro-control-plane:local .
docker build --file apps/a2a-router/Dockerfile --tag nekiro-a2a-router:local .
```

Use NeKiro-Stack when both images must be assembled with Console, SDK, and
sample releases. Stack pins exact revisions and owns teardown, sanitized logs,
browser acceptance, and the complete Register -> Discover -> Install -> Invoke
-> Record proof.
