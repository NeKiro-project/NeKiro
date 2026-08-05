# Satellite documentation

This is the unified reading surface for documentation owned by the NeKiro
satellite repositories. The pages below are immutable-revision snapshots; the
satellite repository remains the canonical owner and the central RepoWiki is
not a second writable source.

Refresh the snapshots deliberately after reviewing an upstream change:

```text
python scripts/sync_repowiki_satellites.py
```

## Integrated repositories

| Repository | Central section | Snapshot source | Wiki state at snapshot time |
| --- | --- | --- | --- |
| **NeKiro Console** | [Console documentation](console/index.md) | 26 committed `.qoder/repowiki` Markdown pages plus `README.md` at `5e577d8` | GitHub Wiki disabled; committed RepoWiki export is the source |
| **nekiro-sdk-go** | [Go SDK documentation](sdk-go/index.md) | `README.md`, `agent/README.md`, and `client/README.md` at `0bc1bd0` | GitHub Wiki enabled but no Wiki git pages were available |
| **NeKiro Samples** | [Samples documentation](samples/index.md) | `README.md`, `runtime-a/README.md`, and `runtime-b/README.md` at `89bf743` | GitHub Wiki enabled but no Wiki git pages were available |
| **NeKiro Stack** | [Stack documentation](stack/index.md) | `README.md` at `20a6f36` | GitHub Wiki enabled but no Wiki git pages were available |
| **nekiro-a2a-transport-go** | [A2A transport documentation](a2a-transport-go/index.md) | `README.md` at `71fb8ee` | GitHub Wiki enabled but no Wiki git pages were available |

“No Wiki git pages” means the public `<repository>.wiki.git` endpoint returned
no repository. It does not authorize Core to invent a replacement owner. When
an upstream Wiki is initialized, add its reviewed pages and revision to the
sync manifest; until then, the repository-owned Markdown above is the
available canonical documentation.

## Ownership and change policy

- Edit Console, SDK, Samples, Stack, and transport facts in their owning repositories.
- Use the central RepoWiki for cross-repository navigation and read-only snapshots.
- Every snapshot page carries its source repository and full commit revision.
- Do not copy satellite source code, credentials, generated build output, or mutable release state into Core.
