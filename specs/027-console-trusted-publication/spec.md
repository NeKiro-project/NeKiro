# Feature Specification: Console Trusted Publication Loop

**Feature Branch**: `codex/027-console-trusted-publication`

**Created**: 2026-07-26

**Status**: Implementation complete; delivery closure pending

**Input**: User request: advance `Agent A -> register -> NeKiro URL -> installer -> B -> Router -> A Agent` and provide a trustworthy Console operations surface, with upstream Issues and independently reviewed PR slices.

## Clarifications

### Session 2026-07-26

- Q: How are provider identity and browser credentials selected? -> A: Use explicit `VITE_NEKIRO_PROVIDER_ID`, `VITE_NEKIRO_PROVIDER_TOKEN`, `VITE_NEKIRO_OWNER_TOKEN`, and `VITE_NEKIRO_DEFAULT_WORKSPACE_ID`; never infer identity from Card owner fields or reuse one token across provider and Workspace-owner operations.
- Q: How does a Workspace owner review an exact Release when Catalog search has no Release ID and Installation accepts only a version constraint? -> A: Require an explicit Release ID handoff in the Console, read it through `GET /v4/releases/{releaseId}`, require `state=published` and exact Agent/Card provenance, install with an exact version constraint, then require the Installation response `installedReleaseId` to match the preflight Release. Do not add a new backend contract in Slice B.

## User Scenarios & Testing

### User Story 1 - Provider registers and proves an Agent endpoint (Priority: P1)

An Agent provider can use the production Console to submit a versioned Agent Card, create an endpoint binding, issue a one-time verification challenge, inspect its current status, and complete verification against the declared endpoint. The Console makes the current state and exact failure category visible so the provider does not need SQL, direct Agent access, or an internal service URL.

**Why this priority**: Trusted publication is the trust boundary that makes an installed Agent safe to invoke. A draft that can be published without endpoint ownership proof is not an acceptable Phase 1 user workflow.

**Independent Test**: Against a fresh platform environment and a reachable sample endpoint, complete registration, binding creation, challenge issuance, challenge completion, and read back a verified binding from the Console.

**Acceptance Scenarios**:

1. **Given** an Agent Card that passes the active Card contract, **when** the provider submits it, **then** the Console shows the server-returned draft with its identity, version, capabilities, and no fabricated publication state.
2. **Given** a registered Card and an explicit endpoint, **when** the provider creates a binding, **then** the Console shows the binding ID, canonical endpoint, verification method, and pending status.
3. **Given** a pending binding, **when** the provider issues a challenge, **then** the Console shows the challenge ID, challenge URL, and expiry, keeps the proof only in transient page state, and does not persist it in browser storage, logs, or unrelated requests.
4. **Given** an eligible pending challenge whose endpoint serves the expected proof, **when** the provider selects verify, **then** the Console refreshes the binding and shows verified status with the evidence digest.
5. **Given** an invalid, expired, reused, disallowed, or unavailable verification attempt, **when** the operation fails, **then** the Console preserves the exact trusted-publication error code, HTTP status, trace ID, and next action without converting the binding to verified.

### User Story 2 - Provider creates and controls an immutable Release (Priority: P1)

After endpoint verification, a provider can create a Release for the exact Agent Card version and binding, read its immutable provenance, publish it, and intentionally suspend or revoke it. The Console presents the allowed state transitions and disables actions that the server would reject.

**Why this priority**: A verified endpoint is evidence for a specific version and endpoint binding; it is not itself a discoverable or installable Release. Explicit Release lifecycle controls make that distinction visible and auditable.

**Independent Test**: Create a Release from a verified binding, verify and publish it, read it back, then suspend and revoke it while observing each returned state and provenance field.

**Acceptance Scenarios**:

1. **Given** a registered Card and pending or verified binding, **when** the provider creates a Release, **then** the Console shows the server-assigned Release ID, Card digest, binding ID, endpoint origin/path, verification method, and state.
2. **Given** a pending Release and a now-verified binding, **when** the provider selects verify, **then** the Release becomes verified only after the server confirms the current binding evidence.
3. **Given** a verified Release, **when** the provider selects publish, **then** discovery exposes the Agent version as available and the Console no longer offers an in-place edit of its immutable provenance.
4. **Given** a verified or published Release, **when** the provider selects suspend or revoke and confirms the destructive action, **then** the Console displays the resulting state and explains that recovery requires a deliberate provider operation; it does not silently republish or unsuspend.
5. **Given** an invalid state transition or a Release owned by another principal, **when** the operation fails, **then** the Console displays the exact server error and leaves the previous state unchanged.

### User Story 3 - Workspace owner installs only trusted Releases (Priority: P1)

