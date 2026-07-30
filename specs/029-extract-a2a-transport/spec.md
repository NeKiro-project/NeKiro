# Feature Specification: Extract Reusable A2A Transport

**Feature Branch**: `codex/029-extract-a2a-transport`

**Created**: 2026-07-30

**Status**: Implemented

**Input**: User description: "Lead the migration of reusable NeKiro A2A transport behavior into NeKiro-project/nekiro-a2a-transport-go and consume it from NeKiro through go.mod."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Maintain One Reusable Transport Source (Priority: P1)

As a NeKiro platform maintainer, I want protocol transport and wire-validation behavior to be maintained and released from one standalone upstream project so the Router consumes a reviewed version instead of owning a private copy.

**Why this priority**: A single versioned source removes implementation drift while preserving the Router as the owner of platform routing, authorization, and Ledger behavior.

**Independent Test**: Publish one standalone transport release, consume that exact release from NeKiro, and verify the existing non-streaming and streaming Router acceptance without a copied transport implementation.

**Acceptance Scenarios**:

1. **Given** the standalone transport has a reviewed release, **when** NeKiro resolves its dependencies, **then** the Router consumes that exact release through its dependency manifest and contains no second implementation of the extracted wire behavior.
2. **Given** the standalone release is unavailable or its version is missing, **when** NeKiro resolves dependencies, **then** dependency resolution fails explicitly and does not use a local, legacy, vendored, or alternate transport copy.
3. **Given** an independently implemented Agent, **when** the migrated Router sends a non-streaming or streaming call, **then** the wire behavior and correlated invocation outcome match the pre-migration contracts.

---

### User Story 2 - Reuse Transport Without NeKiro Platform Coupling (Priority: P1)

As a Go A2A integrator, I want to use the standalone transport without importing NeKiro Router internals, Control Plane contracts, Workspace models, publication facts, credentials, or Ledger code.

**Why this priority**: A purported upstream library is not reusable if it retains a reverse dependency on the platform application that consumes it.

**Independent Test**: Build a fresh external consumer that imports only the standalone release and the supported A2A protocol dependency, sends strict request-response and streaming calls to a conforming test Agent, and observes typed transport failures for invalid wire input.

**Acceptance Scenarios**:

1. **Given** a clean external consumer, **when** it imports the standalone release, **then** it can build and use the supported transport operations without importing a NeKiro application or platform-contract package.
2. **Given** caller-supplied request metadata or credentials, **when** a call is issued, **then** the library injects only the explicitly supplied values and does not invent identity, authorization, endpoint, or context data.
3. **Given** malformed, oversized, mismatched, unsupported, interrupted, canceled, or timed-out transport input, **when** the library processes it, **then** it returns an explicit distinguishable failure and never fabricates a successful result.

---

### User Story 3 - Preserve NeKiro Ownership and Failure Semantics (Priority: P1)

As a platform operator, I want the migration to preserve exact Agent resolution, trusted Router credentials, context propagation, bounded transport, cancellation, public errors, and Ledger terminalization so extracting code does not weaken platform governance.

**Why this priority**: Reuse is acceptable only when no platform trust boundary or observable invocation behavior moves into the generic library or changes silently.

**Independent Test**: Run the existing Router unit, integration, contract, cross-runtime, and invoke-to-record acceptance suites against the standalone release and compare every success, protocol failure, endpoint failure, timeout, cancellation, overflow, and Ledger outcome with the active contracts.

**Acceptance Scenarios**:

1. **Given** an authorized installed exact Release, **when** the Router invokes its Agent, **then** NeKiro still resolves the target and issues request-scoped credentials before delegating protocol transport.
2. **Given** an unsupported endpoint/profile/authentication mode or invalid platform context, **when** dispatch is attempted, **then** the NeKiro adapter rejects it before transport with the same public error semantics.
3. **Given** a known remote Task when the caller cancels or times out, **when** cancellation is propagated, **then** at most one bounded remote cancellation attempt occurs, no retry or alternate route is used, and NeKiro retains terminal-outcome ownership.
4. **Given** a Router or Ledger race, **when** more than one terminal condition occurs, **then** the existing first-committed terminal policy remains authoritative and the transport library creates no Ledger fact.

---

### User Story 4 - Upgrade Through Explicit Compatibility Evidence (Priority: P2)

As a maintainer of both projects, I want every transport release to declare its supported protocol/profile compatibility and pass a reusable conformance corpus before NeKiro upgrades.

