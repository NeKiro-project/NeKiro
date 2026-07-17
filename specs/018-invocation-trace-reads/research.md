# Research: Invocation and Trace Metadata Reads

## Decision 1: Reuse the existing Workspace owner boundary

- **Decision**: Call the Workspace-owned `GetWorkspace` authorization port
  before constructing any Router read request.
- **Rationale**: Workspace already owns immutable owner policy and distinguishes
  invalid, not-found, forbidden, and dependency outcomes. A read service must
  not duplicate SQL or installation policy.
- **Rejected alternatives**: trusting a public owner header, reading
  Workspace tables from Gateway, or treating a missing Workspace as an empty
  Trace.

## Decision 2: Use one Router destination and sibling v3 paths

- **Decision**: Extend the existing Router HTTP client with GET methods that
  preserve the configured Router origin and select only the contract-defined
  `/internal/v3/workspaces/...` paths.
- **Rationale**: Invocation creation and metadata reads belong to the same
  Router service boundary; deriving a sibling path from one explicit operation
  URL does not introduce an alternate destination.
- **Rejected alternatives**: a destination list, localhost default, URL
  userinfo, direct Ledger SQL, retry, or a historical `/v2` path.

## Decision 3: Let Router own stored-response validation

- **Decision**: Router Ledger Store and `LedgerHandler` validate the durable
  Invocation/Event and Trace projections before a 200 response. Gateway
  accepts only `application/json` from the authenticated Router and proxies
  the already validated metadata response without persisting or rewriting it.
- **Rationale**: Ledger owns Event 0.3 sequence and lineage semantics. A
  second Gateway validator would create another source of lifecycle truth and
  would require unbounded result buffering.
- **Rejected alternatives**: Gateway direct database reads, result-content
  extraction, full response replay, or a permissive empty-object fallback.

## Decision 4: Collapse internal failures only at the public boundary

- **Decision**: Router 404 maps to public `NOT_FOUND`; Router auth failures,
  malformed media/body, transport errors, and 5xx map to safe public
  `DEPENDENCY_ERROR`. Workspace errors retain their exact public meanings.
- **Rationale**: Internal service details and raw dependency messages are not
  public contract values. The distinction needed by callers is resource
  absence versus dependency failure.
- **Rejected alternatives**: forwarding internal auth status, returning raw
  error bodies, or converting dependency failures to empty `[]`/success.

## Decision 5: Register Router read routes with the existing Ledger adapter

- **Decision**: Add authenticated GET route registration around the existing
  `LedgerHandler` methods and require the production Ledger Store to satisfy
  both append and read ports.
- **Rationale**: T004 already owns storage and response validation; T008 only
  adds the process boundary wiring required for Control Plane reads.
- **Rejected alternatives**: a second read store, a Control Plane table query,
  or a new unversioned route.

## Decision 6: Reuse the required Gateway deadline

- **Decision**: Bound Workspace authorization and the Router GET with the
  existing required Gateway invocation deadline configuration.
- **Rationale**: A metadata read is a managed dependency operation and must not
  wait forever when Router or PostgreSQL is unavailable. Reusing the approved
  deadline avoids a new inferred timeout policy.
- **Rejected alternatives**: an unbounded HTTP client, a hard-coded timeout,
  or a retry/reconnect loop.

## Decision 7: Bound and validate Router success bodies at Gateway

- **Decision**: Require a separate metadata response limit, read one bounded
  body, reject duplicate/unknown members and trailing JSON, then validate the
  active InvocationDetail/Trace DTO and Workspace/Trace correlation before
  writing HTTP 200.
- **Rationale**: Router is the Ledger source of truth, but the Gateway is the
  public disclosure boundary. A malformed or content-bearing internal response
  must not become a successful public response, and an unbounded body is not a
  safe dependency boundary.
- **Rejected alternatives**: blindly copying 200 bytes, reusing the request or
  Agent limit, silently dropping unknown fields, or buffering without a strict
  bound.

## Evidence Sources

- `specs/010-invocation-routing-ledger/tasks.md` T008 and acceptance profile
- `contracts/openapi/control-plane-invocation.v4.yaml`
- `contracts/openapi/router-internal.v3.yaml`
- `contracts/runtime_contracts_validation.go`
- `apps/a2a-router/internal/ledger/store.go`
- `apps/a2a-router/internal/api/ledger_handler.go`
- `apps/control-plane/internal/workspace/service.go`
- `apps/control-plane/internal/invocation/router_client.go`
