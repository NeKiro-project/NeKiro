# Research: Cross-Runtime Caller Sample

## Decision 1: Runtime framework

- **Decision**: Pin `trpc.group/trpc-go/trpc-agent-go v1.10.0` in `agents/runtime-a/go.mod`.
- **Rationale**: The framework exposes an Agent interface plus Runner, Session, and Event execution, which gives Runtime A a materially different execution model from Runtime B's direct `a2a-go` request handler. The version is an existing tagged release and is reproducibly pinned by the nested module.
- **Alternatives considered**:
  - Reusing Runtime B's handler was rejected because it would not prove runtime independence.
  - Making the framework a root-module dependency was rejected by ADR 0003 and would couple platform code to one Runtime.
  - A second full A2A framework server was rejected because the active Profile already has a verified `a2a-go v0.3.15` adapter in this repository; the framework is used for execution, while the adapter remains protocol-owned.

## Decision 2: Module isolation

- **Decision**: Use a nested module at `agents/runtime-a/` with a local `replace` for the repository root module so the sample can import only versioned contracts and the thin SDK during local development.
- **Rationale**: `go test ./...` in the root does not implicitly absorb nested modules, preventing framework dependencies from entering platform builds or `go.mod`/`go.sum`.
- **Alternatives considered**: A root-module subpackage was rejected because Go dependency resolution would pin the Runtime framework for every platform package.

## Decision 3: A2A boundary

- **Decision**: Use `github.com/a2aproject/a2a-go v0.3.15` `a2asrv` at the sample edge and `a2asrv.CallContextFrom` to read the Router's exact context headers.
- **Rationale**: Runtime B and the repository conformance suite already prove this library against the active Profile. Runtime A can therefore vary its internal execution model without changing the platform wire contract.
- **Alternatives considered**: The framework's bundled A2A server was not used because it uses a separate protocol module/version and would create a second Profile adapter in the sample.

## Decision 4: Failure and fallback policy

- **Decision**: Fail startup/request immediately for missing or invalid required values; propagate SDK/Router errors without retry or alternate route.
- **Evidence**: ADR 0006, Issue #29 acceptance criteria, active SDK contract, and the repository fallback policy in `AGENTS.md`.
- **Fallback delta**: removed 0, retained 0, added 0.
