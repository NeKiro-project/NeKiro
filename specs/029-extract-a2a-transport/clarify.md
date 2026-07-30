# Clarification Record: Extract Reusable A2A Transport

**Date**: 2026-07-30

No critical ambiguity required user interruption. The repository evidence and
the user's selected upstream resolve the high-impact decisions:

- Scope is the reusable client transport, strict JSON-RPC/SSE validation,
  transport-level failure taxonomy, and executable profile conformance.
- NeKiro retains target resolution, Card/capability limits, request-scoped
  signed credentials, platform context, Platform Error and result-event
  mapping, Router orchestration, Ledger, and terminal race ownership.
- The standalone module depends on the active protocol-focused A2A library,
  not on NeKiro platform packages or an Agent Runtime framework.
- The first release is consumed as an immutable tag. A local `replace` may be
  used only while proving the two-repository migration and is not committed.
- Existing wire contracts and acceptance behavior are the compatibility
  baseline. Any discovered behavioral change returns to Spec/ADR review.
- Nil HTTP transport behavior and one bounded remote cancellation propagation
  attempt are retained only under their existing documented policies. No new
  fallback, retry, alternate source, or compatibility path is permitted.

## Coverage Summary

| Category | Status | Evidence |
| --- | --- | --- |
| Functional scope and actors | Clear | Four prioritized user stories and explicit non-goals |
| Domain and ownership | Clear | Key entities plus Router/platform/runtime boundary |
| Failure handling | Clear | Edge-case matrix and FR-005 through FR-010 |
| Security and privacy | Clear | Explicit injection ownership, secrecy, and no-default rules |
| External dependencies | Clear | User-selected upstream and active A2A Profile pin |
| Compatibility and release | Clear | Tagged release, no committed replacement, downstream gates |
| Performance and scale | Deferred to planning | Exact byte bounds already remain caller supplied; no new throughput behavior is introduced |
| UX/accessibility/localization | Not applicable | This is a transport-library and Router dependency extraction |
| Completion signals | Clear | Eight measurable outcomes and two-repository verification |

Specification quality remains 16/16 checklist items passing.