**Why this priority**: Independent release cadence is valuable only when downstream upgrades remain deliberate and evidence-backed.

**Independent Test**: Run the standalone conformance suite, verify the release metadata, and prove that changing the consumed version requires an explicit dependency update and downstream verification.

**Acceptance Scenarios**:

1. **Given** a standalone release candidate, **when** its release checks run, **then** all declared JSON-RPC, SSE, result-kind, Task-state, cancellation, limit, and negative-corpus rules pass.
2. **Given** a future protocol-library version, **when** maintainers consider upgrading it, **then** the standalone project requires an explicit compatibility decision and does not accept the version through a floating dependency or compatibility fallback.

### Edge Cases

- A JSON-RPC response repeats a member, changes the request ID, contains both result and error, contains neither, uses an unsupported ID type, or has trailing data.
- A response uses a valid status with the wrong media type, redirects, exceeds the configured bound, or closes with an unreadable body.
- An SSE response contains duplicate or missing event IDs, multiple or missing data lines, an incomplete delimiter, invalid JSON, repeated terminal data, an identity-changing Task/context, or an event exceeding its bound.
- A stream is canceled before a Task ID is known, after a Task ID is known, while the remote Agent completes, or while terminal Ledger persistence races.
- Metadata injection fails or supplies duplicate authorization/context values; failure remains visible and no second credential source is attempted.
- The standalone module release exists but its declared protocol/profile compatibility does not match NeKiro's active profile; downstream integration stops rather than attempting legacy behavior.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The project MUST publish the reusable A2A client transport and wire-validation behavior from `NeKiro-project/nekiro-a2a-transport-go` as a versioned standalone module.
- **FR-002**: The standalone module MUST NOT import NeKiro application internals, Control Plane types, Router types, Workspace/Installation models, trusted-publication models, platform error contracts, credential claims, Ledger models, or platform storage code.
- **FR-003**: The standalone module MUST use the protocol-focused A2A dependency selected by the active NeKiro A2A Profile and MUST declare the exact supported protocol-library version.
- **FR-004**: The standalone module MUST support the active request-response and server-streaming message operations plus the single remote Task cancellation operation required for cancellation propagation.
- **FR-005**: The standalone module MUST validate JSON-RPC version, request/response identity, result/error exclusivity, supported result/event kinds, duplicate members, trailing data, media types, and strict SSE framing before returning accepted protocol data.
- **FR-006**: The standalone module MUST enforce caller-supplied positive response and event bounds without truncating data into a successful result.
- **FR-007**: The standalone module MUST expose explicit typed failure categories sufficient for the NeKiro adapter to distinguish invalid configuration, protocol violation, remote Agent failure, endpoint unavailability, deadline, cancellation, and response overflow.
- **FR-008**: The standalone module MUST accept request metadata through an explicit caller-owned injection seam and MUST NOT define or infer NeKiro Workspace, Release, Card digest, invocation lineage, credential, or authorization policy.
- **FR-009**: The standalone module MUST reject HTTP redirects and MUST NOT retry, switch endpoints, reuse stale data, use alternate credentials, downgrade protocol behavior, truncate a response, or fabricate a result.
- **FR-010**: The standalone module MUST close and account for request/response bodies deterministically, propagate caller cancellation, and perform no background work after a call has completed except the explicitly requested bounded cancellation operation.
- **FR-011**: NeKiro MUST retain a thin Router-owned adapter that maps exact resolution, active Agent Card limits, platform context, request-scoped Router credentials, transport failures, and stream events between versioned platform contracts and the standalone transport API.
- **FR-012**: NeKiro MUST retain ownership of routing, capability authorization, exact Release provenance, credential issuance, Platform Error mapping, Result Stream Event mapping, Task/Invocation lineage, Ledger writes, and terminal race policy.
- **FR-013**: The migration MUST preserve active public and internal wire contracts and MUST NOT introduce a contract version bump unless implementation discovers an unavoidable behavior change and returns to specification and ADR review first.
- **FR-014**: The standalone module MUST include executable positive and negative conformance tests for all extracted behavior, and NeKiro MUST retain downstream adapter, integration, and end-to-end verification.
- **FR-015**: A clean external consumer MUST be able to import the tagged standalone module without a local replacement, vendored NeKiro source, or dependency on the NeKiro root module.
- **FR-016**: NeKiro MUST consume a tagged immutable standalone version through its dependency manifest; local replacement directives are migration-only and MUST NOT be committed as production dependency behavior.
- **FR-017**: Documentation MUST identify which behavior moved upstream, which platform behavior remains downstream, how compatibility is declared, and how releases and downstream upgrades are verified.
- **FR-018**: The migration MUST add zero fallback behaviors; any retained nil-transport or one-shot cancellation policy MUST cite its existing contract or ADR evidence and remain observably equivalent.
- **FR-019**: Source transfer MUST preserve applicable license and attribution obligations in both repositories.
- **FR-020**: Logs, errors, fixtures, examples, and documentation MUST contain no bearer credential, signing material, endpoint credential, Workspace secret, or invocation credential.

