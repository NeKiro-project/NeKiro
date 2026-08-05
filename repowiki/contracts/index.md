# Contracts and compatibility

Contracts are versioned before cross-boundary implementation. Language-neutral
schemas, OpenAPI documents, A2A profiles, and their Go mappings are the
authoritative sources; service and SDK types are consumers.

## Independent version axes

These values must not be inferred from one another:

- Agent Card Schema version
- Agent version and Release identity
- Northbound HTTP API version
- Internal API version
- Event and result version
- A2A Profile Schema and protocol version
- Router credential version

## Active Phase 1 surface

| Boundary | Active contract | Meaning |
| --- | --- | --- |
| Catalog, Workspace, Installation | Northbound API v3 | Registration, publication, discovery, Workspace, and Installation |
| Invocation and Trace | Northbound API v4 | Workspace-scoped invocation and metadata reads |
| Exact Card resolution | Control Plane Internal v2 | Router resolves the authorized exact Card and provenance |
| Nested installed-version resolution | Control Plane Internal v3 | Router resolves the exact enabled Installation pin |
| Router dispatch | Router Internal v4 | Control Plane dispatches an authorized root Invocation |
| Router metadata reads | Router metadata v3 | Workspace-scoped Invocation and Trace projections |
| Agent-facing A2A | A2A 0.3.0 with profile schema 0.2 | Supported JSON-RPC methods, context headers, and streaming subset |
| Router-to-Agent authentication | Router credential v1 | Fresh Ed25519 request binding for each managed HTTP request |
| Invocation facts | Invocation Event 0.3 | Append-only metadata and lineage, without Agent payloads |
| Stream result | Result Stream Event v2 | Ordered transient events with one immutable terminal outcome |

## Compatibility rules

An optional field is compatible only when omission preserves existing meaning.
Removing or renaming fields, changing types or requiredness, tightening
validation, changing status/media type/error semantics, moving ownership, or
reinterpreting historical Ledger facts is breaking.

Breaking changes require a new contract version, migration guidance, and a
clear compatibility window—or an explicit pre-runtime decision that no window
is justified. Historical files remain unchanged migration evidence; the active
runtime does not add speculative dual-read, dual-write, or fallback behavior.

## Failure and data safety

Missing, invalid, unauthorized, disabled, not-found, timeout, cancellation,
dependency, and protocol failures keep distinct status and error semantics.
Public errors contain fixed safe messages and correlation identifiers only.
Agent inputs, outputs, endpoint details, credentials, raw dependency errors, and
stack data do not cross into Cards, events, logs, or Ledger facts.

## Canonical source

- [Contract Compatibility Policy](../source-docs/contracts/compatibility.md)
- [Trusted Publication v1](../source-docs/contracts/trusted-publication-v1.md)
- [Phase 1 contract sources](../source-docs/architecture/phase-1-spec.md)
- [Source documents](../source-docs/index.md)
