# Quickstart: Verify Standalone A2A Transport Extraction

## Prerequisites

- Go 1.26.0
- Clean clones of `NeKiro-project/nekiro-a2a-transport-go` and
  `NeKiro-project/NeKiro`
- Upstream release tag available before final downstream verification
- Explicit PostgreSQL/Compose values required by existing NeKiro E2E guides

No command may substitute a local endpoint, default limit, mock credential, or
alternate module source when a required value is missing.

## Standalone upstream

From the upstream repository:

```text
go test ./...
go test -race ./...
go vet ./...
go mod tidy
git diff --check
```

Confirm that `go list -deps ./...` contains the official A2A dependency but no
NeKiro platform module.

## External consumer

Create a temporary module outside both repositories, require the tagged
upstream version, and compile one request-response plus one streaming example.
The temporary module must not use `replace`, vendor NeKiro source, or import a
NeKiro package.

Expected outcome: dependency resolution and compilation succeed using only the
public tag.

## NeKiro focused verification

From the NeKiro repository after removing any development replacement:

```text
go test -count=1 ./apps/a2a-router/internal/transport/a2a
go test -count=1 ./apps/a2a-router/internal/api
go test -count=1 ./contracts
go test -count=1 ./agents/runtime-b
go test ./...
go vet ./...
git diff --check
```

From `agents/runtime-a`:

```text
go test ./...
go vet ./...
```

## Acceptance verification

Run the existing invoke-to-record non-streaming, streaming, timeout,
cancellation, credential, and reverse cross-runtime scenarios using the
commands and explicit environment documented by their owning Specs.

Expected outcome: all existing public results, Platform Error codes, Task/Trace
lineage, exact Release provenance, and Ledger terminal facts remain unchanged.

## Final dependency and fallback checks

- Root `go.mod` requires an immutable upstream version.
- Root `go.mod` and `go.work` contain no local replacement for the upstream.
- The former JSON-RPC/SSE core is absent from the Router adapter.
- Upstream contains no NeKiro platform import.
- Fallback delta is `removed 0, retained 2, added 0, net 0`.
- Added fallback evidence is `none`.
