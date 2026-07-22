# Data model: Trusted Agent Publication

## Provider

| Field | Meaning |
| --- | --- |
| provider_id | Stable platform identifier |
| owner_identity | Authenticated owner binding |
| verification_status | `unverified`, `verified`, `suspended` |
| verification_method | `http_well_known` for Slice A |
| verified_at | Timestamp of successful provider verification |
| created_at / updated_at | Registry lifecycle timestamps |

An Agent identity may claim at most one provider identity. The claim is a
Registry ownership fact and is not inferred from the Card's display owner.
The first provider binding establishes the claim; another provider receives a
conflict and cannot replace it.

## Endpoint Binding

| Field | Meaning |
| --- | --- |
| binding_id | Stable binding identity |
| provider_id | Owning Provider |
| agent_id | Bound Agent identity |
| agent_card_version | Exact Card version resolved by the binding |
| endpoint_origin | Canonical scheme/host/port |
| endpoint_path | Exact A2A path |
| verification_status | `pending`, `verified`, `failed`, `revoked` |
| verification_method | Proof method identifier |
| verification_evidence_digest | Non-secret evidence digest |
| verification_failure_code | Typed category, never raw dependency detail |
| verified_at / revoked_at | State timestamps |

## Verification Challenge

| Field | Meaning |
| --- | --- |
| challenge_id | Public lookup identity |
| binding_id | Target endpoint binding |
| proof_digest | Hash of the one-time proof |
| expires_at | Explicit expiry boundary |
| used_at | Single-use marker |
| created_at | Creation timestamp |

The raw proof is transient and is never a persistent field. All identifiers
are validated at the Gateway/Registry boundary. No model has an endpoint
credential, API key, or JWT signing material.