### Key Entities

- **Standalone Transport Release**: An immutable version of reusable A2A wire transport, validation, failure categories, compatibility declaration, and conformance evidence.
- **Transport Target**: The explicit remote endpoint and operation capabilities supplied by a caller without platform-owned resolution or authorization semantics.
- **Request Metadata Injector**: A caller-owned mechanism that supplies exact request metadata or credentials and reports injection failure without defaults or alternate sources.
- **Transport Failure**: A typed, non-platform-specific failure classification that preserves protocol, remote, endpoint, deadline, cancellation, configuration, and overflow distinctions.
- **NeKiro Transport Adapter**: The Router-owned mapping layer between standalone transport concepts and exact NeKiro resolution, credentials, context, errors, streaming results, and Ledger lifecycle.

### Runtime/Platform Boundary *(mandatory when feature touches Agent execution or integration)*

- **Platform-owned behavior**: Exact Release resolution, Workspace authorization, capability checks, Router credential issuance, invocation lineage, Platform Error mapping, Result Stream Event mapping, and Ledger lifecycle remain in NeKiro.
- **Runtime-owned behavior**: Agent model, prompt, tool, workflow, memory, session, Task production, and response generation remain in external Agent Runtimes; the extracted module is only a protocol transport dependency.
- **Cross-runtime proof**: Existing Runtime A and Runtime B acceptance MUST continue to complete Router-mediated calls and one correlated nested lineage while consuming the extracted transport release.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of extracted production transport behavior is maintained in one standalone source, with zero duplicate implementation files retained in NeKiro.
- **SC-002**: A clean external consumer imports one tagged standalone release and completes both supported message modes without importing any NeKiro platform module.
- **SC-003**: 100% of the standalone positive and negative conformance corpus passes on the supported development and CI platforms.
- **SC-004**: 100% of existing NeKiro Router, contract, integration, cross-runtime, and invoke-to-record acceptance checks pass after dependency replacement.
- **SC-005**: Success, protocol failure, endpoint failure, Agent failure, timeout, cancellation, and overflow outcomes remain observably equivalent before and after migration for every covered acceptance case.
- **SC-006**: Dependency and source scans report zero upstream imports of NeKiro application/platform packages and zero downstream duplicate copies of the extracted core.
- **SC-007**: Fallback audit reports added 0, and secrecy scans report zero credentials or signing material introduced by the migration.
- **SC-008**: A future upstream version cannot enter NeKiro without an explicit dependency change and passing standalone plus downstream compatibility checks.

## Assumptions

- The target upstream repository is controlled by the NeKiro project and uses an open-source license compatible with the source being transferred.
- The active A2A Profile remains protocol `0.3.0` with `github.com/a2aproject/a2a-go v0.3.15` until an explicit compatibility decision changes it.
- Existing NeKiro public/internal contracts and runtime acceptance are the behavioral baseline for this extraction.
- Go's documented nil `http.Transport` behavior and the existing one bounded remote cancellation propagation attempt remain the only retained recovery-like policies in scope.
- Upstream and downstream changes are independently reviewed and merged; the upstream tag is created before the downstream production dependency is finalized.

## Non-Goals

- Replacing, forking, or reimplementing the complete A2A protocol library.
- Moving NeKiro Router, routing, resolution, Workspace authorization, trusted publication, credential claims, Platform Error mapping, streaming result contracts, Ledger, or database behavior upstream.
- Adding an Agent Runtime framework, server framework, model/tool/workflow/session behavior, or framework-specific adapter to the standalone transport core.
- Supporting additional A2A transports, protocol versions, endpoint discovery, reconnect, replay, retry, cache, alternate endpoint, credential fallback, or legacy compatibility behavior.
- Changing active Agent Card, A2A Profile, Router, invocation, result, event, credential, or Ledger contracts as part of the extraction.
- Publishing the NeKiro Agent SDK or client SDK from the standalone transport repository.
