---
kind: dependency_management
name: Go/Node 双栈依赖管理策略
category: dependency_management
scope:
    - '**'
source_files:
    - go.mod
    - go.sum
    - package.json
    - pnpm-workspace.yaml
    - apps/control-plane/Dockerfile
---

本仓库采用 Go + Node.js 双栈技术，分别通过 go.mod/go.sum 与 pnpm 管理第三方依赖，并在构建阶段通过 Dockerfile 显式配置 GOPROXY/GOSUMDB。

**Go 依赖管理**
- 根级 go.mod 声明模块路径为 github.com/Nene7ko/NeKiro，固定 Go 版本为 1.26.0；所有直接依赖（如 a2aproject/a2a-go、jackc/pgx/v5、getkin/kin-openapi、santhosh-tekuri/jsonschema/v6）均使用语义化版本号，间接依赖由工具自动解析并记录在 go.sum 中。
- 未启用 vendor/ 目录，依赖从远程模块代理拉取。
- 构建镜像 apps/control-plane/Dockerfile 通过 ARG GOPROXY=https://proxy.golang.org 和 GOSUMDB=sum.golang.org 显式指定官方代理与校验数据库，确保可重复构建。

**Node.js 依赖管理**
- 根级 package.json 声明 packageManager: "pnpm@11.3.0" 与 engines.node >= 22.12.0，并通过 pnpm-workspace.yaml 将 apps/*、sdks/*、agents/*、tests/* 纳入工作区，实现跨包共享依赖。
- 顶层仅保留 TypeScript/Vitest 等开发期依赖，业务前端 SDK 位于 sdks/agent-sdk（当前为空占位）。
- 提供统一脚本 check 串联 go test、go vet、pnpm typecheck/test/build，作为本地一致性检查入口。

**约定与约束**
- 新增 Go 依赖需同步更新 go.mod 与 go.sum，避免提交未锁定的版本。
- 生产构建必须通过 Dockerfile 的 GOPROXY/GOSUMDB 参数进行，禁止在 CI 中覆盖默认值。
- 多语言子模块应遵循 pnpm workspace 规范，在各自 package.json 中声明依赖，由根工作区统一安装与类型检查。