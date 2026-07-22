# Trusted Publication Contract v1 (Slice A)

This is the contract design input for the implementation. JSON Schema and
OpenAPI files are added with the implementation PR and become the runtime
source of truth.

## Create provider binding

`POST /v4/providers/{providerId}/agents/{agentId}/endpoint-bindings`

Request:

```json
{
  "endpoint": "https://agent.example/a2a",
  "method": "http_well_known",
  "version": "1.0.0"
}
```

Response returns `bindingId`, `agentCardVersion`, canonical endpoint,
`verificationStatus`, and `verificationMethod`; it does not return a secret.

## Create challenge

`POST /v4/providers/{providerId}/endpoint-bindings/{bindingId}/challenges`

Response returns a one-time `challengeId`, `challengeUrl`, `expiresAt`, and
the proof exactly once. The proof is not stored or returned by subsequent
reads.

## Complete challenge

`POST /v4/providers/{providerId}/endpoint-bindings/{bindingId}/challenges/{challengeId}/complete`

The Registry performs the exact declared-origin request and returns the
binding state. Typed public failures include `INVALID_ENDPOINT`,
`DISALLOWED_NETWORK`, `ENDPOINT_UNAVAILABLE`, `WRONG_PROOF`,
`CHALLENGE_EXPIRED`, `CHALLENGE_REUSED`, `REDIRECT_NOT_ALLOWED`, and
`DEPENDENCY_ERROR`. The response also includes the exact `agentCardVersion`.

## Read binding

`GET /v4/providers/{providerId}/endpoint-bindings/{bindingId}`

Returns provider, Agent identity, canonical endpoint, method, state, and
timestamps only. Challenge proof and dependency details are omitted.
