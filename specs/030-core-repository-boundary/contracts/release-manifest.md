# NeKiro Stack Release Manifest Contract

The Stack repository records one exact product assembly in `components.json`.
The initial cutover may use commit SHAs before all OCI publishing exists; a
published product release must additionally record exact tags and image digests.

Required logical shape:

```json
{
  "schemaVersion": "1",
  "contractIdentity": "<core contract release identity>",
  "components": {
    "core": { "repository": "NeKiro-project/NeKiro", "commitSha": "<40 hex>" },
    "console": { "repository": "NeKiro-project/NeKiro-Console", "commitSha": "<40 hex>" },
    "sdkGo": { "repository": "NeKiro-project/nekiro-sdk-go", "commitSha": "<40 hex>" },
    "samples": { "repository": "NeKiro-project/NeKiro-Samples", "commitSha": "<40 hex>" },
    "transportGo": { "repository": "NeKiro-project/nekiro-a2a-transport-go", "tag": "v0.1.1", "commitSha": "<40 hex>" }
  }
}
```

Validation rules:

1. `schemaVersion` is exactly `1` until a reviewed breaking manifest change.
2. Each repository name is an approved canonical owner.
3. Every `commitSha` is a full 40-character lowercase hexadecimal Git object.
4. A tag, when present, resolves to the declared commit and is not mutable.
5. An image, when present, uses an exact `sha256:` OCI digest.
6. Branch names, `latest`, local filesystem references, URL archives without a
   checksum, and Go `replace` directives are invalid production inputs.
7. CI checks out or pulls exactly these identities before acceptance.
8. A missing or mismatched component terminates validation with failure before
   services start.
