# Feature Specification: A2A Router Foundation

**Feature Branch**: `codex/013-router-foundation`
**Created**: 2026-07-16
**Status**: Implemented, locally verified, independently reviewed, and converged
**Input**: Spec 010 T003 / GitHub #22.

## Context

This feature creates the smallest independent A2A Router process boundary that
Control Plane Dispatch can call. It does not execute Agent transport, write
Ledger facts, expose SDK/nested invocation, or integrate Compose. Its purpose
is to prove the Data Plane process, strict configuration, service
authentication, readiness, and Control Plane exact-resolution direction before
transport and Ledger features build on it.

## Clarifications

### Session 2026-07-16

- Q: What is the accepted Invocation boundary in this feature? A: None. Router Foundation validates and resolves, but it does not create or persist Invocation events; Ledger acceptance starts in the Ledger/dispatch features.
- Q: Which downstream may Router call? A: Only Control Plane Internal v2 `/internal/v2/resolve-agent` through one required configured destination and one explicit service Bearer token.
- Q: Which inbound callers are trusted? A: Only a separately configured Router service Bearer principal; northbound credentials and Agent credentials are not valid for this internal boundary.
- Q: What readiness means here? A: Configuration and handler assembly are valid; readiness does not probe Control Plane, database, Agent endpoints, or Ledger storage.
- Q: What happens to valid dispatch requests after resolution? A: They receive an explicit `ROUTE_NOT_FOUND` placeholder until T006 implements A2A transport; no Agent endpoint is contacted and no result or Ledger fact is fabricated.

## User Scenarios & Testing

### User Story 1 - Start a Strict Router Process (Priority: P1)

A platform operator can start the Router only when every required Data Plane setting is explicit and valid.

**Independent Test**: Load config tables with missing, blank, malformed, whitespace-padded, credential-bearing, duplicate, unsupported, zero, negative, fractional, exponent, overflow, and out-of-range values and verify startup/readiness fails without defaults.

**Acceptance Scenarios**:

1. **Given** complete valid Router config, **When** the process assembles handlers, **Then** readiness returns OK without probing external dependencies.
2. **Given** any required config is missing, blank, malformed, or out of range, **When** config loads, **Then** startup fails and no default destination, credential, timeout, limit, or port is used.
3. **Given** a configured Control Plane URL containing userinfo, query, fragment, non-HTTP scheme, or the wrong path, **When** config loads, **Then** it is rejected before serving.

### User Story 2 - Authenticate Internal Dispatch Requests (Priority: P1)

Control Plane Dispatch can call the Router internal boundary with a configured service credential, while all other callers are rejected before resolution or routing.

**Independent Test**: Exercise missing, malformed, unknown, duplicate, and valid Bearer credentials against the Router internal handler and prove invalid callers create no resolution request.

**Acceptance Scenarios**:

1. **Given** a valid service Bearer credential, **When** `POST /internal/v3/invocations` arrives with a strict request and supported media, **Then** Router authenticates the service caller and proceeds to exact resolution.
2. **Given** no credential, a northbound credential, an Agent credential, or an unknown token, **When** a dispatch request arrives, **Then** Router returns fixed `UNAUTHENTICATED` or `FORBIDDEN` before resolution and without trusted correlation side effects.
3. **Given** invalid Content-Type, Accept, body size, duplicate fields, unknown fields, malformed IDs, malformed JSON, or trailing content, **When** a dispatch request arrives, **Then** Router returns pre-acceptance v4 validation/media/size errors and performs zero resolution calls.

### User Story 3 - Resolve Through Control Plane Only (Priority: P1)

Router re-resolves the exact Agent and capability through the Control Plane-owned internal contract without importing Control Plane internals or reading its storage.

**Independent Test**: Fake Control Plane Internal v2 returns success and each typed failure; Router sends exactly one request with trusted correlation, preserves response trace/correlation, maps failures exactly, and never calls Agent or Ledger paths.

**Acceptance Scenarios**:

1. **Given** a valid dispatch request and successful Control Plane resolution, **When** Router handles it, **Then** the resolve request repeats the exact invocationId, rootTaskId, traceId, workspaceId, agentId, version, and capability.
2. **Given** Control Plane returns `AGENT_NOT_INSTALLED`, `INSTALLATION_DISABLED`, `AGENT_DISABLED`, `CAPABILITY_NOT_ALLOWED`, or `DEPENDENCY_ERROR`, **When** Router handles dispatch, **Then** Router preserves the typed status/body/correlation and returns no successful routing placeholder.
3. **Given** Control Plane transport fails or returns malformed media/body, **When** Router handles dispatch, **Then** Router returns explicit correlated `DEPENDENCY_ERROR` and does not retry, cache, alternate-source, or fabricate resolution.

