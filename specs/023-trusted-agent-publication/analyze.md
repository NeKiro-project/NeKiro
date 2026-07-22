# Analyze: Trusted Agent Publication

## Consistency result

**PASS for Slice A planning; implementation is scoped to T001–T008.**

- Constitution: provider/release verification is a Control Plane prerequisite
  for the Phase 1 loop and does not introduce runtime-framework behavior.
- Spec ↔ Plan: endpoint proof, network restrictions, typed failures, secret
  safety, and compatibility are represented in both artifacts.
- Plan ↔ Tasks: T001–T008 cover the complete Slice A design; later slices are
  explicitly deferred and dependency-ordered.
- Contracts ↔ ownership: Registry owns provider/binding/challenge facts;
  Gateway exposes only versioned northbound operations; Router and Workspace
  are not given storage access.
- Fallback policy: no default endpoint, localhost exception, retry, redirect,
  empty success, or dependency-to-business-state conversion is introduced.

## Risks carried into implementation review

1. DNS resolution can change between validation and connect. The HTTP client
   boundary must retain the approved policy and avoid following redirects.
2. Challenge proof must be compared in constant time and must not be logged.
3. Schema/migration readiness must fail closed when a new table or constraint
   is missing.
4. Existing legacy published samples must remain readable without being
   silently upgraded to verified.

## Implementation evidence

- Trusted Publication v1 JSON Schema and OpenAPI define exact Card-version
  binding, SemVer prerelease/build compatibility, and typed public verification
  errors.
- Registry persistence uses migration 003, an Agent-to-Provider first-claim
  rule, nullable evidence/failure fields, named FK/status/digest checks, and
  fail-closed readiness checks for required columns and types.
- Verification performs strict endpoint canonicalization, one-time challenge
  reservation, constant-time proof comparison, DNS policy checks, destination
  pinning, redirect rejection, explicit timeout/expiry handling, and TLS hook
  clearing.
- Gateway tests assert authenticated forwarding of provider, Agent, Card
  version, typed public errors, and absence of raw dependency/proof details.
- Catalog tests cover canonical aliases, reserved/private/loopback/link-local/
  multicast/unspecified/CGNAT ranges, DNS dependency/empty results, wrong
  proof, redirect, endpoint unavailability, expiry races, HTTPS pinning,
  suspended providers, endpoint mismatch, and concurrent challenges.
- `go test ./...`, `go vet ./apps/control-plane/...`, `golangci-lint run ./...`,
  `git diff --check`, and Compose config validation with explicit environment
  passed. PostgreSQL integration and Compose E2E require Docker/PostgreSQL;
  the local Docker daemon is not available. Race testing is environment
  blocked because the workstation has no C compiler for CGO.

## Fallback audit

Fallback delta: removed 0, retained 0, added 0, net 0.
Added fallback evidence: none.
