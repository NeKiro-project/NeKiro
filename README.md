<div align="center">

# NeKiro

**A runtime-agnostic framework for registering, discovering, installing, invoking, and auditing Agents.**

[![CI](https://github.com/NeKiro-project/NeKiro/actions/workflows/ci.yml/badge.svg)](https://github.com/NeKiro-project/NeKiro/actions/workflows/ci.yml)
[![Satellite Integration](https://github.com/NeKiro-project/NeKiro/actions/workflows/satellite-integration.yml/badge.svg)](https://github.com/NeKiro-project/NeKiro/actions/workflows/satellite-integration.yml)
[![Codecov](https://codecov.io/gh/NeKiro-project/NeKiro/branch/main/graph/badge.svg)](https://codecov.io/gh/NeKiro-project/NeKiro)
[![Documentation](https://img.shields.io/badge/docs-RepoWiki-blue)](https://nekiro-project.github.io/NeKiro/)
[![License](https://img.shields.io/badge/license-Apache--2.0-green.svg)](LICENSE)

[Documentation](https://nekiro-project.github.io/NeKiro/) ·
[简体中文](README.zh-CN.md) ·
[Architecture](docs/architecture/platform-direction.md) ·
[Contracts](docs/contracts/compatibility.md) ·
[Samples](https://github.com/NeKiro-project/NeKiro-Samples) ·
[Stack](https://github.com/NeKiro-project/NeKiro-Stack)

</div>

NeKiro provides the platform layer around Agent runtimes. It gives Agents a
versioned identity, makes exact releases discoverable and installable, routes
managed calls through a controlled A2A boundary, and records invocation
lineage without taking ownership of the Agent's internal execution model.

Use NeKiro when different Agent frameworks or services need to cooperate under
one consistent contract and security model.

## Why NeKiro?

Agent runtimes such as
[tRPC-Agent-Go](https://github.com/trpc-group/trpc-agent-go) are responsible for
models, prompts, tools, planners, workflows, memory, RAG, and sessions. NeKiro
does not replace those capabilities. It connects independently implemented
Agents through a framework-owned lifecycle:

```text
Register -> Discover -> Install -> Invoke -> Record
```

- **Runtime agnostic**: an Agent may use tRPC-Agent-Go, `a2a-go`, another
  framework, or a custom runtime.
- **Contract first**: Agent Cards, Releases, HTTP APIs, internal APIs, A2A
  profiles, credentials, events, and results are independently versioned.
- **Router mediated**: managed Consumer-to-Provider and Agent-to-Agent calls go
  through the A2A Router instead of using direct Provider addresses.
- **Exact release routing**: discovery and invocation preserve Agent version,
  Release ID, Card digest, endpoint provenance, and audience.
- **Workspace governance**: discovering an Agent does not automatically grant
  permission to invoke it; installation and permissions are explicit.
- **Auditable lineage**: nested calls propagate `root_task_id`,
  `parent_invocation_id`, and `trace_id` into an append-only metadata Ledger.
- **Fail closed**: missing topology, invalid configuration, failed trust
  checks, and unavailable dependencies remain distinct failures without retry,
  alternate endpoints, or stale-success fallback.

## Platform lifecycle

| Stage | What happens | Owning boundary |
| --- | --- | --- |
| Register | A Provider publishes an Agent Card and an immutable Release, then registers a ready runtime instance. | Catalog Registry + Provider deployment |
| Discover | Consumers query published Agent capabilities and exact versions. | Gateway + Catalog |
| Install | A Workspace authorizes an exact Agent version and required permissions. | Workspace |
| Invoke | Gateway dispatches the request to the A2A Router, which resolves the exact Release and a ready instance. | Gateway + A2A Router |
| Record | The Router appends status, failure, timing, Release provenance, and task/trace lineage. | Invocation Ledger |

## What works today

The current product Stack verifies the complete lifecycle against PostgreSQL,
Nacos, the production Console, and two independently implemented sample
runtimes. Core also provides explicit private-CA TLS and mTLS configuration for
the Router's Nacos HTTP and gRPC transports; exercising those Router-side modes
in the deployed Stack is the next acceptance milestone.

- Trusted Agent Card registration and immutable Release publication.
- Capability discovery and public Agent sharing.
- Workspace installation with explicit permissions.
- Exact-Release instance registration and Nacos lifecycle observation.
- Atomic initial-snapshot/watch handoff and fail-closed empty topology.
- Runtime removal, replacement, and routing recovery.
- JSON invocation and Server-Sent Events streaming.
- Cancellation propagation to the Provider.
- Nested Agent calls that re-enter the Router in both directions.
- Queryable Invocation and trace lineage in PostgreSQL.
- Router-issued, short-lived Agent credentials.
- Explicit private-CA TLS and mTLS configuration and Core tests for Router
  Nacos HTTP/gRPC transports.
- Product E2E coverage for Provider Nacos TLS and mTLS registration.
- Linux and Windows Config Center verification.

The Stack acceptance deliberately proves that Consumers cannot bypass the
Router, removed instances do not receive new calls, and secrets or Agent
payloads are not written to the Ledger.

## Architecture

```mermaid
flowchart LR
    User["User / Console / Application"] --> Gateway["Control Plane Gateway"]
    Gateway --> Catalog["Catalog Registry"]
    Gateway --> Workspace["Workspace"]
    Gateway --> Router["A2A Router"]

    Provider["Provider Agent"] -->|"Card + Release"| Gateway
    Provider -->|"Instance lease"| Nacos["Nacos"]
    Nacos -->|"Snapshot + watch"| Router

    Router -->|"Managed A2A call"| Provider
    Provider -->|"Nested managed call"| Router
    Router --> Ledger["Invocation Ledger"]

    Catalog --> Postgres[("PostgreSQL")]
    Workspace --> Postgres
    Ledger --> Postgres
```

The deployment rules are intentionally strict:

1. Console and external applications access the platform through Gateway.
2. Gateway delegates managed Agent execution to the A2A Router.
3. Registry remains the only permanent source of Agent Card and Release facts.
4. Router resolves exact Releases and ready instances without storing a second
   permanent Card.
5. Agent-to-Agent calls re-enter Router and preserve task/trace lineage.
6. Ledger stores metadata facts, never Agent input, output, credentials, or
   secrets.

See [Platform Direction](docs/architecture/platform-direction.md) and the
[Phase 1 Architecture](docs/architecture/phase-1-spec.md) for the complete
ownership and trust model.

## Samples

[NeKiro-Samples](https://github.com/NeKiro-project/NeKiro-Samples) contains two
Agents with the same platform contract but different runtime implementations:

| Sample | Runtime | Demonstrates |
| --- | --- | --- |
| Runtime A | tRPC-Agent-Go | A framework-backed Agent, TLS Nacos registration, Router-mediated invocation, and nested calls to Runtime B. |
| Runtime B | Direct `a2a-go` server | JSON/SSE/task/cancellation behavior, mTLS Nacos registration, exact instance identity, and nested calls to Runtime A. |

The Stack also exercises a replacement Runtime B instance and Provider
registration security fixtures for wrong CAs, wrong TLS server names, and
missing mTLS client identities.

The resulting sample flow is:

```text
Runtime A -> Router -> Runtime B
Runtime B -> Router -> Runtime A
```

Neither runtime receives the other's direct target address.

## Quick start

This example runs the real **Runtime B -> Router -> Runtime A** path. Runtime A
first publishes a lease to Nacos. Router observes the instance through a
snapshot and continuous watch. Runtime B then calls A by Agent ID and
capability; it never receives A's address.

```text
Runtime A --lease--> Nacos --snapshot/watch--> Router
Runtime B --Agent ID + capability--> Router --> Runtime A
                                      |
                                      +--> Invocation Ledger
```

The production code is in
[NeKiro-Samples](https://github.com/NeKiro-project/NeKiro-Samples). The two
complete `package main` programs below run inside that module. They use the
Core [`registry`](https://pkg.go.dev/github.com/NeKiro-project/NeKiro/registry)
and [`registry/nacos`](https://pkg.go.dev/github.com/NeKiro-project/NeKiro/registry/nacos)
packages through a thin Samples adapter:

```text
Runtime main
  -> Samples environment + TLS/mTLS adapter
  -> Core registry models + registry/nacos Registrar
  -> Nacos
```

Core owns the registration, heartbeat, lease, and deregistration semantics.
The Samples adapter only maps explicit `RUNTIME_A_*` / `RUNTIME_B_*`
deployment settings, builds the secured HTTP transport, and connects the lease
to process readiness and shutdown. The endpoint ownership challenge shown in
the programs is a separate trusted-publication check, not part of instance
registration. Because the programs import Samples `internal` packages, copy
their integration pattern into another Agent module rather than importing
those packages directly.

Runtime A starts its managed A2A endpoint, registers an exact instance, keeps
the lease alive, and deregisters during shutdown:

```go
package main

import (
    "context"
    "errors"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/NeKiro-project/NeKiro-Samples/internal/challengeproof"
    "github.com/NeKiro-project/NeKiro-Samples/internal/nacosregistration"
    runtimea "github.com/NeKiro-project/NeKiro-Samples/runtime-a"
)

func main() {
    if err := run(); err != nil {
        log.Fatal(err)
    }
}

func run() error {
    config, err := runtimea.LoadConfig(os.LookupEnv)
    if err != nil {
        return err
    }
    registrationConfig, err := nacosregistration.Load(
        os.LookupEnv, "RUNTIME_A", config.AgentID, config.InstanceID,
    )
    if err != nil {
        return err
    }
    registrationClient, err := nacosregistration.NewHTTPClient(registrationConfig)
    if err != nil {
        return err
    }
    registration, err := nacosregistration.New(registrationConfig, registrationClient)
    if err != nil {
        return err
    }

    handler, err := runtimea.NewHandler(config, http.DefaultClient)
    if err != nil {
        return err
    }
    application, err := challengeproof.NewHandler(
        runtimea.NewHTTPHandlerWithReadiness(handler, registration),
        os.LookupEnv,
    )
    if err != nil {
        return err
    }
    if err := registration.Register(context.Background()); err != nil {
        return err
    }

    ctx, stop := signal.NotifyContext(
        context.Background(), os.Interrupt, syscall.SIGTERM,
    )
    defer stop()
    server := &http.Server{Addr: config.ListenAddress, Handler: application}
    serverErrors := make(chan error, 1)
    go func() {
        err := server.ListenAndServe()
        if errors.Is(err, http.ErrServerClosed) {
            err = nil
        }
        serverErrors <- err
    }()
    registrationErrors := make(chan error, 1)
    go func() { registrationErrors <- registration.Run(ctx) }()

    var runErr error
    registrationStopped := false
    select {
    case <-ctx.Done():
    case runErr = <-serverErrors:
    case runErr = <-registrationErrors:
        registrationStopped = true
    }
    stop()

    shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    shutdownErr := server.Shutdown(shutdownContext)
    if !registrationStopped {
        select {
        case registrationErr := <-registrationErrors:
            runErr = errors.Join(runErr, registrationErr)
        case <-shutdownContext.Done():
            runErr = errors.Join(runErr, shutdownContext.Err())
        }
    }
    deregisterErr := registration.Deregister(shutdownContext)
    return errors.Join(runErr, shutdownErr, deregisterErr)
}
```

B registers the same way, validates the Router-issued credential on its
incoming A2A endpoint, and creates its handler with the Router URL and Agent
token. When B receives the sample's `nested` request, `NewConfiguredHandler`
uses the public Agent SDK to target A by Agent ID and capability. B never
resolves or dials A itself.

```go
package main

import (
    "context"
    "errors"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/NeKiro-project/NeKiro-Samples/internal/challengeproof"
    "github.com/NeKiro-project/NeKiro-Samples/internal/nacosregistration"
    runtimeb "github.com/NeKiro-project/NeKiro-Samples/runtime-b"
    "github.com/NeKiro-project/nekiro-sdk-go/agent/routerauth"
)

func main() {
    if err := run(); err != nil {
        log.Fatal(err)
    }
}

func run() error {
    address, err := runtimeb.ListenAddressFromEnvironment(os.LookupEnv)
    if err != nil {
        return err
    }
    config, err := runtimeb.LoadConfig(os.LookupEnv)
    if err != nil {
        return err
    }
    authenticationConfig, err := routerauth.LoadConfig(os.LookupEnv)
    if err != nil {
        return err
    }
    registrationConfig, err := runtimeb.LoadRegistrationConfig(
        os.LookupEnv, config.AgentID, config.InstanceID,
    )
    if err != nil {
        return err
    }
    registrationClient, err := nacosregistration.NewHTTPClient(registrationConfig)
    if err != nil {
        return err
    }
    registration, err := runtimeb.NewNacosRegistration(
        registrationConfig, registrationClient,
    )
    if err != nil {
        return err
    }

    handler, err := runtimeb.NewConfiguredHandler(config, http.DefaultClient)
    if err != nil {
        return err
    }
    execution, err := runtimeb.NewHTTPHandlerWithAuthAndReadiness(
        handler, authenticationConfig, registration,
    )
    if err != nil {
        return err
    }
    application, err := challengeproof.NewHandler(execution, os.LookupEnv)
    if err != nil {
        return err
    }
    if err := registration.Register(context.Background()); err != nil {
        return err
    }

    ctx, stop := signal.NotifyContext(
        context.Background(), os.Interrupt, syscall.SIGTERM,
    )
    defer stop()
    server := &http.Server{Addr: address, Handler: application}
    serverErrors := make(chan error, 1)
    go func() {
        err := server.ListenAndServe()
        if errors.Is(err, http.ErrServerClosed) {
            err = nil
        }
        serverErrors <- err
    }()
    registrationErrors := make(chan error, 1)
    go func() { registrationErrors <- registration.Run(ctx) }()

    var runErr error
    registrationStopped := false
    select {
    case <-ctx.Done():
    case runErr = <-serverErrors:
    case runErr = <-registrationErrors:
        registrationStopped = true
    }
    stop()

    shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    shutdownErr := server.Shutdown(shutdownContext)
    if !registrationStopped {
        select {
        case registrationErr := <-registrationErrors:
            runErr = errors.Join(runErr, registrationErr)
        case <-shutdownContext.Done():
            runErr = errors.Join(runErr, shutdownContext.Err())
        }
    }
    deregisterErr := registration.Deregister(shutdownContext)
    return errors.Join(runErr, shutdownErr, deregisterErr)
}
```

The handler extracts `PlatformContext` only after `routerauth` verifies the
Router credential, then calls `agentsdk.Client.Invoke` with
`TargetAgentID: "runtime-a"`. See the complete
[B -> A invocation implementation](https://github.com/NeKiro-project/NeKiro-Samples/blob/main/runtime-b/nested.go).

### Run the complete scenario

The immutable [NeKiro-Stack](https://github.com/NeKiro-project/NeKiro-Stack)
supplies the parts intentionally omitted above: trusted Card/Release
publication, Workspace installation and permissions, Router credentials,
PostgreSQL, secured Nacos, exact component revisions, and Ledger assertions.
The following commands require Git, Go 1.26+, Docker, Bash, and network access:

```bash
git clone https://github.com/NeKiro-project/NeKiro-Stack.git
cd NeKiro-Stack

work_root=$(mktemp -d)
backend_env="$work_root/backend.env"
prepared_env="$work_root/prepared.env"

./scripts/write-ci-env.sh backend "$backend_env" "$(pwd)" nekiro-quickstart
set -a
source "$backend_env"
set +a

./scripts/prepare.sh "$(pwd)/components.json" "$work_root/checkouts" "$prepared_env"
set -a
source "$prepared_env"
set +a

go run ./cmd/nacos-secure-fixture generate "$NEKIRO_E2E_TLS_ROOT"
docker compose --project-name "$NEKIRO_E2E_COMPOSE_PROJECT" \
  --file compose.yaml \
  --file "$NEKIRO_E2E_COMPOSE_OVERRIDE_FILE" \
  --profile router-nacos-secure \
  up --detach --wait --wait-timeout 120
go test -tags=e2e -run '^TestInvokeToRecordAcceptance$' -count=1 ./tests/backend

docker compose --project-name "$NEKIRO_E2E_COMPOSE_PROJECT" \
  --file compose.yaml \
  --file "$NEKIRO_E2E_COMPOSE_OVERRIDE_FILE" \
  --profile router-nacos-secure \
  --profile runtime-registration \
  --profile watch-refresh \
  down --volumes --remove-orphans
```

A passing acceptance proves that A registered, Router discovered its exact
Release-scoped instance, B reached A only through Router, and the parent/child
records share `root_task_id` and `trace_id` in Ledger. It also exercises A -> B,
instance removal, fail-closed routing, and replacement recovery.

### Develop Core

For Core development, Go 1.26 or newer is required:

```bash
git clone https://github.com/NeKiro-project/NeKiro.git
cd NeKiro

go mod download
go build ./...
go test ./...
go test -race ./...
go vet ./...
```

Build the Core service images from the repository root:

```bash
docker build --file apps/control-plane/Dockerfile --tag nekiro-control-plane:local .
docker build --file apps/a2a-router/Dockerfile --tag nekiro-a2a-router:local .
```

Running the binaries alone is a Core development check, not product E2E
success. Use NeKiro-Stack to validate registration, discovery, installation,
routing, sample Agents, Console behavior, and committed Ledger records
together.

## Repository map

NeKiro uses separate repositories to keep ownership and release boundaries
explicit.

| Repository | Responsibility |
| --- | --- |
| [NeKiro](https://github.com/NeKiro-project/NeKiro) | Control Plane, A2A Router, contracts, Config Center source semantics, service-owned migrations, and Core verification. |
| [NeKiro-Console](https://github.com/NeKiro-project/NeKiro-Console) | Production web Console. |
| [nekiro-sdk-go](https://github.com/NeKiro-project/nekiro-sdk-go) | Public Go SDKs for applications and Agents. |
| [NeKiro-Samples](https://github.com/NeKiro-project/NeKiro-Samples) | Cross-runtime sample Agents and Provider deployment wiring. |
| [NeKiro-Stack](https://github.com/NeKiro-project/NeKiro-Stack) | Immutable multi-component assembly and product backend/browser E2E. |
| [nekiro-a2a-transport-go](https://github.com/NeKiro-project/nekiro-a2a-transport-go) | Reusable A2A HTTP/JSON-RPC/SSE transport mechanics. |

Core required CI does not check out or build satellite source. A separate
post-merge Satellite Integration workflow invokes satellite-owned reusable
workflows against the exact merged Core SHA.

## Core repository layout

```text
apps/control-plane/   Gateway, Catalog, Workspace, publication, and dispatch
apps/a2a-router/      A2A routing, credentials, topology, policy, and Ledger
config_center/        Provider-neutral byte snapshots, reads, watches, publish
contracts/            JSON Schema, OpenAPI, A2A profiles, and Go mappings
tests/                Core contract and service integration verification
docs/                 Architecture, contracts, ADRs, usage, and operations
```

Catalog, Workspace, and Ledger migrations stay beside the modules that own
their schemas and are embedded in the corresponding service binaries.

## Security model

- No guessed defaults for secrets, production endpoints, database addresses,
  JWT keys, certificates, or trust roots.
- No Consumer-to-Provider direct path in managed invocation.
- No system-root fallback or `InsecureSkipVerify` for secured Nacos modes.
- No automatic retry, alternate endpoint, old component, or stale topology
  success.
- No Agent payloads, credentials, or keys in Ledger metadata.
- No shared internal implementation types across deployment boundaries;
  cross-process data uses versioned contracts.

Security and compatibility decisions are documented as ADRs under
[`docs/decisions`](docs/decisions/), including runtime trust, signed Router
credentials, instance discovery, registration leases, and Nacos transport
security.

## Development and verification

PostgreSQL integration suites require an explicit dedicated database whose
name ends in `_test`:

```bash
export NEKIRO_TEST_DATABASE_URL='postgresql://user:password@127.0.0.1:5432/nekiro_core_test?sslmode=disable'

go test -tags=integration -count=1 ./apps/control-plane/internal/catalog/postgres
go test -tags=integration -count=1 ./apps/control-plane/internal/workspace/postgres
go test -tags=integration -count=1 ./apps/control-plane/internal/workspace/integration
go test -tags=integration -count=1 ./apps/a2a-router/internal/ledger
go test -tags=integration -count=1 ./tests/integration/catalog
```

Useful documentation:

- [Core development](docs/usage/core-development.md)
- [Trusted publication operations](docs/usage/trusted-publication-operations.md)
- [Config Center runtime operations](docs/usage/config-center-runtime-operations.md)
- [External Gateway operations](docs/usage/external-gateway-operations.md)
- [Contract compatibility policy](docs/contracts/compatibility.md)
- [Central RepoWiki](https://nekiro-project.github.io/NeKiro/)

## Project direction

NeKiro started from the idea of “pluggable Agents” and has evolved into an
Agent Framework focused on the links and contracts between independently
implemented nodes. The current direction is to make the complete managed
Provider/Consumer lifecycle increasingly deployable, secure, observable, and
language neutral while leaving Agent-internal intelligence to runtime owners.

## Contributing

Issues and pull requests are welcome. Changes to public behavior, contracts,
data ownership, or architecture should begin with an Issue; ownership or
compatibility decisions should also update an ADR. See
[Core development](docs/usage/core-development.md) for the expected validation
commands.

## History and license

The annotated tag `pre-repository-split-2026-08-04` preserves the accepted
monorepo tree and tracked Spec Kit history. Repository ownership and migration
rationale are recorded in
[ADR 0009](docs/decisions/0009-core-repository-boundary.md).

NeKiro is licensed under the [Apache License 2.0](LICENSE).
