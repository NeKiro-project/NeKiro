# ADR 0021: Pre-release Platform API v1 Reset

- Status: Accepted
- Date: 2026-08-17
- Issue: [NeKiro#120](https://github.com/NeKiro-project/NeKiro/issues/120)

## Context

Before the first product release, NeKiro used URL versions `v2`, `v3`, and
`v4` to make incompatible implementation slices explicit while contracts were
still being discovered. Those numbers describe pre-release iteration order;
they do not represent three supported product generations.

Publishing `v0.1.0` with all of those URL families would make a new adopter
configure several apparently independent API generations and would imply a
compatibility promise that NeKiro has never released. No published Core,
Console, Go SDK, Samples, or Stack release requires the pre-release paths.

Payload contracts are different. Agent Card, Installation, Invocation Event,
Platform Error, result stream, A2A Profile, and Router credential identities
are stored or exchanged independently. Resetting a URL version does not permit
rewriting those schema identities or historical Ledger facts.

## Decision

The first released platform exposes exactly one version for every NeKiro-owned
HTTP trust boundary:

| Boundary | Versioned prefix |
| --- | --- |
| Gateway | `/v1` |
| Control Plane and Router internal APIs | `/internal/v1` |
| Agent-to-Router API | `/agent/v1` |

The current Catalog, Workspace, Installation, trusted publication, public
sharing, Invocation, and Ledger-read behavior moves to `/v1`. Exact Agent
resolution, installed-version resolution, Router dispatch, and Router metadata
reads move to `/internal/v1`. `/agent/v1/invocations` remains unchanged.

Active OpenAPI documents use `info.version: 1.0.0`. Superseded HTTP API
documents named `v2`, `v3`, or `v4` are removed from the embedded contract
tree. Focused v1 documents may describe separate owners or domains, but they
do not create separate URL versions.

There is no redirect, route alias, content negotiation fallback, dual-read,
dual-write, dual-dispatch, or automatic endpoint probing. Requests to retired
paths return the normal unmatched-route `404` and do not enter domain logic.
Strict service endpoint configuration accepts only the v1 paths.

Independently versioned payload schemas keep their current identities. In
particular, Platform Error v2/v3/v4 and Invocation Event v0.3 remain valid
payload identities on their documented surfaces; they are not URL versions.

## Release ordering

1. Review the Core v1 contract and handler change.
2. Update Console, Go SDK, Samples, and Stack against the exact Core revision.
3. Prove the complete cross-runtime product loop with exact component commits.
4. Publish component tags and immutable image digests.
5. Publish the Stack compatibility manifest only after every exact revision is
   green.

Core required CI remains Core-only. Cross-repository acceptance stays in the
satellite-owned reusable workflows pinned by full commit SHA.

## Consequences

- New users configure one platform API generation.
- Generated clients and documentation no longer expose pre-release iteration
  history as supported product surface.
- Every pre-release consumer must migrate atomically before the release.
- Independently versioned payload contracts remain explicit and may still have
  version numbers different from the URL API.
- A future incompatible HTTP change requires a new URL version, migration
  policy, and compatibility window after product release.

## Compatibility

This is an intentional breaking pre-release change. It has no runtime
compatibility window because there is no published product consumer to
preserve. The exact mapping is documented in
[`platform-api-v1-migration.md`](../usage/platform-api-v1-migration.md).

## Fallback report

Fallback delta: removed every pre-release v2/v3/v4 HTTP path, retained 0,
added 0, net negative.

Added fallback evidence: none.
