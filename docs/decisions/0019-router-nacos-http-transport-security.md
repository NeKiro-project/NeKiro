# ADR 0019: Router Nacos HTTP Transport Security

- Status: Accepted for issue #110
- Date: 2026-08-10
- Decision owner: Platform Architecture
- Extends: ADR 0014, ADR 0015, and ADR 0018

## Context

The Router uses two Nacos transports on one exact-Release discovery path. It
reads the binding document and initial Naming snapshot through the Nacos HTTP
API, then optionally follows Naming lifecycle changes through gRPC. ADR 0018
made the gRPC observation trust explicit, but the HTTP client still inherited
system roots for HTTPS and did not bind TLS fields to the configured URL
scheme.

That leaves production discovery with two different trust policies. A secure
gRPC watch cannot compensate for a binding or initial snapshot retrieved from
an ambiently trusted or plaintext HTTP endpoint.

## Decision

1. The exact scheme in `NEKIRO_ROUTER_NACOS_API_ORIGIN` is the explicit HTTP
   transport selector. `http` selects controlled plaintext; `https` selects
   authenticated TLS. Redirects cannot change that selection.
2. An `http` origin requires every Nacos HTTP TLS field to be absent. An
   `https` origin requires a clean absolute CA bundle path and explicit
   canonical lowercase DNS name or canonical IP address.
3. Optional mTLS is selected only by supplying both clean absolute client
   certificate and key paths. Partial pairs are invalid. No client identity is
   inferred from the host or environment.
4. The Router constructs one HTTP client and injects it into both the Nacos
   Config Center reader and the initial Naming snapshot directory. Those
   provider packages do not read deployment files or own trust policy.
5. The HTTP transport disables environment proxies, redirects, and persistent
   connection reuse. HTTPS uses only the configured private CA pool, TLS 1.2
   or newer, normal server-name verification, and the optional configured
   client certificate. System roots and `InsecureSkipVerify` are forbidden.
6. HTTP and gRPC share the same bootstrap material loader: regular files only,
   non-empty, at most 1 MiB, strict CA PEM, and redacted failure classes.
7. A configuration or TLS failure stops Router startup or the exact request.
   It never downgrades HTTPS to HTTP, changes authority, uses a proxy, retries,
   redirects, reconnects, or serves cached topology.
8. Paths, PEM/key bytes, file contents, parser details, credentials, and Nacos
   provider internals do not enter errors, logs, topology status, contracts,
   Ledger, or runtime metadata.

## Consequences

- Binding retrieval, the initial snapshot, and continuous observation can use
  one private PKI trust boundary end to end.
- Existing explicitly configured `http` deployments remain valid without TLS
  fields. Existing `https` deployments must provide explicit private trust
  material and restart Router to rotate it.
- Core still does not own Nacos server deployment or certificate issuance.

## Fallback Delta

Fallback delta: removed 0, retained 0, added 0, net +0.

Added fallback evidence: none.
