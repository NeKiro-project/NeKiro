# Feature Specification: Workspace Installation Inspection

**Feature Branch**: `codex/issue-7-installation-inspection`

**Created**: 2026-07-15

**Status**: Approved for implementation

**Input**: GitHub Issue #7, "[Installation] List and inspect Workspace Installation history"

## Clarifications

The Issue, active Northbound v3 contract, Spec 003, Spec 004, and Spec 005
already resolve the material behavior questions. No additional clarification
is required before implementation. The feature consumes the existing contract
and does not add a new public operation or fallback policy.

## User Scenarios & Testing

### User Story 1 - Read an Installation Fact (Priority: P1)

An authenticated Workspace owner reads one Installation and receives the
original version constraint, exact installed version, accepted permission
snapshot, status, and lifecycle timestamps. A current or uninstalled row is
read as the same durable fact; the read does not consult Catalog or mutate
state.

**Why this priority**: Installation facts are the durable authorization record
needed to audit what a Workspace accepted and to diagnose later operations.

**Independent Test**: Persist one enabled Installation and one uninstalled
Installation, read both as the owner, and compare every response field with the
committed database values after reconstructing the service.

**Acceptance Scenarios**:

1. **Given** an owned Workspace and a current Installation, **when** the owner
   reads its exact identifier, **then** the response contains the immutable
   constraint, exact pin, accepted permissions, status, and timestamps.
2. **Given** an owned Workspace and an uninstalled Installation, **when** the
   owner reads its exact identifier, **then** the terminal row remains
   readable, including `uninstalledAt`.
3. **Given** an Installation identifier belonging to another Workspace, **when**
   the owner reads it under the first Workspace, **then** the response is
   `NOT_FOUND` and no Installation fact is returned.

### User Story 2 - Traverse Installation History (Priority: P1)

An authenticated Workspace owner lists current and historical Installations in
bounded pages. Results use the active contract's deterministic ascending
`installedAt`, then `installationId` order. An opaque cursor resumes strictly
after the last returned tuple, so unchanged data has neither duplicates nor
omissions.

**Why this priority**: Owners need a complete audit history, including rows
released by uninstall, without loading an unbounded result.

**Independent Test**: Create more rows than one page, use every returned cursor
until it is absent, and assert that the concatenated identifiers equal the
single ordered history exactly once.

**Acceptance Scenarios**:

1. **Given** current and uninstalled rows, **when** the owner lists with a
   limit from 1 through 100, **then** each page contains no more than the limit,
   uses stable order, and includes every row across cursor continuation.
2. **Given** unchanged history with equal installation timestamps, **when** the
   owner traverses all pages, **then** the identifier tie-breaker prevents
   duplicates and omissions.
3. **Given** an owned Workspace with no Installation rows, **when** the owner
   lists with a valid limit, **then** the response is an explicit `items: []`
   without `nextCursor`.
4. **Given** a cursor bound to another Workspace or page size, **when** it is
   submitted, **then** the request fails validation and does not restart from
   the first page.

### User Story 3 - Preserve Authorization and Failure Semantics (Priority: P1)

The Gateway and Workspace service distinguish unauthenticated, invalid,
forbidden, not-found, and dependency-failure outcomes without revealing
whether another caller's Workspace or Installation exists. A failed list query
is never represented as an empty successful history.

**Why this priority**: Installation history is authorization evidence; leaking
existence or hiding storage failure would make the audit boundary unreliable.

**Independent Test**: Exercise each active v3 failure mapping through the HTTP
adapter and inject Workspace persistence failures at both Workspace lookup and
Installation read/list boundaries.

**Acceptance Scenarios**:

1. **Given** no trusted bearer identity, **when** a read or list request is
   submitted, **then** the response is `401 UNAUTHENTICATED` and no service
   call occurs.
2. **Given** an authenticated non-owner, **when** a read or list request is
   submitted for a Workspace, **then** the response is `403 FORBIDDEN` without
   Workspace or Installation facts.
3. **Given** an unknown Workspace or Installation, **when** inspection is
   attempted, **then** the response is `404 NOT_FOUND` rather than an empty
   item list.
4. **Given** a malformed limit/cursor or dependency error, **when** inspection
   is attempted, **then** the response is respectively `400 VALIDATION_ERROR`
   or `503 DEPENDENCY_ERROR`, with the trace header and no synthetic data.

## Edge Cases

- `limit` is required and must be an integer in the inclusive range 1-100;
  omitted, repeated, non-integer, and out-of-range values are validation
  failures.
- A cursor is opaque, strictly base64url-decoded, rejects duplicate or unknown
  members and trailing data, and must match the Workspace and requested limit.
- The list includes `enabled`, `disabled`, and `uninstalled` rows. It does not
  filter historical rows and it does not physically delete them.
- A real empty result has a non-null JSON array and no cursor. A database or
  row-scan failure remains `DEPENDENCY_ERROR`.
- Read authorization is evaluated from the Workspace owner before the
  Installation lookup, preventing cross-Workspace fact leakage.
