# Feature Specification: Public Agent Share URL

**Feature Branch**: `codex/028-public-agent-share`

**Created**: 2026-07-30

**Status**: Confirmed; clarification and planning complete; implementation in progress

**Input**: User request: after Agent A is registered, NeKiro assigns it a native public sharing URL so another user can open or paste that URL, review the Agent, install it into an authorized Workspace, and use it through the managed platform path.

## Clarifications

### Session 2026-07-30

- Q: Is the shared value an Agent endpoint or a NeKiro identity? -> A: It is a stable NeKiro-owned public Agent identity represented by a normal HTTPS URL. It never exposes or aliases the Agent's A2A endpoint.
- Q: Does opening a public URL authorize installation or invocation? -> A: No. Public resolution is read-only. Installation requires an authenticated Workspace context, explicit permission acceptance, and an exact trusted Release; invocation remains Gateway -> Router -> Agent.
- Q: Which Release does an Agent-level URL install when more than one trusted Release is published? -> A: The stable Agent URL presents eligible published Releases and B explicitly selects one exact Release. The URL never implies "latest" and never substitutes a Release. A future immutable Release-pinned URL would be an additive feature requiring its own policy.

## User Scenarios & Testing

### User Story 1 - Provider receives a stable NeKiro share identity (Priority: P1)

After registering Agent A, its provider receives a stable public Agent identity and canonical NeKiro HTTPS URL. The URL can be copied and shared independently of Agent versions, Release lifecycle changes, endpoint changes, or the Agent's implementation runtime.

**Why this priority**: A platform-owned identity is the core product value of the feature. Without it, providers must share internal IDs or runtime endpoints and recipients cannot enter through a trustworthy NeKiro boundary.

**Independent Test**: Register one Agent, read its public identity and URL, register later Card versions and change the verified endpoint through the existing trusted publication workflow, then confirm the same public URL still identifies only that Agent.

**Acceptance Scenarios**:

1. **Given** a provider successfully registers a new Agent identity, **when** the registration result is read, **then** NeKiro returns one immutable public Agent ID and one canonical public HTTPS URL for that Agent.
2. **Given** the Agent has multiple Card versions or Releases, **when** the provider reads its public identity again, **then** the public Agent ID and URL remain unchanged.
3. **Given** an Agent display name or endpoint changes through an authorized workflow, **when** an existing share URL is opened, **then** it still resolves the same Agent identity and does not redirect to or expose the runtime endpoint.
4. **Given** a public share URL, **when** it is inspected or copied, **then** it contains no bearer token, challenge proof, Workspace identifier, endpoint credential, invitation secret, or invocation credential.

---

### User Story 2 - Recipient resolves and reviews a shared Agent (Priority: P1)

User B can open the public URL in a browser or paste it into the NeKiro Console. NeKiro resolves the public Agent identity through the Gateway and shows only public, trusted publication information needed to decide whether to install it.

**Why this priority**: The public URL must be useful to a recipient without turning Catalog storage, an Agent endpoint, or an internal service into a second northbound interface.

**Independent Test**: Open and paste the same valid URL without a Workspace credential, verify the public Agent identity and currently installable trusted publication data, and confirm that no installation or invocation occurs.

**Acceptance Scenarios**:

1. **Given** a valid share URL for an Agent with an installable trusted Release, **when** B opens or pastes it, **then** NeKiro shows the Agent's public identity, name, owner attribution, capabilities, declared permissions, Card version, Release identity, and trusted publication provenance required for review.
2. **Given** B is not authenticated, **when** B opens a valid public URL, **then** B may review public Agent information but cannot install or invoke the Agent.
3. **Given** a registered Agent has never published a trusted Release, **when** its URL is opened, **then** NeKiro reports that the Agent is not currently installable and does not expose draft Card data, endpoint data, or verification evidence.
4. **Given** a malformed URL, an unsupported NeKiro host/path, or an unknown public Agent ID, **when** B pastes or opens it, **then** resolution fails with the exact owning-boundary error and does not try another host, ID, endpoint, or legacy format.
5. **Given** a selected Release is suspended, revoked, unpublished, or no longer available, **when** B reviews it, **then** NeKiro reports its exact non-installable lifecycle state and does not substitute another Release.

---

