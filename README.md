# NeKiro Core

NeKiro is the core of a runtime-agnostic Agent operating platform. This
repository owns the Control Plane, A2A Router, language-neutral contracts,
service-owned PostgreSQL migrations, and Core verification.

The platform loop is:

```text
Register -> Discover -> Install -> Invoke -> Record
```

Managed user-to-Agent and Agent-to-Agent calls pass through the A2A Router.
The Invocation Ledger records metadata and lineage, not Agent input or output.

## Repository scope

```text
apps/control-plane/   Gateway, Catalog, Workspace, and Invocation Dispatch
apps/a2a-router/      Routing, transport adaptation, credentials, and Ledger
contracts/            JSON Schema, OpenAPI, A2A profile, and Go mappings
tests/                Core contract and service integration tests
docs/                 Architecture, contracts, decisions, and Core usage
```

Catalog, Workspace, and Ledger SQL migrations stay beside the modules that own
their schemas. They are embedded in the corresponding service binaries and are
not maintained in a separate database repository.

## Satellite repositories

- [NeKiro-Console](https://github.com/NeKiro-project/NeKiro-Console) owns the production web Console.
- [nekiro-sdk-go](https://github.com/NeKiro-project/nekiro-sdk-go) owns the public Go Agent and application SDKs.
- [NeKiro-Samples](https://github.com/NeKiro-project/NeKiro-Samples) owns the cross-runtime sample Agents.
- [NeKiro-Stack](https://github.com/NeKiro-project/NeKiro-Stack) owns Compose assembly, immutable component pins, and product acceptance.
- [nekiro-a2a-transport-go](https://github.com/NeKiro-project/nekiro-a2a-transport-go) owns reusable A2A wire transport mechanics.

Core PR required CI never vendors or checks out satellite source. After every
merge to `main`, a separate Satellite Integration workflow calls immutable,
satellite-owned reusable workflows for SDK compatibility, both sample
Runtimes, and NeKiro-Stack backend/browser acceptance against that exact Core
commit. Product-level source, commands, and success criteria remain owned by
the satellite repositories.

## Build and test

Go 1.26 or newer is required.

```powershell
go mod download
go build ./...
go test ./...
go test -race ./...
go vet ./...
```

PostgreSQL integration suites require an explicit dedicated database whose
name ends in `_test`:

```powershell
$env:NEKIRO_TEST_DATABASE_URL = 'postgresql://user:password@127.0.0.1:5432/nekiro_core_test?sslmode=disable'
go test -tags=integration -count=1 ./apps/control-plane/internal/catalog/postgres
go test -tags=integration -count=1 ./apps/control-plane/internal/workspace/postgres
go test -tags=integration -count=1 ./apps/control-plane/internal/workspace/integration
go test -tags=integration -count=1 ./apps/a2a-router/internal/ledger
go test -tags=integration -count=1 ./tests/integration/catalog
```

Build the two Core images from the repository root:

```powershell
docker build --file apps/control-plane/Dockerfile --tag nekiro-control-plane:local .
docker build --file apps/a2a-router/Dockerfile --tag nekiro-a2a-router:local .
```

See [Core development](docs/usage/core-development.md) for service commands and
[trusted publication operations](docs/usage/trusted-publication-operations.md)
for the publication lifecycle. The complete architecture is documented in
[Phase 1 Architecture](docs/architecture/phase-1-spec.md).

## History

The annotated tag `pre-repository-split-2026-08-04` preserves the accepted
monorepo tree and tracked Spec Kit history. Repository ownership and migration
rationale are recorded in [ADR 0009](docs/decisions/0009-core-repository-boundary.md).

NeKiro is licensed under the Apache License 2.0.
