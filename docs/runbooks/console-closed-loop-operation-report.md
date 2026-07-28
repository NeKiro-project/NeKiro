# Console Closed-Loop Operation Report

**Date:** 2026-07-29
**Scope:** Spec 027 production Console and trusted publication loop
**Merged upstream commit:** `adb1a3873f62f4ff08f3ef29fb944e36d8fdbc4c`
**Pull request:** [NeKiro #65](https://github.com/NeKiro-project/NeKiro/pull/65)
**Acceptance run:** [CI run 30401796416](https://github.com/NeKiro-project/NeKiro/actions/runs/30401796416)

## Result

The reviewed Console delivery is merged into `upstream/main`. The fresh
Compose backend acceptance and the production-route Playwright acceptance both
passed in CI. The acceptance covers:

```text
Register -> Verify -> Publish -> Discover -> Install -> Invoke -> Record
                                      Runtime B -> Router -> Runtime A
                                      Trace / Ledger
```

No new product defect was found during the final green acceptance run, so no
additional upstream issue was opened from this sweep. The earlier Console
issues handled by this delivery were #66, #67, #68, and #69; their fixes were
reviewed and included in PR #65.

## Verified CI Evidence

| Check | Evidence | Result |
| --- | --- | --- |
| Frontend typecheck, unit tests, and production build | `frontend` job in run `30401796416` | Passed |
| Backend trusted publication and reverse lineage | `backend-acceptance` job `90418109782` | Passed |
| Production browser route | `console-browser-acceptance` job `90418251841` | Passed |
| Go quality, runtime samples, workspace integration, and Compose config | Remaining CI jobs in run `30401796416` | Passed |
| PR merge state | PR #65, merge commit `adb1a387` | Merged |

The browser job used a fresh PostgreSQL/Compose stack, built the production
Console with explicit configuration, installed Chromium, and ran
`pnpm --dir apps/console run test:e2e`. It did not use route interception, a
mock Gateway, a direct Agent call, or a Router-internal browser endpoint.

## Operator Preconditions

Use explicit, non-empty values for:

```text
VITE_NEKIRO_API_BASE_URL
VITE_NEKIRO_PROVIDER_ID
VITE_NEKIRO_PROVIDER_TOKEN
VITE_NEKIRO_OWNER_TOKEN
VITE_NEKIRO_DEFAULT_WORKSPACE_ID
```

The browser acceptance also requires the Compose variables documented in
[Spec 027 quickstart](../../specs/027-console-trusted-publication/quickstart.md),
including the database URL, development principals, Router credential key,
Runtime A/B credentials, byte limits, and deadlines. Map the Gateway origin to
the local machine without changing the configured origin:

```text
127.0.0.1 gateway.nekiro.test
```

Do not replace the explicit Gateway origin with `localhost`, an IP literal, a
mock server, or an alternate endpoint.

## Standard Operation

From the platform repository:

```text
pnpm install --frozen-lockfile
docker compose --project-name nekiro-root-console-browser --file deploy/compose.yaml up --build --detach --wait --wait-timeout 120
pnpm --dir apps/console exec playwright install --with-deps chromium
pnpm --dir apps/console run build
pnpm --dir apps/console run test:e2e
docker compose --project-name nekiro-root-console-browser --file deploy/compose.yaml down --volumes --remove-orphans
```

The final teardown is required even when an earlier command fails.

## Console Walkthrough

1. Open the production preview route and confirm `Agent Card Catalog` and
   `API: configured` are visible.
2. Create the configured Workspace.
3. In `Registry`, register Runtime A and Runtime B Agent Cards. Confirm each
   Card uses the explicit provider identity, version `1.0.0`, its A2A endpoint,
   and the `runtime.echo` capability.
4. In `Trusted Publication`, select each Card, create an Endpoint Binding,
   issue a Challenge, and confirm the endpoint has served the one-time proof.
5. Complete verification. Confirm the Binding is read back as `verified` and
   the proof is no longer displayed or retained in browser storage, request
   bodies, or browser console messages.
6. Create a Release for the exact Card version and Binding. Verify it when
   required, publish it, and record the displayed Release ID and Card digest.
7. In `Installations`, select each published Agent, enter its Release ID, and
   run `Preflight`. Confirm the Release is published and its Card, digest,
   Binding, endpoint, and version provenance match. Install the exact pin and
   confirm the returned `installedReleaseId` equals the preflight Release ID.
8. Enter an unknown Release ID and run `Preflight`. Confirm the Gateway
   returns `HTTP 404`, `NOT_FOUND`, and a trace ID rather than a success state.
9. In `Invocations`, select Runtime B, use capability `runtime.echo`, and
   invoke JSON input. Confirm the response contains the Gateway-generated
   `invocationId`, `rootTaskId`, and `traceId`, and that the nested Runtime A
   result is present.
10. Repeat with `Stream result over SSE` enabled. Confirm sequence `0` is an
    `accepted`/`pending` event and the final event is
    `completed`/`succeeded`; correlation IDs remain unchanged throughout.
11. In `Ledger`, read the returned Trace ID. Confirm the Runtime B root
    Invocation and Runtime A child Invocation share the root Task and Trace,
    and that the child `parentInvocationId` equals the root Invocation ID.
    Confirm both Release IDs and Card digests are visible in the operation
    record.
12. Open the isolated demo routes. Confirm they render without any Gateway,
    Agent, or Router-internal request.

## Acceptance Matrix

| Area | Asserted behavior |
| --- | --- |
| Browser boundary | Every API request originates at the configured Gateway; no `/internal/`, `/agent/`, Runtime A, or Runtime B request originates from the browser. |
| Publication | Binding challenge, verification, Release verification, and publication are server-authoritative and visible as distinct states. |
| Installation | Workspace installation uses a published exact Release handoff and validates the returned Release provenance. |
| JSON invocation | Runtime B is reached through Gateway and Router, and its nested Runtime A result is returned with correlated metadata. |
| SSE invocation | Accepted and terminal events are ordered, size-bounded, and share invocation, root task, and trace identifiers. |
| Trace | The root/child relationship is queryable by Workspace-scoped Trace ID. |
| Ledger | Runtime A and Runtime B facts, Release provenance, Card digests, and terminal status are visible without secrets. |
| Negative path | Unknown Release preflight returns an explicit `NOT_FOUND` error and trace ID. |
| Isolation | Demo routes do not call the backend. |

## Local Verification Note

The local machine has Docker Desktop and the required PostgreSQL image. A
local clean Compose attempt was made with the same explicit configuration, but
the machine could not complete external dependency downloads:

1. The first build hit multiple `unexpected EOF` failures from
   `proxy.golang.org` while downloading Go modules.
2. The second build completed the Control Plane images but failed to obtain
   Docker Hub authorization for the uncached `alpine:3.22` and
   `golang:1.26-alpine` layers.
3. A Host-process fallback attempt also lacked the uncached Router JWT module
   and the Runtime A `trpc-agent-go` module.

These are environment dependency failures, not application assertions. The
authoritative fresh-environment browser and backend runs are the green CI
jobs listed above. No code or contract fallback was added to work around the
local network condition.

## Recovery and Safety

- Keep the Gateway, Catalog, Workspace, Router, and Ledger ownership boundaries
  intact.
- Treat Release IDs, installed Release IDs, Invocation IDs, Root Task IDs, and
  Trace IDs as runtime facts; never substitute hand-written values.
- Do not reuse a challenge proof, put it in storage or URLs, or send it in a
  later request.
- On an acceptance failure, preserve the sanitized CI/backend logs and the
  failing response status, error code, and trace ID.
- If a new product defect is found, open an upstream issue with reproduction
  steps and evidence before changing code. Implement it in a scoped PR, run an
  independent issue/spec review, then rerun the full acceptance matrix.
