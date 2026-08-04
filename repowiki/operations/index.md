---
layout: default
title: Operations
description: Core development checks and trusted publication operations.
permalink: /operations/
nav_order: 5
---

# Operations

This section is the operational entry point for the Core repository. The
Console, sample Agents, Compose stack, and product acceptance remain owned by
the satellite repositories in the [repository map]({{ '/repositories/' | relative_url }}).

## Local Core checks

Core requires Go 1.26 or newer and PostgreSQL for integration suites. The
baseline quality sequence is:

```text
gofmt -l apps contracts tests
go mod tidy
go build ./...
go test ./...
go test -race ./...
go vet ./...
```

PostgreSQL integration tests must use an explicitly configured dedicated
database whose name ends in `_test`. Missing or invalid required configuration
is a readiness failure; it does not receive a guessed default.

## Trusted publication path

Trusted Publication v1 is a Registry-owned proof step before an endpoint can
participate in a verified Agent Release. It does not deploy an Agent and never
allows the Console or a caller to bypass the Router.

The operator runbook follows this order:

1. Prepare an explicit Gateway request and deployment configuration.
2. Register and publish the versioned Agent Card and trusted Release facts.
3. Inspect trust and Invocation provenance through Gateway APIs.
4. Suspend or revoke through the owning lifecycle operation when required.
5. Validate the complete acceptance path through the pinned Stack revisions.

The runbook does not use SQL to mutate domain state, call an Agent directly as
a recovery path, mutate an immutable Release, or hide a dependency failure
behind an alternate endpoint.

## Recovery posture

Failure categories remain explicit. A missing Card, invalid request,
unauthorized Workspace, disabled Installation, disabled Agent version, failed
dependency, timeout, cancellation, and rejected trust proof each retain their
own error and audit meaning. Fallback budget is zero unless an existing
contract, ADR, runbook, or SLO provides evidence for a different policy.

## Canonical source

- [Core development]({{ '/source-docs/usage/core-development/' | relative_url }})
- [Trusted publication operations]({{ '/source-docs/usage/trusted-publication-operations/' | relative_url }})
- [Trusted Publication v1]({{ '/source-docs/contracts/trusted-publication-v1/' | relative_url }})
