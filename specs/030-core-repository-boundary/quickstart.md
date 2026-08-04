# Quickstart: Verify the Repository Split

## 1. Provenance

```text
git show pre-repository-split-2026-08-04^{commit}
git fsck --full
```

For each extracted repository, compare the export commit tree to the matching
source subtree or selected path union and retain the comparison in its pull
request.

## 2. Core

```text
gofmt -l apps contracts tests
go mod tidy
git diff --exit-code -- go.mod go.sum
go build ./...
go test ./...
go test -race ./...
go vet ./...
go test -tags=integration -count=1 ./apps/control-plane/internal/catalog/postgres
go test -tags=integration -count=1 ./apps/control-plane/internal/workspace/postgres
go test -tags=integration -count=1 ./apps/control-plane/internal/workspace/integration
go test -tags=integration -count=1 ./apps/a2a-router/internal/ledger
go test -tags=integration -count=1 ./tests/integration/catalog
```

The final tracked-tree scan must find no `apps/console`, `sdks`, `agents`,
`deploy`, `tests/e2e`, `specs`, `.specify`, or repository-local Spec Kit skill.

## 3. SDK

```text
gofmt -l .
go mod tidy
git diff --exit-code -- go.mod go.sum
go build ./...
go test ./...
go test -race ./...
go vet ./...
```

Scan for core `internal` imports and `replace` directives; both must be absent.

## 4. Samples

```text
gofmt -l .
go mod tidy
git diff --exit-code -- go.mod go.sum
go build ./...
go test ./...
go test -race ./...
go vet ./...
docker build -f runtime-a/Dockerfile .
docker build -f runtime-b/Dockerfile .
```

Both sample Runtimes must consume explicit released core/SDK identities.

## 5. Console

```text
pnpm install --frozen-lockfile
pnpm typecheck
pnpm test
pnpm build
```

Console CI must not check out core or assemble the backend stack.

## 6. Stack

```text
docker compose --file compose.yaml config --quiet
go test -tags=e2e -count=1 ./tests/backend
pnpm --dir console run test:e2e
```

Before execution, validate `components.json`, resolve every exact component
identity, and reject local or floating sources. Run backend and browser
acceptance in a fresh Compose project and remove volumes afterward.

## 7. GitHub Gates

Every repository must report workflow `CI` with a successful aggregate
`required` job. Public repositories require pull requests, one independent
approval, resolved review conversations, the aggregate check, and protection
against force push and deletion. The private Console currently cannot enable
the same ruleset on the organization's GitHub plan; its CI and review process
remain mandatory and the limitation is recorded visibly.

## 8. Final Product Proof

The pinned Stack assembly must pass trusted registration, verification,
publication, discovery, exact installation, JSON invocation, SSE invocation,
nested Runtime B -> Router -> Runtime A invocation, Ledger lineage queries,
authorization/lifecycle/dependency negative paths, cancellation races, and
secrecy scans.

Fallback delta: removed copied-source and local-replacement paths; added 0.

Added fallback evidence: none.
