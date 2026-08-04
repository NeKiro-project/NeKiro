# Feature Specification: Core Repository Boundary

**Feature Branch**: `codex/core-repository-split`

**Created**: 2026-08-02

**Status**: Approved for implementation

**Input**: User description: "Specification only: keep the NeKiro repository focused on core implementation and define how SQL and other artifacts should be separated."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Classify Core and Satellite Artifacts (Priority: P1)

As a NeKiro maintainer, I want every tracked artifact assigned to an explicit owner and release boundary so the core repository contains only platform core, its contracts, and the evidence needed to govern and verify that core.

**Why this priority**: Extraction cannot be safe while "core" means a file type or directory preference rather than an ownership rule. A complete classification prevents accidental removal of required state, contracts, or verification.

**Independent Test**: Review the repository inventory and confirm that every tracked artifact family has exactly one disposition, one canonical owner, and one validation gate, with no unclassified or multiply authoritative production source.

**Acceptance Scenarios**:

1. **Given** the current repository inventory, **when** maintainers apply the boundary policy, **then** every artifact is classified as core-owned, satellite-owned, duplicate-to-remove, or retained historical evidence.
2. **Given** an artifact used by both core and a satellite, **when** its ownership is evaluated, **then** one canonical owner is named and all other consumers use a versioned release rather than a maintained copy.
3. **Given** an artifact whose owner or compatibility boundary cannot be proven, **when** extraction is proposed, **then** that extraction is blocked and recorded for policy resolution rather than guessed.

---

### User Story 2 - Keep Database Evolution With Its Owner (Priority: P1)

As a platform operator, I want Catalog, Workspace, and Ledger schema migrations to remain owned and released by the services that understand those schemas so a core service release can validate and advance its own persistent state without a separately drifting database repository.

**Why this priority**: The SQL currently represents core data ownership and upgrade history. Moving it by file extension would separate executable behavior from the state contract it requires.

**Independent Test**: Starting from every supported existing schema version, use the corresponding core service release to apply its owned forward migrations and pass readiness, while verifying that each migration version has one canonical source.

**Acceptance Scenarios**:

1. **Given** a Catalog, Workspace, or Ledger migration, **when** the inventory is classified, **then** the migration remains with the module that owns the affected schema.
2. **Given** the same migration is represented in more than one maintained file or embedded copy, **when** the boundary cleanup completes, **then** one canonical representation remains and all migration execution and verification consume it.
3. **Given** an existing supported database, **when** a post-extraction service release starts its explicit migration workflow, **then** the same ordered forward upgrade and readiness semantics are preserved.
4. **Given** an operational database artifact that does not define a core-owned application schema, **when** it is classified, **then** it belongs to the deployment or operations satellite rather than the core service source.

---

### User Story 3 - Maintain Satellites Without Core Source Mirrors (Priority: P1)

As a Console, SDK, or sample Agent maintainer, I want my component to have an independent source and release lifecycle so changes do not require copying production source into the NeKiro core repository.

**Why this priority**: Mirrored source creates competing authorities, broadens core CI, and makes a dependency failure indistinguishable from stale copied code.

**Independent Test**: Release each satellite from its canonical project, verify it against the published NeKiro contracts, and confirm the core checkout contains no maintained production copy and can complete core build and verification without checking out satellite source.

**Acceptance Scenarios**:

1. **Given** the standalone Console is canonical, **when** a Console release is produced, **then** no production Console source or browser-only toolchain is maintained in the core repository.
2. **Given** a public SDK release, **when** an external application or Agent consumes it, **then** the SDK depends only on published contracts and supported public boundaries, not core service internals.
3. **Given** sample Agents implemented with different runtimes, **when** their releases are produced, **then** their runtime logic, fixtures, and sample-specific tests are owned outside the core repository.
4. **Given** a required satellite release is missing or incompatible, **when** integration is attempted, **then** the integration fails explicitly and does not use a copied source tree, local replacement, older version, or alternate component.

---

### User Story 4 - Prove the Product Across Repository Boundaries (Priority: P2)

As a release operator, I want one integration boundary to assemble immutable core and satellite releases and prove the complete NeKiro loop so repository separation does not weaken product acceptance.

**Why this priority**: Core isolation is useful only if Register -> Discover -> Install -> Invoke -> Record and cross-runtime lineage remain demonstrably complete.

**Independent Test**: Assemble one explicit set of released component versions in a clean environment and complete the trusted publication, installation, direct invocation, nested cross-runtime invocation, Ledger query, negative-path, and secrecy acceptance.

**Acceptance Scenarios**:

