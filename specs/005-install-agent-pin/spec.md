# Feature Specification: Install a Published Agent and Pin an Exact Version

**Feature Branch**: `codex/005-install-agent-pin`
**Created**: 2026-07-15
**Status**: Approved
**Input**: GitHub issue #5, dependent on issues #3 and #4

## Clarifications

- Only the authenticated Workspace owner may create an Installation. Owner
  identity comes from trusted Gateway authentication and is never accepted in
  the request body or a public header.
- The submitted `versionConstraint` is required, parsed by the active SemVer
  policy, and is never replaced by a wildcard, exact version, or default.
- Selection considers only currently published Catalog versions. Draft,
  disabled, unknown, and non-matching versions are not candidates.
- The highest eligible SemVer precedence wins. A pre-release participates only
  when the matching constraint branch explicitly contains a pre-release
  comparator. Equal precedence caused only by build metadata is resolved by
  the bytewise-greatest exact version string.
- The selected exact Card is the Catalog selection linearization point. A
  Catalog disable that commits after that point does not rewrite or undo the
  already persisted Installation; later exact resolution observes the disabled
  Catalog state.
- `acceptedPermissions` is required in the request and may be an explicit empty
  array. Every value must be an exact, case-sensitive ID declared by the
  selected Card; unknown or duplicate values fail before persistence. The
  stored snapshot is ascending bytewise order and is never expanded by later
  Card versions.
- A Workspace may have at most one enabled or disabled Installation for an
  Agent. A current duplicate is a conflict, including under concurrent
  requests. Reinstall after uninstallation creates a new ID.
- Publishing a newer matching version never upgrades or rewrites an existing
  Installation. The submitted constraint and selected exact version are
  immutable authorization evidence.
- No endpoint probe, Agent call, deployment, cache, retry, or alternate
  Catalog source is part of installation.

## User Scenarios & Testing

### User Story 1 - Install and Pin an Agent Version (Priority: P1)

A Workspace owner installs a published Agent by submitting its ID, a SemVer
constraint, and the exact permissions they accept. The platform chooses one
eligible published version through the Catalog boundary and stores a durable,
enabled Installation with a server-assigned ID and exact version pin.

**Why this priority**: This closes the `Discover -> Install` step and creates
the authorization fact required before managed invocation.

**Independent Test**: Publish stable, pre-release, build-metadata, draft, and
disabled versions; install with valid and invalid constraints/permission sets;
verify the selected pin, owner authorization, duplicate behavior, restart
durability, and no mutation after a newer version is published.

#### Acceptance Scenarios

1. **Given** an owned Workspace and multiple published matches, **when** the
   owner installs with a valid constraint, **then** the highest eligible
   SemVer is persisted as `installedVersion`.
2. **Given** only pre-release matches and no pre-release comparator in the
   matching constraint branch, **when** install is requested, **then** no
   version is selected and the fixed not-found failure is returned.
3. **Given** a pre-release-aware constraint, **when** stable and eligible
   pre-release versions are available, **then** SemVer precedence selects the
   highest eligible version.
4. **Given** versions equal in SemVer precedence because only build metadata
   differs, **when** install is requested, **then** the bytewise-greatest exact
   version string is pinned.
5. **Given** a permission subset declared by the selected exact Card, **when**
   install succeeds, **then** the exact canonical snapshot is persisted,
   including an explicit empty set.
6. **Given** an unknown or duplicate permission ID, **when** install is
   requested, **then** validation fails before any Installation is written.
7. **Given** a non-owner, missing Workspace, invalid request, unavailable
   Catalog, or unavailable Workspace store, **when** install is requested,
   **then** the distinct fixed failure is returned and no synthetic success is
   produced.
8. **Given** an enabled or disabled current Installation, **when** another
   install for the same Workspace and Agent is requested, **then** conflict is
   returned and no duplicate current row is created.
9. **Given** concurrent install attempts, **when** they complete, **then** one
   deterministic winner exists and every other attempt returns conflict.
10. **Given** a newer matching version is published after installation, **when**
    the Installation is read, **then** its constraint, exact pin, permissions,
    ID, status, and timestamps are unchanged.
11. **Given** Catalog disables the selected version after successful selection,
    **when** the installation transaction completes, **then** the exact pin
    remains historical truth and no automatic rollback or rewrite occurs.
12. **Given** a committed Installation and process reconstruction, **when** it
    is read, **then** the stored exact pin, permissions, status, identity, and
    timestamps remain unchanged.

## Edge Cases