A Workspace owner can discover published Agents, review declared permissions and exact Release provenance, install the selected version into the current Workspace, and enable, disable, or uninstall the resulting Installation. The installation view distinguishes the installed Release ID from the Card version.

**Why this priority**: Installation is the authorization handoff from provider trust to Workspace use. The owner must be able to see what exact immutable Release was accepted.

**Independent Test**: Discover a published verified Agent, install it into a fresh Workspace with explicit permissions, read the Installation, disable and re-enable it, and verify that unpublished, suspended, revoked, or untrusted candidates remain unavailable.

**Acceptance Scenarios**:

1. **Given** a published trusted Release ID supplied by the provider or deployment handoff, **when** the owner opens installation, **then** the Console reads the Release through the Gateway, shows its exact Card version, immutable provenance, declared permissions, and the permissions that will be accepted before enabling installation.
2. **Given** a pending, unpublished, suspended, or revoked Release, **when** the owner attempts installation, **then** the Console shows the exact lifecycle error and does not create a successful-looking Installation.
3. **Given** an enabled Installation, **when** the owner disables it, **then** the Console shows disabled status and the invocation surface prevents a new call from being submitted through that Installation.
4. **Given** a disabled Installation, **when** the owner re-enables it or uninstalls it, **then** the Console displays the returned status and does not invent an enabled or removed state before the Gateway response arrives.

### User Story 4 - User invokes through Router and inspects the complete lineage (Priority: P1)

A Workspace user invokes an installed Agent through the public Gateway, receives a validated JSON or SSE result, and opens the resulting Invocation or Trace metadata. The visible flow proves that Agent B can call Agent A through the Router and return the result through the original caller without direct endpoint access.

**Why this priority**: The product proof is the complete `Register -> Verify -> Publish -> Discover -> Install -> Invoke -> Record` loop, including a cross-runtime nested call that remains governed and traceable.

**Independent Test**: With two independently implemented sample Agents installed in one Workspace, invoke B, have B call A through the Agent SDK and Router, and inspect one Trace containing the root and child Invocation with distinct Release provenance.

**Acceptance Scenarios**:

1. **Given** an enabled trusted Installation, **when** the user submits a JSON invocation, **then** the result contains the returned invocation, root Task, and Trace identifiers and no endpoint, Router, or Agent credential is entered in the form.
2. **Given** an enabled streaming Installation, **when** the user submits an SSE invocation, **then** the Console enforces accepted-first, contiguous sequence, stable correlation, and terminal-event semantics before showing completion.
3. **Given** Agent B receives a managed request and needs Agent A, **when** B uses the Agent SDK, **then** the child request traverses the Router, preserves the root Task, Trace, and parent Invocation relationship, and returns through B to the original Gateway caller.
4. **Given** a timeout, cancellation, unavailable endpoint, disabled Installation, suspended or revoked Release, or malformed stream, **when** the operation fails, **then** the Console shows the exact failure category and related trace/invocation identifiers where available; it does not display fabricated success data or retry.
5. **Given** a completed nested invocation, **when** the user opens its Trace, **then** the Console shows both root and child records and the child points to its parent Invocation without exposing result payloads as Ledger facts.

### User Story 5 - Operator verifies a real production frontend (Priority: P2)

The repository maintainers can run the Console's typecheck, focused contract tests, production build, and a browser-level acceptance against an explicit Gateway origin. Comparison demos remain available but are separate from the production route and never substitute for live API evidence.

**Why this priority**: A trustworthy operational surface must be reproducible in CI. A green job that matches no frontend project or exercises only mock demos is not evidence of the user workflow.

**Independent Test**: Run the frontend checks and a fresh-environment browser acceptance that completes the visible trusted lifecycle and reads the resulting invocation lineage.

**Acceptance Scenarios**:

1. **Given** an explicit frontend project configuration, **when** repository CI runs, **then** typecheck, tests, build, and browser acceptance execute against the production Console.
2. **Given** a `#/demo`, `#/demo/glass`, `#/demo/terminal`, or `#/demo/saas` URL, **when** it is opened, **then** it renders comparison content using mock data without being mistaken for a production API result.
3. **Given** missing, blank, whitespace-containing, credential-bearing, wildcard, or otherwise invalid browser configuration, **when** the Console starts or submits a request, **then** it fails explicitly and does not fall back to localhost, another endpoint, trimmed credentials, local storage, retry, reconnect, or legacy publish routes.

### Edge Cases

