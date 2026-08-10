# ADR 0017: Router Topology Status Projection

- Status: Accepted for issues #88, #89, #90, and #105
- Date: 2026-08-10
- Decision owner: Platform Architecture
- Extends: ADR 0015 and ADR 0016

## Context

The Router continuously observes Nacos topology for each exact Release used by
an Invocation. Stack acceptance can stop a Provider lease and observe a later
`AGENT_UNAVAILABLE`, but that public failure does not prove which internal
state caused it. The Router may have consumed an empty topology, or it may have
selected an older endpoint and failed at transport. Both correctly fail closed
at the public boundary, but only the first proves the watch lifecycle required
by #89.

Readiness cannot supply this evidence. It reports whether the selected provider
boundary is usable at process scope, not the state and revision of every exact
Release observation. Querying Nacos from Stack would prove provider state but
not that Router consumed it.

## Decision

1. Add the authenticated Router-owned
   `GET /internal/v1/instance-topology/status` operation and the independently
   versioned `router-topology-status.v1` JSON Schema.
2. The continuous `WatchSelector` owns the in-memory projection. A status read
   copies existing sessions only. It never creates an observation, calls
   `Snapshot` or `Observe`, probes an endpoint, refreshes, retries, reconnects,
   or changes observation state.
3. Each observation exposes only Agent ID, Agent Card version, Release ID,
   `initializing|missing|empty|populated|unavailable`, observation-local
   revision, and the Router-local observation timestamp. The response also
   identifies the one selected provider.
4. Endpoint/address, provider source token, instance ID/metadata, Card digest,
   canonical endpoint, audience, configuration payload, credential, Agent
   input/output, and provider response content are forbidden.
5. `localRevision` is ordered only within one Router observation session. It is
   not a Nacos revision, persistent version, recovery cursor, or ordering value
   across restarts or exact Release targets.
6. `observedAt` is when Router locally accepted the current safe state or made
   the session unavailable. It is freshness evidence for that local projection,
   not Provider health, Nacos server time, lease expiry time, or an SLO claim.
7. A terminal watch failure changes the state to `unavailable`, preserves the
   last accepted local revision, and records the terminal Router observation
   time. Wall-clock timestamps are not ordering evidence; consumers use local
   revision only within the same observation session. Failure does not retain
   a selectable snapshot or create stale success.
8. The route is registered only when the configured selector implements
   continuous observation. Direct and snapshot-only routing do not return an
   empty status as if observation were enabled.
9. The existing Router service Bearer boundary authenticates the operation.
   Responses use `Cache-Control: no-store`; missing and unknown credentials
   remain distinct 401/403 Platform Error v4 outcomes.

## Consequences

- Stack can record a populated revision, stop the exact Provider lease, and
  wait for a newer `empty` revision before asserting fail-closed invocation and
  Ledger behavior.
- Operators can distinguish initializing, absent, empty, populated, and
  terminal observation state without receiving routable topology or secrets.
- The status surface is evidence only. It does not become Registry, health
  discovery, remediation, or a control API.

## Fallback Delta

Fallback delta: removed 0, retained 0, added 0, net +0.

Added fallback evidence: none.
