# Trusted Publication v1

Trusted Publication v1 adds a Registry-owned proof step before an endpoint can
participate in a verified Agent Release. It does not deploy the Agent and it
does not permit Console or client applications to call the Agent directly.

## Provider flow

1. Register Agent Card v0.2 through the existing Catalog API.
2. Create an endpoint binding with the authenticated provider ID, Agent ID,
   exact registered Card version, endpoint, and method `http_well_known`.
3. Create a single-use challenge. The response returns the proof exactly once.
4. Serve that exact proof at the returned
   `/.well-known/nekiro/challenges/{challengeId}` URL.
5. Complete the challenge. A successful response exposes the binding status
   and SHA-256 evidence digest, never the proof.

Creating a verified endpoint binding does not publish an Agent Release. The
immutable release state machine is delivered in Spec 023 Slice B / Issue #49.

## Network policy

Verification accepts only `http` or `https` endpoints without credentials,
query strings, or fragments. Redirects are rejected. DNS is resolved once,
every returned address is checked, and the request is pinned to an approved
address to prevent a second resolution from bypassing the policy.

Loopback, private, link-local, multicast, and unspecified addresses are denied
unless their hostname appears in the explicit
`NEKIRO_ENDPOINT_ALLOWED_PRIVATE_HOSTS_JSON` configuration. The allowlist has
no implicit localhost or development value.

The following settings are required:

- `NEKIRO_ENDPOINT_CHALLENGE_TTL_SECONDS`
- `NEKIRO_ENDPOINT_VERIFICATION_TIMEOUT_MS`
- `NEKIRO_ENDPOINT_ALLOWED_PRIVATE_HOSTS_JSON` (use `[]` to deny all private
  hosts)

The verification timeout must be shorter than the challenge TTL.

## Failure semantics

Failure responses use the trusted-publication error contract. Invalid endpoint,
wrong proof, and redirect are distinct typed failures; disallowed network,
unknown resources, challenge expiry/reuse, endpoint unavailability, and
dependency failure each retain their own public code. The exact codes are
`INVALID_ENDPOINT`, `DISALLOWED_NETWORK`, `ENDPOINT_UNAVAILABLE`,
`WRONG_PROOF`, `CHALLENGE_EXPIRED`, `CHALLENGE_REUSED`,
`REDIRECT_NOT_ALLOWED`, and `DEPENDENCY_ERROR`.

Failure responses include the platform trace ID but never proof values,
dependency details, tokens, or endpoint credentials.

## Source contracts

- `contracts/schemas/trusted-publication.v1.schema.json`
- `contracts/openapi/trusted-publication.v1.yaml`
