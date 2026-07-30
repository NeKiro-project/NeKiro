# Implementation Plan: Extract Reusable A2A Transport

**Branch**: `codex/029-extract-a2a-transport` | **Date**: 2026-07-30 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/029-extract-a2a-transport/spec.md`

## Summary

Extract the Router's reusable HTTP/JSON-RPC/SSE client mechanics into the
standalone `NeKiro-project/nekiro-a2a-transport-go` module, publish a tagged
release, and replace the private implementation with a thin Router-owned
adapter. The upstream module owns explicit call configuration, redirect
rejection, strict JSON-RPC envelope checks, bounded response/event reads, exact
SSE framing, raw stream-result preservation, supported event-kind checks, and
transport-level failure categories. NeKiro continues to own the active A2A
Profile contract, exact Agent resolution, Card/capability limits, signed
credential issuance, platform context, Task/event semantics, Platform Error
mapping, result-event mapping, cancellation policy, Ledger, and terminal races.

## Technical Context

**Language/Version**: Go 1.26.0 in both repositories

**Primary Dependencies**: Go `net/http`, `encoding/json`, and `iter`;
`github.com/a2aproject/a2a-go v0.3.15`; NeKiro active language-neutral A2A
Profile 0.2 / protocol 0.3.0 remains downstream contract authority

**Storage**: N/A in upstream; existing Router-owned PostgreSQL Ledger remains
downstream and is not imported by the standalone module

**Testing**: Go unit and `httptest` wire tests in upstream; existing NeKiro
contract, Router unit/integration, cross-runtime, race, and invoke-to-record E2E
tests downstream

**Target Platform**: Portable Go library and Linux CI; downstream Router
container remains the production consumer

**Project Type**: Standalone protocol transport library plus an existing
data-plane service adapter

**Performance Goals**: Preserve streaming behavior without full-stream
buffering; bound each non-stream response and complete SSE event using explicit
caller-supplied byte limits; add no retry, polling, reconnect, or background
processing

**Constraints**: No NeKiro package import upstream; no default endpoint,
credential, byte limit, retry, alternate route, legacy mode, truncation, or
success fabrication; committed downstream dependency must use an immutable tag
with no local `replace`

**Scale/Scope**: One root upstream package, three public call operations
(`message/send`, `message/stream`, `tasks/cancel`), one typed failure taxonomy,
one NeKiro adapter, and the current positive/negative wire corpus

## Constitution Check

*GATE: Passed before research and re-checked after design.*

- **Phase 1 loop**: PASS. The extraction strengthens the existing Invoke leg
  by making the Router's protocol transport independently releasable and
  reusable without altering the user-facing loop.
- **Ownership**: PASS. Upstream owns transport mechanics only. Router retains
  routing, resolution, authorization, credentials, task policy, event mapping,
  and Ledger ownership.
- **Runtime independence**: PASS. The standalone dependency is protocol-only,
  imports no Runtime framework, and is exercised by Runtime A and Runtime B.
- **Contracts**: PASS. No language-neutral contract changes. A2A Profile 0.2 /
  protocol 0.3.0 and `a2a-go v0.3.15` remain explicit compatibility inputs.
- **Invocation lineage**: PASS. Lineage headers and signed claims remain in the
  Router adapter; upstream accepts only explicit caller-owned interceptors.
- **Failure safety**: PASS. Transport failures are typed and mapped
  downstream; no retry, alternate endpoint, default, truncation, or secret
  logging is introduced.
- **SDD traceability**: PASS. Upstream implementation precedes its mapped tests;
  downstream replacement follows a released tag and is covered by existing
  acceptance suites.
- **Cross-runtime proof**: PASS. Existing Runtime B direct A2A and Runtime A
  framework-backed acceptance continue through the same Router adapter.

**Post-design gate**: PASS. The public API exposes no NeKiro-owned domain type,
and the dependency direction remains Router -> standalone transport ->
official protocol library.

## Research Decisions

See [research.md](research.md). The binding decisions are:

1. Publish one root package named `a2atransport` at
   `github.com/NeKiro-project/nekiro-a2a-transport-go`.
2. Require explicit per-call endpoint and byte bounds. Do not add defaults.
3. Accept caller-owned `a2aclient.CallInterceptor` values for request metadata;
   the library owns no NeKiro credentials or headers.
4. Return official A2A result/event values plus an immutable copy of the raw
   streaming JSON-RPC `result`; preserve platform event mapping downstream.
5. Expose a small typed failure taxonomy and preserve original causes through
   `errors.Is`/`errors.As` without including cause text in the public error
   string.
6. Keep Profile fixtures and Task-state mapping in NeKiro contracts; upstream
   declares compatibility and independently tests extracted wire behavior.
7. Release upstream `v0.1.0`, then the conformance-complete patch `v0.1.1`, before finalizing the NeKiro dependency. A local
   replacement is allowed only during verification and is removed before commit.
8. Preserve Apache-2.0 licensing and add source-origin attribution.

## Public API and Ownership

The planned API is documented in
[contracts/transport-public-api.md](contracts/transport-public-api.md). The
important direction is:

```text
NeKiro Router adapter
  -> a2atransport Client
     -> a2a-go JSON-RPC client
        -> external Agent

