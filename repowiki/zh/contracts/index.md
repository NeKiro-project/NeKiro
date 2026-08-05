# 契约与兼容性

跨边界实现之前先定义契约。语言无关 schema、OpenAPI、A2A profile 及其 Go
映射是权威来源；服务和 SDK 类型都是消费者。

## 独立版本轴

以下值不能相互推导：

- Agent Card Schema version
- Agent version 与 Release identity
- Northbound HTTP API version
- Internal API version
- Event 与 result version
- A2A Profile Schema 与 protocol version
- Router credential version

## Phase 1 当前契约

| 边界 | 当前契约 | 含义 |
| --- | --- | --- |
| Catalog、Workspace、Installation | Northbound API v3 | 注册、发布、发现、Workspace 和 Installation |
| Invocation 与 Trace | Northbound API v4 | Workspace-scoped Invocation 和 metadata read |
| 精确 Card 解析 | Control Plane Internal v2 | Router 解析授权后的精确 Card 与 provenance |
| 嵌套版本解析 | Control Plane Internal v3 | Router 解析 enabled Installation pin |
| Router dispatch | Router Internal v4 | Control Plane dispatch authorized root Invocation |
| Router metadata read | Router metadata v3 | Workspace-scoped Invocation 与 Trace projection |
| Agent-facing A2A | A2A 0.3.0，profile schema 0.2 | 支持的 JSON-RPC 方法、上下文 header 和 streaming subset |
| Router-to-Agent authentication | Router credential v1 | 每个 managed HTTP request 的 Ed25519 binding |
| Invocation facts | Invocation Event 0.3 | 不含 Agent payload 的追加式 metadata 和 lineage |
| Stream result | Result Stream Event v2 | 有唯一不可变 terminal outcome 的有序临时事件 |

## 兼容规则

新增可选字段只有在省略时保持既有语义才兼容。删除或重命名字段、修改类型
或 requiredness、收紧校验、改变 status/media type/error 语义、移动 owner 或
重新解释历史 Ledger fact 都是 breaking change。

Breaking change 需要新契约版本、迁移说明和明确兼容窗口；或者由 pre-runtime
决策明确说明无需兼容窗口。历史文件作为 migration evidence 保持不变，当前
Runtime 不增加推测性的双读、双写或 fallback。

## 失败与数据安全

缺失、非法、未授权、禁用、未找到、超时、取消、依赖和协议失败保持不同的
status 与 error 语义。公开错误只包含固定安全消息和 correlation identifier。
Agent input、output、endpoint 细节、credential、原始依赖错误和 stack data
不能进入 Card、event、log 或 Ledger fact。

## Canonical source

- [Contract Compatibility Policy](../source-docs/contracts/compatibility.md)
- [Trusted Publication v1](../source-docs/contracts/trusted-publication-v1.md)
- [Phase 1 contract sources](../source-docs/architecture/phase-1-spec.md)
- [源文档](../source-docs/index.md)
