# PR #42 review convergence

## Symptom

The latest PR review round left fourteen unresolved inline findings across
nested Agent authorization, Control Plane Internal v3 error handling, root
dispatch contract shape, SDK limits/SSE/error exposure, handler coverage, and
Spec-Driven Development artifacts.

## Scope

Only the PR #42 Agent SDK nested-invocation slice and its active contracts,
tests, ADR/compatibility records, and package documentation are in scope.
No unrelated runtime, deployment, or frontend work is included.

## Reproduction / expected result

Review the unresolved threads on PR #42, exercise `go test ./...` and
`go vet ./...`, and inspect the changed contract/SDK paths. Every fixed inline
finding should have a corresponding code, contract, test, or governance change;
unsupported race validation remains an environment limitation rather than a
silent pass.

## Actual result before this fix

Agent credentials were Agent-only, v3 errors accepted phase-invalid shapes, the
SDK supplied a 16 MiB default and buffered SSE, RouterError exposed raw bytes,
root OpenAPI advertised parent lineage, and SDD/active-contract records were
stale.
