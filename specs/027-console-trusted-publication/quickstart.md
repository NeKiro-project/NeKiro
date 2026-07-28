# Quickstart: Console Trusted Publication Loop

## Prerequisites

- Node and pnpm versions required by the root `package.json`.
- Go version from `go.mod`.
- Docker Engine with Compose v2 for the fresh backend environment.
- Explicit, non-empty environment values for Gateway origin, provider ID,
  separate provider/owner bearer values, one Workspace, and the existing
  acceptance secrets.
- A sample endpoint that can serve the configured trusted-publication challenge.

Do not use localhost, a wildcard origin, an empty token, a trimmed token, a
mock Gateway, SQL state mutation, direct Agent calls, retry, reconnect, or an
alternate endpoint to satisfy a missing prerequisite.

## Frontend checks

From the platform repository:

```text
pnpm install --frozen-lockfile
pnpm typecheck
pnpm test
pnpm build
```

The commands must discover `apps/console` and execute its non-empty scripts.

From the standalone Console repository, the same focused commands remain:

```text
npm install --no-package-lock
npm run typecheck
npm run lint
npm test
npm run build
```

Slice B review evidence: the independent post-fix review reported `PASS` with
no actionable High/Medium findings. The Console test command includes API,
policy, and production component static-render checks.

Standalone implementation PR: https://github.com/NeKiro-project/NeKiro-Console/pull/6

## Live Console walkthrough

1. Configure the exact Gateway origin, `VITE_NEKIRO_PROVIDER_ID`,
   `VITE_NEKIRO_PROVIDER_TOKEN`, `VITE_NEKIRO_OWNER_TOKEN`, and
   `VITE_NEKIRO_DEFAULT_WORKSPACE_ID`. The old single `VITE_NEKIRO_TOKEN`
   input is not accepted.
2. Open the production Console route, create or load one Workspace, and keep
   the Workspace identity explicit.
3. Register an Agent Card and inspect the returned draft.
4. Create an Endpoint Binding, issue a Challenge, and confirm the endpoint has
   served the challenge material.
5. Complete the Challenge and read the Binding until the server returns
   `verified`.
6. Create a Release for the exact Card version and Binding, verify it, and
   publish it.
7. Discover the published Agent. Select or enter the provider handoff Release
   ID, read it through the Gateway, verify `state=published`, exact Card
   version, Card digest, Binding, and endpoint provenance, then install the
   exact version into the Workspace. Confirm the returned
   `installedReleaseId` is the same ID before showing success.
8. Install the second sample Agent, invoke B through Gateway, and inspect the
   JSON/SSE result and Invocation/Trace metadata.
9. Open the Trace and verify the B root and A child share the root Task/Trace
   and preserve the parent Invocation relationship.

## Backend reverse-lineage acceptance

Use the existing clean Compose environment variables from
`specs/026-trusted-publication-acceptance/quickstart.md`, then run:

```text
docker compose --project-name nekiro-console-acceptance --file deploy/compose.yaml down --volumes --remove-orphans
docker compose --project-name nekiro-console-acceptance --file deploy/compose.yaml up --build --detach --wait --wait-timeout 120
go test -tags=e2e -count=1 ./tests/e2e/invoke-record
docker compose --project-name nekiro-console-acceptance --file deploy/compose.yaml down --volumes --remove-orphans
```

The acceptance must include the reverse B -> Router -> A lineage and must
report exact Release IDs, parent/child Invocation IDs, and Trace IDs on failure.
Runtime A and Runtime B must each receive separate explicit Router Agent
tokens and response/event limits; missing values must fail Compose interpolation.
