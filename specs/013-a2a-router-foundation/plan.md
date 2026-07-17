# Implementation Plan: A2A Router Foundation

**Branch**: `codex/013-router-foundation` | **Date**: 2026-07-16 | **Spec**: [spec.md](spec.md)

## Summary

Create the first standalone `apps/a2a-router` process with strict required
configuration, service authentication, readiness, a Router Internal v3 dispatch
handler, and a Control Plane Internal v2 resolution client. The handler
authenticates and validates dispatch requests, re-resolves exact Agent facts
through Control Plane only, preserves typed failures, and returns correlated
`ROUTE_NOT_FOUND` after successful resolution until Agent transport exists.

## Technical Context

**Language**: Go 1.26
**Dependencies**: Go standard HTTP/JSON packages and existing contract DTO/validators
**Storage**: None in this feature
**Testing**: Unit and HTTP tests with fakes/`httptest`; full `go test` and `go vet`
**Constraints**: Router must not import Control Plane internals; no Ledger or Agent transport writes; zero fallback

## Architecture and Ownership

```text
Control Plane Dispatch
  -> Router Internal v3 /internal/v3/invocations
  -> Router auth + strict request/media/body validation
  -> Resolution client POST /internal/v2/resolve-agent
  -> typed resolution result/failure
  -> correlated ROUTE_NOT_FOUND until T006 transport
```

- `apps/a2a-router/cmd/a2a-router` owns process assembly and readiness.
- `apps/a2a-router/internal/config` owns strict environment parsing.
- `apps/a2a-router/internal/auth` owns Router service credentials.
- `apps/a2a-router/internal/resolution` owns Control Plane Internal v2 calls.
- `apps/a2a-router/internal/api` owns Router Internal v3 HTTP surface.
- Ledger, Agent transport, SDK, and Compose wiring are explicitly out of scope.

## Constitution Check

- Phase 1 loop: PASS; this is the Data Plane entry needed for `Invoke`.
- Runtime agnostic: PASS; no Agent framework or runtime internals are imported.
- Control/Data Plane boundary: PASS; Router calls only versioned Control Plane internal API.
- Contracts first: PASS; consumes existing v3/v2/v4 contracts without schema changes.
- Failure and secret safety: PASS; no default credentials, no endpoint probes, no raw secrets in output.
- SDD/review: PASS; this plan produces mapped tasks, tests, review, and converge gates.

## Write Scope

- `specs/013-a2a-router-foundation/`
- `apps/a2a-router/cmd/a2a-router/`
- `apps/a2a-router/internal/config/`
- `apps/a2a-router/internal/auth/`
- `apps/a2a-router/internal/resolution/`
- `apps/a2a-router/internal/api/dispatch_handler.go`
- `apps/a2a-router/Dockerfile`

Excluded: `apps/a2a-router/internal/ledger/`, Agent transport, SDK, sample Agents, Compose/CI orchestration, Control Plane runtime code, and shared contracts.

## Fallback Report

```text
Fallback delta: removed 0, retained 0, added 0, net 0
Added fallback evidence: none
```
