# NeKiro RepoWiki

这是 NeKiro 平台的统一知识入口。它把已接受的架构、契约、决策和运维
指南组织成一套可导航的文档，同时保持每条事实由 canonical owner 维护。

平台闭环：

```text
Register -> Discover -> Install -> Invoke -> Record
```

## 从这里开始

- [架构与所有权](architecture/index.md)：Core 管什么、信任边界在哪里、服务如何协作。
- [平台生命周期](architecture/lifecycle.md)：五步闭环及其失败语义。
- [契约与兼容性](contracts/index.md)：当前版本、迁移规则和数据安全约束。
- [运维](operations/index.md)：本地验证和受信发布流程。
- [决策记录](decisions/index.md)：解释平台边界为何如此设计的 ADR。
- [仓库地图](repositories.md)：所有 NeKiro 仓库的统一入口。
- [卫星仓库文档](satellites/index.md)：各卫星仓库的只读文档快照。
- [源文档](source-docs/index.md)：从 docs/ 自动生成的规范页面。

## 一图理解平台

NeKiro 从 Agent 外部管理 Agent。模型、Prompt、Tool、Planner、Workflow、
Memory、RAG、Session 和 Runtime telemetry 属于 Agent Runtime；发布、发现、
Workspace 授权、精确版本解析、受控路由和 metadata-only lineage 属于 Core。

```text
Caller / Console
       |
       v
Gateway / Control Plane -----> Catalog + Workspace
       |
       v
A2A Router ------------------> Agent endpoint
       |
       v
Invocation Ledger (metadata and lineage only)
```

## 所有权规则

中央 Wiki 是统一阅读入口，不是卫星仓库的第二个可写事实源。Core 文档从
docs/ 构建；Console、SDK、Samples、Stack 和 A2A Transport 的源码与事实仍由
各自仓库维护，并从[仓库地图](repositories.md)进入。

## Wiki 的边界

- 不定义 Agent 内部执行行为。
- 不替代 contracts/ 下的版本化契约文件。
- 不复制卫星仓源码，也不推测卫星仓的发布状态。
- 不把依赖失败伪装为空结果、成功响应或备用 endpoint。
