# Research: Install a Published Agent and Pin an Exact Version

## Observed Baseline

- The active v3 route, Workspace service, Catalog selection port, PostgreSQL
  uniqueness transaction, and Installation v2 validator already exist on the
  dependent branch.
- Existing service tests cover one stable selection, build-metadata tie, and a
  basic owner/lifecycle workflow, while integration covers a broad workflow and
  a 100-request race.
- Focused #5 evidence is missing for HTTP required-array presence, explicit
  empty permissions, dependency precedence, restart equality, and newer
  publication immutability.
- The current subset helper can turn an explicit empty slice into a nil slice;
  the PostgreSQL `NOT NULL` array and JSON response require a non-nil empty
  representation.
- The current Installation insert returns the pre-storage Go value, so
  PostgreSQL timestamp precision can differ from a later read.

## Decisions

### Keep Catalog selection behind the existing port

Catalog remains the sole owner of published visibility, SemVer precedence,
pre-release policy, and exact Card facts. Installation calls the port and never
reads Catalog tables or creates an alternate candidate list.

### Validate request presence at the Gateway boundary

The shared Go request DTO represents the decoded values, not JSON member
presence. The HTTP adapter therefore uses a private wire shape for the install
request and rejects missing/null `acceptedPermissions` while accepting `[]`.
This preserves the language-neutral contract without adding transport state to
the domain type.

### Preserve empty permissions as an explicit array

The domain constructs a non-nil zero-length slice for an empty accepted set.
This is the approved product value required by the contract, not a fallback.

### Return committed database facts

The insert uses `RETURNING` for all Installation fields. The returned row is the
single response fact, so timestamps and array representation are identical to
restart reads.

## Fallback Inventory

| Candidate | Classification | Evidence |
| --- | --- | --- |
| Empty accepted-permission array | Keep | Explicit #3/#5 product policy; it authorizes only permission-free capabilities |
| Wildcard/default SemVer constraint | Remove | Invalid or missing input is `VALIDATION_ERROR` |
| Stale/disabled Card as a selection source | Remove | Catalog publishes only eligible candidates |
| Alternate Catalog query/cache/retry | Remove | Catalog port is the sole approved source |
| Auto-upgrade after newer publication | Remove | Exact pin is immutable authorization evidence |
| Anonymous/default owner | Remove | Trusted Gateway identity is mandatory |

Fallback delta for this feature: removed `0`, retained `1`, added `0`, net `0`.
Added fallback evidence: the explicit empty permission set is approved product
semantics; no degraded dependency behavior is retained.