NeKiro contracts/Profile
  -> validate/map official A2A results and events

a2atransport -X-> NeKiro contracts, Router, credentials, Ledger, Runtime
```

The upstream call boundary receives only endpoint, explicit response/event
bounds, official A2A request types, and caller-supplied interceptors. The
downstream adapter computes effective configured/Card bounds, creates a fresh
credential interceptor for every operation, and maps `FailureKind` into the
active Platform Error contract.

## Fallback Inventory

| Behavior | Disposition | Evidence | Migration rule |
| --- | --- | --- | --- |
| Nil `http.Client.Transport` uses `http.DefaultTransport` | Keep | Go `net/http` contract and ADR 0007 | Preserve exactly; nil `*http.Client` still fails construction |
| One bounded `tasks/cancel` propagation after local timeout/cancel | Keep downstream | ADR 0006 and Spec 017 FR-011 | Do not move policy into generic transport; no retry or result substitution |
| Error cause classification into platform code | Keep downstream | Platform Error v4 and Specs 016/017 | Upstream exposes transport kind; adapter remains sole platform mapper |
| Local `replace` during two-repository verification | Remove before commit | Migration tooling only; Spec FR-016 | Final dependency uses tagged upstream release |
| Retry, alternate endpoint, alternate credential, legacy decoder, truncation, default limit | Remove/forbidden | AGENTS.md and Specs 016/017 | Added count remains zero |

No item is `Needs policy`.

## Release and Migration Sequence

1. Complete Spec/ADR/design in the isolated NeKiro branch.
2. Implement and test the standalone module on
   `codex/extract-a2a-transport`.
3. Prove the NeKiro adapter against the local module only as a temporary
   development step.
4. Independently review upstream, commit, push, and merge its PR.
5. Tag the reviewed upstream commits `v0.1.0` and `v0.1.1`, and verify the public module fetch.
6. Remove any local replacement, require `v0.1.1` in NeKiro, and run full
   downstream verification.
7. Independently review NeKiro, converge findings, commit, push, and open the
   downstream PR.

The downstream PR must not merge before the immutable upstream tag exists.

## Project Structure

### Documentation

```text
specs/029-extract-a2a-transport/
|-- spec.md
|-- clarify.md
|-- plan.md
|-- research.md
|-- data-model.md
|-- quickstart.md
|-- contracts/transport-public-api.md
|-- checklists/requirements.md
`-- tasks.md

docs/decisions/0008-standalone-a2a-transport-module.md
```

### Standalone upstream

```text
E:/nekiro-a2a-transport-go/
|-- go.mod
|-- go.sum
|-- LICENSE
|-- NOTICE
|-- README.md
|-- doc.go
|-- client.go
|-- errors.go
|-- jsonrpc.go
|-- stream.go
|-- client_test.go
|-- jsonrpc_test.go
`-- stream_test.go
```

### NeKiro downstream

```text
go.mod
go.sum
apps/a2a-router/internal/transport/a2a/
|-- client.go          # Router adapter and credential interceptor
|-- errors.go          # FailureKind -> Platform Error mapping only
|-- nonstreaming.go    # platform request/result mapping
|-- streaming.go       # profile/event/Ledger-neutral stream mapping and cancel policy
|-- target.go          # exact Card/Release/capability resolution mapping
`-- *_test.go
```

**Structure Decision**: Keep the standalone package at its module root to make
the import path stable and short. Delete only transport mechanics that the
downstream adapter now delegates; retain all platform/Profile behavior under
the Router's `internal` boundary.

## Verification Matrix

| Layer | Required proof |
| --- | --- |
| Upstream unit | Constructor/call validation, redirect rejection, body closure, typed causes |
| Upstream JSON-RPC | duplicate members, ID mismatch/type, XOR result/error, trailing data, media, overflow |
| Upstream SSE | exact media/framing, unique ID, one data line, delimiter, raw result, event bound, interrupted I/O |
| Upstream integration | official `a2asrv` request-response, streaming, cancellation, interceptor per operation |
| Dependency isolation | no `github.com/NeKiro-project/NeKiro` or legacy fork imports; clean external consumer builds |
| Downstream focused | Router transport and API packages, contracts, Runtime B |
| Downstream full | `go test ./...`, Runtime A nested module tests, `go vet ./...`, race tests |
| Acceptance | invoke-to-record non-stream/stream, timeout/cancel, reverse cross-runtime lineage |
| Hygiene | `git diff --check`, fallback scan, secrecy scan, committed `go.mod` has tag and no `replace` |

## Complexity Tracking

No constitution violation requires justification. The second repository is the
user-selected product boundary and reduces, rather than adds, platform-module
coupling.
