# Data Model: Invocation Ledger

## `ledger.invocation_events`

Primary key `event_id`; unique `(invocation_id, sequence)`. Required scalar
columns hold Event 0.3 identity, exact context, type/status, and timestamp.
Nullable scalar columns are limited to parent ID, chunk index/bytes, terminal
latency, and fixed error code. Checks enforce type/status field combinations
and non-negative counters. There is no payload, endpoint, credential, message,
or generic JSON column.

## `ledger.invocations`

Primary key `invocation_id`. Immutable context mirrors sequence zero. Mutable
projection fields are current status, optional terminal latency/error code,
`created_at`, and `updated_at`. Checks enforce terminal metadata shape and
timestamp order.

Indexes:

- `(workspace_id, trace_id, created_at, invocation_id)` for Trace reads.
- `(workspace_id, root_task_id, created_at, invocation_id)` for task lineage.
- `(workspace_id, parent_invocation_id, created_at, invocation_id)` for children.

## Invariants

- Sequence zero is `created/pending`; later sequence is exactly previous + 1.
- Context never changes after sequence zero.
- Lifecycle follows the active Event 0.3 state machine.
- Stream chunk index starts at zero and advances without gaps.
- First terminal is final.
- Child parent is running, in the same Workspace, has matching root/Trace, and
  its target Agent equals the child's authenticated Agent caller.
- Event and projection mutation are one transaction.