- Missing `acceptedPermissions` is invalid; `"acceptedPermissions": []` is the
  only valid empty-set representation. `null` is invalid.
- Permission IDs are not trimmed, case-folded, deduplicated, or silently
  normalized at the request boundary.
- Empty, whitespace-only, malformed, or parser-invalid constraints are invalid.
- A valid constraint with no published match is not found, not a dependency
  failure. A Catalog query failure is a dependency failure.
- An Installation response must retain the exact immutable pin and canonical
  permission order. Lifecycle or later publication cannot alter it.
- Database timestamp precision must be represented by the values returned from
  the committed row; a pre-storage in-memory timestamp is not a second fact.
- No API key, token, Card body, endpoint, or dependency detail appears in an
  Installation response, error, or log.

## Requirements

### Functional Requirements

- **FR-001**: Install MUST require trusted authentication and MUST permit only
  the Workspace owner through the owner-policy boundary.
- **FR-002**: The request MUST contain an Agent ID, valid SemVer constraint,
  and required `acceptedPermissions` array; invalid or missing fields MUST
  return `VALIDATION_ERROR`.
- **FR-003**: The Catalog boundary MUST select only currently published
  versions and MUST return the highest eligible SemVer precedence.
- **FR-004**: Pre-release and equal-precedence build-metadata behavior MUST
  follow the clarified active Catalog policy.
- **FR-005**: Unknown, draft, disabled, and non-matching candidates MUST NOT
  be selected; no match is `NOT_FOUND` and Catalog failure is
  `DEPENDENCY_ERROR`.
- **FR-006**: Every accepted permission MUST be declared by the selected exact
  Card; unknown or duplicate IDs MUST fail before persistence.
- **FR-007**: An explicit empty accepted-permission array MUST be valid and MUST
  remain an explicit empty stored/returned snapshot.
- **FR-008**: Success MUST persist the Workspace ID, Agent ID, submitted
  constraint, exact installed version, canonical permission snapshot, enabled
  status, server-assigned ID, and committed server timestamps.
- **FR-009**: A Workspace/Agent pair MUST have at most one non-uninstalled
  Installation, including under concurrent requests; losing attempts MUST
  return conflict.
- **FR-010**: The constraint, exact pin, accepted permissions, Installation ID,
  identity, status history, and timestamps MUST NOT be rewritten by a newer
  published version.
- **FR-011**: Catalog selection and Workspace persistence MUST use the
  approved transaction/linearization semantics; a later Catalog disable MUST
  not create a false rollback or rewrite.
- **FR-012**: Installation behavior MUST cross the Catalog-owned controlled
  port and MUST NOT query Catalog tables or invoke an Agent.
- **FR-013**: HTTP, PostgreSQL, restart, concurrency, and dependency failures
  MUST have explicit mapped tests.
- **FR-014**: This feature MUST add zero fallback behavior and MUST NOT add
  deployment, endpoint probing, cache, retry, or alternate-source behavior.

## Success Criteria

- **SC-001**: An owner can complete one authenticated `POST /v3/workspaces/{workspaceId}/installations` workflow and receive an exact enabled pin.
- **SC-002**: 100% of selection, permission, owner, duplicate, and dependency
  cases return their distinct contract-defined result without an invalid row.
- **SC-003**: 100% of concurrent same-Agent install races leave one current
  Installation and explicit conflicts for all losers.
- **SC-004**: 100% of committed Installation facts survive process
  reconstruction without changing exact pin, permissions, identity, or times.
- **SC-005**: A newer matching publication changes zero fields of an existing
  Installation.
- **SC-006**: No installation path invokes, probes, deploys, caches, retries,
  or falls back to another Catalog source.

## Assumptions

- Issue #3 freezes Installation v2, Northbound v3, Catalog selection behavior,
  and the Workspace store/Catalog reader ports.
- Issue #4 supplies the durable owner-controlled Workspace root and its active
  readiness boundary.
- The active contracts do not require a new public version for this behavior;
  #5 consumes the already approved v3 Installation route.

## Non-Goals

- Installation inspection, lifecycle enable/disable/uninstall, exact internal
  resolution, Invocation Dispatch, A2A Router, Ledger, or Agent execution.
- Automatic upgrade, rollback, reconciliation, deployment, endpoint health,
  Kubernetes binding, or Runtime framework behavior.
- Membership/RBAC, OIDC, organization governance, billing, rating, or
  Marketplace behavior.
- New contract versions, historical dual-read routes, cache, queue, retry, or
  alternate Catalog persistence.
