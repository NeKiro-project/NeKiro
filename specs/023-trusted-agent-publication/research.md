# Research: Trusted Agent Publication

## Existing repository facts

- `apps/control-plane/internal/catalog` is the Registry and owns Agent Card
  persistence, publication status, and discovery projection.
- Agent Card v0.2 currently stores a display owner, endpoint, and declared
  authentication type, but no provider identity or verification evidence.
- Workspace installation already selects a published exact version through a
  Catalog port; it is the correct seam for the later verified-release gate.
- Router transport currently accepts only `authentication.type = none` and
  carries platform trace headers. Router-to-Agent credentials belong in the
  transport boundary, not in Registry or Agent Card.
- The repository's configuration loaders reject missing security values and
  do not permit localhost fallbacks, so endpoint verification must follow the
  same explicit-policy approach.

## Decision

Start with an HTTP well-known challenge and a Registry-owned endpoint binding.
Use an injected network policy and HTTP client so SSRF checks and redirect
rejection are testable. Defer DNS/TXT and organization attestation until the
first method has a stable contract.

## Rejected alternatives

- Treating `owner.id` as provider proof: it is display metadata and is not
  evidence of endpoint control.
- Accepting any successful HTTP response: it would let a third party publish a
  URL it does not control.
- Fetching localhost/private addresses by default: it creates an SSRF boundary
  and violates the explicit security configuration policy.
- Storing a provider secret in Agent Card or Ledger: it leaks credentials into
  platform discovery and audit data.
