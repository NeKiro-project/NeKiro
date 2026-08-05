---
layout: default
title: NeKiro RepoWiki
description: A unified map of the NeKiro platform, its contracts, operating rules, decisions, and owning repositories.
permalink: /
nav_order: 1
---

# NeKiro RepoWiki

This is the central reading surface for the NeKiro platform. It turns the
repository's accepted architecture, contracts, decisions, and operating guides
into one navigable map while keeping each fact in its canonical owner.

The platform loop is:

```text
Register -> Discover -> Install -> Invoke -> Record
```

## Start here

- [Architecture and ownership]({{ '/architecture/' | relative_url }}) — what Core owns, where the trust boundaries are, and how the services interact.
- [Platform lifecycle]({{ '/architecture/lifecycle/' | relative_url }}) — the five-step operating loop and its failure semantics.
- [Contracts and compatibility]({{ '/contracts/' | relative_url }}) — active versions, migration rules, and data-safety constraints.
- [Operations]({{ '/operations/' | relative_url }}) — local verification and trusted-publication procedures.
- [Decision record]({{ '/decisions/' | relative_url }}) — accepted ADRs that explain why the boundaries exist.
- [Repository map]({{ '/repositories/' | relative_url }}) — one entry point for every NeKiro repository.
- [Source documents]({{ '/source-docs/' | relative_url }}) — the original `docs/` files rendered into the Wiki at build time.

## The platform in one view

NeKiro operates Agents from the outside. Agent Runtimes own model, Prompt,
Tool, Planner, Workflow, Memory, RAG, Session, and runtime telemetry behavior.
Core owns publication, discovery, Workspace authorization, exact-version
resolution, managed routing, and metadata-only invocation lineage.

```text
Caller / Console
       |
       v
Gateway / Control Plane -----> Catalog + Workspace
       |
       v
A2A Router ------------------> Agent endpoint
       |
       v
Invocation Ledger (metadata and lineage only)
```

## Ownership rule

The central Wiki is a portal, not a second writable source for satellite
repositories. Core documentation is rendered here from `docs/`. Console, SDK,
Samples, Stack, and A2A Transport facts remain owned by their respective
repositories and are linked from the [repository map]({{ '/repositories/' | relative_url }}).

## What this Wiki deliberately does not promise

- It does not define Agent-internal execution behavior.
- It does not replace versioned files under `contracts/`.
- It does not copy satellite source code or silently infer their release state.
- It does not turn a dependency failure into an empty result, a success, or a fallback endpoint.