1. **Given** compatible immutable releases of core, Console, SDK/sample dependencies, and deployment assets, **when** the integration acceptance runs, **then** the complete user-visible loop and correlated nested lineage pass.
2. **Given** a contract-incompatible or unpinned component, **when** release acceptance begins, **then** acceptance stops before deployment and identifies the incompatible boundary.
3. **Given** a core-only change, **when** core verification runs, **then** it does not require satellite source, a browser toolchain, or sample runtime source.
4. **Given** a satellite-only change, **when** its own verification runs, **then** it validates against an explicit core contract release before product-level acceptance.

---

### User Story 5 - Preserve Governance and Provenance (Priority: P3)

As a project maintainer, I want accepted specifications, architecture decisions, compatibility history, authorship, and licenses to remain traceable after extraction so a smaller core checkout does not erase why the platform behaves as it does.

**Why this priority**: Repository cleanup must not trade away auditability, compatibility evidence, or contribution attribution.

**Independent Test**: Trace each moved source family from its original core commit to its canonical satellite history and release, and resolve every retained specification, decision, and compatibility reference without relying on an undocumented location.

**Acceptance Scenarios**:

1. **Given** source moved to a satellite, **when** its provenance is inspected, **then** original authorship, applicable license, and source lineage are available.
2. **Given** an accepted historical specification, **when** the repository is reduced, **then** its provenance remains reachable through the pre-split tag and Git history while permanent architecture, contract, and usage facts remain in the core repository.
3. **Given** future work that changes only a satellite, **when** it is proposed, **then** its Issue, implementation, tests, and review are owned by that satellite while any cross-boundary contract change starts with the contract owner.

### Edge Cases

- A file is generated for a satellite from a core-owned contract. The contract remains canonical in core; generated output belongs to the satellite release and must identify its exact contract version.
- A migration is old but still required to upgrade a supported database. It remains part of the owning service release history and is not archived merely because no new installation starts at that version.
- Two migration representations differ while claiming the same schema version. Extraction stops until the owning module identifies the authoritative behavior; content is not merged or selected heuristically.
- A core test currently imports sample Agent implementation code. The test must be converted to a core-owned protocol fixture or moved to cross-repository acceptance before the sample source is removed.
- A full-stack runbook mixes core service operations with Console, sample, or environment-specific steps. Core-owned operations remain with core; product assembly steps move with the integration or deployment owner, and links are updated together.
- A satellite release disappears, is retagged, or cannot be verified. Core and integration gates fail visibly; no vendored copy, floating branch, cached old release, or alternate source is accepted.
- An existing Go consumer imports `github.com/Nene7ko/NeKiro`. It must update to `github.com/NeKiro-project/NeKiro`; the migration does not provide a legacy module, forwarding package, or `replace` compatibility shim.
- An extraction overlaps an unfinished feature. Cutover waits for a coherent reviewed release boundary so one feature is not split between two writable authorities.
- Historical documents refer to old paths. References are updated or preserved as explicit historical paths; history is not silently rewritten to imply the new ownership always existed.
- A transferred history contains secret material. Transfer is blocked for security review; provenance preservation does not authorize copying credentials or signing material.

## Requirements *(mandatory)*

### Repository Ownership Boundary

| Artifact family | Required disposition | Ownership reason |
|---|---|---|
| Control Plane runtime | Keep in core | Owns Gateway, Catalog, Workspace, and Invocation Dispatch behavior and data |
| A2A Router runtime | Keep in core | Owns routing, task context, policy hooks, credentials, transport adaptation, and Ledger behavior |
| Language-neutral contracts and core conformance corpus | Keep canonical in core | Define the platform language and cross-boundary compatibility |
| Catalog, Workspace, and Ledger schema migrations | Keep with the owning core module | Define core persistent-state evolution and readiness expectations |
| Core service build definitions and unit, contract, and service integration tests | Keep in core | Required to release and verify core independently |
| Architecture decisions, compatibility records, usage documentation, and core CI policy | Keep in core | Govern core behavior and preserve durable operational facts |
| Production Console source and browser tests | Move to the existing Console owner | User interface has an independent source and release lifecycle |
| Public Agent and application SDK source | Move to versioned SDK owner(s) | Consumer libraries require independent releases and must not import service internals |
| Sample Agent runtimes and sample-only fixtures | Move to a sample owner | Demonstrate interoperability without becoming platform runtime logic |
| Multi-component development stack, environment assembly, and cross-product acceptance | Move to an integration or distribution owner | Compose released components and prove the whole product without owning component source |
| Reusable A2A wire transport mechanics | Keep in the existing standalone transport owner | Generic transport is reusable; core retains only NeKiro policy and contract adaptation |
| Service-specific packaging metadata | Keep with its service | Must version and release with the executable it packages |
| Frontend-only or satellite-only root tooling | Remove from core after cutover | A core checkout must not carry tooling solely for extracted source |
| Transient delivery records with no active governance role | Retain or archive only after reference and retention review | Repository size alone is not evidence that project history is disposable |

### Functional Requirements