### User Story 3 - Recipient installs the exact trusted Agent Release (Priority: P1)

After reviewing the shared Agent, authenticated user B chooses an authorized Workspace, reviews the exact Release and its declared permissions, explicitly accepts those permissions, and installs that Release. Successful installation makes the Agent available through the existing Workspace invocation path.

**Why this priority**: A share link becomes product value only when it hands off safely to Workspace authorization without weakening exact-Release provenance or permission consent.

**Independent Test**: Starting only from a shared URL and an authenticated Workspace-owner session, install one exact published Release, verify the returned Installation records the same Release ID and accepted permissions, then invoke it through the existing Gateway and Router path.

**Acceptance Scenarios**:

1. **Given** B has resolved an installable shared Agent, **when** B starts installation, **then** NeKiro requires an authenticated principal and an explicitly selected Workspace that B is authorized to manage.
2. **Given** an exact published Release is selected, **when** B reviews installation, **then** NeKiro shows the Release identity, Card version, immutable provenance, and declared permissions before B can confirm.
3. **Given** B explicitly accepts the requested permissions, **when** installation succeeds, **then** the Installation records the exact selected Release and permissions; the client rejects a successful-looking response that names a different or missing Release.
4. **Given** B lacks Workspace authority, rejects a permission, or the Release becomes non-installable before confirmation, **when** installation is submitted, **then** no enabled Installation is created and the exact authorization, permission, or lifecycle error remains visible.
5. **Given** the Installation is enabled, **when** B invokes the Agent, **then** the request follows the existing Gateway -> Router -> Agent flow and produces the existing Invocation Ledger facts; the public URL is never used as a runtime endpoint.

---

### User Story 4 - Provider and operator control public availability (Priority: P2)

Providers and operators can understand why a public Agent is or is not installable. Existing trusted publication lifecycle controls determine installability; the share feature does not create an independent publication state or an automatic recovery path.

**Why this priority**: Public sharing must remain a projection of Registry and trusted Release facts, not a second source of truth that can drift from publication governance.

**Independent Test**: Resolve one Agent URL while its Release moves through published, suspended, and revoked states, and confirm the public result follows the authoritative lifecycle without republishing, retrying, or switching Releases.

**Acceptance Scenarios**:

1. **Given** a published Release is suspended or revoked, **when** its Agent URL is resolved again, **then** the public result reflects the authoritative lifecycle and prevents a new installation of that Release.
2. **Given** publication state changes while B has an installation confirmation open, **when** B submits, **then** the server revalidates the exact Release and fails explicitly if it is no longer installable.
3. **Given** Catalog projection or a dependent read is unavailable, **when** the URL is resolved, **then** NeKiro reports dependency failure and does not return stale, empty, or fabricated installable data.

### Edge Cases

- Two Agents have the same display name or owner-selected label; their immutable public Agent IDs and canonical URLs remain distinct.
- Letter case, Unicode, trailing separators, query parameters, fragments, percent encoding, or surrounding whitespace could change URL interpretation; only the canonical documented form is accepted and invalid input is rejected rather than normalized into another identity.
- A public Agent ID is syntactically valid but unknown; the response does not infer another Agent from display name, Card ID, Release ID, or endpoint.
- A URL resolves while a Release lifecycle transition occurs; the installation write revalidates the exact Release instead of trusting the earlier read.
- A provider registers an Agent but never verifies or publishes a Release; the stable identity exists, but no draft or verification detail becomes public and installation remains unavailable.
- The selected Workspace already has the exact Release installed, has another Release installed, or has a disabled Installation; the existing Installation contract returns its precise conflict or transition semantics and the share flow does not invent replacement or upgrade behavior.
- Public metadata contains a credential-shaped value in a Card field or provenance field; the owning boundary rejects or redacts it according to the existing secrecy contract and never emits it through the share response.
- A public share page is indexed, cached, or embedded by an external service; the URL remains non-authorizing and contains no secret, while cache and indexing policy require an explicit deployment decision before implementation.

## Requirements

### Functional Requirements

