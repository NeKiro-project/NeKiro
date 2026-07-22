# ADR 0004: Trusted Agent Publication and Endpoint Ownership

## Status

Accepted for the first Trusted Agent Publication vertical slice.

## Context

A2A standardizes Agent communication but does not prove who controls an
endpoint or whether a published Card still refers to the same release. The
Registry currently stores a display owner and endpoint, which is insufficient
for cross-organization trust.

## Decision

NeKiro introduces Registry-owned Provider, Endpoint Binding, Verification
Challenge, and immutable Agent Release facts. The first endpoint ownership
method is a single-use HTTP well-known challenge over the exact declared
origin. Verification rejects credentials, redirects, and private/loopback/
link-local destinations by default. Any development exception must be an
explicit validated allowlist configuration.

Provider identity is not inferred from `AgentCard.owner.id`. A provider is
created from the authenticated Gateway principal and an Agent identity may be
claimed by only one provider; the first binding establishes that Registry
claim and a different provider receives a conflict. This explicit claim rule
allows the provider principal to differ from the Card display owner. A verified
binding is a prerequisite for the later published-release state; Workspace
resolution and Router authentication consume the exact release and binding
through versioned ports. No secret is stored in Agent Card, public responses,
logs, events, or Ledger.

## Consequences

- Existing Card v0.2 remains compatible and legacy samples remain readable,
  but they are not silently upgraded to trusted releases.
- Registry gains new data ownership and migration responsibility.
- Endpoint verification adds a network security boundary and must be tested
  for SSRF and redirect behavior.
- DNS/TXT, organization attestation, mTLS, and automated deployment remain
  follow-up decisions after this contract is proven.
