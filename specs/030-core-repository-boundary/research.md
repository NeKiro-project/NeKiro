# Research: Core Repository Boundary

## Decision 1: Split by ownership and release unit

**Decision**: Keep Control Plane, Router, contracts, owned migrations, and core
tests in `NeKiro`; use `NeKiro-Console`, `nekiro-sdk-go`, `NeKiro-Samples`, and
`NeKiro-Stack` for the extracted production families.

**Rationale**: These components have different toolchains, consumers, and
release cadence. Core services do not need their source to build or verify
their own behavior.

**Rejected**: A single `extras` repository. It would preserve unrelated release
coupling and would not establish clear canonical owners.

## Decision 2: Keep SQL with the owning service

**Decision**: Move canonical migration files under the Catalog, Workspace, and
Ledger PostgreSQL packages and load them through `go:embed`.

**Rationale**: The SQL defines service-owned persistent-state evolution. A
separate database repository would reverse ownership and introduce release
skew. Current duplicates include identical root/internal SQL copies and raw Go
string copies.

**Rejected**: Extract all `.sql` files by extension. File type is not an
architecture boundary.

## Decision 3: Preserve source history before cleanup

**Decision**: Create an annotated tag at the accepted pre-split commit, export
`sdks/` and `agents/` with `git subtree split`, and export the union of
`deploy/` plus `tests/e2e/` from a clean temporary clone. Push only explicit
export branches to empty target repositories.

**Rationale**: `git filter-repo` is unavailable, while Git 2.54 provides
`subtree`. Stack needs a union of paths and therefore uses an isolated
history-filtering clone. The source tag preserves original commit identities
when filtered histories necessarily rewrite commits.

**Rejected**: Copying current files into new initial commits. It would discard
authorship and source lineage. `--mirror` and `--all` are rejected because they
could push unrelated refs.

## Decision 4: Adopt the official Go module identity

**Decision**: Change the core module to
`github.com/NeKiro-project/NeKiro` before downstream modules are released. The
SDK module is `github.com/NeKiro-project/nekiro-sdk-go`.

**Rationale**: The current `github.com/Nene7ko/NeKiro` path does not match the
canonical repository and has no published tag compatibility to preserve. New
repositories must not depend on a personal namespace or local `replace`.

**Migration impact**: Existing Go source consumers must change their module
requirements and imports to `github.com/NeKiro-project/NeKiro`. No legacy
module, forwarding package, or compatibility replacement is published.

**Rejected**: Retaining the personal path indefinitely or masking it with Go
module replacements. Both make the published dependency identity ambiguous.

## Decision 5: Product acceptance belongs to Stack

**Decision**: `NeKiro-Stack` owns Compose, release pins, backend acceptance,
browser acceptance, log sanitization, and the complete cross-runtime loop.

**Rationale**: Only an integration owner should assemble multiple independently
released components. Core and Console CI must remain independently runnable.

**Rejected**: Having Console check out core, or core build Console and samples.
Those patterns recreate cross-repository source coupling.

## Decision 6: Use immutable component identities

**Decision**: Stack records exact Git commit SHAs initially and moves to exact
release tags and OCI digests as images are published. CI rejects `latest`,
branches, local paths, `replace`, and local production `build:` entries.

**Rationale**: Explicit manifest updates retain review evidence and make an
incompatible dependency fail before deployment.

**Rejected**: Automatically following the newest release, floating branches,
or cached older components. They make product identity nondeterministic.

## Decision 7: Align CI contracts, not job contents

**Decision**: Every repository uses workflow name `CI`, least privilege,
`pull_request` plus `push` to `main`, concurrency cancellation, explicit
timeouts, Actions pinned to full commit SHAs, pinned toolchain versions, and a
final `required` job. Go owners run
format, build, test, race, vet, and tidy checks; each owner adds only its own
integration jobs.

**Rationale**: A stable aggregate gate makes repository rules consistent while
allowing Console, libraries, services, samples, and Stack to verify the behavior
they own.

**Rejected**: One reusable workflow hosted by core. It would make every
repository's CI availability depend on another repository.

## Decision 8: Retire tracked Spec Kit after the transition

**Decision**: Spec 030 is the final tracked feature. The post-cutover core tree
removes `specs/`, `.specify/`, and `.agents/skills/speckit-*`. Durable facts move
to AGENTS.md, ADR 0009, contracts, usage docs, GitHub records, releases, and the
Wiki. The annotated tag and Git history retain historical Spec content.

**Rationale**: The owner selected repository-local Issues, ADRs, pull requests,
checks, and releases as the future workflow and wants the core checkout focused
on implementation.

**Rejected**: Keeping historical Spec directories in the default branch. Git
already preserves them and the tracked tree would continue carrying delivery
narrative rather than current core behavior.

## Constraints Found During Audit

- The standalone Console must first receive the public Agent share files and
  related accepted changes from core.
- Two Router tests import `agents/runtime-b`; they must use core-owned protocol
  fixtures before `agents/` is deleted.
- Runtime A currently uses `replace github.com/Nene7ko/NeKiro => ../..`; this is
  forbidden after extraction.
- Sample Dockerfiles copy contracts and SDK source from the monorepo; standalone
  builds must consume published Go modules.
- Core CI does not currently execute the Ledger PostgreSQL integration suite.
- The private Console repository cannot receive GitHub rulesets on the current
  organization plan. CI is still mandatory; equivalent branch protection
  requires making it public or changing the GitHub plan.

## Fallback Assessment

No new fallback is authorized. The preliminary phase temporarily retains three
existing paths: Runtime A's local core-module replacement and the two sample
Docker builds that consume monorepo source. They are explicitly removed by
T027/T028 after history export. Copied source, local module replacements,
floating refs, alternate components, and stale component substitution are all
absent from the final state.
