# ADR 0013: Nacos Runtime Provider Slice

- Status: Accepted for issues #88 and #89 Nacos runtime slice
- Date: 2026-08-08
- Decision owner: Platform Architecture
- Related issues: [#88](https://github.com/NeKiro-project/NeKiro/issues/88), [#89](https://github.com/NeKiro-project/NeKiro/issues/89)

## Context

The File-backed Router integration proves the application boundary but does not
prove a production middleware path. Nacos can own opaque configuration bytes
and ephemeral instance topology, but it cannot become an authority for Agent
Cards, Releases, Workspace authorization, credential audience, or invocation
policy.

Nacos clients commonly add local snapshots, multiple-server failover, retry,
and automatic listener recovery. Those behaviors conflict with Core's zero
fallback policy unless separately approved.

## Decision

1. Add a snapshot-only `config_center/nacos` reader for one explicit API origin,
   namespace, group, and injected request executor. It performs exactly one
   bounded `GET /nacos/v1/cs/configs` request. It has no cache, server list,
   retry, watch, resubscription, publisher, or inferred authentication.
2. Add a snapshot-only `registry/nacos` directory for one explicit API origin,
   namespace, exact binding source, and injected request executor. It performs
   exactly one bounded `GET /nacos/v1/ns/instance/list` request per snapshot.
3. Add `router-nacos-instance-bindings.v1`. Each binding maps one exact
   `ReleaseTarget` to one explicit Nacos group, service, and cluster. The
   Router owns this typed document; Config Center continues to expose only
   opaque bytes.
4. The Router reads the binding document and naming snapshot once per
   Invocation. Only healthy and enabled Nacos instances are ready. Disabled or
   unhealthy instances remain explicit unavailable instances. Nacos does not
   imply a draining state. Runtime identity is the explicit
   `nekiro.instanceId` Naming metadata value; the provider-generated Nacos
   instance key is not treated as an application identity, and a missing or
   invalid metadata identity fails the snapshot.
5. Runtime B owns its optional Nacos registration lifecycle. Registration,
   heartbeat, and deregistration use one explicit service tuple and advertised
   IP. A failed initial registration fails startup; a failed heartbeat makes
   the sample not-ready and terminates serving. No alternate service or Nacos
   origin is selected.
6. Stack owns only exact-revision assembly, Nacos deployment, and publication
   of the exact Release-to-service binding after Catalog publication. It does
   not copy component source or publish instance addresses.
7. Authentication mode is explicit `none` or `access_token`. Tokens are
   bootstrap secrets and are never returned in errors, status, logs, contracts,
   or Ledger facts.
8. Observe, multiple Nacos servers, provider failover, load-balancing,
   registration federation, and external Gateway consumption remain outside
   this slice. ADR 0014 subsequently defines Config Center observation without
   listener recovery; Naming observation remains outside this decision.

## Consequences

- A live Stack can prove Router-owned Nacos discovery while Runtime B owns its
  ephemeral registration.
- Snapshot failure is visible on the affected Invocation and never falls back
  to File, direct routing, a cached response, or another endpoint.
- This slice's Nacos adapters advertise only snapshot capability. ADR 0014
  subsequently adds explicit Config Center watch capability for #88; Naming
  watch conformance remains separately visible in #89.

## Fallback Delta

Fallback delta: removed 0, retained 0, added 0, net +0.

Added fallback evidence: none.
