# Data Model: Public Agent Share URL

## Public Agent Identity

Catalog-owned, one-to-one with an existing Agent identity.

| Field | Type | Rules |
| --- | --- | --- |
| `agent_id` | existing varchar(128) | Provider/Card identity primary key |
| `public_agent_id` | varchar(36) | Required, globally unique, `^agt_[0-9a-f]{32}$`, immutable |
| `owner_id` | existing varchar(128) | Unchanged |
| `created_at` | existing timestamptz | Unchanged |

New identities receive 128 random bits. Existing identities receive a stable value exactly once in migration v5. A database unique index arbitrates the theoretical collision; registration fails explicitly rather than choosing another identity inside the transaction.

## Canonical Agent Share URL

Derived response value, not a stored second fact:

```text
{configured_public_origin}/a/{public_agent_id}
```

The configured origin has scheme and authority only. The path has exactly two segments and no query, fragment, credentials, Workspace ID, token, endpoint, or Release ID.

## Public Agent View

A Catalog-owned read projection:

```text
public_agent_id
public_url
registered_at
availability: installable | not_installable
releases[]
```

The identity envelope contains no Card-derived fields. Draft Card version, Agent ID, name, description, owner, protocol, endpoint, authentication, permissions, capabilities, and verification evidence are not emitted when there is no eligible published trusted Release.

## Public Trusted Release

Each item maps one exact `catalog.agent_releases` row and matching Agent Card version.

```text
release_id
agent_id
name
description
owner { id, display_name }
agent_card_version
card_digest
published_at
skills[] { id, name, description, input_schema, output_schema, required_permissions }
permissions[] { id, description }
limits { timeout_ms, max_input_bytes, max_output_bytes, streaming }
authentication_type
```

Eligibility requires all of:

- Agent version `publication_status = published`;
- Release `state = published`;
- Release Agent/version/digest exactly match the stored Card;
- non-null Release publication time.

Rows are ordered by `published_at DESC`, then Release ID for deterministic presentation. Order never chooses a Release.

## State Semantics

```text
registered identity + zero eligible Releases -> not_installable, with no Card-derived fields
registered identity + one or more eligible Releases -> installable
unknown public identity -> NOT_FOUND
malformed identity -> VALIDATION_ERROR
Catalog inconsistency or query failure -> DEPENDENCY_ERROR
```

Suspended, revoked, draft, pending-verification, verified-only, disabled, or legacy-unverified versions are excluded from the eligible list. Installation revalidates the selected exact Release at write time, so a lifecycle race cannot create an enabled Installation.

## Workspace Installation

No schema change. A share-originated install is the existing Installation:

```text
workspace_id
agent_id
version_constraint = exact selected agent_card_version
installed_version
installed_release_id = selected release_id
accepted_permissions
enabled
installed_at
```

The navigation source is not persisted because it does not change Installation meaning.
