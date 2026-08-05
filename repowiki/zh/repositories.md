# 仓库地图

NeKiro 由一组独立发布的仓库组成。本页面提供统一入口，但不把卫星仓源码
复制进 Core。

| 仓库 | Canonical responsibility | Repository | Wiki |
| --- | --- | --- | --- |
| **NeKiro Core** | Control Plane、A2A Router、contracts、service-owned migrations、Core tests、architecture 和 Core usage | [NeKiro-project/NeKiro](https://github.com/NeKiro-project/NeKiro) | [本 RepoWiki](https://nekiro-project.github.io/NeKiro/) |
| **NeKiro Console** | Production Console 与 browser behavior | [NeKiro-project/NeKiro-Console](https://github.com/NeKiro-project/NeKiro-Console) | [中央 RepoWiki](https://nekiro-project.github.io/NeKiro/zh/satellites/console/) |
| **nekiro-sdk-go** | Public Go Agent 和 application SDK | [NeKiro-project/nekiro-sdk-go](https://github.com/NeKiro-project/nekiro-sdk-go) | [中央 RepoWiki](https://nekiro-project.github.io/NeKiro/zh/satellites/sdk-go/) |
| **NeKiro Samples** | 独立 sample Agent Runtime、Card 和 sample-specific tests | [NeKiro-project/NeKiro-Samples](https://github.com/NeKiro-project/NeKiro-Samples) | [中央 RepoWiki](https://nekiro-project.github.io/NeKiro/zh/satellites/samples/) |
| **NeKiro Stack** | Multi-component assembly、immutable release pins 和 product acceptance | [NeKiro-project/NeKiro-Stack](https://github.com/NeKiro-project/NeKiro-Stack) | [中央 RepoWiki](https://nekiro-project.github.io/NeKiro/zh/satellites/stack/) |
| **nekiro-a2a-transport-go** | 可复用的 A2A HTTP、JSON-RPC 和 SSE transport mechanics | [NeKiro-project/nekiro-a2a-transport-go](https://github.com/NeKiro-project/nekiro-a2a-transport-go) | [中央 RepoWiki](https://nekiro-project.github.io/NeKiro/zh/satellites/a2a-transport-go/) |

## 发布与验证顺序

1. Core 发布版本化契约和经过验证的 Core revision。
2. SDK、Console、Samples 和 transport 仓库针对该 identity 验证。
3. Stack 固定 immutable compatible revisions 并运行产品验收。
4. 对精确 Core commit 保留可见的 compatibility evidence。

中央 Wiki 记录这个顺序，但不成为备用发布 authority。详见
[ADR 0009](source-docs/decisions/0009-core-repository-boundary.md)。
