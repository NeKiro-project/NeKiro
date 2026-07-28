# Console to Gateway Contract Mapping

This is a consumer mapping over existing versioned contracts. It does not
introduce a new wire contract.

| Console operation | Gateway route | Request | Success shape | Failure contract |
| --- | --- | --- | --- | --- |
| Register Card | `POST /v3/agents` | Card 0.2 wrapper | Catalog entry | Catalog/Northbound error |
| Discover | `GET /v3/agents` | query/capability/owner/limit/cursor | Catalog search | Catalog/Northbound error |
| Create Binding | `POST /v4/providers/{providerId}/agents/{agentId}/endpoint-bindings` | `{version, endpoint, method}` | Trusted Publication `endpointBinding` | Trusted Publication v1 error |
| Read Binding | `GET /v4/providers/{providerId}/endpoint-bindings/{bindingId}` | path only | `endpointBinding` | Trusted Publication v1 error |
| Issue Challenge | `POST /v4/providers/{providerId}/endpoint-bindings/{bindingId}/challenges` | no body | `challenge` | Trusted Publication v1 error |
| Complete Challenge | `POST /v4/providers/{providerId}/endpoint-bindings/{bindingId}/challenges/{challengeId}/complete` | no body | `endpointBinding` | Trusted Publication v1 error |
| Create Release | `POST /v4/providers/{providerId}/agents/{agentId}/releases` | `{version, endpointBindingId}` | `agentRelease` | Trusted Publication v1 error |
| Read Release | `GET /v4/releases/{releaseId}` | path only | `agentRelease` | Trusted Publication v1 error |
| Verify/Publish/Suspend/Revoke | `POST /v4/releases/{releaseId}/{action}` | no body | `agentRelease` | Trusted Publication v1 error |
| Installation Release preflight | `GET /v4/releases/{releaseId}` | path only; explicit handoff ID | `agentRelease` with `state=published` and selected Card identity | Trusted Publication v1 error or client identity mismatch |
| Install | `POST /v3/workspaces/{workspaceId}/installations` | Installation v2 request | Installation v2 | Platform Error v3/v4 as active route defines |
| Invoke JSON/SSE | `POST /v4/workspaces/{workspaceId}/invocations` | Invocation v4 request | Result v1 / Stream Event v2 | Platform Error v4 |
| Read Invocation/Trace | `GET /v4/workspaces/{workspaceId}/invocations/{id}` / `traces/{id}` | path only | Invocation/Trace metadata | Platform Error v4 |

## Validation Rules

- All success payloads reject unknown fields and missing required fields.
- Path identifiers are encoded and validated against the active identifier
  grammar before a request is made.
- The error-body `traceId` is authoritative. If `x-nek-trace-id` is present,
  it must equal the error-body `traceId`; absence remains valid because the
  active OpenAPI does not require the header.
- Mutating operations do not retry, redirect, switch origins, or send challenge
  proof in a request body.
- Catalog search does not expose Release IDs and the active Installation request
  accepts only an Agent/version constraint. The Console therefore requires an
  explicit Release-ID handoff, reads it before installation, submits the exact
  returned Card version as the constraint, and rejects a returned
  `installedReleaseId` that differs from the preflight ID.
- The active Card, Release, Installation, Invocation, and Trace IDs must agree
  across related views before the UI renders a trusted success state.