- **FR-001**: NeKiro MUST allocate exactly one immutable, globally unique public Agent ID when a new Agent identity is successfully registered; later Card versions and Releases MUST reuse it.
- **FR-002**: NeKiro MUST expose one canonical, ordinary HTTPS share URL under an operator-controlled NeKiro public origin. The URL MUST encode only the public Agent identity and routing syntax, and MUST NOT contain credentials, Workspace context, endpoint data, or authorization grants.
- **FR-003**: Public Agent identity, Card version, Release identity, endpoint binding, and Workspace Installation MUST remain distinct domain concepts; the public Agent URL MUST NOT be accepted as an Agent A2A endpoint or invocation target.
- **FR-004**: Frontends and external clients MUST resolve a public Agent URL only through the Gateway's versioned northbound contract. The Gateway MUST obtain authoritative Agent and trusted Release facts through Catalog ownership boundaries.
- **FR-005**: Public resolution MUST expose only explicitly public Agent metadata and trusted, published Release provenance required for installation review. It MUST NOT expose draft Cards, private owner data, raw endpoints, binding challenges, evidence material, credentials, internal service locations, or Ledger data.
- **FR-006**: The public Agent URL MUST present eligible published Releases and require B to explicitly select one exact Release before installation. The implementation MUST NOT infer "latest", substitute another Release, or use a version-only fallback. An immutable Release-pinned URL is not part of this feature.
- **FR-007**: A registered Agent without an eligible published trusted Release MUST have a stable public identity but MUST resolve to an explicit not-installable state without exposing unpublished details.
- **FR-008**: Opening or resolving a share URL MUST be read-only and MUST NOT create an Installation, authorize a Workspace, accept permissions, invoke an Agent, or mutate publication state.
- **FR-009**: Installation from a share URL MUST require B's authenticated identity, an explicitly chosen authorized Workspace, review of the exact Release provenance and declared permissions, and explicit acceptance of the permissions.
- **FR-010**: A successful share-originated Installation MUST persist and return the exact selected published Release identity, installed Card version, accepted permissions, Workspace identity, enabled state, and installation time under the existing Installation ownership model.
- **FR-011**: The installation boundary MUST revalidate Agent identity, exact Release eligibility, Workspace authority, and accepted permissions at write time. A stale public read MUST grant no installation right.
- **FR-012**: Unknown IDs, malformed URLs, unsupported origins or paths, unpublished Releases, suspended Releases, revoked Releases, authorization failures, permission mismatches, conflicts, and dependency failures MUST retain distinct contract-defined errors and MUST NOT become empty results or successful-looking states.
- **FR-013**: URL parsing and resolution MUST NOT trim, redirect, retry, query alternate origins, accept legacy aliases, infer IDs, follow Agent-controlled endpoints, or fall back to Catalog search.
- **FR-014**: Existing invocation semantics MUST remain unchanged: all calls to an installed shared Agent go through Gateway and A2A Router, and all root and nested calls retain existing Task, Trace, parent/child, exact Release, credential, and Ledger behavior.
- **FR-015**: Release publication, suspension, revocation, and future approved lifecycle operations MUST remain the sole authority for public installability; the share feature MUST NOT own a parallel publication state.
- **FR-016**: The feature MUST provide a Console entry path for both opening a canonical share URL and pasting one into an installation surface, with the same strict identity, provenance, authentication, and permission behavior.
- **FR-017**: Contract, integration, and end-to-end acceptance MUST cover stable identity allocation, public resolution, exact Release selection, permission review, authorized installation, invocation through Router, Ledger recording, lifecycle changes, invalid input, unauthorized Workspace access, and secrecy scanning.
- **FR-018**: Documentation MUST distinguish public Agent identity, trusted Release, Workspace Installation, and Agent endpoint, and MUST explain that a share URL is public discovery metadata rather than a secret invitation or direct execution URL.

### Key Entities

- **Public Agent Identity**: The immutable, globally unique public identifier assigned to one registered Agent identity. It is stable across Agent Card versions, Releases, endpoint changes, display-name changes, and Workspace Installations.
- **Canonical Agent Share URL**: The operator-owned HTTPS representation of a Public Agent Identity. It is safe to disclose, non-authorizing, and never an Agent endpoint.
- **Public Agent View**: A read-only projection containing only public Agent metadata and eligible trusted Release provenance needed for review. Registry and trusted publication remain its sources of truth.
- **Trusted Release Candidate**: An exact immutable published Release that the recipient may review and select for installation; lifecycle transitions can make it non-installable before the installation write.
- **Share-Originated Installation**: A normal Workspace Installation whose user journey began from a public URL. It retains existing ownership, exact-Release, permission, enabled-state, and lifecycle semantics rather than becoming a new installation type.

