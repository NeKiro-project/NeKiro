# Tasks: Streaming A2A Result Delivery

**Input**: Design documents from `specs/017-streaming-a2a-events/`

**Scope**: Implement Router-owned streaming A2A delivery, bounded SSE framing,
and metadata-only lifecycle facts. No contract version changes or result
persistence are allowed.

**Tests**: Add tests after the corresponding approved implementation, mapped to
the Spec acceptance scenarios and active contract validators.

## Phase 1: Setup

- [X] T001 Confirm the active Router Internal v3, Result Stream Event v2, A2A Profile 0.2, ADR 0006, and Runtime B stream fixtures in `specs/017-streaming-a2a-events/research.md` and `plan.md`.
- [X] T002 [P] Add the Spec 017 focused validation commands and bounded-stream operating notes to `specs/017-streaming-a2a-events/quickstart.md` and `docs/runbooks/local-development.md`.

## Phase 2: Foundational

- [X] T003 Define Router streaming transport and SSE writer interfaces in `apps/a2a-router/internal/api/dispatch_handler.go` without importing Control Plane internals or changing active contracts.
- [X] T004 [P] Define bounded A2A event serialization and strict one-line SSE framing helpers in `apps/a2a-router/internal/transport/a2a/streaming.go` with explicit byte-limit errors and no truncation.
- [X] T005 [P] Add shared Result Stream Event v2 mapping/sequence helpers in `apps/a2a-router/internal/transport/a2a/streaming.go` that preserve invocation, root-task, trace, event, and chunk indexes.
- [X] T006 [P] Add required `NEKIRO_ROUTER_SSE_EVENT_LIMIT_BYTES` parsing, tests, `.env.example`, `deploy/compose.yaml`, and `docs/runbooks/local-development.md` without inferring it from another limit.

**Checkpoint**: Router can represent one bounded, contract-valid Result Stream
Event and SSE frame but has not yet opened an Agent stream.

## Phase 3: User Story 1 - Stream an Agent Result (Priority: P1)

**Goal**: Deliver one exact A2A `message/stream` call as transient ordered
Result Stream Event v2 SSE values.

**Independent Test**: Runtime B emits its deterministic stream; Router output
contains accepted, ordered chunks, one terminal event, and exact correlation.

### Implementation

- [X] T007 [US1] Implement `message/stream` in `apps/a2a-router/internal/transport/a2a/client.go` or `streaming.go`, propagating the same trusted platform headers as non-streaming dispatch.
- [X] T008 [US1] Map approved A2A message/task/status/artifact events to transient Result Stream Event v2 chunks in `apps/a2a-router/internal/transport/a2a/streaming.go` and reject unsupported or identity-changing events.
- [X] T009 [US1] Wire `stream=true` dispatch, exact target/input validation, response commitment, accepted/chunk/terminal sequencing, and writer flushing in `apps/a2a-router/internal/api/dispatch_handler.go`.
- [X] T010 [US1] Add focused Runtime B streaming success and correlation tests in `apps/a2a-router/internal/transport/a2a/streaming_test.go` and `apps/a2a-router/internal/api/dispatch_handler_test.go`.

**Checkpoint**: A valid stream is independently demonstrable with accepted,
ordered chunks, and one terminal event; `stream=false` behavior remains green.

## Phase 4: User Story 2 - Enforce Bounded SSE Delivery (Priority: P1)

**Goal**: Enforce separate A2A-event and SSE-event limits with exact framing.

**Independent Test**: Boundary-sized and oversized upstream events produce
bounded forwarding or explicit `AGENT_RESPONSE_TOO_LARGE` failure without
truncation or whole-stream buffering.

### Implementation

- [X] T011 [US2] Enforce the effective configured/Card A2A event bound while reading each upstream event in `apps/a2a-router/internal/transport/a2a/streaming.go`.
- [X] T012 [US2] Enforce the full compact UTF-8 SSE frame bound before writing in `apps/a2a-router/internal/api/dispatch_handler.go` and map overflow to the approved correlated failure.
- [X] T013 [US2] Add raw-wire SSE framing and A2A/SSE limit boundary tests in `apps/a2a-router/internal/transport/a2a/streaming_test.go` and `apps/a2a-router/internal/api/dispatch_handler_test.go`.

**Checkpoint**: No emitted frame is truncated, multiline, unbounded, or larger
than the required effective limit.

## Phase 5: User Story 3 - Preserve Streaming Lifecycle and Failure Facts (Priority: P1)

**Goal**: Preserve accepted/routing/running/stream/terminal semantics and
explicit post-commit failure behavior.

**Independent Test**: Invalid event, interrupted EOF, timeout/cancellation,
Agent failure, and Ledger failure cases produce valid correlated outcomes and
never fabricate clean success.

### Implementation

