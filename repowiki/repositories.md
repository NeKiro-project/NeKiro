---
layout: default
title: Repository map
description: Canonical ownership and Wiki entry points for every NeKiro repository.
permalink: /repositories/
nav_order: 7
---

# Repository map

NeKiro is maintained as a set of independently released repositories. This
page gives one entry point without copying their source into Core.

| Repository | Canonical responsibility | Repository | Wiki |
| --- | --- | --- | --- |
| **NeKiro Core** | Control Plane, A2A Router, contracts, service-owned migrations, Core tests, architecture, and Core usage | [NeKiro-project/NeKiro](https://github.com/NeKiro-project/NeKiro) | [This RepoWiki](https://nekiro-project.github.io/NeKiro/) |
| **NeKiro Console** | Production Console and browser behavior | [NeKiro-project/NeKiro-Console](https://github.com/NeKiro-project/NeKiro-Console) | [Wiki](https://github.com/NeKiro-project/NeKiro-Console/wiki) |
| **nekiro-sdk-go** | Public Go Agent and application SDKs | [NeKiro-project/nekiro-sdk-go](https://github.com/NeKiro-project/nekiro-sdk-go) | [Wiki](https://github.com/NeKiro-project/nekiro-sdk-go/wiki) |
| **NeKiro Samples** | Independent sample Agent Runtimes, Cards, and sample-specific tests | [NeKiro-project/NeKiro-Samples](https://github.com/NeKiro-project/NeKiro-Samples) | [Wiki](https://github.com/NeKiro-project/NeKiro-Samples/wiki) |
| **NeKiro Stack** | Multi-component assembly, immutable release pins, and product acceptance | [NeKiro-project/NeKiro-Stack](https://github.com/NeKiro-project/NeKiro-Stack) | [Wiki](https://github.com/NeKiro-project/NeKiro-Stack/wiki) |
| **nekiro-a2a-transport-go** | Reusable A2A HTTP, JSON-RPC, and SSE transport mechanics | [NeKiro-project/nekiro-a2a-transport-go](https://github.com/NeKiro-project/nekiro-a2a-transport-go) | [Wiki](https://github.com/NeKiro-project/nekiro-a2a-transport-go/wiki) |

## Release and verification order

1. Core publishes versioned contracts and a verified Core revision.
2. SDK, Console, Samples, and transport repositories verify against that identity.
3. Stack pins immutable compatible revisions and runs product acceptance.
4. Compatibility evidence remains visible for the exact Core commit.

The central Wiki documents this order; it does not become an alternate release
authority. See [ADR 0009]({{ '/source-docs/decisions/0009-core-repository-boundary/' | relative_url }}) for the accepted ownership decision.
