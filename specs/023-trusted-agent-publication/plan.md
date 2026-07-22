# Implementation plan: Trusted Agent Publication

## Constitution Check

- Phase 1 Loop: this feature gates publication and exact installation, making
  the existing Register -> Discover -> Install -> Invoke -> Record loop
  trustworthy.
- Runtime agnostic: provider/release facts and endpoint proof are Control
  Plane concerns; no Agent Runtime framework is added.
- Ownership: Catalog/Registry owns Provider, Challenge, Binding, and Release;
  Workspace owns Installation; Router owns transport and Ledger facts.
- Contracts first: versioned JSON Schema/OpenAPI and Go mappings precede
  handlers and persistence.
- Failure/secret safety: invalid network targets, wrong proofs, dependency
  failures, and expired challenges remain distinct; no credential is returned
  or logged.

## Delivery slices

### Slice A — Provider identity and endpoint binding (#48)

1. Define Provider, Endpoint Binding, Challenge, and verification result
   contracts.
2. Add Registry-owned persistence and migrations without changing historical
   Agent Card payloads.
3. Add explicit endpoint parsing and network policy validation. The validator
   rejects credentials, fragments, unsupported schemes, redirects, and
   disallowed IP ranges; it performs no network fallback.
4. Add Gateway operations to create/complete a challenge and inspect a
   binding, with provider ownership authorization.
5. Add focused unit, contract, and integration tests for success and negative
   paths. Existing v3 routes remain compatible.

### Slice B — Immutable release lifecycle (#49)

1. Add release digest/binding identity and publication state transitions.
2. Make Workspace exact-version resolution require a verified published
   release.
3. Add migration, contract, and lifecycle tests.

### Slice C — Router-to-Agent trust (#50)

1. Define the Router credential profile and key configuration contract.
2. Sign short-lived credentials at Router transport boundary and validate them
   in the sample/runtime adapters.
3. Add forged, expired, wrong-audience, direct-request, JSON/SSE, and nested
   invocation tests.

### Slice D — Client SDK (#51)

1. Add a lightweight Go Client SDK for Gateway invocations.
2. Keep endpoint/version/router/Agent credentials out of its public request
   model.
3. Add contract tests and application example.

### Slice E — Acceptance and operations (#52)

1. Run a clean Register -> Verify -> Publish -> Install -> Invoke -> Record
   flow and all required negative paths.
2. Add operator/provider runbook for verification, suspension, revocation,
   evidence inspection, and recovery ownership.

## Slice A technical design

The first implementation uses a Registry-owned challenge service with an
injected HTTP doer so tests can prove network policy without depending on
external hosts. A challenge request is persisted as a hash and expiry; the
raw proof is returned only once to the authenticated provider. Completion
requires an exact response from the declared endpoint and an unchanged
binding. Redirects are disabled at the HTTP client boundary. DNS resolution
must be checked against the explicit network policy before the request is
sent. Challenge material is never included in response, logs, or stored Card
data.

## Compatibility and migration

- Agent Card v0.2 remains unchanged in this slice.
- Existing published sample Cards remain readable. They remain explicitly
  `legacy/unverified` until a provider completes the new proof flow; no code
  silently upgrades them to trusted.
- New fields are additive in a new `trusted-publication` contract version.
- Catalog schema version advances only after migration and readiness checks are
  updated together.

## Verification commands

```text
go test ./contracts/...
go test ./apps/control-plane/...
go vet ./apps/control-plane/...
go test -tags=integration ./apps/control-plane/internal/catalog/postgres
```

Fallback delta for Slice A: removed 0, retained 0, added 0, net 0. No
fallback is needed to complete a verification attempt; every failure remains
visible and typed.
