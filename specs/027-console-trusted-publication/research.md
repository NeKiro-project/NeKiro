# Research: Console Trusted Publication Loop

## Existing Evidence

### Console source

The cloned `NeKiro-Console` repository already contains the production Glass
2.0 route and isolated `#/demo`, `#/demo/glass`, `#/demo/terminal`, and
`#/demo/saas` routes. Its live client already implements strict-ish mappings for
Agent Card 0.2, Catalog v3, Workspace/Installation v2, Gateway Invocation v4,
Invocation/Trace reads, JSON results, and SSE results. Its production forms use
explicit environment values and do not persist the bearer token in local
storage.

The missing product behavior is the Trusted Publication v1 client and UI. The
current Registry action calls the legacy Catalog publish/disable route. That is
not an acceptable trust path for this Spec and must be replaced or hidden for
the production flow; no compatibility fallback is allowed.

### Active Gateway routes

The platform repository exposes the following public routes through the Gateway:

- `POST/GET /v4/providers/{providerId}/agents/{agentId}/endpoint-bindings`
- `GET /v4/providers/{providerId}/endpoint-bindings/{bindingId}`
- `POST /v4/providers/{providerId}/endpoint-bindings/{bindingId}/challenges`
- `POST /v4/providers/{providerId}/endpoint-bindings/{bindingId}/challenges/{challengeId}/complete`
- `POST /v4/providers/{providerId}/agents/{agentId}/releases`
- `GET /v4/releases/{releaseId}`
- `POST /v4/releases/{releaseId}/verify|publish|suspend|revoke`

Responses use strict additional-property-free Trusted Publication v1 shapes and
stable error codes. The routes return a trace header and safe error payload; the
browser must not read internal Router or Catalog routes.

### Backend acceptance

The existing clean Compose acceptance proves trusted Register -> Verify ->
Publish -> Install -> Invoke -> Record and an A -> Router -> B nested path.
The requested flow needs the reverse call direction as a separately asserted
fixture. The existing Agent SDK already supports nested calls and the Router
already owns correlation and credentials, so the acceptance should change the
fixture request/expected target direction rather than add a new runtime.

### CI behavior

Root `package.json` runs workspace scripts recursively with `--if-present`.
Because `apps/console` is currently absent, root `pnpm typecheck`, `pnpm test`,
and `pnpm build` can succeed without running a production frontend. After the
standalone Console slices are reviewed, importing the resulting source with a
real `apps/console/package.json` makes discovery observable. Browser acceptance
must be an explicit job or test command and must fail if its required
Gateway/Workspace configuration is absent.

## Decisions

| Question | Decision | Evidence / reason |
| --- | --- | --- |
| Where is production Console source? | `apps/console` in the platform repo | AGENTS.md target structure and root pnpm workspace/CI contract |
| How are trust operations called? | Existing public Gateway Trusted Publication v1 routes | Gateway handlers and OpenAPI/schema in `contracts/` |
| What is the Card publication authority? | Trusted Release state, not legacy Catalog publish | Trusted Release lifecycle and Installation gates already merged |
| How is challenge proof handled? | Transient memory only, never persisted or echoed into later requests | Trust contract returns it once; constitution forbids secret persistence/leakage |
| How are mutations reflected? | Use returned value and explicit follow-up read where a view depends on authoritative state | Avoid optimistic/fabricated state and stale destructive controls |
| How is reverse nesting proven? | Extend the existing single acceptance fixture | Avoid a second stack and preserve one Ledger/Trace authority |
| Is a new fallback allowed? | No | Constitution VII and Spec FR-012; missing policy is not a reason to continue |
| What is the trace header rule? | Error-body `traceId` is authoritative; an optional `x-nek-trace-id` header must match when present | The active OpenAPI declares the header but not its requiredness or equality semantics; accepting absence preserves compatibility while rejecting disagreement prevents ambiguous correlation |
| Where does Slice A write? | Standalone `NeKiro-Console` | Issue #2 is filed in that repository; the reviewed source is imported into `apps/console` only in Slice D |

### Phase 7 SemVer compatibility

Installation response validation now uses `semver@7.8.5` with strict numeric
parsing. The adapter preserves the active Go constraint forms `=>`, `=<`,
`~>`, partial prerelease versions such as `>=0-0`, and wildcard `!=` by
translating only those syntax forms before comparison. It also preserves the
backend's 512-byte range and 32-OR-branch limits, rejects empty OR branches
and repeated comma separators, and keeps npm's prerelease tuple rule so a
prerelease comparator does not open unrelated prerelease tuples.

The Go `Masterminds/semver` parser accepts leading-zero range components while
the browser parser deliberately rejects them as non-canonical input, matching
the strict SemVer policy already used for Agent Card versions. The exact
boundary is covered by the API regression matrix rather than hidden behind a
coercion or fallback.

### Phase 9 boundary remediation

The Console keeps active backend `uint64` SemVer compatibility without
converting components to JavaScript `number`. Components at or above
`Number.MAX_SAFE_INTEGER` are ranked with `BigInt` before the maintained npm
parser evaluates the normalized range; this preserves implicit successors in
caret and wildcard ranges. Failed authoritative Binding reads clear the
Binding, Release, handoff IDs, and destructive confirmation state as one
transient UI invalidation. The remediation added no fallback, retry, alternate
endpoint, or secret handling path.

## Rejected Alternatives

- **Keep the standalone Console as a CI submodule**: rejected because root
  workspace CI would not own or reliably discover the production package, and
  it would create a second repository synchronization boundary.
- **Continue using Catalog publish for a fast UI**: rejected because it would
  expose an untrusted publication path and violate the immutable Release gate.
- **Poll endpoint health or automatically retry challenge/invocation**:
  rejected because no product/SLO policy authorizes those behaviors.
- **Add a new backend endpoint for the Console**: rejected because current
  Gateway routes already cover the required operations and no contract gap is
  evidenced.
- **Use optimistic UI state**: rejected because a failed or concurrent server
  transition must remain visible and authoritative.
