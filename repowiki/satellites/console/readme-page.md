<div class="satellite-source-note">Read-only mirror of <a href="https://github.com/NeKiro-project/NeKiro-Console/blob/5e577d86825e2ff80434e752342990b9b313d947/README.md"><code>NeKiro-project/NeKiro-Console/README.md</code></a> at <code>5e577d86825e2ff80434e752342990b9b313d947</code>. The canonical document remains in the satellite repository; edit it there and refresh this snapshot.</div>

# NeKiro Console

The production React/Vite/TypeScript Console for the NeKiro Agent Operating
Platform. This repository is the canonical Console source; the core platform
repository does not maintain a production UI copy after the repository split.

The Console talks only to the NeKiro Gateway. It supports trusted Agent
publication, public Agent share URLs, Catalog discovery, exact Release
installation, managed JSON/SSE invocation, and Workspace-scoped Ledger reads.

## Configuration

Copy `.env.example` to `.env.local` and provide every value explicitly:

```text
VITE_NEKIRO_API_BASE_URL=https://gateway.example.test
VITE_NEKIRO_PROVIDER_ID=provider-id
VITE_NEKIRO_PROVIDER_NAME=Provider Name
VITE_NEKIRO_PROVIDER_TOKEN=provider-token
VITE_NEKIRO_OWNER_TOKEN=workspace-owner-token
VITE_NEKIRO_DEFAULT_WORKSPACE_ID=workspace-id
VITE_NEKIRO_PUBLIC_AGENT_ORIGIN=https://agents.example.test
```

Provider and Workspace credentials are sent only as authorization headers and
are not written to browser storage. Missing, blank, whitespace-padded, or
otherwise invalid required configuration fails at startup.

## Development

```text
pnpm install --frozen-lockfile
pnpm dev
```

## Verification

```text
pnpm typecheck
pnpm test
pnpm build
```

Console CI verifies only Console-owned behavior. Full backend/browser product
acceptance is owned by
[NeKiro-Stack](https://github.com/NeKiro-project/NeKiro-Stack), which checks out
an exact Console commit and invokes the retained Playwright suite against its
immutable component manifest.

## Related repositories

- [NeKiro core](https://github.com/NeKiro-project/NeKiro)
- [NeKiro Stack](https://github.com/NeKiro-project/NeKiro-Stack)
- [NeKiro Samples](https://github.com/NeKiro-project/NeKiro-Samples)
- [NeKiro Go SDK](https://github.com/NeKiro-project/nekiro-sdk-go)

Fallback behavior is not implemented in the Console. Dependency, contract,
authorization, lifecycle, timeout, cancellation, and malformed-response states
remain explicit failures.
