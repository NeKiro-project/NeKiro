# 决策记录

ADR 解释平台边界背后的长期决策。本页面提供统一索引；源文件仍是 canonical
source，并在[源文档](../source-docs/index.md)中展示。

| ADR | 决策 | 作用 |
| --- | --- | --- |
| 0001 | [Go Backend And Router](../source-docs/decisions/0001-go-backend-stack.md) | 固定 Core service 和 Router 使用 Go。 |
| 0002 | [Invocation Result Transport and Internal API Direction](../source-docs/decisions/0002-invocation-result-transport-and-internal-api-direction.md) | 保持 result delivery 临时化，内部 API 方向明确。 |
| 0003 | [Runtime-Agnostic Platform Boundary](../source-docs/decisions/0003-runtime-agnostic-platform-boundary.md) | 将 Agent Runtime 行为留在 Core 外部。 |
| 0004a | [Catalog Persistence and Strong Discovery Consistency](../source-docs/decisions/0004-catalog-persistence-and-consistency.md) | 明确 Registry ownership 和 discovery consistency。 |
| 0004b | [Trusted Agent Publication and Endpoint Ownership](../source-docs/decisions/0004-trusted-agent-publication.md) | 建立 trusted Release provenance 和 endpoint ownership。 |
| 0005 | [Minimal Workspace and Installation Boundary](../source-docs/decisions/0005-minimal-workspace-installation-boundary.md) | 固定最小化的授权与精确版本事实。 |
| 0006 | [Invocation Runtime Trust and Failure Policy](../source-docs/decisions/0006-invocation-runtime-trust-and-failure-policy.md) | 定义 deadline、cancel、failure、size 和 Ledger 语义。 |
| 0007 | [Router-to-Agent Signed Invocation Credential](../source-docs/decisions/0007-router-agent-signed-credential.md) | 将 managed request 绑定到 provenance，且不持久化 secret。 |
| 0008 | [Standalone A2A Transport Module](../source-docs/decisions/0008-standalone-a2a-transport-module.md) | 将可复用 wire mechanics 与 Core policy 分离。 |
| 0009 | [Core Repository and Satellite Ownership](../source-docs/decisions/0009-core-repository-boundary.md) | 建立多仓所有权和发布顺序。 |

当前 source set 中存在两个描述性文件都使用 0004 编号。Wiki 使用 0004a 和
0004b 仅用于导航区分，canonical 文件及其历史保持不变。
