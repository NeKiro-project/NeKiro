# Research: Invocation Ledger

## Decision 1: Projection-row locking

Use `SELECT ... FOR UPDATE` on the Invocation projection for every event after
creation. It serializes lifecycle validation per Invocation while allowing
different Invocations to append concurrently. Rejected alternative: deriving
the next sequence with `MAX(sequence)` because it cannot atomically protect
context and terminal state.

## Decision 2: Validate history inside the append transaction

Load ordered prior events after acquiring the projection lock and replay them
through `RuntimeInvocationSequenceValidator`, then accept the candidate. This
keeps the active contract validator authoritative instead of duplicating its
state machine in Ledger code. Rejected alternative: trusting only caller input
or adding a second independent transition table.

## Decision 3: Fixed scalar schema

Use explicit columns and persist only `error_code`, not an event JSON document
or error message. This makes prohibited content structurally unrepresentable.
The fixed safe Platform Error v4 message is reconstructed from the code.

## Decision 4: Parent locking for nested creation

A child sequence-zero transaction locks the parent projection and validates
running state plus exact lineage before child insert. This gives deterministic
behavior when parent terminalization races child acceptance.

## Decision 5: No database retry

Serialization, constraint, connectivity, and commit failures are returned as
explicit conflict or dependency errors. No retry, alternate connection/store,
or reconciliation worker is authorized.
