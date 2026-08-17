# Contract Compatibility Policy

## Released HTTP API identity

NeKiro `v0.1.0` exposes one version at each NeKiro-owned HTTP boundary:

| Owner boundary | Active prefix | Active OpenAPI documents |
| --- | --- | --- |
| Gateway | `/v1` | `control-plane.v1.yaml`, `control-plane-invocation.v1.yaml`, `trusted-publication.v1.yaml`, `public-agent-share.v1.yaml` |
| Control Plane internal | `/internal/v1` | `control-plane-internal.v1.yaml`, `control-plane-installed-version.v1.yaml` |
| Router internal | `/internal/v1` | `router-internal.v1.yaml`, `router-metadata.v1.yaml`, `router-topology-status.v1.yaml` |
| Agent-to-Router | `/agent/v1` | `router-agent.v1.yaml` |

The pre-release `/v2`, `/v3`, and `/v4` routes and OpenAPI documents are not
released compatibility surfaces. They are removed rather than served as
aliases. See [ADR 0021](../decisions/0021-pre-release-platform-api-v1-reset.md)
and the [migration guide](../usage/platform-api-v1-migration.md).

## Independent payload versions

HTTP API version, Agent Card Schema, Agent version, Workspace/Installation
schema, Invocation Event, Platform Error, result, A2A Profile, A2A protocol,
Router credential, and topology projection versions are independent. A URL
version must never be inferred from a payload `schemaVersion`, and resetting
the URL API does not rewrite persisted facts.

| Contract | Active identity |
| --- | --- |
| Agent Card Schema | `0.2` |
| Workspace Schema | `1` |
| Installation Schema | `2` |
| Public Agent Share | `1` |
| Invocation Result | `1` |
| Invocation Result Stream Event | `2` |
| Invocation Event | `0.3` for runtime/Ledger |
| Platform Error | surface-specific `2`, `3`, or `4` |
| A2A Profile Schema / protocol | `0.2` / `0.3.0` |
| Router Invocation Credential | `1` |
| Router Topology Status | `1` |

Historical payload schemas remain readable where required for immutable
Release or Ledger provenance. Active validators do not silently upgrade,
downgrade, dual-read, or reinterpret them.

## Compatible changes

- Adding an optional field is compatible only when omission preserves the
  existing meaning.
- Adding an endpoint is compatible only when existing clients and ownership
  remain valid.
- Adding an enum value requires explicit consumer-impact review because an
  exhaustive consumer may treat it as breaking.
- Documentation may clarify behavior only when it does not change accepted
  input, output, failure, ownership, or security semantics.

## Breaking changes

The following require a new contract version, migration guidance, and an
explicit compatibility window after `v0.1.0`:

- removing or renaming a field;
- changing type, requiredness, status code, media type, or fixed error meaning;
- tightening accepted values or semantic validation;
- moving behavior or data to a different owner;
- reinterpreting an immutable Release, Installation, Invocation, or Ledger
  fact;
- changing the trusted caller, credential, or correlation source;
- changing an existing route without retaining its documented compatibility
  policy.

## Failure and data semantics

Missing input, invalid input, not found, forbidden, disabled, dependency
failure, timeout, cancellation, and protocol failure remain distinct. They
must not collapse into `null`, an empty collection, a normal success response,
an automatic retry, or a request to an alternate endpoint.

Platform Error public messages and correlation are fixed by their payload
contract. Agent input, output, credentials, endpoint details, raw dependency
errors, and stack data are forbidden from metadata-only Ledger contracts.

## Pre-release v1 cutover

All first-release consumers move to v1 together. There is no redirect, route
alias, dual-read, dual-write, dual-dispatch, downgrade, or old-Core fallback.
Retired paths return `404` before domain behavior. Required service URLs accept
only their exact v1 destination.

Fallback delta: removed every pre-release v2/v3/v4 HTTP path, retained 0,
added 0, net negative.

Added fallback evidence: none.
