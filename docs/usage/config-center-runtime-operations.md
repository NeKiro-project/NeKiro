# Config Center Runtime Operations

## Bootstrap

Router bootstrap must set `NEKIRO_ROUTER_INSTANCE_ROUTING_MODE` to exactly
`direct`, `config_center_file`, or `nacos`. File mode additionally requires an absolute
root, a positive payload limit, a strict configuration key, and an instance
port name. The mapped file must contain a valid
`router-instance-directory.v1` document before the Router starts.

Nacos mode requires one exact API origin ending in `/nacos`, namespace ID,
configuration group, authentication mode, response byte limit, request
timeout, strict directory key, and instance port name. Authentication is
explicitly `none` or `access_token`; the token must be absent in `none` mode.
The configured Nacos key must contain a valid
`router-nacos-instance-bindings.v1` document before the Router starts. Each
exact Release target maps to one Nacos service, group, and cluster.
Keys that are already legal Nacos dataIds are passed without translation. A
slash-separated key, or a key beginning with the reserved `nekiro.key.v1.`
prefix, maps to `nekiro.key.v1.` followed by the unpadded Base64URL encoding of
the complete key. This mapping is collision-free; the existing
`router.nacos-bindings` dataId remains unchanged.

Secrets, database URLs, Router signing keys, and service authentication remain
bootstrap configuration. They are not accepted in the instance directory.

## Rollout

1. Publish and authorize the exact Agent Release through the Control Plane.
2. Build a directory target from the published Release ID, Card digest, Agent
   identity, canonical endpoint, and canonical audience.
3. In File mode, publish the complete replacement directory document
   atomically. In Nacos mode, publish the complete binding document after the
   Agent service has registered its ephemeral instance.
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

Deleting the configured key, removing the File source root, losing
permissions, interrupting the File watcher, or making the configured Nacos
origin unavailable makes the directory unavailable. Readiness returns `503`,
and affected new Invocations fail with dependency semantics.
Configuration bytes, paths, source errors, and credentials are never returned
by readiness.

## Recovery

Restore the same configured source and publish one complete valid document.
If the File reader has entered a terminal interrupted state, restart the Router
after repairing the source. Nacos mode performs a fresh binding read and one
fresh Naming snapshot for each new Invocation; it has no local cache, retry,
alternate server, or watch recovery. Confirm readiness, then issue a new
Invocation. The Router never retries or reroutes an already failed Invocation.
