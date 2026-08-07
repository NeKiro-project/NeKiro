# ADR 0011: Provider-Neutral External Gateway Foundation

- Status: Accepted for issue #90 first delivery slice
- Date: 2026-08-07
- Decision owner: Platform Architecture
- Related issue: [#90](https://github.com/NeKiro-project/NeKiro/issues/90)

## Context

NeKiro has two different meanings of "Gateway" that must remain separate.
`apps/control-plane/internal/gateway` is the semantic northbound API boundary:
it authenticates callers, establishes invocation identity, and hands managed
invocations to the A2A Router. External products such as Envoy Gateway, APISIX,
and Higress are infrastructure providers that may configure routes, observe
their reconciliation, and, in a future approved integration, forward traffic.

Those products have incompatible control surfaces and different support for
streaming, affinity, draining, readiness, discovery, and retry policy. Letting
the Control Plane or Router import a product SDK would couple platform semantics
to one provider. Treating an administrative write as proof that an Agent route
is live would also hide asynchronous data-plane failure.

The related Instance Registry (#89) owns only ephemeral topology for an
already exact Release. A route must name exactly one discovery owner. Gateway
and Router may not independently select from the same instance topology.

## Decision

1. Add the root Go package `gateway/` with package name `gateway`. It is the
   **External Gateway integration boundary**, not the Control Plane semantic
   Gateway and not a reverse proxy.
2. Its v1 provider-neutral API contains immutable exact `RouteSpec` values,
   `Provider.Reconcile`, `Status`, `BeginDrain`, `Delete`, `Close`, explicit
   capability reporting, typed outcomes, and a deterministic fake/conformance
   harness. It does not expose a provider SDK, administrative endpoint,
   credential, request body, response body, or header map.
3. A `RouteSpec` preserves an exact Release ID and Card digest, Agent ID and
   version, canonical endpoint and Router credential audience, opaque backend
   reference, desired revision, and one explicit discovery owner. It cannot
   substitute another Release, endpoint, audience, or discovery mode.
4. `accepted` reports acceptance of desired state and `programmed` reports a
   provider observation. Neither means the data plane is ready. A provider may
   claim readiness only through `data_plane_readiness` plus its own documented
   evidence; readiness remains a separately observed route state.
5. Forwarding, SSE forwarding/flush, cancellation affinity, drain, readiness,
   retry-policy control, and instance selection are optional capabilities. A
   provider must reject a route whose declared requirements it cannot meet. It
   must never silently emulate a capability or choose an alternate provider.
6. A route has one discovery owner: `gateway` for provider-owned stable-origin
   discovery, or `router` for separately approved Router-side Directory and
   Selector work. The v1 fake deliberately supports neither data-plane mode;
   it refuses Router-owned discovery rather than simulating selection.
7. Desired and observed revisions remain distinct opaque values. `stale` is an
   explicit outcome for an operation conditioned on a different desired
   revision; a `stale_revision` status requires distinct desired and observed
   revisions. No revision ordering, cache, or recovery path is inferred.
8. This first slice has no Envoy Gateway, APISIX, Higress, Config Center, or
   `cmd` composition-root adapter. Each requires a later approved provider
   contract, capability evidence, configuration ownership, and Stack-owned
   integration coverage.

## Consequences

- The semantic Gateway remains the only northbound API entry point. An
  external Gateway provider cannot authenticate a NeKiro caller, authorize a
  Workspace, resolve an Agent Release, construct Router credentials, dispatch
  an Invocation, or write Ledger facts.
- Provider implementations can be tested through one capability-aware
  conformance suite without importing provider types into Control Plane or
  Router domain packages.
- Callers can distinguish invalid input, unsupported capability, unauthorized
  provider access, provider unavailability, provider rejection, not-ready,
  not-found, stale revision, cancellation, and local closure.
- Provider adapters must prove their own safe forwarding behavior before they
  advertise it. In particular, JSON/SSE payload transparency, no Agent
  execution retry, cancellation affinity, drain bounds, and data-plane
  readiness are not asserted by an administrative success response.
- The v1 foundation intentionally provides no implicit default provider,
  provider switch, upstream/endpoint switch, retry, stale route cache,
  alternate Release, background polling, or recovery behavior.

## Rejected Alternatives

### Reuse the Control Plane Gateway package

Rejected because external infrastructure reconciliation and semantic API
handling have different data ownership, failure semantics, and dependencies.
Sharing the name at a filesystem root does not make them the same owner.

### Put Envoy, APISIX, and Higress SDK types in the Router

Rejected because Router is a data-plane invocation owner, not an external
Gateway control-plane client. It would reverse the dependency boundary and
bind Agent dispatch to one provider's management API.

### Treat accepted or programmed as ready

Rejected because external Gateway reconciliation is asynchronous. An accepted
configuration can still be rejected, empty upstream, unavailable, or not ready
in the data plane.

### Infer a fallback discovery owner or provider

Rejected because simultaneous selection by Gateway and Router creates
untraceable affinity and authority conflicts. Explicit failure preserves the
exact Release and Invocation lineage.

## Fallback Delta

Fallback delta: removed 0, retained 0, added 0, net +0.

Added fallback evidence: none.
