# ADR 0015: Nacos Naming Observation Boundary

- Status: Accepted as the issue #89 observation adaptation boundary
- Date: 2026-08-09
- Decision owner: Platform Architecture
- Related issue: [#89](https://github.com/NeKiro-project/NeKiro/issues/89)
- Extends: [ADR 0013](0013-nacos-runtime-provider-slice.md)

## Context

Nacos Naming HTTP list provides snapshots but no reliable watch. Nacos 2.x
Naming subscription uses a gRPC push connection. The general-purpose Go SDK
adds a local ServiceInfo cache, background refresh, connection recovery, and
redo-subscribe behavior. Those policies violate Core's fallback budget and
cannot be injected into the directory unchanged. Periodic HTTP list requests
would be polling rather than Naming observation.

## Decision

1. The Nacos directory continues to use the exact HTTP list endpoint for its
   optional snapshot-only mode.
2. Observe is advertised only when an explicit Naming subscription executor
   and bounded pending-change capacity are supplied.
3. The executor must declare v1 guarantees: atomic initial/push handoff, one
   subscribe attempt, and no retry, reconnect, response cache, failover,
   implicit polling, or hidden reauthentication. Construction rejects an
   executor that cannot make every declaration.
4. Core constructs the exact namespace, service, group, and cluster request.
   The executor returns the initial raw Nacos ServiceInfo together with an
   already-established push stream. Core owns bounded payload validation,
   ServiceInfo decoding, identity validation, topology deltas, local revision
   order, delivery queueing, cancellation, and terminal outcomes.
5. A push carries one complete ServiceInfo. Unchanged public topology is
   suppressed. A changed instance set emits `instances_changed`; an empty host
   list is an empty bound target, not target deletion.
6. Invalid or oversized push data, delivery overflow, stream EOF, or transport
   failure terminates the observation. There is no resubscription or snapshot
   fallback.
7. This decision does not approve the stock Nacos SDK client. Core provides a
   minimal executor over the official published protobuf API. It performs one
   explicit TCP connection, ServerCheck, push-stream setup, and Subscribe
   request; it ACKs client detection and Naming pushes and refuses every
   reconnect attempt. Router composition is explicit and disabled unless all
   Naming gRPC observation bootstrap fields are present.

## Consequences

- The provider adaptation can pass shared Registry conformance using the same
  raw initial/push boundary a production executor must implement.
- Transport recovery policy cannot leak into Registry through a generic SDK
  callback.
- The executor is verified against Nacos 2.5.1 as well as an in-process
  protocol fixture. Existing Router deployments remain snapshot-only; an
  explicitly enabled deployment advertises Naming observation.

## Fallback Delta

Fallback delta: removed 0, retained 0, added 0, net +0.

Added fallback evidence: none.
