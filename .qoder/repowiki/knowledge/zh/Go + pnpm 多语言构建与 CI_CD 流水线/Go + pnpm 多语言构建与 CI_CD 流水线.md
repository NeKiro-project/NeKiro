---
kind: build_system
name: Go + pnpm 多语言构建与 CI/CD 流水线
category: build_system
scope:
    - '**'
source_files:
    - go.mod
    - apps/control-plane/Dockerfile
    - .github/workflows/ci.yml
    - deploy/compose.yaml
    - package.json
---

## 构建系统概览

NeKiro 平台采用 **Go (1.26) + pnpm (11.3)** 双栈构建体系，通过 GitHub Actions 实现统一的 CI 流水线。后端以 Go modules 管理依赖，前端使用 pnpm workspace 组织多个 Node.js 子项目。

## 核心构建工具链

### Go 构建（后端）
- **模块管理**: `go.mod` 声明根模块 `github.com/Nene7ko/NeKiro`，Go 版本锁定为 1.26.0
- **编译参数**: Dockerfile 中使用 `CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w"` 生成静态二进制，剥离调试信息并最小化镜像体积
- **依赖缓存**: CI 通过 `cache-dependency-path: go.sum` 缓存 Go 模块下载
- **测试覆盖**: `go test -coverprofile coverage.txt ./...` 生成覆盖率报告上传至 Codecov
- **代码质量**: 集成 `golangci-lint v2.12.2`、`go vet` 和标准单元测试

### Node.js 构建（前端）
- **包管理器**: pnpm 11.3.0，通过 `pnpm-workspace.yaml` 管理多包工作区
- **脚本命令**: `build`、`typecheck`、`test` 通过 `-r --if-present` 递归执行各子包脚本
- **Node 版本**: 要求 >= 22.12.0，CI 固定使用 24.16.0
- **类型检查**: TypeScript ~6.0.3 + Vitest ^4.1.10

## 容器化策略

### Control Plane 镜像
- **多阶段构建**: `apps/control-plane/Dockerfile` 使用 golang:1.26.4-bookworm 作为构建器，最终镜像基于 scratch
- **安全加固**: 以非 root 用户 (UID 65532) 运行，仅暴露 8080 端口
- **证书注入**: 从构建阶段复制 `/etc/ssl/certs/ca-certificates.crt`
- **健康检查**: 通过 `/control-plane healthcheck http://127.0.0.1:8080/readyz` 验证服务就绪状态

### 本地开发编排
- **Docker Compose**: `deploy/compose.yaml` 定义 PostgreSQL 17.9 + Control Plane 的完整开发环境
- **数据库迁移**: 独立的 `control-plane-migrate` 服务在应用启动前执行 `migrate up`
- **网络隔离**: 使用 `platform-internal`（内部网络）和 `local-access` 两个网络分离访问权限
- **环境变量**: 所有关键配置通过 `${VAR:?message}` 语法强制校验必填项

## CI/CD 流水线（GitHub Actions）

### 并行作业
1. **go-quality**: 代码质量检查（构建、测试、覆盖率、vet、lint）
2. **workspace-integration**: 带 PostgreSQL 17.9 服务的集成测试，按 tag `integration` 过滤
3. **frontend**: 前端类型检查、测试与构建
4. **compose-config**: 验证 docker-compose 配置文件有效性

### 关键配置
- **PostgreSQL 服务**: 通过 `services.postgres` 启动，使用 `pg_isready` 健康检查
- **测试数据库**: `NEKIRO_TEST_DATABASE_URL` 环境变量注入连接字符串
- **Codecov 集成**: 上传覆盖率报告并设置 `fail_ci_if_error: true`

## 构建约定与最佳实践

### 目录结构规范
- 应用入口位于 `apps/<app>/cmd/<app>/main.go`
- 每个可部署组件独立 Dockerfile
- 契约定义集中在 `contracts/` 目录，支持 JSON Schema 验证

### 环境变量约定
- 数据库连接: `NEKIRO_DATABASE_URL` / `NEKIRO_COMPOSE_DATABASE_URL`
- 监听地址: `NEKIRO_LISTEN_ADDRESS`
- 认证模式: `NEKIRO_AUTH_MODE`, `NEKIRO_INTERNAL_AUTH_MODE`
- 开发凭据: `NEKIRO_DEV_AUTH_PRINCIPALS_JSON`, `NEKIRO_INTERNAL_DEV_AUTH_PRINCIPALS_JSON`

### 版本管理
- Go 模块版本由 `go.mod` 统一管理
- Node.js 版本通过 `package.json` engines 字段约束
- 容器镜像使用 SHA256 摘要锁定基础镜像版本（如 `postgres:17.9-bookworm@sha25:...`）