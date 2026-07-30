# Public Agent Share v1 Contract Mapping

## Canonical browser URL

```text
GET {NEKIRO_PUBLIC_AGENT_ORIGIN}/a/{publicAgentId}
```

This is a Console route. It is not an Agent endpoint and performs no mutation.

## Registration response extension

`POST /v3/agents` keeps its existing request and status behavior. Its `CatalogEntry` response gains two optional additive fields:

```json
{
  "publicAgentId": "agt_0123456789abcdef0123456789abcdef",
  "publicUrl": "https://agents.nekiro.dev/a/agt_0123456789abcdef0123456789abcdef"
}
```

The implementation returns both fields together after successful registration. Existing clients may ignore them; partial presence is invalid for the updated Console mapping.

## Anonymous public resolution

```text
GET /v4/public/agents/{publicAgentId}
Authorization: absent
```

Success is Public Agent Share v1:

```json
{
  "schemaVersion": "1",
  "publicAgentId": "agt_0123456789abcdef0123456789abcdef",
  "publicUrl": "https://agents.nekiro.dev/a/agt_0123456789abcdef0123456789abcdef",
  "registeredAt": "2026-07-30T00:00:00Z",
  "availability": "installable",
  "releases": [{
    "releaseId": "rel_...",
    "agentId": "runtime-a",
    "name": "Runtime A",
    "description": "Sample Agent",
    "owner": {"id": "provider-a", "displayName": "Provider A"},
    "agentCardVersion": "1.0.0",
    "cardDigest": "64-lowercase-hex",
    "publishedAt": "2026-07-30T00:01:00Z",
    "authenticationType": "http_bearer",
    "skills": [],
    "permissions": [],
    "limits": {"timeoutMs": 30000, "maxInputBytes": 1048576, "maxOutputBytes": 1048576, "streaming": true}
  }]
}
```

The identity envelope never contains Card-derived metadata. An identity with no eligible Release returns `availability=not_installable` and `releases=[]`. The response never contains `protocol`, `endpoint`, `endpointBindingId`, `endpointOrigin`, `endpointPath`, `verificationEvidenceDigest`, challenge proof, bearer material, Workspace ID, or Ledger facts.

## Resolution failures

| HTTP | Code | Meaning |
| --- | --- | --- |
| 400 | `VALIDATION_ERROR` | malformed public ID or unsupported path syntax |
| 404 | `NOT_FOUND` | syntactically valid but unknown public identity |
| 503 | `DEPENDENCY_ERROR` | Catalog read or stored-contract dependency failed |
| 500 | `INTERNAL_ERROR` | unclassified platform failure |

All errors include the Gateway `traceId`. No error changes to an empty successful view.

## Exact Installation handoff

After B selects one item and explicitly accepts permissions, Console calls the existing authenticated boundary:

```text
POST /v3/workspaces/{workspaceId}/installations
Authorization: Bearer <Workspace-owner credential>
```

```json
{
  "agentId": "runtime-a",
  "versionConstraint": "1.0.0",
  "acceptedPermissions": ["text.read"]
}
```

Console accepts success only when:

```text
installedVersion == selected agentCardVersion
installedReleaseId == selected releaseId
acceptedPermissions == explicitly accepted permissions
status == enabled
```

The server revalidates Release state and Workspace authority at the write boundary. No client read acts as authorization.
