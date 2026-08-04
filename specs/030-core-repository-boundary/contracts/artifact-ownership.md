# Artifact Ownership Contract

| Current family | Canonical owner | Release unit | Disposition | Required gate |
|---|---|---|---|---|
| `apps/control-plane/**` | `NeKiro` | Control Plane image/binary | Retain | Core quality and PostgreSQL integration |
| `apps/a2a-router/**` | `NeKiro` | Router image/binary | Retain | Core quality and Ledger integration |
| `contracts/**` | `NeKiro` | Core module/contract identity | Retain | Schema/OpenAPI conformance |
| service migrations | `NeKiro` owning module | Owning service release | Deduplicate | Fresh, upgrade, version, readiness tests |
| `tests/contract/**` | `NeKiro` | Core verification | Retain | Core CI |
| `tests/integration/**` | `NeKiro` | Core verification | Retain | Core CI with PostgreSQL |
| `apps/console/**` | `NeKiro-Console` | Console release | Extract | Typecheck, unit, build, owned browser tests |
| `sdks/**` | `nekiro-sdk-go` | Go module | Extract | Format, build, test, race, vet, tidy, boundary |
| `agents/**` | `NeKiro-Samples` | Go module and two images | Extract | Per-runtime quality and image builds |
| `deploy/**` | `NeKiro-Stack` | Product manifest | Extract | Manifest and Compose validation |
| `tests/e2e/**` | `NeKiro-Stack` | Product acceptance | Extract | Backend and browser acceptance |
| root Node manifests | `NeKiro-Console` | Console toolchain | Remove from core | Console frozen install and build |
| `.env.example` full stack values | `NeKiro-Stack` | Stack configuration | Extract | Explicit configuration validation |
| core environment reference | `NeKiro` | Core usage docs | Retain as documentation | Core config tests/docs review |
| `docs/architecture/**` | `NeKiro` | Core documentation | Retain | Link and scope review |
| `docs/contracts/**` | `NeKiro` | Contract documentation | Retain | Contract conformance |
| `docs/decisions/**` | `NeKiro` | Architecture record | Retain | ADR review |
| core runbooks | `NeKiro` | `docs/usage/**` | Retain/move | Operator command validation |
| roadmap/handoff/bugfix/delivery reports | GitHub/Wiki/history | Historical record | Remove from default tree | Pre-split tag and link audit |
| `specs/**`, `.specify/**`, Spec Kit skills | Git history/tag | Historical record | Remove after Spec 030 | Final tracked-tree scan |
| reusable A2A transport | `nekiro-a2a-transport-go` | Go module | Already external | Transport CI and SemVer release |

## Dependency Rules

- Core imports only its own public contracts and approved external libraries.
- SDK imports public core contracts; it never imports `apps/**` or internal
  packages.
- Samples import released core contracts and SDK packages; they never use a
  local replacement to a sibling checkout.
- Console calls Gateway through published northbound contracts.
- Stack consumes immutable releases and contains no maintained component source.
- Core never checks out Console, SDK, Samples, or Stack during CI.
