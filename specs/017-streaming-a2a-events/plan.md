# Implementation Plan: Streaming A2A Result Delivery

**Branch**: `017-streaming-a2a-events` | **Date**: 2026-07-16 | **Spec**:
[spec.md](spec.md)

**Input**: Feature specification from
`specs/017-streaming-a2a-events/spec.md`, ADR 0006, Router Internal v3,
Result Stream Event v2, Invocation Event 0.3, and the pinned A2A Profile.

## Summary

Extend the Router's existing exact A2A transport and dispatch boundary to
support `stream=true`: call `message/stream`, validate the A2A event sequence,
map each event to a transient Result Stream Event v2, enforce separate A2A and
SSE byte limits, and emit strict one-data-line SSE frames. Streaming must use
the existing accepted Invocation boundary and metadata-only Ledger facts; no
result content or chunk is persisted.

## Technical Context

**Language/Version**: Go 1.26.0

**Primary Dependencies**: Go `net/http`, `bufio`, `encoding/json`,
`github.com/a2aproject/a2a-go v0.3.15`, existing contracts validators, Router
auth/config/resolution/API/Ledger packages.

**Storage**: Router-owned PostgreSQL Ledger for metadata-only lifecycle facts;
no result or chunk storage.

**Testing**: Go unit/HTTP tests, `httptest`, Runtime B stream fixture, active
A2A Profile conformance fixtures, runtime Result Stream v2 validators, race,
full `go test ./...`, `go vet ./...`, Compose config, and `git diff --check`.

**Target Platform**: Windows developer shell and Linux/WSL/container runtime.

**Project Type**: Router Data Plane HTTP service.

**Performance Goals**: Forward each valid event without buffering the complete
stream; hold at most one bounded upstream event and one bounded serialized SSE
frame at a time.

**Constraints**: Required limits remain strict base-10 values in
`1..2147483647` (`NEKIRO_ROUTER_A2A_EVENT_LIMIT_BYTES` and the separate
`NEKIRO_ROUTER_SSE_EVENT_LIMIT_BYTES`); effective Agent output is the minimum configured/Card bound;
SSE accepts exact `text/event-stream`; no retries, caches, fallback endpoints,
default credentials, result persistence, or full Agent Runtime dependencies.

**Scale/Scope**: One streaming root Invocation per request, with arbitrary
finite event count and one first terminal event; nested Agent calls and
cross-runtime proof consume this stream path without sharing Runtime types.

## Constitution Check

- **Phase 1 loop**: PASS. This feature completes the streaming form of Invoke
  and preserves Record metadata.
- **Ownership**: PASS. Router owns A2A transport, SSE framing, dispatch
  sequencing, and Ledger append calls; Control Plane is accessed only through
  the existing versioned resolution contract.
- **Runtime independence**: PASS. A2A events are opaque contract values; no
  model, tool, workflow, memory, or Runtime-specific API enters core code.
- **Contracts**: PASS. Consumes active Router Internal v3, Result Stream Event
  v2, Invocation Event 0.3, Platform Error v4, A2A Profile 0.2, and protocol
  0.3.0 without version changes.
- **Invocation lineage**: PASS. Accepted, chunk, and terminal events repeat
  invocation/root-task/trace identifiers; Ledger facts remain metadata only.
- **Failure safety**: PASS. Overflow, malformed event, timeout, cancellation,
  interrupted EOF, Agent failure, and Ledger failure remain distinct; no
  fallback behavior is introduced.
- **SDD traceability**: PASS. Each story and requirement maps to tasks and
  post-implementation tests; implementation follows the approved spec.
- **Cross-runtime proof**: PASS. Runtime B stream fixtures exercise the same
  Router path that future independently implemented Agents will use.

## Project Structure

### Documentation (this feature)

```text
specs/017-streaming-a2a-events/
|-- spec.md
|-- checklists/requirements.md
|-- research.md
|-- data-model.md
|-- quickstart.md
|-- plan.md
`-- tasks.md
```

### Source Code

```text
apps/a2a-router/internal/api/dispatch_handler.go
apps/a2a-router/internal/api/dispatch_handler_test.go
apps/a2a-router/internal/transport/a2a/client.go
apps/a2a-router/internal/transport/a2a/streaming.go
apps/a2a-router/internal/transport/a2a/streaming_test.go
apps/a2a-router/internal/transport/a2a/errors.go
apps/a2a-router/internal/config/config.go
apps/a2a-router/cmd/a2a-router/
agents/runtime-b/
contracts/runtime_contracts_validation.go
```

**Structure Decision**: Keep streaming transport beside the existing
non-streaming A2A client, keep HTTP/SSE ownership in the Router API package,
and reuse shared runtime validators rather than introducing a second event
schema or a generic streaming framework. Any change to `contracts/` requires a
separate contract Spec/ADR and is outside this plan.

The Router configuration gains one required `SSEEventLimitBytes` field sourced
from `NEKIRO_ROUTER_SSE_EVENT_LIMIT_BYTES`; it is independent from the A2A event
limit and is passed explicitly to the SSE writer.

## Design Decisions

1. **One bounded event at a time**: the A2A client reads one SSE event, bounds
   its serialized bytes, validates it, and yields it; no whole-stream buffer or
   result replay is introduced.
2. **Result Stream Event v2 is the northbound frame**: the Router emits an
   `accepted` event before the first Agent event, maps each valid Agent event to
   a `chunk`, and emits one terminal event after the source stream terminates
   or fails.
3. **Profile validation precedes forwarding**: event kind, task/context
   identity, terminal ordering, and A2A protocol errors are rejected before a
   chunk reaches the caller.
4. **Ledger and live stream stay separate**: `stream` Ledger facts contain
   chunk index/byte metadata only; Result Stream Event `chunk` carries the
   transient opaque value and is never appended to the Ledger.
5. **Post-commit failures are in-band**: once SSE headers or the accepted frame
   are committed, writable failures become one correlated terminal event; a
   failure before commitment remains an HTTP platform error.

## Failure Mapping

| Condition | HTTP/SSE behavior | Ledger behavior |
| --- | --- | --- |
| Invalid request/media or unresolved route | Pre-correlation/typed HTTP error | No accepted fact |
| Unsupported Card/profile/auth/endpoint | HTTP 502 or correlated failed SSE from routing | `created -> routing -> failed` if accepted |
| A2A event/protocol invalid | HTTP 502 before SSE commitment or failed SSE after commitment | Terminal failure if accepted |
| A2A/SSE event exceeds limit | 502 / failed SSE with `AGENT_RESPONSE_TOO_LARGE` | Terminal failure if committed; no fabricated success |
| Deadline/cancellation | 504/409 before commitment or timed-out/canceled SSE | First committed terminal wins |
| Source EOF before terminal | Explicit interrupted delivery; no completed event | Non-terminal history remains non-success |
| Ledger append fails after Agent side effect | 503 before commitment or failed SSE if writable | Durable history remains at last committed fact |

## Write Scope

- Owned: `specs/017-streaming-a2a-events/**`,
  `apps/a2a-router/internal/api/dispatch_handler.go` and focused tests,
  `apps/a2a-router/internal/transport/a2a/**` and focused tests,
  `agents/runtime-b/` stream fixtures, and Router runbook/handoff updates.
- Referenced but not re-owned: `contracts/**`,
  `apps/a2a-router/internal/ledger/**`, and existing Control Plane resolution
  contracts.
- Not owned: Catalog, Workspace, Frontend, SDK runtime features, new A2A
  authentication types, contract version changes, result persistence, or
  deployment defaults.

## Complexity Tracking

No constitution violations require justification.
