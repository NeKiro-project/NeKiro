# ADR 0016: Nacos Runtime Registration Lease

- Status: Accepted for issue #105 first registration slice
- Date: 2026-08-10
- Decision owner: Platform Architecture
- Related issues: #89, #105
- Extends: ADR 0013 and ADR 0015

## Context

The Nacos Directory can now observe an exact authorized Release continuously,
but Core did not expose a provider-neutral registration lifecycle. Sample
Runtime code therefore had to own Nacos wire details directly, making the
Register -> Discover boundary difficult to reuse and verify consistently.

Registration is not a Directory write operation. Directory consumers are
Router and Gateway selection paths; registration authority belongs to the
Runtime that owns the advertised process and endpoint.

## Decision

1. Add a separate `registry.InstanceRegistrar` capability. It accepts one
   immutable exact `ReleaseTarget` and one immutable ready `Instance`; it does
   not resolve or authorize Releases.
2. `Register` performs one explicit initial provider request and returns only
   after provider acknowledgement. Failure prevents the Runtime from claiming
   a live lease.
3. A returned `InstanceLease` maintains one explicit heartbeat interval.
   Heartbeats never retry, reconnect, switch origin, change binding, or use a
   cached success. The first failure terminates the lease permanently with a
   typed safe outcome.
4. The Runtime must observe `Done` and stop readiness/serving after a terminal
   heartbeat. Core does not silently keep the Runtime available.
5. `Close(context.Context)` stops heartbeat and makes exactly one explicit
   deregistration attempt. Cancellation and deregistration failure remain
   visible; they are not converted into success.
6. The Nacos registrar is configured with one exact binding, namespace,
   advertised port name, positive weight, heartbeat interval, heartbeat
   timeout, IP deletion timeout, API origin, authentication mode, and injected
   request executor. No value is inferred from Directory state or provider
   defaults. Provider attempts to replace the configured heartbeat interval
   terminate the lease.
7. Registration publishes only safe instance metadata plus the exact
   `nekiro.instanceId`. Authentication tokens and provider response bodies are
   never exposed through public errors or status.
8. Runtime process integration stays in the owning Samples repository. Stack
   owns exact-revision acceptance proving registration, discovery, invocation,
   lease termination, and post-removal fail-closed behavior.

## Consequences

- Core provides one reusable registration contract without becoming a Runtime
  supervisor or a second Catalog owner.
- Kubernetes directories remain read-only and do not advertise registration.
- A failed heartbeat is terminal by design; recovery requires a new process or
  an explicitly initiated new registration lifecycle.
- Provider freshness/status projection and cross-repository acceptance remain
  separate deliverables in issue #105.

## Fallback Delta

Fallback delta: removed 0, retained 0, added 0, net +0.

Added fallback evidence: none.
