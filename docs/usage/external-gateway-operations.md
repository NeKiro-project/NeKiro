# External Gateway Operations

The root `gateway/` package currently defines the provider-neutral lifecycle;
it does not yet ship an Envoy Gateway, APISIX, or Higress adapter. Operators
must not treat the foundation or a successful administrative call as a usable
data-plane route.

## Rollout

A provider adapter must receive one exact `RouteSpec`, reject unsupported
capabilities, reconcile the desired revision, and report observed status.
`accepted` and `programmed` are not readiness. Traffic may begin only when the
adapter has explicit `data_plane_readiness` capability and reports its
documented evidence.

## Reconciliation Failure

Keep desired and observed revisions distinct. A rejected, unavailable,
unauthorized, stale, or not-ready outcome must remain visible. Do not switch
providers, upstreams, Releases, discovery owners, or cached route state.

## Drain And Deletion

Begin drain only when the provider advertises drain capability. Preserve
existing streaming and cancellation affinity for the adapter's documented
grace period, stop new traffic, observe the drained state, and then delete the
exact route revision. A missing route is not equivalent to successful drain.

## Backend Outage And Recovery

Provider management availability and Agent data-plane readiness are separate.
During a backend outage, expose the provider's typed state and leave the exact
Release and discovery owner unchanged. Recovery consists of reconciling the
same desired route and observing its readiness evidence; no automatic retry,
alternate endpoint, or fallback provider is permitted.