- A binding or Release response contains unknown fields, missing required fields, inconsistent IDs, an invalid timestamp, or a Card/Release mismatch; the client rejects it as an invalid response.
- A challenge is expired or already used while the user has the page open; the Console requires an explicit fresh challenge and does not resubmit the old challenge.
- The endpoint redirects, resolves to a disallowed network, is unreachable, or serves the wrong proof; the error remains visible as its exact category.
- A Release is pending, verified, published, suspended, or revoked while the page has stale data; a successful mutation is followed by a server read before actions are re-enabled.
- An Installation response omits `installedReleaseId` for a trusted installation or returns a Card version that differs from the selected version; the client rejects the inconsistent response.
- A Release preflight is missing, not found, not published, or does not match the selected Catalog Card; the Console blocks installation and preserves the exact Gateway error without attempting a version-only install.
- An SSE response ends before a terminal event, emits a duplicate or reordered event, changes correlation, or emits data after terminal; the client reports interruption/protocol failure.
- A request is canceled after the Gateway has assigned an Invocation; the UI does not replace the canceled result with a later success and can inspect the durable Ledger fact separately.
- Provider and Workspace owner credentials are missing or belong to the wrong principal; the operation fails at the Gateway boundary and the Console does not attempt another credential.
- Browser refresh occurs after a challenge is issued or a mutation starts; no challenge proof, token, or pending success is restored from storage.

## Requirements

### Functional Requirements

- **FR-001**: The production Console MUST use only public Gateway routes for registration, discovery, trusted publication, Workspace, Installation, invocation, and Ledger operations.
- **FR-002**: The production Console MUST expose Endpoint Binding creation/read, Challenge issuance/completion, Release creation/read, verification, publication, suspension, and revocation using the active Trusted Publication v1 semantics.
- **FR-003**: The Console MUST model Binding and Release state from server responses and MUST not use Catalog publication as a substitute for trusted Release publication.
- **FR-004**: The Console MUST retain challenge proof only in transient memory for the minimum operation display and MUST never persist provider, Workspace, Agent, Router, or challenge credentials in local storage, URLs, logs, or unrelated requests.
- **FR-005**: The Console MUST preserve the exact trusted-publication HTTP status, stable error code, and error-body trace ID for failed operations; if an `x-nek-trace-id` response header is present, it MUST agree with the error-body trace ID, while a missing header remains valid under the active OpenAPI contract. Static recovery guidance belongs to the UI and MUST NOT be represented as a fabricated server field.
- **FR-006**: Release controls MUST enforce the server-defined immutable lifecycle and MUST not add in-place edits, automatic republish, unsuspend, revoke reversal, retry, or alternate endpoint behavior.
- **FR-007**: Workspace installation MUST require an explicit published Release preflight, show its immutable provenance, submit explicit accepted permissions and an exact version constraint, and distinguish installed Card version from installed immutable Release identity. The client MUST reject a successful-looking Installation whose `installedReleaseId` does not match the preflight Release.
- **FR-008**: The Console MUST reject an installation, invocation, or ledger response whose required fields, identity relationships, status transitions, timestamps, or correlation fields are missing or inconsistent.
- **FR-009**: The Console MUST support JSON and SSE invocation through Gateway v4 and preserve the active validation rules for correlation, sequence, chunk order, terminal status, and failure categories.
- **FR-010**: The Console MUST display Invocation and Trace metadata from Gateway responses, including nested parent/child lineage and Release provenance when returned, without fabricating records or persisting result payloads as Ledger data.
- **FR-011**: The platform acceptance MUST include a canonical cross-runtime path in which Agent B invokes Agent A through the Router and returns through the original Gateway request, with one correlated Trace.
- **FR-012**: Frontend configuration MUST require an exact Gateway origin, explicit `VITE_NEKIRO_PROVIDER_ID`, provider bearer, owner bearer, and Workspace identity as applicable; invalid or missing values MUST fail at the owning boundary without inferring identity, sharing credentials across principals, localhost, wildcard, trimming, storage, retry, reconnect, or legacy fallback.
- **FR-013**: Comparison demo routes MUST remain available and visually distinct from the production route; demo mock data MUST not be used by production operations or acceptance evidence.
- **FR-014**: Focused frontend tests MUST cover request construction, strict response mapping, trusted lifecycle transitions, exact error preservation, JSON/SSE validation, correlation, and secret absence.
- **FR-015**: Repository automation MUST execute frontend typecheck, focused tests, production build, and browser acceptance against an explicit fresh environment, and MUST fail when no production frontend is discovered.
- **FR-016**: Documentation MUST describe the provider trust workflow, Workspace installation workflow, B-to-A nested invocation, credential boundaries, failure categories, and recovery ownership without recommending SQL or direct Agent access.
- **FR-017**: The implementation MUST be split into independently reviewable PR scopes, and each scope MUST have an upstream Issue, mapped acceptance evidence, a pre-implementation Issue review by a subagent, and an independent post-implementation code review before the next scope starts.

### Key Entities

