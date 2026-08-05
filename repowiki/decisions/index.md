---
layout: default
title: Decision record
description: Accepted architecture and ownership decisions for NeKiro Core.
permalink: /decisions/
nav_order: 6
---

# Decision record

The ADRs explain the durable boundaries behind the platform. The Wiki provides
one index; the source files remain canonical and are rendered under
[Source documents]({{ '/source-docs/' | relative_url }}).

| ADR | Decision | Why it matters |
| --- | --- | --- |
| 0001 | [Go Backend And Router]({{ '/source-docs/decisions/0001-go-backend-stack/' | relative_url }}) | Fixes Go for Core services and the Router boundary. |
| 0002 | [Invocation Result Transport and Internal API Direction]({{ '/source-docs/decisions/0002-invocation-result-transport-and-internal-api-direction/' | relative_url }}) | Keeps result delivery transient and internal APIs directional. |
| 0003 | [Runtime-Agnostic Platform Boundary]({{ '/source-docs/decisions/0003-runtime-agnostic-platform-boundary/' | relative_url }}) | Keeps Agent Runtime behavior outside Core. |
| 0004a | [Catalog Persistence and Strong Discovery Consistency]({{ '/source-docs/decisions/0004-catalog-persistence-and-consistency/' | relative_url }}) | Makes Registry ownership and discovery consistency explicit. |
| 0004b | [Trusted Agent Publication and Endpoint Ownership]({{ '/source-docs/decisions/0004-trusted-agent-publication/' | relative_url }}) | Establishes trusted Release provenance and endpoint ownership. |
| 0005 | [Minimal Workspace and Installation Boundary]({{ '/source-docs/decisions/0005-minimal-workspace-installation-boundary/' | relative_url }}) | Freezes the smallest durable authorization and pinning fact. |
| 0006 | [Invocation Runtime Trust and Failure Policy]({{ '/source-docs/decisions/0006-invocation-runtime-trust-and-failure-policy/' | relative_url }}) | Defines deadlines, cancellation, failure, size, and Ledger semantics. |
| 0007 | [Router-to-Agent Signed Invocation Credential]({{ '/source-docs/decisions/0007-router-agent-signed-credential/' | relative_url }}) | Binds each managed request to exact provenance without persisting secrets. |
| 0008 | [Standalone A2A Transport Module]({{ '/source-docs/decisions/0008-standalone-a2a-transport-module/' | relative_url }}) | Keeps reusable wire mechanics outside Core policy. |
| 0009 | [Core Repository and Satellite Ownership]({{ '/source-docs/decisions/0009-core-repository-boundary/' | relative_url }}) | Establishes the multi-repository ownership and release order. |

The source set currently contains two descriptive files numbered `0004`. The
Wiki labels them `0004a` and `0004b` only for navigation; the canonical files
and their history are unchanged.
