# Research: Public Agent Share URL

## Decision 1: Separate opaque public identity

**Decision**: Store one immutable `public_agent_id` per Catalog Agent identity, formatted `agt_` plus 32 lowercase hexadecimal characters.

**Rationale**: The Agent Card `agentId`, trusted Release ID, endpoint, and Workspace Installation identify different domain objects. A generated platform identifier remains stable when versions, endpoints, names, or runtimes change and cannot be confused with an A2A destination.

**Alternatives considered**:

- Reuse Agent Card `agentId`: rejected because it is provider-selected and collapses platform public identity into the Card identity.
- Mutable slug: rejected because rename and collision policy would make shared links unstable.
- Release ID as URL identity: rejected because the confirmed product requirement is a stable Agent-level URL with explicit Release selection.

## Decision 2: Migration and immutability

**Decision**: Catalog schema v5 adds a non-null unique column, deterministically backfills existing rows during the one-time migration, and installs a trigger that rejects later public-ID changes. New identities use 128 bits from `crypto/rand`.

**Rationale**: Existing registered Agents must receive stable URLs after upgrade. Database uniqueness and immutability make the Registry the fact source even under concurrency or process restart.

**Alternatives considered**:

- Compute the ID on each read: rejected because changing algorithms would mutate public links and collisions would not be transactionally owned.
- Maintain a second share table: rejected because one-to-one identity data belongs on `agent_identities` and a second table adds unnecessary lifecycle.

## Decision 3: Explicit public origin

**Decision**: Control Plane and Console each require one exact public Agent origin. Production operations require HTTPS; explicitly configured local/test HTTP origins remain environment-scoped test inputs.

**Rationale**: An absolute canonical URL is required after registration. Inferring from request headers would trust proxies or callers and would create multiple canonical identities. No localhost, alternate origin, or default is supplied.

**Alternatives considered**:

- Derive from `Host`/forwarded headers: rejected as caller-controlled identity and routing inference.
- Return only a relative path: rejected because the provider asked for a shareable NeKiro URL.

## Decision 4: Anonymous safe projection

**Decision**: `GET /v4/public/agents/{publicAgentId}` is anonymous and read-only. Its identity envelope returns only the public identity, canonical URL, registration time, availability, and zero to 100 exact published trusted Releases. Every Card-derived display, owner, capability, and permission field is scoped to an eligible Release item.

**Rationale**: B must be able to review a shared Agent before authentication, but endpoint bindings, evidence, credentials, draft Cards, and Ledger data are unnecessary and sensitive. An identity with no eligible Release must not reveal the newest draft Card merely to populate a page header.

**Alternatives considered**:

- Return the full Agent Card: rejected because `protocol.endpoint` would expose the runtime destination.
- Reuse provider Release read: rejected because it includes endpoint/binding/evidence fields and enforces provider authentication.
- Return no data until authentication: rejected because the requested artifact is public sharing.

## Decision 5: Explicit Release selection

**Decision**: The response lists eligible Releases and the UI leaves the selection empty until B explicitly chooses one.

**Rationale**: The user confirmed explicit selection. Display ordering is not an installation policy. Exact version plus post-install Release equality preserves immutable provenance.

**Alternatives considered**:

- Latest semantic version or most recent publication: rejected because no latest policy exists and automatic selection would be a fallback.
- Immutable Release-pinned URL: deferred as an additive future feature.

## Decision 6: Reuse Installation and invocation boundaries

**Decision**: The share flow submits the existing exact-version Installation request and validates the returned `installedReleaseId`; all invocation and Ledger APIs remain unchanged.

**Rationale**: Installation already owns Workspace authorization, permission acceptance, exact version selection, lifecycle revalidation, and conflict semantics. A parallel share-install state machine would violate data ownership.

**Alternatives considered**:

- A public endpoint that installs directly: rejected because anonymous resolution must be non-authorizing.
- A share-specific Installation table: rejected because origin of navigation is not a durable Installation fact.

## Fallback Audit

| Surface | Classification | Decision |
| --- | --- | --- |
| Missing public origin | Remove/forbid | Fail configuration; no inferred Host or localhost |
| Unknown/malformed public ID | Remove/forbid | Distinct `NOT_FOUND` / `VALIDATION_ERROR` |
| No published Release | Explicit business state | `not_installable` with an empty Release list |
| Multiple Releases | Needs policy resolved | B selects exactly one; no preselection |
| Selected Release changed | Remove/forbid | Installation revalidation fails; no substitution |
| Public API dependency failure | Remove/forbid | Propagate `DEPENDENCY_ERROR`; no empty success |

Fallback delta: removed 0, retained 0, added 0, net 0. Added fallback evidence: none.