### Runtime/Platform Boundary

- **Platform-owned behavior**: Public identity allocation, canonical URL resolution, public metadata projection, trusted Release eligibility, Workspace authorization, exact-Release installation, Router-mediated invocation, and Ledger lineage are NeKiro platform responsibilities.
- **Runtime-owned behavior**: Agent prompts, models, tools, workflow, memory, sessions, deployment, endpoint implementation, and response generation remain in external Agent Runtimes. A Runtime never creates or interprets NeKiro public Agent URLs.
- **Cross-runtime proof**: Install Runtime A from its NeKiro share URL, invoke Runtime B through its own Installation, and have B call A through Router in one Trace. Neither Runtime receives the public share URL as an endpoint or shares framework-internal types.

## Success Criteria

### Measurable Outcomes

- **SC-001**: 100% of newly registered Agent identities receive exactly one canonical public Agent ID and URL, and repeated reads after new versions, Releases, or endpoint changes return the same values.
- **SC-002**: Starting only from a valid public URL, an authenticated Workspace owner can review public metadata, select one exact trusted Release, accept permissions, and create a matching enabled Installation without entering an Agent ID, Release ID, or endpoint manually.
- **SC-003**: 100% of share-originated successful Installations record the same exact Release identity and permission set shown at final confirmation; mismatched or missing Release identity yields zero successful-looking Installations.
- **SC-004**: Unauthenticated visitors can cause zero Installations and zero Invocations by opening or resolving a public URL.
- **SC-005**: The acceptance matrix distinguishes all specified malformed, unknown, unpublished, suspended, revoked, unauthorized, conflict, and dependency-failure cases, with zero retries, alternate-source lookups, inferred identities, Release substitutions, or fabricated success results.
- **SC-006**: Secrecy scans of canonical URLs, public responses, Console state, logs, tests, and browser history find zero bearer tokens, challenge proofs, endpoint credentials, Router credentials, Workspace identifiers, or invocation credentials introduced by this feature.
- **SC-007**: An Agent installed from a share URL completes the existing managed invocation path and produces queryable exact-Release Invocation Ledger facts; a cross-runtime nested acceptance retains one Trace and the correct parent/child lineage.
- **SC-008**: Suspending or revoking a previously visible Release prevents 100% of new installation attempts for that exact Release without changing the stable public Agent identity or silently selecting another Release.

## Assumptions

- Trusted Publication v1, Agent Card 0.2, Catalog/Northbound v3, Installation v2, Northbound Invocation v4, Router dispatch v4, and existing Ledger metadata contracts remain authoritative until planning identifies an explicitly versioned contract change.
- The canonical URL uses HTTPS under an operator-controlled NeKiro domain. The exact production hostname is a deployment decision, but it has no localhost, inferred, or alternate-origin fallback.
- Existing provider and Workspace authentication/authorization remain in force. A public Agent view is anonymous read metadata only and does not weaken mutation authorization.
- A public Agent identity is allocated after successful registration, but installability begins only after an exact trusted Release is published.
- Existing Installation conflict, disable, uninstall, and invocation behavior is reused; this feature adds a trusted entry and handoff path, not a second Installation state machine.

## Non-Goals

- Treating the public URL as an Agent endpoint, direct invocation URL, bearer invitation, Workspace grant, permission acceptance, or embedded credential.
- Automatic invocation immediately after opening or installing a shared Agent.
- An implicit latest-version policy, automatic upgrade, Release substitution, retry, alternate endpoint/origin, redirect resolver, legacy URL compatibility, or Catalog-search fallback.
- Custom provider vanity domains, mutable slugs as identity, URL shortening, social previews, ranking, recommendations, ratings, billing, certification, or a general Marketplace.
- Agent deployment, runtime health repair, model/tool/workflow/memory behavior, or framework-specific installation.
- New enterprise identity, RBAC, approval, organization federation, or credential issuance behavior.
- Search-engine indexing and cache policy until an explicit privacy, freshness, and invalidation policy is approved.
