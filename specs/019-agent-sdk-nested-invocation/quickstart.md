# Quickstart: Agent SDK Nested Invocation

This child is currently validated with in-process HTTP handlers and explicit
constructor credentials. A later parent acceptance task wires deployment
variables and the second Runtime process.

```powershell
go test -count=1 ./sdks/agent-sdk ./apps/a2a-router/internal/nested ./apps/a2a-router/internal/api ./contracts
go vet ./...
```

The SDK constructor receives an explicit Router v1 URL, bearer token,
transport, JSON response limit, and SSE event limit. Use `Invoke` for JSON and
`InvokeStream`/`Recv` for incremental SSE until `io.EOF`. A Runtime passes the
inherited `PlatformContext` and only the target Agent, capability, input
object, and stream mode. It must not provide a child ID, Workspace, root Task,
Trace, endpoint, or credential field.

## Fallback report

```text
Fallback delta: removed 1, retained 0, added 0, net -1
Added fallback evidence: none
```
