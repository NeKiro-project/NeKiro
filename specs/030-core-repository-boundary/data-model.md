# Data Model: Repository Extraction

This feature changes source and release ownership, not runtime database data.
The entities below are governance records used to make cutover deterministic.

## Artifact Ownership Record

Fields:

- `artifactFamily`: stable path or source family name
- `canonicalRepository`: one GitHub repository
- `ownerBoundary`: domain or product responsibility
- `consumers`: repositories or release processes that depend on it
- `releaseUnit`: module, application, image, or manifest publication
- `dependencyDirection`: owner-to-consumer direction
- `compatibilityEvidence`: contract version, module version, commit, or digest
- `verificationGate`: repository-local check that must pass
- `disposition`: `retain`, `extract`, `deduplicate`, or `remove-after-cutover`

Invariants:

1. Every tracked family has exactly one record and one canonical repository.
2. An extracted family cannot be deleted from core until its satellite gate
   passes and its canonical documentation resolves.
3. No production family has two writable authorities after cutover.

## Component Release

Fields:

- `repository`
- `commitSha`
- `releaseTag` when published
- `modulePath` for Go modules
- `contractIdentity`
- `artifactChecksums`
- `ociDigest` for images
- `ciRunUrl`

Invariants:

1. Commit SHA is exactly 40 lowercase hexadecimal characters.
2. Tags and digests are immutable after acceptance.
3. A missing identity fails assembly; it is not replaced by an older release.

## Integration Manifest

Fields:

- `schemaVersion`
- `core`
- `console`
- `sdkGo`
- `samples`
- `transportGo`
- component image digests
- contract compatibility identity
- acceptance timestamp and run URL after success

Invariants:

1. Every component is explicit and immutable.
2. The manifest contains no branch, `latest`, local path, or unverified digest.
3. Product acceptance runs exactly the manifest it reports.

## Extraction Cutover

States:

```text
inventoried -> history_exported -> satellite_verified -> core_copy_removed -> product_verified -> merged
```

Transitions:

- `inventoried -> history_exported`: source tree and provenance checks pass.
- `history_exported -> satellite_verified`: target repository builds and tests
  independently against explicit upstream identities.
- `satellite_verified -> core_copy_removed`: core imports and tooling no longer
  require the family.
- `core_copy_removed -> product_verified`: Stack assembles exact revisions and
  passes the complete acceptance.
- `product_verified -> merged`: independent review and all required CI pass.

Failure semantics:

- Any provenance mismatch stops export.
- Any missing or incompatible component stops before Compose deployment.
- Any runtime or acceptance failure remains a failed cutover; no partial-success
  state is published.

## Owned Migration

Fields:

- `owner`: `catalog`, `workspace`, or `ledger`
- `version`: positive ordered integer
- `canonicalPath`
- `contentDigest`
- `expectedSchemaVersion`
- `freshInstallEvidence`
- `upgradeEvidence`
- `readinessEvidence`

Invariants:

1. One canonical file exists for each owner/version pair.
2. Embedded execution reads the canonical file set directly.
3. Existing forward-only ordering and readiness shape remain unchanged.
