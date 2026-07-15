# Invocation Runtime Semantic Rules v1

| Rule | Requirement |
| --- | --- |
| `IRT-ERR-001` | Pre-correlation error has exactly code/message/Trace; correlated error additionally requires exact Invocation/root Task. |
| `IRT-ERR-002` | Every Platform Error v4 code has one fixed public message. |
| `IRT-NEST-001` | Child parent/root/Trace/Workspace and caller Agent match the trusted running parent; child ID differs. |
| `IRT-LIFE-001` | Lifecycle begins `created/pending` at sequence zero and preserves immutable context. |
| `IRT-LIFE-002` | Only declared pending/routing/running transitions are legal; success is running-only. |
| `IRT-LIFE-003` | Event and chunk indexes are gapless and no event follows the first terminal. |
| `IRT-MEDIA-001` | Non-stream accepts exactly `application/json`, `application/*`, or `*/*`; stream accepts exactly `text/event-stream`. |

The conformance corpus is authoritative executable evidence for these rules.
