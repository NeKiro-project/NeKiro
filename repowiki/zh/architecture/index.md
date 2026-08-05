# 架构与所有权

NeKiro 是 runtime-agnostic 的 Agent operating platform，长期职责是建立
调用方与独立运行 Agent 之间的组织级信任边界。

## 产品边界

Core 管理 Agent 的外部生命周期：

1. 注册不可变、版本化的 Agent Card。
2. 发现符合条件的已发布能力。
3. 在 Workspace 中按精确版本安装并接受权限。
4. 通过 Gateway 和 A2A Router 调用。
5. 记录 metadata-only 生命周期事实和跨 Agent lineage。

Agent Runtime 仍在 Core 边界之外，可以用任意支持的语言或框架实现推理、
模型访问、工具、工作流、Memory、RAG、Session 和 Runtime telemetry。

## 部署单元

```text
Console -> Control Plane -> A2A Router -> Agent
                     \          |
                      \------ Ledger
```

| 单元 | 职责 | 所有权约束 |
| --- | --- | --- |
| Gateway | Northbound HTTP、调用方和 Workspace 上下文、响应形状 | 不持久化 Agent Card，不执行 A2A transport |
| Registry | Agent Card 版本与发布状态 | 不运行 Agent，不执行 Invocation |
| Discovery | 从已发布 Card 派生能力查询 | 不能成为第二事实源 |
| Workspace | Installation 与接受的权限 | 不负责部署 Agent |
| Invocation Dispatch | Invocation identity 与 dispatch 前授权 | 不成为 A2A protocol executor |
| A2A Router | transport、上下文传播、超时/取消、临时结果转发、事件 | 不拥有永久 Card，不直接访问 Registry/Workspace 存储 |
| Ledger | 追加式 invocation 事件和查询投影 | 不做路由或授权决策 |

## 信任边界

- Console 和外部应用只能访问 Gateway。
- Gateway 不直接访问 Agent；托管调用全部经过 A2A Router。
- Router 通过受控 Control Plane API 解析精确 Card 和 Release。
- 嵌套 Agent-to-Agent 调用必须回到 Router，并保留 root_task_id、parent_invocation_id 和 trace_id。
- Ledger 只保存 metadata 和 lineage，不保存 Agent payload、credential 或 key。
- 进程间数据使用 contracts/ 下的版本化工件，不共享内部实现类型。

## Canonical source

- [第一阶段架构规范](../source-docs/architecture/phase-1-spec.md)
- [NeKiro 平台方向](../source-docs/architecture/platform-direction.md)
- [Core 仓库边界 ADR](../source-docs/decisions/0009-core-repository-boundary.md)
- [仓库地图](../repositories.md)
