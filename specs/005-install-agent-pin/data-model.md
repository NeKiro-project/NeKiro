# Data Model: Installation Pin

## Installation

The Workspace module owns one durable Installation row for each install
attempt. A current row is enabled or disabled; an uninstalled row is historical.

| Field | Meaning | Mutability in #5 |
| --- | --- | --- |
| `installation_id` | Server-assigned safe identity | Immutable |
| `workspace_id` | Authorization root | Immutable |
| `agent_id` | Exact Catalog Agent identity | Immutable |
| `version_constraint` | Submitted owner intent | Immutable |
| `installed_version` | Exact selected Card version | Immutable |
| `accepted_permissions` | Canonical exact subset snapshot | Immutable |
| `status` | Starts `enabled` | Set only by later lifecycle feature |
| `installed_at` | Committed server time | Immutable |
| `updated_at` | Committed current time | Set only by later lifecycle feature |
| `uninstalled_at` | Terminal history time | Not set by #5 |

## Installation Invariants

- `workspace_id` references an existing Workspace.
- `(workspace_id, agent_id)` has at most one row whose status is not
  `uninstalled`.
- `accepted_permissions` is a non-null, unique, ascending bytewise array;
  empty is represented as an empty array.
- `installed_version` is the exact Card version selected by Catalog and
  satisfies the submitted constraint.
- New rows are `enabled` and have `installed_at <= updated_at`.
- All response timestamps are values returned by the committed database row.

## Operation Ordering and Failure Ownership

The Gateway authenticates and validates shape; Workspace authorizes and
validates the permission subset; Catalog selects; PostgreSQL persists. A
failure at each boundary remains owned by that boundary and is not converted
to empty success or a different source.
