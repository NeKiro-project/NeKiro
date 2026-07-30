# Research: Extract Reusable A2A Transport

## Decision 1: Extract protocol mechanics, not Router policy

**Decision**: Move HTTP client cloning, redirect rejection, official A2A client
construction, strict JSON-RPC envelope validation, bounded body reads, strict
SSE framing, raw stream-result capture, and transport error classification.
Keep target resolution, active Profile validation, credentials, platform
headers, result mapping, cancellation policy, and Ledger behavior in NeKiro.

**Rationale**: The current package mixes reusable wire mechanics with
`contracts.DispatchInvocationRequestV4`, `ResolveAgentResponse`, Router signed
credential context, Platform Error codes, and internal stream events. Moving it
verbatim would make the upstream module depend on the platform consuming it.

**Alternatives considered**:

- Move the complete Router package: rejected because routing, authorization,
  provenance, Ledger, and service assembly are NeKiro-owned.
- Keep all code private: rejected because it preserves duplication risk and
  prevents independent protocol reuse.
- Reimplement the full A2A protocol: rejected because ADR 0001 selects the
  official protocol-focused library.

## Decision 2: Use an explicit call-options boundary

**Decision**: Every operation receives an endpoint, a positive response bound,
and, for streaming, a positive event bound. Request metadata is supplied only
through explicit official client interceptors.

**Rationale**: Per-call options let the Router pass the minimum of required
deployment limits and exact Card limits without exporting Card or platform
types. Required limits prevent unbounded reads and avoid invented defaults.

**Alternatives considered**:

- Global default limits: rejected because existing policy requires explicit
  deployment values and prohibits inferred limits.
- Platform-specific Target type upstream: rejected because Agent ID, Release,
  digest, capability, and audience are not transport facts.
- Raw header map owned by the library: rejected because credential issuance and
  trusted lineage must remain caller-owned and operation-scoped.

## Decision 3: Preserve raw stream results alongside official events

**Decision**: A streaming item contains the official decoded A2A event plus an
immutable copy of the raw JSON-RPC `result` bytes that produced it.

**Rationale**: NeKiro forwards transient result data without reserializing it.
The official decoder yields typed events but not the exact result bytes, while
the strict streaming body already observes each complete envelope.

**Alternatives considered**:

- Re-marshal decoded events: rejected because it changes caller bytes and can
  change number/text representation.
- Expose the full raw SSE frame: rejected because downstream requires the
  protocol result, not transport framing.
- Keep raw extraction downstream: rejected because it requires access to
  private upstream body state and duplicates the framing implementation.

## Decision 4: Use a transport-only failure taxonomy

**Decision**: Expose `invalid_argument`, `protocol`, `remote_agent`,
`unavailable`, `deadline_exceeded`, `canceled`, and `response_too_large` kinds.
The error string contains only the kind, while `Unwrap` preserves the original
cause for programmatic inspection.

**Rationale**: External consumers need stable transport distinctions, but
Platform Error v4 belongs to NeKiro. Hiding cause text from `Error()` reduces
accidental leakage while retaining `errors.Is`/`errors.As` behavior.

**Alternatives considered**:

- Export Platform Error codes: rejected as reverse platform coupling.
- Return raw errors only: rejected because current Router behavior depends on
  stable cancellation, timeout, remote, unavailable, overflow, and protocol
  distinctions.
- Convert all failures to protocol failure: rejected because it collapses
  operationally distinct states.

## Decision 5: Keep the language-neutral Profile in NeKiro contracts

**Decision**: The upstream module declares compatibility with protocol 0.3.0
and `a2a-go v0.3.15`, but NeKiro's JSON Profile, conformance corpus, Task-state
mapping, and Platform Error mapping remain the contract source of truth.

**Rationale**: The constitution fixes `contracts/` as the language-neutral fact
source. Moving or copying those artifacts would create a second authority.

**Alternatives considered**:

- Copy Profile files upstream: rejected because two hand-maintained facts can
  drift.
- Make upstream depend on NeKiro contracts: rejected because it defeats module
  independence and can create a dependency cycle.
- Move all contracts upstream: rejected as a broader contract-ownership change
  absent from the request.

## Decision 6: Release upstream v0.1.x before downstream finalization

**Decision**: The first public API remains pre-1.0 while its consumer surface
is proven. Merge and tag upstream first, add the official-server conformance
patch as `v0.1.1`, then commit NeKiro's immutable dependency without a local
replacement.

**Rationale**: The target repository contains only its license, so no prior API
compatibility exists. A real tag is necessary to prove external consumption.

**Alternatives considered**:

- Commit a permanent local replacement: rejected because CI and other
  consumers cannot reproduce it.
- Use a pseudo-version from an unmerged branch: rejected because it weakens
  review/release ownership.
- Publish v1.0.0 immediately: rejected because the newly extracted API has not
  yet completed an independent release cycle.

## Decision 7: Preserve source provenance

**Decision**: Retain Apache-2.0 in both repositories and add a NOTICE identifying
that the initial transport implementation was extracted from NeKiro.

**Rationale**: Both repositories use Apache-2.0, and explicit provenance makes
the source move auditable.

**Alternatives considered**:

- Omit attribution because ownership is shared: rejected because a source move
  should remain traceable even within one organization.

## Decision 8: Fallback audit remains zero-addition

**Decision**: Keep only Go's documented nil-Transport behavior and NeKiro's
one-shot bounded cancellation propagation. Remove the development replacement
before commit and add no retry, alternate route, alternate credential, legacy
decoder, truncation, default limit, or degraded result.

**Rationale**: ADRs 0006/0007 and Specs 016/017 are explicit policy evidence.

**Alternatives considered**:

- Add retries for package consumers: rejected because operation semantics and
  idempotency policy belong to callers.
- Fall back to the former private implementation: rejected because it would
  preserve two production paths and hide dependency failure.
