# ADR 0020: Layered Error Ownership

- Status: Accepted for issue #116
- Date: 2026-08-12
- Decision owner: Platform Architecture
- Related: `nekiro-sdk-go#3`, `nekiro-sdk-go#4`, `NeKiro-Samples#11`, and
  `NeKiro-Samples#12`

## Context

Core domain packages, cross-process contracts, and Agent Runtime lifecycle
code need different error semantics. Treating them as one global taxonomy
would couple independent owners and make dependency details easier to leak
across trust boundaries.

This is the same useful separation visible in Dubbo-Go: a protocol package owns
wire error codes and translation, while individual subsystems classify their
own failures. A caller can handle a stable boundary result without importing
every implementation package or parsing provider text.

The Runtime lifecycle also repeats setup, serving, lease observation, and
shutdown handling in each Sample. The SDK needs to provide that mechanism
without taking ownership of Runtime-specific configuration, handlers,
transport security, or policy.

## Decision

Errors remain layered by ownership. There is no global Core `errors` package,
catch mechanism, or flattened taxonomy.

| Owner | Owns | Boundary rule |
| --- | --- | --- |
| `config_center` | Provider-neutral source outcomes such as missing, invalid, unauthorized, unavailable, cancellation, and revision/watch failures. | Providers map their dependency responses into these classifications without exposing provider payloads. |
| `registry` | Agent Card, Release, registration, lease, and watch invariants. | Registry callers use `errors.Is/As` for local decisions; they do not infer meaning from error strings. |
| Control Plane and Router domains | Catalog, Workspace, Gateway, dispatch, authentication, resolution, transport, and Ledger failures. | Each owning package maps its dependencies explicitly before returning to another Core boundary. |
| `contracts` | Stable cross-process `PlatformErrorCode` values and versioned error shapes. | HTTP, A2A, SSE, and Ledger adapters emit only the contract-defined code, message, phase, and correlation fields. |
| `nekiro-sdk-go/agent/host` | Runtime lifecycle stages: `config`, `registration`, `handler`, `serve`, and `shutdown`. | Host errors expose the stage and static safe text while preserving the cause for `errors.Is/As`; Runtime packages own configuration and policy errors. |

### Translation and safety rules

1. A package may introduce a sentinel or typed error only for a fact it owns.
   Wrapping preserves the owner error for local `errors.Is/As` checks.
2. A process or protocol boundary maps the complete error chain once. It must
   select a versioned contract code and safe message; it must not pass through
   dependency response bodies, URLs, credentials, PEM/key bytes, request or
   Agent payloads, SQL details, or arbitrary cause text.
3. `PlatformErrorCode` is not a replacement for local errors. A local error
   may map to one code in one boundary and a different code in another phase
   when the contract requires it; the mapping is documented and tested at that
   boundary.
4. Cancellation, deadline, missing, invalid, unauthorized, unavailable,
   dependency, and internal outcomes remain distinct. No empty result, retry,
   alternate endpoint, stale snapshot, or old revision may hide one outcome as
   another without explicit policy evidence.
5. Logs and metrics may classify an error more coarsely, but that projection is
   not a new public error contract and must use redacted fields.

### Runtime host boundary

`agent/host` starts the supplied HTTP server, registers and observes the
optional lease, and performs bounded shutdown and deregistration. It does not
load deployment configuration, construct a registry provider, validate Router
credentials, or implement Agent behavior. Runtime setup wraps failures with a
host stage; the host owns only lifecycle failures after setup.

## Compatibility

- Existing owner-local errors and `contracts.PlatformErrorCode` values remain
  unchanged.
- The SDK host is additive. Runtime implementations may adopt it without
  changing the Agent Card, Release, Router, or Ledger contracts.
- Adding a new public code, changing a code/message, or moving an error owner
  requires a contract/ADR review and an explicit compatibility window.

## Consequences

- Core packages remain independently testable and do not depend on a shared
  implementation-only error taxonomy.
- Boundary handlers have a single, reviewable place for redaction and wire
  translation.
- Runtime Samples share lifecycle behavior while retaining their own setup,
  security, and handler ownership.
- Operators can distinguish lifecycle stage and stable boundary code without
  receiving secrets or dependency internals.

## Rejected alternatives

- A root-level `errors` package containing every Core failure.
- A global `recover`/catch layer that turns all failures into one response.
- Parsing `error.Error()` strings to decide authorization, availability, or
  retry behavior.
- Returning raw Nacos, HTTP, SQL, Router, or Agent error text to callers.

## Fallback Delta

Fallback delta: removed 0, retained 0, added 0, net +0.

Added fallback evidence: none.
