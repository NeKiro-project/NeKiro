# Quickstart: Streaming A2A Result Delivery

## Prerequisites

- PostgreSQL Ledger schema is migrated before Router `serve`.
- Required Router configuration is set, including
  `NEKIRO_ROUTER_AGENT_RESPONSE_LIMIT_BYTES` and
  `NEKIRO_ROUTER_A2A_EVENT_LIMIT_BYTES` plus the separate required
  `NEKIRO_ROUTER_SSE_EVENT_LIMIT_BYTES`.
- Runtime B is available as the deterministic streaming A2A fixture.

## Focused validation

```powershell
go test -count=1 ./apps/a2a-router/internal/transport/a2a ./apps/a2a-router/internal/api ./agents/runtime-b
go test -count=1 ./apps/a2a-router/internal/transport/a2a -run 'Streaming|SSE|Limit'
```

The focused tests must prove one `accepted` event, zero-based ordered chunks,
one terminal event, exact correlation, raw single-line SSE framing, configured
A2A/SSE boundary behavior, interrupted EOF, and no Agent content in Ledger
facts.

## Full validation

```powershell
go test -count=1 ./...
go vet ./...
git diff --check
wsl.exe -d Ubuntu-26.04 -- bash -lc 'cd /mnt/e/NeKiro && go test -race -count=1 ./apps/a2a-router/... ./agents/runtime-b'
docker compose --file deploy/compose.yaml config --quiet
```

## Expected wire behavior

Every successful stream returns `200 text/event-stream` and emits exactly one
compact JSON Result Stream Event v2 per frame:

```text
data: {"schemaVersion":"2",...}\n
\n
```

No frame contains extra SSE fields, multiple data lines, unescaped CR/LF, a
truncated JSON value, or a body larger than the required SSE event limit.
EOF before a terminal event is an interrupted non-success outcome.
