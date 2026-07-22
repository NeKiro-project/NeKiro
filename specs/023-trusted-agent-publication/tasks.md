# Tasks: Trusted Agent Publication

## Slice A — Provider identity and endpoint ownership (#48)

- [x] T001 Add versioned trusted-publication contract models and JSON Schema.
- [x] T002 Add Registry Provider, Endpoint Binding, and Challenge data models
      with explicit states and typed errors.
- [x] T003 Add catalog migration and readiness checks for provider/binding/
      challenge ownership.
- [x] T004 Add strict endpoint parser and network policy validator; reject
      credentials, unsupported schemes, redirects, and disallowed addresses.
- [x] T005 Add challenge creation/completion service with single-use expiry,
      exact proof comparison, and no secret persistence.
- [x] T006 Add authenticated Gateway routes and OpenAPI contract mappings.
- [x] T007 Add unit, contract, and integration tests for success and all
      specified negative paths.
- [x] T008 Update provider/operator documentation and link #48/#47.

## Slice B — Immutable release lifecycle (#49)

- [ ] T009 Add immutable release identity, digest, endpoint binding, and state
      transitions.
- [ ] T010 Gate Workspace resolution on verified published releases.
- [ ] T011 Add release migration and lifecycle contract tests.

## Slice C — Router-to-Agent trust (#50)

- [ ] T012 Define Router credential contract and explicit key configuration.
- [ ] T013 Sign and validate short-lived Router-to-Agent credentials.
- [ ] T014 Add forged, expired, audience, direct, JSON/SSE, and nested tests.

## Slice D — Client SDK (#51)

- [ ] T015 Add lightweight Go Client SDK through Gateway.
- [ ] T016 Add SDK contract tests and application example.

## Slice E — Acceptance (#52)

- [ ] T017 Add clean Compose E2E for Register -> Verify -> Publish -> Install
      -> Invoke -> Record and all negative paths.
- [ ] T018 Add operator/provider runbook and convergence evidence.

## Dependency order

```text
T001 -> T002 -> T003 -> T004 -> T005 -> T006 -> T007 -> T008
T008 -> T009 -> T010 -> T011
T011 -> T012 -> T013 -> T014
T014 -> T015 -> T016
T016 -> T017 -> T018
```

## Ownership / parallelism

- T001/T004 can be prepared in parallel after the Spec review; they touch
  contracts and a new validation package respectively.
- T002/T003/T005 are Catalog-owned and must not modify Workspace or Router
  tables.
- T006 owns Gateway routes only; it consumes Catalog ports.
- T007/T008 may run after the service contract stabilizes.