- **FR-001**: The project MUST define NeKiro core as the platform-owned Control Plane, A2A Router, language-neutral contracts, owned persistent-state migrations, core verification, and governance artifacts required to release those capabilities.
- **FR-002**: The project MUST produce a complete artifact ownership record covering every tracked top-level family and root toolchain file before any source is removed.
- **FR-003**: Each ownership record MUST identify one canonical owner, intended consumers, release unit, dependency direction, compatibility evidence, and verification gate.
- **FR-004**: Control Plane and A2A Router production source MUST remain in the core repository and MUST NOT be split by internal domain directory or database schema as part of this feature.
- **FR-005**: Language-neutral Agent Card, API, event, credential, result, and A2A Profile contracts MUST remain canonical core assets and MUST be publishable for satellite conformance without exposing core service internals.
- **FR-006**: Every Catalog, Workspace, and Ledger migration required by a supported core release MUST remain versioned with its owning service.
- **FR-007**: Each owned schema version MUST have exactly one canonical migration representation; manually synchronized SQL files, embedded text copies, or generated mirrors MUST NOT remain competing sources.
- **FR-008**: Existing forward-only migration ordering, schema-version checks, database ownership, and explicit failure behavior MUST remain observably unchanged by repository extraction.
- **FR-009**: Operational database assets that do not define a core-owned application schema MUST be owned by the deployment or operations satellite and MUST NOT become an alternate schema authority.
- **FR-010**: The existing standalone Console project MUST become the sole production Console source; the core repository MUST contain no copied Console implementation or browser-only verification after cutover.
- **FR-011**: Public SDK source MUST be released outside the core repository and MUST consume only explicit published contracts and public platform endpoints.
- **FR-012**: Sample Agent runtime source MUST be released outside the core repository and MUST have no access to platform databases or core internal implementation packages.
- **FR-013**: Multi-component environment assembly, sample wiring, browser acceptance, and full Register -> Discover -> Install -> Invoke -> Record orchestration MUST have one external integration or distribution owner.
- **FR-014**: The core repository MUST retain NeKiro-specific adaptation for the standalone A2A transport release, including exact resolution, platform context, credential issuance, error mapping, result mapping, cancellation policy, and Ledger ownership.
- **FR-015**: Dependency direction MUST remain satellites -> published contracts/public endpoints and integration owner -> immutable component releases; core services MUST NOT depend on Console, SDK, sample, or distribution source.
- **FR-016**: Every cross-repository production dependency MUST identify an immutable reviewed version and declared compatibility range or identity.
- **FR-017**: Missing, unverified, retagged, or incompatible component releases MUST fail at dependency resolution or release acceptance with a distinct visible outcome.
- **FR-018**: The split MUST add zero copied-source, vendored-source, local-replacement, legacy-version, floating-branch, alternate-component, or stale-artifact fallback paths.
- **FR-019**: Each extraction cutover MUST establish the satellite as canonical and pass its independent release checks before removing the core copy, but no released state may contain two writable production authorities.
- **FR-020**: The extraction MUST NOT change wire APIs, language-neutral contract semantics, schema semantics, authentication, authorization, invocation routing, cancellation, failure mapping, or Ledger lineage. It MUST intentionally migrate the public Go source identity from `github.com/Nene7ko/NeKiro` to `github.com/NeKiro-project/NeKiro`, document the consumer import change, and provide no compatibility shim or legacy module fallback.
- **FR-021**: Core CI MUST verify core builds, unit tests, contract conformance, service integration, schema migration/readiness, static analysis, secrecy, and dependency boundaries without satellite source.
- **FR-022**: Each satellite MUST verify its own build, tests, contract compatibility, dependency boundaries, license, and secrecy before publication.
- **FR-023**: A separate product release gate MUST assemble explicit immutable releases and prove trusted publication, discovery, exact installation, JSON and streaming invocation, nested cross-runtime invocation, Ledger reads, negative failures, cancellation, and secrecy.
- **FR-024**: Root manifests, lockfiles, workflow jobs, configuration examples, and developer prerequisites used only by extracted artifacts MUST leave the core repository after their owner has a verified replacement.
- **FR-025**: Architecture decisions, contract compatibility history, and service usage documentation MUST remain tracked with the behavior they govern. Historical specifications MUST remain reachable through the pre-split tag and Git history but MUST leave the post-cutover tracked core tree together with Spec Kit. Satellite-only future delivery records MUST live with their satellite owner.
- **FR-026**: Source moves MUST preserve applicable license, authorship, provenance, issue and review references, and release traceability without transferring secret material.
- **FR-027**: Repository documentation MUST identify the canonical location, release channel, compatibility owner, and support boundary for every extracted artifact family.
- **FR-028**: The repository target structure in project governance MUST be amended through an explicit architecture decision before implementation removes currently mandated top-level product components.

### Key Entities

