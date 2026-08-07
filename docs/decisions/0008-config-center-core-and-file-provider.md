# ADR 0008: Config Center Core and Deterministic File Provider

- Status: Accepted
- Date: 2026-08-07
- Feature: Spec 025 Config Center Core and File Adapter / GitHub #88

## Context

The trusted invocation loop has strict, independently owned environment-backed Control Plane and Router configuration. Future operational governance needs an authoritative-source boundary without becoming an Agent Runtime, Registry, authorization system, or global configuration singleton.

Issue #88 is a parent tracker. Nacos, loader injection, typed service documents, readiness, and runtime policy capture remain unresolved in ownership and failure policy. The accepted first slice is only generic source semantics and one deterministic local adapter.

## Decision

Add root `config_center/` (Go package `configcenter`) to own opaque-byte source semantics: strict keys; immutable present/missing snapshots; one-shot read and atomic initial-state/subscription observation; observation-scoped revisions; explicit update/delete events; typed failures; immutable reader composition; and separate administrative publisher authority.

`config_center/file` is the sole approved provider. It uses one explicit root
watch, deterministic reversible flat mapping, explicit payload/buffer/mode
inputs, and independent reader/publisher constructors. It uses
`github.com/fsnotify/fsnotify` v1.10.1 for non-Darwin watching and
`golang.org/x/sys/unix` for the native Darwin kqueue source. Go 1.26 `os.Root`
confines mapped access to the root opened at construction; an observed symlink
or non-regular leaf is rejected as typed `unsafe_state`. Root acquisition
requires a before-open non-symlink-directory `Lstat`, `Root.Stat(".")`, and
after-open `Lstat` with one `os.SameFile` identity. Reader repeats the
configured-path/pinned-identity check after platform-source registration before it
serves; an observed substitution is rejected without retry, reopen, or source
switch.

The File adapter is only local-development/conformance infrastructure. It does
not claim a hostile-filesystem race guarantee: a privileged actor can replace a
root pathname or in-root leaf after the final applicable check, and `os.Root`
intentionally retains a moved root.
Direct external writers must use atomic replacement and must not create mapped
symlinks or non-regular files. When the single root watch receives rename,
delete, replacement, or root-path-symlink evidence, the reader terminally
interrupts and never reopens or auto-switches to a replacement root. Linux,
Darwin, and Windows are the native test matrix; JavaScript and Plan 9 are
explicitly unsupported.

The publisher saves the root pathname and pinned identity and verifies it at
operation start, immediately before its final `Rename`/`Remove`, and before a
success return. Missing root is `unavailable`; permission is `unauthorized`;
and a symlink, non-directory, or identity replacement is `unsafe_state`. It
writes an explicit-mode temporary leaf inside the pinned root, syncs/closes it,
then atomically replaces the mapped file. This guarantees complete old-or-new
visibility to live-process readers only; it does not promise crash/power-loss
directory-entry durability or add a directory `fsync`. Initial reading,
subscription registration, and root-event processing share one synchronization
boundary: a concurrent state change is initial state or next event, never
silently between them.

On Windows, `FileMode` is limited to the exact projections supported by
`os.Chmod`: writable `0666` or read-only `0444`. POSIX-granular modes such as
`0640` are invalid rather than silently accepted with weaker semantics.

Missing and present-empty are distinct. Delete is explicit, including typed
missing for delete of an absent key. Snapshot bytes are copied at every
ownership/return boundary. `Get` returns an unscoped, non-comparable snapshot
revision; only `Observe.Initial` and delivered events carry a local observation
scope/order. File revisions are not persistent versions, restart cursors, or
external revisions. The core separately identifies duplicate, stale, gap, and
out-of-order input.

`Key` is exact ASCII `segment ("/" segment)*`, where each segment matches
`[A-Za-z0-9][A-Za-z0-9._:-]{0,31}` and a key is at most 160 bytes.
`ProviderID` is the no-slash form with a 128-byte maximum. The flat RawURL
Base64 mapping has a maximum 227-byte leaf (`7 + 214 + 6`), below the
native-platform 255-byte component bound.

Repeated OS notifications are suppressed only when observable state is unchanged,
with a safe aggregate count. `EACCES`/`EPERM` maps to `unauthorized`; observed
symlink/non-regular leaf to `unsafe_state`; oversize to `payload_too_large`; and
other File I/O to `unavailable`. After subscription registration, a failed state
sample, platform watcher overflow, closed watcher channel, root event, or delivery
overflow is terminal `watch_interrupted`. A received non-Darwin fsnotify error
is also terminal unless `errors.Is(err, fs.ErrNotExist)` and one immediate check
proves the configured root path and pinned root identity are exactly intact; that
exact case reconciles only registered keys once and continues the existing
watcher. Darwin kqueue errors are terminal. It is provider-signal handling, not
polling, retry, reopen, resubscription, cache, or source fallback. A state-sample interruption can carry only the safe
cause kind, never a raw path/error/payload. There is no poll, reopen, retry,
resubscribe, cache, old snapshot, alternate source, default directory/provider,
or inferred permission.

On Darwin, Reader uses one native root-directory kqueue watcher as its sole
source. A root Write reconciles registered keys once; root delete/rename,
watcher error, and closure are terminal; `NOTE_ATTRIB` revalidates root identity
and interrupts only when unsafe. fsnotify is not
opened or composed on Darwin, and no marker leaf is used. This observes final
state from Publisher and independent external atomic writers; it is not polling,
retry, reopen, resubscription, cache, alternate source, or fallback. Non-Darwin
retains fsnotify as its platform-primary watcher.
An `EINTR` from the Darwin kqueue syscall completes the same registration,
receive, or close-wake operation on that already registered source. It is not a
received watcher error, provider retry, reopen, resubscription, or fallback.

Errors/status metadata may include only provider, key, operation, revision/order, and typed classification; never payload, decoded configuration, credentials, tokens, private keys, or inferred values.

## Deferred Boundary

This ADR does **not** approve Nacos; Control Plane/Router loader injection; service/governance schemas; typed validation, readiness, or dynamic application; per-Invocation policy capture; secret distribution; Registry/Directory/external Gateway/load balancing; retry/failover; or dynamic authorization/release selection.

Each deferred area needs a separate Spec/ADR. This is a Phase 2 operational-governance blocking prerequisite, not authorization to implement Phase 2 behavior early.

## Compatibility

This adds a root library and semantic contract only. Existing service loaders, schemas, startup behavior, APIs, Agent Card/Release provenance, Workspace authorization, Invocation behavior, and Ledger remain unchanged. No runtime dual-read or compatibility source is introduced.

Future providers must pass the v1 conformance suite. Breaking semantics need a new version and migration decision, never a hidden provider compatibility path.

## Consequences

- Future services can receive read authority without publish/delete authority or provider SDK imports.
- File is deterministic local-development/conformance infrastructure, not production availability or durable event replay.
- Watch interruption is visible; future readiness/governance integration must choose its response explicitly.
- The platform remains runtime agnostic: this is not an Agent Runtime or Agent Card/Registry replacement.

## Rejected Alternatives

- Implement Nacos/service integration now: exceeds Spec 025 and guesses validation, readiness, credential, and recovery policy.
- Use `Get` followed by `Watch`: loses handoff changes.
- Use nested paths/recursive watches: fsnotify has no public portable recursive watch guarantee.
- Poll/reopen/resubscribe/cache after failure: hides interruption and violates zero-fallback policy.

## Fallback report

```text
Fallback delta: removed 0, retained 0, added 0, net +0
Added fallback evidence: none
```
