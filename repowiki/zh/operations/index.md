# 运维

本节是 Core 仓库的运维入口。Console、sample Agent、Compose stack 和产品
验收仍由[仓库地图](../repositories.md)中的卫星仓库负责。

## Core 本地检查

Core 要求 Go 1.26 或更新版本；PostgreSQL integration suite 需要显式配置。
基线质量检查：

```text
gofmt -l apps contracts tests
go mod tidy
go build ./...
go test ./...
go test -race ./...
go vet ./...
```

PostgreSQL integration test 必须使用名称以 _test 结尾的专用数据库。缺少或
非法的必需配置属于 readiness failure，不使用猜测的默认值。

## 受信发布流程

Trusted Publication v1 是 endpoint 参与 verified Agent Release 之前由 Registry
拥有的 proof step。它不部署 Agent，也不允许 Console 或调用方绕过 Router。

运维 runbook 按以下顺序执行：

1. 准备显式的 Gateway request 和 deployment configuration。
2. 注册并发布带版本的 Agent Card 及 trusted Release facts。
3. 通过 Gateway API 检查 trust 和 Invocation provenance。
4. 需要时通过 owner lifecycle operation suspend 或 revoke。
5. 使用固定版本的 Stack 完整验证 acceptance path。

Runbook 不使用 SQL 修改 domain state，不直接调用 Agent 作为 recovery path，不
修改 immutable Release，也不通过备用 endpoint 隐藏依赖失败。

## 恢复姿态

Card 缺失、request 非法、Workspace 未授权、Installation disabled、Agent version
disabled、依赖失败、超时、取消和 trust proof rejected 保持各自的错误与审计含义。
Fallback budget 默认为零，除非已有 contract、ADR、runbook 或 SLO 提供政策证据。

## Canonical source

- [Core development](../source-docs/usage/core-development.md)
- [Trusted publication operations](../source-docs/usage/trusted-publication-operations.md)
- [Trusted Publication v1](../source-docs/contracts/trusted-publication-v1.md)