- **Core Artifact**: Source, contract, migration, test, or governance evidence required to define, release, or verify platform-owned Control Plane or Data Plane behavior.
- **Satellite Artifact**: Console, SDK, sample Runtime, deployment assembly, or product acceptance source with an independent owner and release lifecycle.
- **Artifact Ownership Record**: The classification of an artifact family, including canonical owner, consumers, release unit, dependency direction, compatibility evidence, and verification gate.
- **Owned Migration**: An ordered persistent-state transition released by the single core module that owns the affected schema.
- **Component Release**: An immutable reviewed publication of core or satellite behavior with a declared contract compatibility identity.
- **Integration Manifest**: The exact set of compatible component releases accepted together as one demonstrable NeKiro product build.
- **Extraction Cutover**: The reviewed transition at which a satellite release becomes canonical and the previous core copy and satellite-only tooling cease to be production sources.

### Runtime/Platform Boundary *(mandatory when feature touches Agent execution or integration)*

- **Platform-owned behavior**: Registration, trusted publication, discovery, Workspace installation, exact resolution, Router-mediated invocation, signed platform context, failure mapping, and Ledger lineage remain core-owned and behaviorally unchanged.
- **Runtime-owned behavior**: Sample and real Agent model, prompt, tool, workflow, memory, session, response generation, and runtime-specific fixtures remain outside core; moving samples does not make their behavior a platform contract.
- **Cross-runtime proof**: Product release acceptance MUST consume immutable releases of two independently implemented sample Agents, route a managed nested call through the core Router, and query one correlated Ledger lineage without importing either Runtime's internal source into core.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 100% of tracked artifact families and root toolchain files have exactly one approved ownership record before extraction begins.
- **SC-002**: The completed core repository contains zero production Console, public SDK, sample Runtime, full-stack assembly, or cross-product browser acceptance source files.
- **SC-003**: 100% of active Catalog, Workspace, and Ledger migration versions remain available from their owning core service, with exactly one canonical migration representation per version.
- **SC-004**: A fresh core checkout completes all core build, contract, unit, service integration, migration/readiness, static, dependency, and secrecy gates without checking out satellite source or installing satellite-only tooling.
- **SC-005**: A clean product acceptance run using one explicit immutable release set completes 100% of the existing Register -> Verify -> Publish -> Discover -> Install -> Invoke -> Record scenarios, including JSON, streaming, and nested cross-runtime lineage.
- **SC-006**: Public contract, persistent-state upgrade, error, authorization, cancellation, and Ledger acceptance comparisons report zero unintended behavioral differences before and after extraction.
- **SC-007**: Source and dependency scans report zero maintained cross-repository production source copies, zero local production replacements, zero floating production dependencies, and zero compatibility fallback implementations.
- **SC-008**: 100% of moved source families retain license, authorship, provenance, canonical documentation links, and independently passing release checks.
- **SC-009**: Every component compatibility failure is detected before product release, and no incompatible release set is reported as a degraded or partial success.

## Assumptions

- "Core" follows the amended project charter: platform-owned Control Plane and Data Plane behavior, canonical contracts, owned data evolution, and the verification and governance required to release them.
- The SQL currently in scope consists of core-owned Catalog, Workspace, and Ledger migrations rather than a separate database product; it therefore stays with those owners while duplicate representations are consolidated.
- The existing standalone Console and A2A transport projects provide established precedents for independent source ownership and immutable downstream consumption.
- Repository separation is source and release-boundary work only; it does not relocate live data, split the Control Plane into microservices, or change the separate Router deployment boundary.
- Exact satellite repository count, names, hosting, and release tooling will be selected during planning, but the ownership classes and dependency directions in this specification are binding.
- Historical accepted specifications remain available through the annotated pre-split tag and Git history. Architecture decisions, contracts, compatibility records, and usage documentation remain tracked core evidence.

## Non-Goals

- Moving all SQL into a standalone database repository or treating SQL as non-core solely because of its file type.
- Separating Catalog, Workspace, Invocation Dispatch, Ledger, or Router internals into additional service repositories.
- Changing PostgreSQL, migration direction, data ownership, schema meaning, or supported database upgrade history.
- Redesigning public APIs, contracts, authentication, authorization, invocation, streaming, cancellation, or Ledger behavior.
- Removing core tests, architecture decisions, compatibility records, usage documentation, or required historical migrations merely to reduce repository size.
- Reimplementing the already extracted A2A transport, retaining a private fallback transport, or moving NeKiro policy into the transport project.
- Adding deployment targets, runtime orchestration, Marketplace behavior, SDK languages, or Agent Runtime capabilities.
- Selecting a source hosting vendor, repository naming convention, history-transfer tool, package registry, or release automation mechanism during specification.
- Supporting an indefinitely mixed state in which core and a satellite both accept production edits to the same source.
