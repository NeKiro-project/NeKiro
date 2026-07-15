# Data Model: Installation Inspection

## Workspace-Owned Facts

Issue #7 reads the existing Workspace-owned tables; it does not add columns or
change ownership.

| Entity | Fields used by inspection | Ownership |
| --- | --- | --- |
| Workspace | `workspace_id`, `owner_id`, `created_at`, `updated_at` | Workspace module |
| Installation | `installation_id`, `workspace_id`, `agent_id`, `version_constraint`, `installed_version`, `accepted_permissions`, `status`, `installed_at`, `updated_at`, `uninstalled_at` | Workspace module |

## Installation History Invariants

- `workspace_id` is the authorization scope and query partition.
- `installation_id` is globally unique and is the second ordering key.
- `installed_at` is the first ascending ordering key.
- `status` can be `enabled`, `disabled`, or `uninstalled`; all three are
  included in inspection.
- An uninstalled row remains durable and has `uninstalled_at` equal to its
  terminal `updated_at`.
- The original constraint, exact pin, permission snapshot, and installation ID
  are immutable facts regardless of lifecycle status.

## Cursor Position

The public cursor is an opaque encoding of:

```text
version
workspaceId
limit
installedAt
installationId
```

The internal position is only `(installedAt, installationId)`. A valid cursor
is accepted only when its version, Workspace ID, page size, safe identifier,
and non-zero timestamp all match the current request. The next query applies a
strict lexicographic greater-than predicate, preventing duplicates and
omissions on unchanged data.

## Response Shapes

- Non-empty page: `items` contains at most the requested limit and may include
  `nextCursor` when more rows exist.
- Final non-empty page: `items` contains the remaining rows and omits
  `nextCursor`.
- Genuine empty history: `items` is a non-null empty array and omits
  `nextCursor`.
- Failure: no InstallationList or Installation fact is returned.
