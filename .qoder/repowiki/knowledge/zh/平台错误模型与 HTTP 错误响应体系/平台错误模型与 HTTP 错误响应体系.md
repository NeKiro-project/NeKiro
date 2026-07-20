---
kind: error_handling
name: 平台错误模型与 HTTP 错误响应体系
category: error_handling
scope:
    - '**'
source_files:
    - contracts/contracts.go
    - contracts/schemas/platform-error.v4.schema.json
    - apps/control-plane/internal/gateway/errors.go
    - apps/control-plane/internal/gateway/invocation_handler.go
    - apps/control-plane/internal/gateway/workspace_handler.go
    - apps/control-plane/internal/gateway/catalog_handler.go
    - apps/control-plane/internal/invocation/service.go
---

## 1. 系统/方法概述
仓库采用「契约驱动」的错误处理方案：所有跨进程错误通过 `contracts` 包集中定义的 `PlatformErrorCode` 枚举与多版本 `PlatformError*Vn` 结构体表达，HTTP 网关层负责将内部错误码映射为 HTTP 状态码并序列化为统一的 JSON 错误体。错误分为两类——**预关联（pre-correlation）** 与 **已关联（correlated）**，后者携带 `invocationId`、`rootTaskId` 以便在调用链中追踪。

- 错误码定义：`contracts.PlatformErrorCode` + 常量（`VALIDATION_ERROR`、`UNAUTHENTICATED`、`FORBIDDEN`、`NOT_FOUND`、`CONFLICT`、`AGENT_NOT_INSTALLED`、`INSTALLATION_DISABLED`、`AGENT_DISABLED`、`CAPABILITY_NOT_ALLOWED`、`ROUTE_NOT_FOUND`、`A2A_PROTOCOL_ERROR`、`AGENT_UNAVAILABLE`、`AGENT_EXECUTION_FAILED`、`DEPENDENCY_ERROR`、`TIMEOUT`、`CANCELED`、`INTERNAL_ERROR`）
- 错误体版本：v2/v3/v4 三种结构体，分别对应不同 API 面；`contracts.NewPlatformError*` 工厂函数按版本构造，并通过内嵌 JSON Schema 校验消息文本与字段组合。
- HTTP 映射：各 Handler 中的 `platformErrorStatus` / `workspaceErrorStatus` / `invocationErrorStatus` 将错误码映射到具体 HTTP 状态码，`writePlatformError` / `writeCorrelatedError` 统一写入 `x-nek-trace-id` 头与 JSON body。
- 业务层错误传播：领域服务返回带 `Code contracts.PlatformErrorCode` 的自定义错误类型，Handler 再将其转换为平台错误响应。

## 2. 关键文件与包
- `contracts/contracts.go` — 错误码常量、`PlatformError = PlatformErrorV2` 别名、`NewPlatformError` 入口
- `contracts/schemas/platform-error.v4.schema.json` — v4 错误体的 JSON Schema（含 code→message 约束）
- `apps/control-plane/internal/gateway/errors.go` — 通用 `platformErrorStatus` / `writePlatformError` 工具
- `apps/control-plane/internal/gateway/invocation_handler.go` — Invocation 面的 `writePreError` / `writeCorrelatedError` / `invocationErrorStatus`，使用 v4 错误体
- `apps/control-plane/internal/gateway/workspace_handler.go` — Workspace 面的 `writeWorkspaceError` / `workspaceErrorStatus`，使用 v3 错误体
- `apps/control-plane/internal/gateway/catalog_handler.go` — Catalog 面的 `catalogErrorCode` 映射与失败日志
- `apps/control-plane/internal/invocation/service.go` — 领域层返回带 `Code` 字段的错误结构
- `apps/control-plane/cmd/control-plane/main.go` — 启动期错误使用裸 `errors.New` 与 `errors.Join`，不进入平台错误模型

## 3. 架构与约定
- **分层职责**：领域层只返回语义化错误码（`PlatformErrorCode`），HTTP 层负责状态码映射与序列化，避免业务逻辑耦合 HTTP。
- **版本隔离**：不同 API 面使用不同版本的错误体（Invocation→v4，Workspace→v3，Catalog→v2），通过独立的 `NewPlatformErrorVn` 工厂与 Schema 保证向后兼容。
- **可观测性**：所有平台错误响应均附带 `x-nek-trace-id` 请求头，便于链路追踪；测试中通过 `requireAcceptanceError` 断言响应体结构与错误码。
- **无 panic/recover**：当前代码未发现 `panic`/`recover` 的使用，错误均以返回值形式向上冒泡。
- **中间件缺失**：尚未发现全局 HTTP 中间件统一捕获未处理错误，每个 Handler 显式调用 `writePlatformError` 系列函数。

## 4. 开发者应遵循的规则
1. 在领域/服务层仅返回包含 `Code contracts.PlatformErrorCode` 的结构或标准 error，不要直接构造 HTTP 响应。
2. 在 Gateway Handler 中使用对应的 `writePreError` / `writeCorrelatedError` / `writePlatformError` 输出错误，禁止手写 JSON 响应体。
3. 新增错误码时同步更新 `contracts/contracts.go` 常量、`platform-error.*.schema.json` 的 enum 列表及 `messageRules` 约束，并确保 `NewPlatformErrorVn` 能构造合法实例。
4. 对依赖层错误使用 `errors.Is` / `errors.As` 判断后再映射为合适的 `PlatformErrorCode`，不要透传底层错误信息给客户端。
5. 启动/迁移等基础设施路径可使用裸 `errors.New` 与 `errors.Join`，但不应出现在对外 API 路径中。