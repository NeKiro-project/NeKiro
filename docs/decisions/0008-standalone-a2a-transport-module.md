# ADR 0008: Standalone A2A Transport Module

- Status: Accepted for Spec 029 implementation
- Date: 2026-07-30
- Decision owners: A2A Router and Platform Architecture

## Context

NeKiro's Router currently contains a private package that combines reusable
HTTP/JSON-RPC/SSE client mechanics with NeKiro-specific target resolution,
signed credentials, platform context, active Profile mapping, Platform Error
codes, result events, cancellation policy, and Ledger orchestration.

The platform already depends on the protocol-focused
`github.com/a2aproject/a2a-go v0.3.15`. Reimplementing the complete A2A protocol
would contradict ADR 0001. Keeping all hardening private, however, prevents
independent reuse and couples its release cadence to the entire platform.

## Decision

- `github.com/NeKiro-project/nekiro-a2a-transport-go` becomes the standalone
  upstream for reusable A2A HTTP/JSON-RPC/SSE client mechanics.
- The module depends only on Go standard packages and the A2A protocol library
  pinned by its declared compatibility identity.
- Public calls require an explicit endpoint and explicit positive byte bounds.
  Caller metadata is supplied only through explicit interceptors.
- The module rejects redirects, malformed or mismatched JSON-RPC envelopes,
  invalid media/framing, duplicate SSE IDs/members, trailing data, unsupported
  event kinds, and responses/events above explicit limits.
- Streaming returns official decoded events and immutable raw result bytes so
  callers do not need to reserialize Agent data.
- Transport failures use a transport-only typed taxonomy. NeKiro maps them to
  Platform Error v4 in its Router adapter.
- NeKiro contracts remain the source of truth for A2A Profile, Task-state and
  Message validation, context headers, credential claims, public errors, and
  result events.
- NeKiro retains routing, exact resolution, Card limits, credentials,
  cancellation policy, platform event mapping, Ledger writes, and terminal
  races.
- Upstream releases use immutable semantic-version tags. NeKiro production
  manifests must not contain a local replacement or a fallback copy.

## Consequences

- External Go consumers can reuse the hardened transport without importing the
  NeKiro platform.
- NeKiro gains a second repository and explicit upstream-before-downstream
  release ordering.
- Transport API changes require upstream compatibility review and downstream
  conformance verification.
- The language-neutral A2A Profile remains in one authoritative location, so
  upstream tests must test transport mechanics without redefining platform
  Task/permission/Ledger semantics.
- Source provenance and Apache-2.0 attribution are preserved.

## Rejected Alternatives

### Move the complete Router upstream

Rejected because routing, resolution, authorization, credentials, result
mapping, and Ledger behavior are platform responsibilities.

### Make upstream import NeKiro contracts

Rejected because it reverses the dependency direction and prevents standalone
use.

### Fork or replace the official A2A library

Rejected because the existing Profile already selects a protocol-focused
upstream and the reusable value here is strict transport policy around it.

### Keep a private implementation as fallback

Rejected because dual production paths can drift and hide dependency or
compatibility failures.

## Fallback Delta

Fallback delta: removed 0, retained 2, added 0, net 0.

Retained policies are Go's documented nil `http.Transport` behavior and ADR
0006's one bounded remote cancellation propagation attempt. Added fallback
evidence: none.