- [X] T014 [US3] Extend metadata-only Ledger orchestration in `apps/a2a-router/internal/api/dispatch_handler.go` to append stream chunk indexes/bytes without Agent content and commit terminal facts before clean terminal SSE events.
- [X] T015 [US3] Map A2A protocol, endpoint, overflow, timeout, cancellation, interrupted EOF, and Ledger persistence failures to the active Platform Error v4 semantics in `apps/a2a-router/internal/transport/a2a/streaming.go` and `dispatch_handler.go`.
- [X] T016 [US3] Add one bounded `tasks/cancel` attempt for known A2A task IDs on local deadline/disconnect in `apps/a2a-router/internal/transport/a2a/streaming.go`, with no retry or alternate route.
- [X] T017 [US3] Add lifecycle, interruption, terminal-race, cancellation, and Ledger failure tests in `apps/a2a-router/internal/api/dispatch_handler_test.go` and `apps/a2a-router/internal/transport/a2a/streaming_test.go`.

**Checkpoint**: Accepted streams have one immutable terminal outcome or an
explicit interrupted-delivery failure; Ledger facts remain metadata-only.

## Phase 6: Verification, Review, and Converge

- [X] T018 [P] Run focused Router/Runtime B tests, active contract validators, and raw SSE assertions; record evidence in `specs/017-streaming-a2a-events/tasks.md`.
- [X] T019 [P] Run `go test -count=1 ./...`, `go vet ./...`, WSL race, Compose config, and `git diff --check`; record exact commands and outcomes in `specs/017-streaming-a2a-events/tasks.md` and `docs/handoffs/CURRENT.md`.
- [X] T020 Obtain an independent Standards/Spec Review against `spec.md`, `plan.md`, `tasks.md`, ADR 0006, active contracts, and AGENTS.md; resolve all blocking findings before convergence.
- [X] T021 Run Converge, update `specs/017-streaming-a2a-events/spec.md` and `specs/017-streaming-a2a-events/tasks.md` for any remaining work, and rerun the independent Review after fixes.

## Dependencies & Execution Order

```text
T001 -> T003/T004/T005/T006 -> T007/T008/T009 -> T010
T010 -> T011/T012 -> T013
T013 -> T014/T015/T016 -> T017
T017 -> T018/T019 -> T020 -> T021
```

T002 is documentation-only and can run in parallel with T001. T004 and T005
can run in parallel after the interface boundary is confirmed. User stories
are sequential because each later story consumes the stream seam established
by the prior story.

## Requirement Coverage

| Requirement | Tasks |
| --- | --- |
| FR-001–FR-004 | T003, T007–T010 |
| FR-005–FR-008 | T004, T008, T011–T013 |
| FR-009–FR-011 | T014–T017 |
| FR-012–FR-013 | T001, T003, T014, T020–T021 |
| SC-001 | T010, T018 |
| SC-002–SC-003 | T011–T013, T018 |
| SC-004–SC-005 | T014–T021 |

## Validation Evidence

- Focused: `go test -count=1 ./apps/a2a-router/internal/transport/a2a ./apps/a2a-router/internal/api ./agents/runtime-b` — PASS.
- Full: `go test -count=1 ./...` — PASS.
- Static: `go vet ./...` and `git diff --check` — PASS.
- Race: `wsl.exe -d Ubuntu-26.04 -- bash -lc 'cd /mnt/e/NeKiro && go test -race -count=1 ./apps/a2a-router/... ./agents/runtime-b'` — PASS.
- Compose: `docker compose --file deploy/compose.yaml config --quiet` with required non-empty validation variables — PASS.

## Review and Convergence

- Independent Spec/Standards reviews found and resolved Card streaming/timeout
  preflight, effective Card deadline, five trusted context headers, artifact
  append/last-chunk ordering, terminal lookahead, caller-writer failure
  terminalization, strict unique upstream SSE IDs, raw result preservation,
  cancellation races, and unsupported fallback behavior.
- Convergence keeps the upstream complete SSE block bounded before parsing and
  separately forwards/checks the raw JSON-RPC `result` bytes. The block guard
  intentionally includes framing/envelope bytes so a dependency cannot bypass
  the Router's memory bound with unbounded protocol overhead; downstream SSE
  frames have their own full-frame limit.
- The one-second cancel and post-cancellation Ledger grace are explicit
  bounded operational policies documented in `research.md`; they do not retry,
  reroute, or fabricate success.
- The pinned `a2a-go` SSE decoder has its own 10 MiB scanner token ceiling;
  larger configured values remain fail-closed as classified protocol errors.
  Supporting streams above that dependency ceiling requires a separate
  transport/ADR decision and is outside this slice's approved scope.
- No additional Spec/ADR or contract work remains for this slice.

## Completion State

- Implementation: complete for T001–T019; streaming Router code and tests are
  present and validated.
- Tests: focused, full, race, contract-validator, raw SSE, failure, and
  cancellation coverage are passing.
- Review/Converge: T020–T021 complete; independent reviews are non-blocking and
  convergence has been recorded above.
- Fallback delta: removed 0, retained 0, added 0, net 0.
