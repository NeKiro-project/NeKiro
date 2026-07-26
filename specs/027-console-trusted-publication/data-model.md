# Data Model: Console Trusted Publication Loop

The platform remains the source of truth. These are Console view models and
the backend acceptance facts they consume; they are not new database tables.

## Console View Models

### EndpointBinding

| Field | Meaning | Validation |
| --- | --- | --- |
| `bindingId` | Immutable binding identity | Trusted Publication identifier |
| `providerId` | Provider principal owning the binding | Must match configured provider for provider operations |
| `agentId` | Registered Agent identity | Must match selected Card |
| `agentCardVersion` | Exact Card version | Active semver |
| `endpoint` | Canonical endpoint | HTTPS/HTTP URI returned by Gateway; never rewritten by UI |
| `verificationMethod` | Trust method | `http_well_known` |
| `verificationStatus` | `pending`, `verified`, `failed`, or `revoked` | Enum; no guessed state |
| `verificationFailureCode` | Safe server failure reason | Optional; display only |
| `verificationEvidenceDigest` | Evidence digest | Optional lowercase 64-hex |
| timestamps | Created/updated/verified/revoked facts | Strict date-time parsing |

### VerificationChallenge

| Field | Meaning | Boundary |
| --- | --- | --- |
| `challengeId`, `bindingId` | Challenge identity and association | Must match selected Binding |
| `challengeUrl` | Server-issued verification URL | Display/copy only; no browser fetch to Agent |
| `proof` | One-time proof returned by Gateway | In-memory transient display only; never stored or sent to another Gateway route |
| `expiresAt` | Expiration fact | Strict date-time; expired challenge cannot be reused |

### AgentRelease

| Field | Meaning | Validation |
| --- | --- | --- |
| `releaseId` | Immutable publication identity | Trusted Publication identifier |
| `providerId`, `agentId`, `agentCardVersion` | Release ownership and exact version | Must match selected Card/Binding |
| `cardDigest` | Immutable Card content digest | Lowercase 64-hex |
| `endpointBindingId` | Trust evidence association | Must match selected Binding |
| `endpointOrigin`, `endpointPath` | Snapshot of endpoint provenance | Read-only display |
| `verificationMethod`, `verificationEvidenceDigest` | Trust evidence snapshot | Method enum and optional digest |
| `state` | `draft`, `pending_verification`, `verified`, `published`, `suspended`, `revoked` | Server-defined transition matrix |
| lifecycle timestamps | State transition evidence | Strict date-time |

### Workspace Installation

Existing Installation v2 fields remain authoritative. Console adds the optional
`installedReleaseId` to its mapping and displays it whenever present. A trusted
installation must not be displayed as fully valid when the response has a
Card/version mismatch or an invalid Release identity.

Before submitting an installation, the Console requires an explicit Release ID
handoff and reads that Release through the Gateway. The preflight must return a
published Release whose Agent ID, Card version, and immutable provenance match
the selected Catalog Card. The active Installation request still sends only the
Agent ID, exact version constraint, and accepted permissions; after the request,
`installedReleaseId` must equal the preflight Release ID. The Console does not
infer a Release from Catalog `publicationStatus`.

### Invocation and Trace

Existing Invocation Result v1, Result Stream Event v2, Invocation Event 0.3,
and Gateway v4 metadata fields remain authoritative. The Console keeps:

```text
invocationId
rootTaskId
parentInvocationId
traceId
workspaceId
targetAgentId
agentCardVersion
agentReleaseId / release provenance when returned
status / error code
```

It does not store invocation input/output as Ledger data and does not construct
missing child nodes locally.

## State Transition Ownership

```text
Endpoint Binding:
  pending -> verified | failed | revoked

Release:
  pending_verification -> verified
  verified -> published | suspended | revoked
  published -> suspended | revoked
  suspended -> revoked

Installation:
  enabled <-> disabled
  disabled -> uninstalled
```

The Console renders server responses and only enables actions allowed by the
known state. It never invents a reverse transition or a recovery transition not
defined by the active contract.

## Secret and Persistence Rules

- Bearer configuration is read from explicit environment values and held in the
  provider or owner API client instance; provider and owner tokens are not
  interchangeable, are not exposed as public getters, and are not stored in
  browser storage. The required names are `VITE_NEKIRO_PROVIDER_ID`,
  `VITE_NEKIRO_PROVIDER_TOKEN`, `VITE_NEKIRO_OWNER_TOKEN`, and
  `VITE_NEKIRO_DEFAULT_WORKSPACE_ID`.
- Challenge `proof` is held only in the component state needed to show the
  one-time delivery result and is cleared when the binding flow is closed,
  replaced, or the page unloads.
- No provider, Agent, Router, or application credential appears in Card JSON,
  error objects, Ledger view models, test output, URLs, or logs.
