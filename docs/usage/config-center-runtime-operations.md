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

Nacos Naming observation is optional and disabled by default. Enabling it
requires all of `NEKIRO_ROUTER_NACOS_GRPC_TARGET`,
`NEKIRO_ROUTER_NACOS_GRPC_CLIENT_IP`,
`NEKIRO_ROUTER_NACOS_GRPC_REQUEST_TIMEOUT_MS`,
`NEKIRO_ROUTER_NACOS_PENDING_CHANGES`,
`NEKIRO_ROUTER_NACOS_MAX_OBSERVATIONS`, and the explicit
`NEKIRO_ROUTER_NACOS_GRPC_TRANSPORT_SECURITY` together with
`NEKIRO_ROUTER_NACOS_OBSERVE_ENABLED=true`. Transport security must be exactly
one of:

- `insecure`: explicit plaintext for local or controlled deployments. All TLS
  fields must be absent.
- `tls`: requires `NEKIRO_ROUTER_NACOS_GRPC_TLS_CA_FILE` as a clean absolute
  path and `NEKIRO_ROUTER_NACOS_GRPC_TLS_SERVER_NAME` as a canonical lowercase
  DNS name or canonical IP address.
- `mtls`: requires the TLS fields plus clean absolute
  `NEKIRO_ROUTER_NACOS_GRPC_TLS_CLIENT_CERT_FILE` and
  `NEKIRO_ROUTER_NACOS_GRPC_TLS_CLIENT_KEY_FILE` paths.

TLS files must be regular, non-empty, at most 1 MiB, and valid for their role.
The CA bundle supplies the complete private trust pool; system roots are not
used, hostname verification stays enabled, and client certificate fields are
accepted only in mTLS mode. TLS material is read only during Router bootstrap,
so rotation requires a Router restart. Partial or mode-incompatible gRPC/TLS
configuration is rejected. Errors never include a configured path, PEM data,
private-key bytes, or file contents. The executor opens one connection and
never falls back to plaintext, retries, reconnects,
polls, switches authority, or serves cached topology after failure. The first
Invocation for an exact Release atomically establishes its observation and
uses the initial snapshot. Later Invocations read the latest immutable watched
snapshot without issuing a Naming list request. One observation is retained
per exact Release until Router shutdown, bounded by the configured maximum.

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

For an observe-enabled Router, use the configured Router service Bearer
credential to read `GET /internal/v1/instance-topology/status`. The response is
safe evidence of topology already consumed by Router; it is not a refresh
operation. Record the exact target's `localRevision`, `state`, and `observedAt`
before a lifecycle change. After deregistration, wait for the same Agent,
version, and Release to report `state=empty` with a greater local revision
before issuing the fail-closed acceptance Invocation. An `AGENT_UNAVAILABLE`
response without that status transition does not prove watch consumption.

The status route is absent in direct and snapshot-only modes. Never treat a
404 there as an empty observed topology. Revisions are local to the current
Router observation and cannot be compared across restarts or targets.
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
after repairing the source. Nacos observation mode does not recover a terminal
watch. Restart the Router after repairing the same configured source to
establish fresh observations. It has no stale snapshot fallback, retry,
polling, or alternate server. The Router never retries or reroutes an already
failed Invocation.
