# ADR 0012: Router Configured Instance Routing

- Status: Accepted for issues #88 and #89 runtime integration slice
- Date: 2026-08-08
- Decision owner: Platform Architecture
- Related issues: [#88](https://github.com/NeKiro-project/NeKiro/issues/88), [#89](https://github.com/NeKiro-project/NeKiro/issues/89)

## Context

The Control Plane resolves one exact authorized Release, while the Instance
Registry reports ephemeral network topology for that Release. The Router must
be able to use the topology without allowing it to replace the Agent identity,
Release, Card digest, canonical endpoint, or credential audience. Streaming
and cancellation also require one stable instance choice for the complete
Invocation.

The initial Stack acceptance environment uses Docker Compose rather than a
Kubernetes API. It therefore needs an explicit deployment-owned topology
document while the Kubernetes adapter remains independently testable.

## Decision

1. Add the versioned `router-instance-directory.v1` configuration document.
   It contains exact Release targets and provider-neutral instance facts. The
   Router owns this typed document; `config_center/` continues to treat it as
   opaque bytes.
2. Router bootstrap explicitly selects `direct` or `config_center_file`.
   There is no inferred provider, alternate source, or fallback between modes.
3. The File-backed directory reads one immutable document snapshot for each
   Invocation. Missing, malformed, unauthorized, unavailable, or unmatched
   state fails that Invocation with dependency semantics.
4. This slice selects only when exactly one ready TCP endpoint exists for the
   configured port name. Zero ready endpoints fail. Multiple ready endpoints
   fail because load-balancing policy is not yet approved.
5. Selection may replace only the network destination. Agent ID, Card version,
   Release ID, Card digest, capability, and canonical credential audience are
   immutable. A selector that changes any of them is rejected.
6. Streaming sends and their bounded cancellation attempt retain the selected
   target captured before the stream begins. The directory is not read again
   during the Invocation.
7. File mode starts only with a readable, valid document. Runtime readiness
   revalidates the current document and exposes only `ok` or `not_ready`.
8. Invalid updates and deletion fail closed. The Router does not retain a
   last-known-good document, retry another source, or return to direct routing.

## Consequences

- Stack can prove runtime instance routing with an exact Release and a second
  sample replica while preserving Router credential validation.
- Multi-instance selection, weighting, zones, retries, and failover remain
  outside this slice and require an explicit policy decision.
- The configured directory is suitable for controlled local and Stack
  acceptance. Production Kubernetes composition still owns API credentials,
  bindings, watches, and recovery policy.

## Fallback Delta

Fallback delta: removed 0, retained 0, added 0, net +0.

Added fallback evidence: none.
