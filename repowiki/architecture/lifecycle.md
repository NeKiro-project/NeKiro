# Platform lifecycle

The five verbs describe one governed flow. Each step produces a fact owned by
one domain and passes only the versioned information required by the next
boundary.

| Step | Owner and fact | What must be true before the next step |
| --- | --- | --- |
| Register | Registry stores a draft or published Agent Card version and its publication facts | The Card is valid, immutable when published, and tied to the intended release provenance |
| Discover | Discovery returns a derived view of eligible published versions | Results never become an independent Card authority |
| Install | Workspace stores accepted permissions and an exact resolved Agent version | Authorization is explicit, Workspace-scoped, and can be disabled without rewriting history |
| Invoke | Gateway authorizes; Router dispatches; nested calls return through Router | Root/parent/trace lineage and exact Card/Release provenance are preserved |
| Record | Ledger appends lifecycle metadata and derives read projections | Payloads, results, credentials, and keys are excluded from stored facts |

## Invocation path

```text
Northbound request
  -> Gateway creates trace and authorizes Workspace installation
  -> Control Plane resolves the exact installed Agent version
  -> Router creates the managed Agent hop and propagates lineage
  -> Agent may create child Invocations through the Router
  -> Router returns one transient JSON or SSE result
  -> Ledger retains metadata-only events for inspection
```

The active invocation surface is Workspace-scoped. A caller may inspect
metadata after a disconnect, but Phase 1 does not persist, replay, poll, or
recover result content. Receiving chunks before a non-success terminal event
does not make the Invocation successful.

## Failure semantics

Missing input, invalid input, not found, forbidden, disabled, dependency
failure, timeout, cancellation, and protocol failure are different states.
They must not collapse into `null`, an empty collection, a normal success, or a
silent retry. The first terminal Invocation outcome is immutable, and an EOF
before a terminal stream event means interrupted delivery.

## Acceptance proof

The product proof is cross-runtime: two independently implemented Agents must
be registered, installed, invoked through the Router, and visible in one
parent-child trace. The acceptance suite is owned by
[NeKiro-Stack](https://github.com/NeKiro-project/NeKiro-Stack), which pins the
exact component revisions.

## Canonical source

- [Phase 1 Architecture Specification](../source-docs/architecture/phase-1-spec.md)
- [Invocation result and internal API ADR](../source-docs/decisions/0002-invocation-result-transport-and-internal-api-direction.md)
- [Invocation runtime trust and failure policy ADR](../source-docs/decisions/0006-invocation-runtime-trust-and-failure-policy.md)
- [Router signed credential ADR](../source-docs/decisions/0007-router-agent-signed-credential.md)
