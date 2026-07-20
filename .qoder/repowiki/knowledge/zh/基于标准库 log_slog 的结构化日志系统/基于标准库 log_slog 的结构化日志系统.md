---
kind: logging_system
name: 基于标准库 log/slog 的结构化日志系统
category: logging_system
scope:
    - '**'
source_files:
    - apps/control-plane/cmd/control-plane/main.go
    - apps/control-plane/internal/gateway/catalog_handler.go
    - apps/control-plane/internal/gateway/workspace_handler.go
    - apps/control-plane/internal/gateway/invocation_handler.go
---

本仓库的日志系统基于 Go 标准库 log/slog，采用结构化 JSON 输出模式，贯穿 Control Plane 应用的所有 HTTP 网关层。

## 架构与初始化
- 入口初始化：apps/control-plane/cmd/control-plane/main.go 在进程启动时创建全局 logger：slog.New(slog.NewJSONHandler(os.Stderr, nil))，通过 JSON Handler 将结构化日志输出到 stderr，便于容器编排环境收集。
- 依赖注入：logger 以 *slog.Logger 形式作为构造参数注入到各 Gateway Handler（Catalog、Workspace、Invocation），遵循显式依赖传递而非包级单例的模式。
- 测试隔离：所有测试使用 slog.NewTextHandler(io.Discard, nil) 丢弃日志输出，避免污染测试输出。

## 结构化字段约定
Gateway 层统一使用 Context-aware API（InfoContext/WarnContext/ErrorContext）并附带以下固定字段：
- trace_id：请求追踪 ID（由 TraceGenerator 生成）
- operation：操作名称（如 register、get、publish）
- component：组件标识（如 catalog）
- code：错误码（Platform Error Code）

典型调用模式：handler.logger.WarnContext(ctx, "catalog request failed", "trace_id", traceID, "operation", operation, "code", code)

## 日志级别策略
- Info：服务生命周期事件（如 control plane listening）
- Warn：可恢复异常（认证失败、SSE 中断、Router 响应关闭）
- Error：不可恢复错误（readiness check 失败、Platform Error 写入失败）
- Debug：当前代码库中未使用 Debug 级别

## 设计决策
1. 无自定义 Logger 封装：直接使用 slog.Logger，未建立统一的 logger 抽象或中间件。
2. 无日志级别配置：运行时不暴露日志级别开关，全部通过 JSON 输出交由外部采集器处理。
3. 无异步写入：默认同步写入 stderr，未启用异步缓冲。
4. 上下文关联：HTTP 请求链路通过 request.Context() 传递，结合 trace_id 实现请求级日志关联。

## 开发者规范
- 在 Gateway Handler 中使用 ErrorContext/WarnContext/InfoContext 而非非 Context 版本。
- 始终携带 trace_id 和 operation 字段以便跨组件追踪。
- 业务逻辑层（catalog/workspace/invocation service）目前未直接记录日志，错误通过返回值向上冒泡至 Gateway 层统一记录。