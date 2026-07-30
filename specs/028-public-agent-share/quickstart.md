# Quickstart: Public Agent Share URL

## Prerequisites

- Go, Node, pnpm, Docker Engine, and Compose versions required by the repository.
- All existing explicit Control Plane, Router, Runtime A/B, PostgreSQL, and Console acceptance values.
- One explicit backend public origin and the exact matching Console origin:

```text
NEKIRO_PUBLIC_AGENT_ORIGIN=https://agents.nekiro.test
VITE_NEKIRO_PUBLIC_AGENT_ORIGIN=https://agents.nekiro.test
```

Do not omit the origin or replace it with an inferred Host, alternate domain, redirect, localhost fallback, mock URL, or Agent endpoint.

## Static verification

```text
go test ./...
go vet ./...
pnpm typecheck
pnpm test
pnpm build
docker compose --file deploy/compose.yaml config --quiet
git diff --check
```

Expected result: contract, migration, Catalog, Gateway, Console mapping, URL parser, and component tests pass; production build contains no embedded test credential.

## Clean backend acceptance

Use the explicit environment values from `.github/workflows/ci.yml`, set `NEKIRO_PUBLIC_AGENT_ORIGIN`, and run:

```text
docker compose --project-name nekiro-public-share-acceptance --file deploy/compose.yaml down --volumes --remove-orphans
docker compose --project-name nekiro-public-share-acceptance --file deploy/compose.yaml up --build --detach --wait --wait-timeout 120
go test -tags=e2e -count=1 ./tests/e2e/invoke-record
docker compose --project-name nekiro-public-share-acceptance --file deploy/compose.yaml down --volumes --remove-orphans
```

The acceptance must start with empty volumes and prove:

1. Registration returns a stable public Agent ID and canonical URL.
2. Anonymous resolution before publication is `not_installable` with zero Releases.
3. After trusted publication, resolution returns the exact eligible Release without endpoint, evidence, or credentials.
4. B selects that exact Release, installs it into the authorized Workspace, and the returned `installedReleaseId` matches.
5. Runtime B invocation and Runtime A nested invocation retain existing Router/Ledger lineage.
6. Unknown/malformed IDs and suspended/revoked Releases retain exact failure/non-installable semantics.

## Production Console acceptance

Map the explicitly configured public host and Gateway host to the test machine, build with all existing `VITE_NEKIRO_*` values plus `VITE_NEKIRO_PUBLIC_AGENT_ORIGIN`, then run:

```text
pnpm --dir apps/console exec playwright install chromium
pnpm --dir apps/console run build
pnpm --dir apps/console run test:e2e
```

Verify both entry paths:

- open `/a/{publicAgentId}` directly;
- paste the same full canonical URL in Installations.

Neither entry may preselect a Release. Installation becomes enabled only after B selects one exact Release and explicitly accepts its permissions.

## Verification Evidence

The Spec 028 acceptance was run on an empty Compose volume with the explicit
`https://agents.nekiro.test` public origin:

- `go test -tags=e2e -count=1 ./tests/e2e/invoke-record`: passed. This covered
  stable identity allocation, unpublished and lifecycle projections, malformed
  and unknown IDs, exact public installation, cross-runtime Router invocation,
  Ledger lineage, dependency failures, and secrecy scans.
- Production Console Playwright acceptance: `1 passed (17.2s)`. It covered
  direct `/a/{publicAgentId}` and pasted canonical URL entry, anonymous public
  GETs, no Release preselection, explicit `text.read` consent, exact
  `installedReleaseId`, Invocation JSON/SSE, Trace, and Gateway-only browser
  requests.
- Static checks: `go test ./...`, `go vet ./...`, Console typecheck, 48 Console
  tests, production build, Compose config validation, and `git diff --check`
  all passed.

Fallback delta: removed 0, retained 0, added 0, net 0.
Added fallback evidence: none.
