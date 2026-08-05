<div class="satellite-source-note">Read-only mirror of <a href="https://github.com/NeKiro-project/nekiro-sdk-go/blob/0bc1bd0495ef877f8583301d6ba8ff128d6cae5f/agent/README.md"><code>NeKiro-project/nekiro-sdk-go/agent/README.md</code></a> at <code>0bc1bd0495ef877f8583301d6ba8ff128d6cae5f</code>. The canonical document remains in the satellite repository; edit it there and refresh this snapshot.</div>

# Agent SDK

`github.com/NeKiro-project/nekiro-sdk-go/agent` is a small Go client for the managed Agent Router v1 nested
invocation boundary. It carries the trusted platform context supplied by the
managed transport, validates the untrusted target request, and performs one
HTTP call through the Router.

The SDK does not implement a model, tool, workflow, memory, retry, cache,
fallback route, or Agent Runtime. `NewClient` requires explicit response and
SSE event byte limits; there are no size defaults. Use `Invoke` for JSON and
`InvokeStream` for incremental SSE delivery. A stream must be consumed with
`Recv` through `io.EOF` so the terminal event and sequence can be validated.

Router errors are accepted only when their media type, v4 Platform Error
shape, trace header, HTTP status, and error code agree. The SDK exposes safe
status/code/correlation fields through `RouterError`; it never exposes raw
error response bytes.
