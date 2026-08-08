# Config Center Runtime Operations

## Bootstrap

Router bootstrap must set `NEKIRO_ROUTER_INSTANCE_ROUTING_MODE` to exactly
`direct` or `config_center_file`. File mode additionally requires an absolute
root, a positive payload limit, a strict configuration key, and an instance
port name. The mapped file must contain a valid
`router-instance-directory.v1` document before the Router starts.

Secrets, database URLs, Router signing keys, and service authentication remain
bootstrap configuration. They are not accepted in the instance directory.

## Rollout

1. Publish and authorize the exact Agent Release through the Control Plane.
2. Build a directory target from the published Release ID, Card digest, Agent
   identity, canonical endpoint, and canonical audience.
3. Publish the complete replacement document atomically.
4. Wait for `/readyz` to return `200` with `{"status":"ok"}`.
5. Invoke through the Control Plane and verify the normal Ledger lineage.

The current selector requires exactly one ready TCP endpoint for the configured
port name. Do not publish multiple ready endpoints until a selection policy is
approved and deployed.

## Rejected Updates

Malformed JSON, duplicate members, unknown fields, unsupported schema versions,
invalid Release facts, duplicate Release IDs, invalid addresses, and invalid
instance topology make readiness return `503`. New Invocations fail closed.
Existing streams keep their already selected instance and credential audience.

No last-known-good document is retained. Fix the document at the configured
source; do not switch sources or enable direct routing as an incident fallback.

## Deletion And Outage

Deleting the configured key, removing the source root, losing permissions, or
interrupting the File watcher makes the directory unavailable. Readiness
returns `503`, and affected new Invocations fail with dependency semantics.
Configuration bytes, paths, source errors, and credentials are never returned
by readiness.

## Recovery

Restore the same configured source and publish one complete valid document.
If the File reader has entered a terminal interrupted state, restart the Router
after repairing the source. Confirm readiness, then issue a new Invocation.
The Router never retries or reroutes an already failed Invocation.
