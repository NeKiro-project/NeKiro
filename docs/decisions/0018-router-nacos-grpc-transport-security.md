# ADR 0018: Router Nacos gRPC Transport Security

- Status: Accepted for issue #108
- Date: 2026-08-10
- Decision owner: Platform Architecture
- Extends: ADR 0015

## Context

The Router's Nacos Naming observer uses a long-lived gRPC connection. The
initial slice required the operator to select explicit plaintext `insecure`,
which is suitable only for local or otherwise controlled networks. Production
deployments need authenticated encryption without moving deployment secrets
into Config Center or making the Nacos registry package own Router bootstrap
configuration.

Ambient system trust, hostname-verification bypass, and TLS downgrade would
make the selected Nacos authority depend on host state or turn a configuration
error into plaintext traffic. Those behaviors conflict with the Router's
fail-closed discovery boundary.

## Decision

1. `NEKIRO_ROUTER_NACOS_GRPC_TRANSPORT_SECURITY` accepts exactly `insecure`,
   `tls`, or `mtls`. Plaintext remains an explicit local/controlled deployment
   choice; it is never selected as a fallback.
2. `tls` and `mtls` require a clean absolute CA bundle path and an explicit
   canonical lowercase DNS name or canonical IP address for certificate
   verification. mTLS additionally requires clean absolute client certificate
   and private-key paths. Mode-incompatible or partial fields are rejected.
3. Router bootstrap reads each TLS file as a regular file with a 1 MiB limit.
   Empty, missing, unreadable, non-regular, oversized, malformed, non-CA, or
   mismatched material stops startup before serving.
4. The Router builds a private root pool containing only certificates in the
   configured bundle. System roots are not copied or consulted. Every PEM
   block must be an unadorned CA certificate authorized for certificate
   signing.
5. TLS uses Go hostname verification with TLS 1.2 as the minimum version.
   `InsecureSkipVerify` is forbidden. mTLS presents only the explicitly loaded
   client certificate.
6. `apps/a2a-router/internal/config` owns field compatibility and syntactic
   validation. `apps/a2a-router/cmd/a2a-router` owns bounded file reads and
   credential construction. `registry/nacos` continues to receive constructed
   `credentials.TransportCredentials` and does not read files or service
   configuration.
7. Errors identify only the material role and failure class. Paths, PEM bytes,
   key bytes, file contents, and parser details do not enter errors, logs,
   topology status, Ledger, contracts, or runtime metadata.
8. A TLS handshake or established watch failure does not trigger plaintext,
   another CA, another server name, another authority, reconnect, polling, or
   cached topology.

## Consequences

- Production deployments can authenticate Nacos with private PKI and can
  require a Router client identity.
- Certificate rotation is an explicit deployment restart because TLS material
  is bootstrap configuration, not dynamically watched configuration.
- A host's system root configuration cannot silently expand Router trust.

## Fallback Delta

Fallback delta: removed 0, retained 0, added 0, net +0.

Added fallback evidence: none.
