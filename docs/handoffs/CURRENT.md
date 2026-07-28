# Current Handoff: Console Trusted Publication Loop

**Updated**: 2026-07-28 (Asia/Hong_Kong)

**State**: Spec 027 implementation is complete on the stacked delivery
branches. The production Console proves the Gateway-only
`Register -> Verify -> Publish -> Discover -> Install -> Invoke -> Record`
workflow and displays the reverse `Runtime B -> Router -> Runtime A` lineage.

## Delivery

- Slice C: upstream PR #63,
  `codex/027-slice-c-reverse-lineage`, reverse cross-runtime acceptance.
- Slice D: upstream PR #64,
  `codex/027-slice-d-console-browser`, stacked on Slice C; production Console,
  CI discovery, fresh Compose browser acceptance, and isolated demos.
- Standalone Console source: NeKiro-Console PRs #5, #6, and #7.
- Required commit identity: `Nene7ko_ <1604009816@qq.com>`.

## Verification

- Slice D CI run `30322101411`: seven workflow jobs plus the Codecov patch
  check passed, including frontend, backend acceptance, runtime samples, Compose configuration, and
  `console-browser-acceptance`.
- Frontend local checks: `pnpm typecheck`, `pnpm test` (40/40), `pnpm build`,
  and `git diff --check` passed.
- Browser acceptance asserts Gateway origin, JSON/SSE correlation, reverse
  Trace/Release provenance, exact negative behavior, challenge-proof absence,
  and zero API traffic on the four `#/demo*` routes.
- PR #63's fork-only Codecov upload check is a non-blocking infrastructure
  limitation; its executable Go, backend, workspace, runtime, frontend, and
  Compose checks passed.

## Process Record

T005 remains unchecked because Slice A's issue-scope review was performed
retrospectively during convergence. This historical ordering deviation is
preserved and must not be rewritten as a satisfied pre-implementation gate.
No new runtime, contract, fallback, retry, alternate endpoint, direct Agent,
or SQL mutation path is approved by this handoff.