- Inspection does not call Catalog, probe endpoints, refresh pins, mutate
  lifecycle state, or add retry/cache/alternate-source behavior.

## Requirements

### Functional Requirements

- **FR-001**: The Northbound GET Installation operation MUST read one exact
  current or historical Installation under `/v3/workspaces/{workspaceId}/installations/{installationId}`.
- **FR-002**: A successful exact read MUST return the committed
  `versionConstraint`, `installedVersion`, `acceptedPermissions`, `status`,
  `installedAt`, `updatedAt`, and conditional `uninstalledAt` values.
- **FR-003**: The Northbound list operation MUST accept a required `limit` in
  the inclusive range 1-100 and an optional opaque cursor.
- **FR-004**: List results MUST include all Installation statuses and MUST be
  ordered by `installedAt ASC, installationId ASC`.
- **FR-005**: Cursor continuation MUST start strictly after its encoded
  `(installedAt, installationId)` tuple and MUST bind to the Workspace and
  requested limit.
- **FR-006**: On unchanged data, complete cursor traversal MUST return every
  matching Installation exactly once, without duplicate or omitted rows.
- **FR-007**: A successful empty history MUST serialize as `items: []` and
  omit `nextCursor`.
- **FR-008**: Workspace inspection MUST authorize the caller against the
  Workspace owner before reading Installation facts; cross-Workspace IDs MUST
  return `NOT_FOUND`.
- **FR-009**: Unknown Workspace, unknown Installation, mismatched
  Workspace/Installation, invalid cursor/query, unauthenticated caller, and
  non-owner caller MUST map to the active Northbound v3 failure semantics
  without returning existence or Installation facts.
- **FR-010**: Workspace persistence, query, scan, and restart failures MUST
  remain explicit dependency failures and MUST NOT become empty success.
- **FR-011**: Inspection MUST read only Workspace-owned facts and MUST NOT
  query or mutate Catalog data.
- **FR-012**: The implementation MUST preserve the active Installation v2 and
  Northbound v3 contracts without introducing a second contract or fallback.
- **FR-013**: Read/list responses and errors MUST include the Gateway trace
  header and MUST not contain credentials, tokens, or permission secrets beyond
  the accepted permission IDs already in the Installation contract.

### Key Entities

- **Workspace**: The owner-controlled authorization root identified by
  `workspaceId` and immutable `ownerId`.
- **Installation**: A Workspace-owned immutable authorization pin with original
  version constraint, exact installed version, accepted permissions, lifecycle
  status, and timestamps. Uninstalled rows are terminal history.
- **InstallationList**: A bounded ordered page of Installation facts with an
  optional opaque continuation cursor.
- **InstallationPosition**: The internal ordering tuple represented by
  `installedAt` and `installationId`; it is not exposed as a public mutable
  filter.

### Runtime/Platform Boundary

- **Platform-owned behavior**: Workspace owner authorization, Workspace-owned
  Installation reads, deterministic history pagination, HTTP error mapping,
  and trace propagation.
- **Runtime-owned behavior**: None. This feature does not invoke or inspect an
  Agent Runtime.
- **Cross-runtime proof**: Not applicable; the feature reads platform
  authorization facts before the later Router invocation boundary.

## Success Criteria

### Measurable Outcomes

- **SC-001**: Every successful exact read returns all committed Installation
  fields, including terminal timestamps, with equality after process restart.
- **SC-002**: For an unchanged history of at least 101 rows, traversing pages
  with limits 1, 25, or 100 returns the ordered set of all rows exactly once.
- **SC-003**: A Workspace with no rows returns an explicit empty JSON array;
  injected lookup, query, scan, or restart failures return no successful empty
  response and map to `503 DEPENDENCY_ERROR` at HTTP.
- **SC-004**: Unknown, cross-Workspace, invalid-cursor, unauthenticated, and
  non-owner cases return the active v3 status/code pair with no Installation
  payload or existence detail.
- **SC-005**: Contract, unit, PostgreSQL integration, restart, pagination,
  authorization, dependency, and HTTP tests cover every acceptance scenario.

## Assumptions

- The active `contracts/openapi/control-plane.v3.yaml` and
  `contracts/schemas/installation.v2.schema.json` are the source of truth.
- Workspace and Installation PostgreSQL migrations from Spec 004/005 are
  already present and remain Workspace-owned.
- A dedicated PostgreSQL database whose name ends in `_test` is required for
  real integration tests; without it, those tests are reported as not run.
- The Gateway authenticates the caller; the inspection service receives only
  the trusted caller identity and never accepts owner identity from request
  data.

## Non-Goals

- No Installation lifecycle mutation, reinstall behavior, exact Catalog
  resolution, invocation dispatch, Router, Ledger, SDK, or Console work.
- No Catalog probing, endpoint health check, version refresh, cache, retry,
  alternate data source, default limit, or compatibility fallback.
- No physical deletion or redaction of uninstalled Installation history.