### User Story 4 - Stop Before Agent Transport (Priority: P1)

A caller receives a truthful non-success response showing Router foundation is wired but Agent transport is not yet implemented.

**Independent Test**: After successful resolution, verify Router returns correlated `ROUTE_NOT_FOUND`, not success, not a mock Agent result, not a Ledger fact, and not an endpoint probe.

## Edge Cases

- Auth succeeds but the dispatch request contains duplicate JSON members.
- The request asks for SSE but sends a JSON-only Accept header, or vice versa.
- Control Plane Internal v2 returns a valid correlated error with a different trace header.
- Control Plane returns `200` with a non-JSON media type or malformed resolved Card.
- Context deadline/cancellation occurs before the Control Plane response.

## Requirements

- **FR-001**: Router MUST run as an independent Go process under `apps/a2a-router` and MUST NOT import `apps/control-plane/internal/*`.
- **FR-002**: Router MUST expose `POST /internal/v3/invocations` and readiness endpoints through its own handler assembly.
- **FR-003**: Router configuration MUST require explicit listen address, Router service principals, Control Plane resolve URL, Control Plane service Bearer token, internal request body limit, response body limit, and resolution deadline, all strictly parsed with no defaults.
- **FR-004**: Router readiness MUST validate local configuration and handler assembly only; it MUST NOT probe Control Plane, Agent endpoints, Ledger, or databases.
- **FR-005**: Router internal authentication MUST use a dedicated service credential set and MUST NOT accept northbound or Agent credentials implicitly.
- **FR-006**: Router MUST authenticate before parsing trusted dispatch semantics and MUST reject invalid media/body/shape/identifier requests before Control Plane resolution.
- **FR-007**: Router MUST consume the frozen Router Internal v3 and Platform Error v4 contracts without modifying shared contracts in this feature.
- **FR-008**: Router MUST call only Control Plane Internal v2 `/internal/v2/resolve-agent` to resolve exact Agent facts; it MUST NOT read Catalog/Workspace storage or maintain a permanent Card copy.
- **FR-009**: Each valid dispatch request MUST produce exactly one Control Plane resolution attempt and MUST NOT retry, cache, use alternate destinations, or call an Agent endpoint.
- **FR-010**: Resolution requests MUST preserve exact invocationId, rootTaskId, traceId, workspaceId, targetAgentId, agentCardVersion, and capability from the dispatch request.
- **FR-011**: Router MUST preserve Control Plane typed resolution failures and trace/correlation semantics, and MUST distinguish Control Plane dependency failure from authorization, disabled, not-installed, and validation failures.
- **FR-012**: After successful resolution, Router Foundation MUST return correlated `ROUTE_NOT_FOUND` until Agent transport is implemented; it MUST NOT return mock success, persist Ledger facts, or probe the Agent endpoint.
- **FR-013**: Router logs/errors/readiness MUST NOT expose service tokens, Agent endpoint credentials, raw dependency details, or Agent input/output payloads.
- **FR-014**: The feature MUST add zero fallback behavior: no default localhost, no weak token, no anonymous mode, no retry, no cache, no alternate Control Plane source, and no direct Agent path.

## Key Entities

- **Router service principal**: A configured internal caller credential allowed to invoke Router Internal v3.
- **Resolution client**: Router-owned client for Control Plane Internal v2 exact resolution.
- **Dispatch envelope**: Trusted Router Internal v3 request received from Control Plane Dispatch.
- **Readiness state**: Local configuration/assembly health; not dependency reachability.

## Non-Goals

A2A Agent transport, Ledger events or storage, result streaming from Agents, cancellation propagation, SDK/nested invocation, metadata read APIs, Compose/CI process orchestration, Frontend, Marketplace, and any Agent Runtime behavior.

## Success Criteria

- **SC-001**: 100% of required Router config absence, blank, malformed, unsafe, and out-of-range cases fail startup without defaults.
- **SC-002**: Invalid auth/media/body requests make zero Control Plane resolution calls.
- **SC-003**: Valid dispatch makes exactly one Control Plane Internal v2 resolution call with exact trusted correlation and target fields.
- **SC-004**: Every mapped Control Plane resolution failure returns its distinct public status/code/correlation without collapsing to success or generic dependency failure.
- **SC-005**: Successful resolution returns a truthful correlated `ROUTE_NOT_FOUND` placeholder and proves no Agent endpoint, Ledger path, retry, cache, or fallback was used.

## Fallback Report

```text
Fallback delta: removed 0, retained 0, added 0, net 0
Added fallback evidence: none
```
