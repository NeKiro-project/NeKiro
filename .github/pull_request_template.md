## Summary

<!-- Describe the Core behavior changed and its owning domain. -->

## Architecture and contract impact

- Owner: Control Plane / A2A Router / contracts / owned migrations
- Affected contract versions:
- Compatibility or migration decision:
- Satellite repositories requiring verification:

- [ ] No public contract, data model, or ownership boundary changed.
- [ ] Compatible changes preserve omitted-field and failure semantics.
- [ ] Breaking changes include an ADR, new contract version, and migration plan.

## Verification

Commands run:

```text
go build ./...
go test -count=1 ./...
go test -race ./...
go vet ./...
# relevant PostgreSQL integration suites
```

Observed success signals:

<!-- Include contract, PostgreSQL, image, and Stack evidence appropriate to the change. -->

## Security and failure semantics

- [ ] Gateway/Router/Registry/Ledger ownership remains intact.
- [ ] Secrets, Agent payloads, and credentials are absent from Card/log/event/Ledger facts.
- [ ] Missing, invalid, unauthorized, timeout, cancellation, and dependency failures remain distinct.

Fallback delta: removed 0, retained 0, added 0, net 0

Added fallback evidence: none

## Checklist

- [ ] Tests cover affected success and failure paths.
- [ ] Architecture, contract, or usage documentation was updated where required.
- [ ] Dependencies, Actions, and reusable workflows use immutable revisions.
- [ ] Required satellite and Stack follow-up work is linked.
