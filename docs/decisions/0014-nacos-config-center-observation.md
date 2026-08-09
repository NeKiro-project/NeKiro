# ADR 0014: Nacos Config Center Observation

- Status: Accepted for issue #88 watch conformance
- Date: 2026-08-09
- Decision owner: Platform Architecture
- Related issue: [#88](https://github.com/NeKiro-project/NeKiro/issues/88)
- Extends: [ADR 0013](0013-nacos-runtime-provider-slice.md)

## Context

ADR 0013 introduced an intentionally snapshot-only Nacos Config Center reader.
Issue #88 also requires the provider-neutral observation contract: a gap-free
initial/watch handoff, ordered local revisions, explicit deletion, bounded
delivery, and terminal interruption without cached-state fallback.

Nacos Config long polling compares an MD5 digest supplied in a
`Listening-Configs` record. An empty successful response is a normal long-poll
timeout. A changed-key response identifies the tuple but does not carry the new
configuration content, so the reader must perform one exact bounded GET after
the notification.

Provider-neutral keys allow slash-separated segments, while Nacos rejects `/`
in a dataId. Passing only the intersection of both key spaces would make the
Nacos reader fail the provider-neutral conformance contract.

## Decision

1. Nacos observation is an explicit reader option. Snapshot-only construction
   remains valid and continues to return `unsupported` from `Observe`.
2. An enabled observation supplies one bounded long-poll timeout and one
   bounded subscription queue. The reader issues
   `POST /nacos/v1/cs/configs/listener` with the exact namespace, group, dataId,
   current MD5, and `Long-Pulling-Timeout`.
3. The initial GET completes before the subscription starts. Its exact digest
   is used by the first listener request, so a concurrent transition is either
   represented by the initial snapshot or reported by Nacos. There is no
   Get-then-watch loss window.
4. An empty successful listener response continues the same observation with a
   new long-poll request. It is protocol continuation, not recovery from a
   failure. HTTP, authorization, response, read, revision, or queue failure is
   terminal and interrupts the reader. There is no retry after failure,
   resubscription, cache, alternate server, or stale-value delivery.
5. A changed tuple causes one exact bounded GET. A present value emits update;
   absence emits delete; an unchanged state is suppressed and counted. Config
   content and listener metadata use separate bounds.
6. Legal Nacos dataIds that do not start with `nekiro.key.v1.` retain their
   exact text. Keys containing `/`, plus keys occupying that reserved prefix,
   map to `nekiro.key.v1.` followed by unpadded Base64URL of the complete key.
   The reserved-prefix rule makes this mapping collision-free while preserving
   existing deployed dataIds such as `router.nacos-bindings`.
7. Observation revisions remain local to one `Observe` call. Nacos MD5 and
   provider revisions are not exposed as Core revisions or recovery cursors.
8. The Router continues to compose the Nacos Config reader in snapshot-only
   mode for this change. Adopting dynamic Router configuration is a separate
   application policy decision with its own readiness and field-acceptance
   semantics.

## Consequences

- The Nacos Config reader passes the shared read, observation, interruption,
  and close conformance suites when observation is enabled.
- Existing Nacos runtime assembly keeps its current exact binding dataId.
- Operators publishing a slash-separated key must publish to its documented
  mapped dataId; Core does not gain publishing authority.
- Nacos Naming observation in issue #89 remains a separate protocol slice.

## Fallback Delta

Fallback delta: removed 0, retained 0, added 0, net +0.

Added fallback evidence: none.
