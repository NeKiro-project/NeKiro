# External Gateway Foundation v1

- Status: Active for issue #90 first delivery slice
- Go package: `github.com/NeKiro-project/NeKiro/gateway`
- Architecture decision: [ADR 0011](../decisions/0011-external-gateway-foundation.md)

## Ownership

This contract describes desired-state reconciliation and observation for an
external Gateway provider. The root `gateway/` package is not
`apps/control-plane/internal/gateway`: the latter remains NeKiro's semantic
northbound API boundary.

The external Gateway contract does not authenticate callers, authorize a
Workspace, resolve or change an Agent Card/Release, issue Router credentials,
invoke an Agent, proxy traffic, select an instance, or write Invocation/Ledger
facts. It never carries Agent input/output, headers, credentials, secrets, or
provider administrative endpoints.

## Root API

```go
type Provider interface {
    Name() ProviderName
    Capabilities() Capabilities
    Reconcile(context.Context, RouteSpec) (ReconcileResult, error)
    Status(context.Context, RouteKey) (RouteStatus, error)
    BeginDrain(context.Context, RouteKey, DrainRequest) (ReconcileResult, error)
    Delete(context.Context, RouteKey, DeleteRequest) (ReconcileResult, error)
    Close() error
}
```

All public values are immutable. Constructors validate input and every
accessor for collections returns a copy. `Close` is idempotent. A nil context
is invalid; a canceled or expired context returns the typed `canceled` outcome
and exposes only its local `context.Canceled` or `context.DeadlineExceeded`
cause.

## Exact Route Identity

`RouteSpec` v1 has the following explicit facts:

| Field | Meaning |
| --- | --- |
| Route key | Provider-neutral desired-route identity |
| Desired revision | Opaque exact desired-state token; it has no ordering semantics |
| Release ID | Exact immutable Catalog Release identity |
| Card digest | Lower-case 64-hex exact Agent Card provenance |
| Agent ID / version | Exact Agent identity and strict semantic version |
| Endpoint origin / path | Canonical HTTPS or HTTP origin plus canonical path, forming the exact endpoint |
| Audience | Canonical Router credential audience derived from the exact endpoint |
| Discovery owner | Exactly `gateway` or `router` |
| Backend reference | Opaque provider-local reference, never an arbitrary URI or secret |
| Required capabilities | Explicit feature set the selected provider must support |

The route retains all facts byte-for-byte after validation. It does not accept
an alternate Release, endpoint, audience, backend, or discovery owner.

An `endpoint_origin` has scheme and host only, no user info, path, query, or
fragment. It must already be canonical. An `endpoint_path` begins with one
slash, has no query, fragment, percent escaping, authority, or redundant slash
prefix. The assembled canonical endpoint is at most 2048 bytes. `audience`
must equal the existing `CanonicalRouterAgentAudience` result for that exact
endpoint.

## Discovery Ownership

`gateway` means the external Gateway owns discovery and selection behind the
stable exact-Release origin. `router` means a separately approved Router-side
Instance Directory and Selector owns discovery and pins one instance for an
Invocation. A route has exactly one value; the v1 contract has no dual-owner,
union, failover, or automatic conversion mode.

Provider implementations that cannot support the selected mode return typed
`unsupported`, rather than attempting an alternate owner. The deterministic
test fake implements no data plane and returns
`unsupported(router_discovery_unsupported)` for Router-owned discovery.

## Capabilities

The v1 capability vocabulary is closed:

| Capability | Provider evidence required before accepting it |
| --- | --- |
| `forwarding` | Forward the approved exact route without rewriting platform semantics |
| `sse_forwarding` | Preserve SSE framing and event order |
| `sse_flush` | Flush SSE events without buffering the stream |
| `cancellation_affinity` | Route a cancellation to the same accepted Invocation target |
| `drain` | Begin provider-supported graceful drain for an exact desired revision |
| `data_plane_readiness` | Report readiness using provider-specific data-plane evidence |
| `retry_policy_control` | Explicitly disable Agent execution and cancellation retry behavior |
| `instance_selection` | Select one upstream only in the declared Gateway-owned mode |

Capabilities are not implications. In particular, `forwarding` does not imply
SSE flush, affinity, drain, readiness, retry control, or instance selection.
A `RouteSpec` with an unsupported required capability must return typed
`unsupported(required_capability)` and must not record a partial route.

## Reconciliation And Status

`Reconcile` declares the exact desired route. Repeating an identical desired
revision is idempotent. Reusing a desired revision with different immutable
route facts is typed `invalid(revision_reused)`.

`RouteStatus` carries one route key, one desired revision, an optional separate
observed revision, and one state:

| State | Meaning |
| --- | --- |
| `absent` | Known desired route is absent at the provider |
| `accepted` | Desired state was accepted; no readiness claim |
| `programmed` | Provider observed the route programmed; no readiness claim |
| `not_ready` | Provider reports the route is not ready |
| `rejected` | Provider rejected the desired configuration |
| `empty_upstream` | Provider has no usable upstream for the route |
| `draining` | Drain has begun under provider-specific evidence |
| `deleting` | Desired deletion was accepted; deletion is not yet observed complete |
| `deleted` | Provider observed deletion complete |
| `stale_revision` | Desired and observed revisions are both present and differ |

`programmed` is never a data-plane-ready alias. A concrete adapter must publish
the evidence and capability semantics that allow a caller to use a readiness
claim. There is no polling, cache, stale-status reuse, or hidden reconciliation
recovery in v1.

`BeginDrain` and `Delete` each include an explicit expected desired revision.
An unknown route returns `not_found`; a mismatched expected revision returns
`stale`; unsupported drain returns `unsupported`. Deleting and an observed
`deleted` state remain different, and repeated exact deletion is idempotent.

## Outcomes

| Situation | Public result |
| --- | --- |
| Invalid or incomplete contract input | typed `invalid` |
| Required capability/mode unavailable | typed `unsupported` |
| Provider HTTP 401/403 | typed `unauthorized` |
| Provider outage, rate limit, or dependency failure | typed `unavailable` |
| Provider rejects a valid desired route | typed `rejected` |
| Provider explicitly reports not ready | typed `not_ready` |
| Unknown route key | typed `not_found` |
| Expected desired revision differs | typed `stale` |
| Caller cancellation/deadline | typed `canceled` for that call |
| Explicit local provider closure | typed `closed` |

`OutcomeError` exposes only a closed, safe cause vocabulary. It must not expose
provider response bodies, headers, URLs, credentials, Agent payloads, or
arbitrary transport text.

## Testkit

`gateway/testkit` supplies a deterministic `FakeProvider` and a reusable
capability-aware provider conformance harness. The fake has no network client,
vendor SDK, proxy, retry, polling loop, data-plane readiness, forwarding, SSE,
affinity, drain, or instance-selection capability. It is a contract fixture,
not an Envoy/APISIX/Higress emulator.

## Non-Goals

This v1 foundation does not provide Envoy Gateway, APISIX, or Higress adapters;
configuration snapshots or provider selection in a `cmd` composition root;
metrics/operator projection; actual JSON or SSE forwarding; a reverse proxy;
load balancing; Router Selector integration; traffic policy; drain timing;
provider federation; retry; failover; stale cache; or Stack acceptance. Each
requires a later versioned provider-specific contract and integration evidence.

## Compatibility

This is a Go package v1 contract. Additive optional capabilities may be added
only if existing provider behavior is unchanged. Changing route identity,
canonical endpoint/audience validation, a state, an outcome, a required
operation, revision semantics, or discovery-owner semantics is breaking and
requires a new contract version plus migration policy.