- **Endpoint Binding**: A provider-owned association between an Agent Card version and a canonical endpoint, with verification method, state, failure code, and evidence timestamps/digest.
- **Verification Challenge**: A one-time, expiring server-issued challenge used to prove endpoint control; its proof is transient UI data and not a persisted Console credential.
- **Agent Release**: An immutable, version-specific publication record linking a Card digest, endpoint binding, verification evidence, and lifecycle state.
- **Workspace Installation**: A Workspace owner's explicit authorization for an exact Agent version/Release and accepted permission set. The Console obtains the Release identity through an explicit preflight handoff because the active Installation request remains version-constraint based.
- **Invocation Lineage**: Gateway, Router, Invocation, Task, Trace, parent Invocation, and Release metadata that connects a root request to nested Agent calls.
- **Console Operation State**: Server-derived loading, success, validation failure, conflict, dependency failure, and recovery-required states shown for each operation.

### Runtime/Platform Boundary

- **Platform-owned behavior**: Agent Card registration, endpoint trust, immutable Release lifecycle, discovery, Workspace authorization, Router-mediated invocation, and append-only lineage are owned by NeKiro services and their versioned contracts.
- **Console-owned behavior**: Request forms, transient operation state, strict contract mapping, safe error presentation, confirmation of destructive lifecycle actions, and navigation are owned by the Console; it does not decide trust or authorization.
- **Runtime-owned behavior**: Agent model calls, prompts, tools, workflows, memory, and runtime-specific execution remain in the independently implemented Agents and are not added to the Console or platform core.
- **Cross-runtime proof**: Runtime B and Runtime A use only A2A/Agent SDK boundaries for the nested call; the Console supplies no endpoint or runtime-specific type and the resulting Trace contains both Releases.

## Success Criteria

### Measurable Outcomes

- **SC-001**: A provider can complete registration, endpoint verification, and trusted Release publication from the production Console in one fresh environment without SQL or direct Agent requests.
- **SC-002**: A Workspace owner can discover and install two published trusted Agents with explicit permissions, and 100% of installation records shown by the Console include the exact installed version and Release identity when the Gateway returns it.
- **SC-003**: The canonical B -> Router -> A invocation completes with one root Task and Trace, one parent/child Invocation relationship, and distinct Release provenance for both Agents.
- **SC-004**: 100% of covered invalid, expired, reused, forbidden, unavailable, disabled, suspended, revoked, timeout, cancellation, and interrupted-stream cases retain their exact stable error/failure category and never become fabricated success data.
- **SC-005**: Secret scans across source, test fixtures, browser storage, request bodies, visible response state, logs, and CI artifacts find zero bearer tokens, Router credentials, Agent credentials, or persisted challenge proofs outside the explicit one-time challenge delivery boundary.
- **SC-006**: Every frontend CI run executes and passes typecheck, focused tests, production build, and browser acceptance against the production route; a missing project or skipped test is a failing condition.
- **SC-007**: Independent reviews for every PR scope report zero High or Medium findings before the next scope is implemented, and all review findings are either fixed or returned to the Spec/Tasks with explicit policy.
- **SC-008**: The four live user stories can be demonstrated from the Console using only Gateway URLs, with comparison demos remaining isolated and no production operation reading mock data.

## Assumptions

- Trusted Publication v1, Agent Card 0.2, Catalog/Northbound v3, Workspace/Installation v2, Northbound Invocation v4, Router metadata v3, Invocation Event 0.3, and Result Stream Event v2 remain the active contracts.
- Backend trusted publication routes and the existing clean Compose acceptance are already available from upstream main; this feature consumes them unless a separate Issue identifies a missing contract or acceptance gap.
- Phase 1 uses explicit development-static provider/owner bearer values and one active Workspace; credential issuance, rotation, OIDC, RBAC, and multi-Workspace administration remain outside this feature.
- The provider's Agent endpoint is responsible for serving the challenge material; the Console can issue and complete a challenge but does not deploy or modify the Agent.
- The existing production Console repository remains the source of frontend code for Console Issues #2/#4/#3. The platform repository owns the SDD artifacts, upstream coordination, the integrated `apps/console` CI package, and the backend acceptance Issue #60.

## Non-Goals

- New backend trust, installation, invocation, or Ledger semantics without a separate contract/ADR and Issue.
- Agent deployment, endpoint health polling, automatic repair, retry, reconnect, alternate endpoint selection, automatic Release recovery, or fallback publication.
- Credential issuance, secret management, browser identity federation, complete RBAC, approval workflows, quotas, billing, rating, certification, or Marketplace ranking.
- LLM, prompt, tool, workflow, memory, RAG, session, or other Agent Runtime features.
- Persisting challenge proofs, invocation inputs/outputs, or arbitrary provider secrets in the Console.
- Replacing the production Console with one of the existing comparison demo routes.
