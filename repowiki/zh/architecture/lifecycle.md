# 平台生命周期

五个动词描述一条受治理的调用链。每一步都产生由一个领域拥有的事实，
并只向下一个边界传递所需的版本化信息。

| 步骤 | Owner 与事实 | 进入下一步前必须满足 |
| --- | --- | --- |
| Register | Registry 保存 draft 或 published Agent Card 版本及发布事实 | Card 有效；发布后不可变，并绑定预期 Release provenance |
| Discover | Discovery 返回符合条件的已发布版本派生视图 | 查询结果不能成为独立 Card 权威 |
| Install | Workspace 保存接受的权限和精确 Agent 版本 | 授权明确、按 Workspace 隔离，可禁用但不改写历史 |
| Invoke | Gateway 授权；Router dispatch；嵌套调用回到 Router | root/parent/trace lineage 和精确 Card/Release provenance 保持完整 |
| Record | Ledger 追加生命周期 metadata 并派生读取投影 | 不保存 payload、result、credential 或 key |

## 调用路径

```text
Northbound request
  -> Gateway 创建 trace 并授权 Workspace Installation
  -> Control Plane 解析精确的已安装 Agent 版本
  -> Router 创建受控 Agent hop 并传播 lineage
  -> Agent 可以通过 Router 创建 child Invocation
  -> Router 返回一次性的 JSON 或 SSE result
  -> Ledger 保留 metadata-only 事实供查询
```

当前 Invocation surface 按 Workspace 隔离。断开连接后可以查询 metadata，但
Phase 1 不持久化、重放、轮询或恢复 result 内容。非成功 terminal event 之前
收到的 chunks 不代表 Invocation 成功。

## 失败语义

缺失输入、非法输入、未找到、禁止、禁用、依赖失败、超时、取消和协议失败
是不同状态，不能折叠为 null、空集合、普通成功或静默重试。第一个 terminal
Invocation outcome 不可变；terminal stream event 之前的 EOF 表示传输中断。

## 验收证明

产品证明必须跨 Runtime：两个独立实现的 Agent 需要完成注册、安装、经 Router
调用，并出现在一条 parent-child trace 中。验收套件由
[NeKiro-Stack](https://github.com/NeKiro-project/NeKiro-Stack) 所有，并固定精确
组件版本。

## Canonical source

- [第一阶段架构规范](../source-docs/architecture/phase-1-spec.md)
- [Invocation result 与内部 API ADR](../source-docs/decisions/0002-invocation-result-transport-and-internal-api-direction.md)
- [Invocation runtime trust 与 failure policy ADR](../source-docs/decisions/0006-invocation-runtime-trust-and-failure-policy.md)
- [Router signed credential ADR](../source-docs/decisions/0007-router-agent-signed-credential.md)
