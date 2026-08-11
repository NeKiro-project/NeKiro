<div align="center">

# NeKiro

**一套面向 Agent 注册、发现、安装、调用与审计的运行时无关框架。**

[![CI](https://github.com/NeKiro-project/NeKiro/actions/workflows/ci.yml/badge.svg)](https://github.com/NeKiro-project/NeKiro/actions/workflows/ci.yml)
[![Satellite Integration](https://github.com/NeKiro-project/NeKiro/actions/workflows/satellite-integration.yml/badge.svg)](https://github.com/NeKiro-project/NeKiro/actions/workflows/satellite-integration.yml)
[![Codecov](https://codecov.io/gh/NeKiro-project/NeKiro/branch/main/graph/badge.svg)](https://codecov.io/gh/NeKiro-project/NeKiro)
[![Documentation](https://img.shields.io/badge/docs-RepoWiki-blue)](https://nekiro-project.github.io/NeKiro/zh/)
[![License](https://img.shields.io/badge/license-Apache--2.0-green.svg)](LICENSE)

[English](README.md) ·
[文档](https://nekiro-project.github.io/NeKiro/zh/) ·
[架构](docs/architecture/platform-direction.md) ·
[契约](docs/contracts/compatibility.md) ·
[Samples](https://github.com/NeKiro-project/NeKiro-Samples) ·
[Stack](https://github.com/NeKiro-project/NeKiro-Stack)

</div>

NeKiro 提供围绕 Agent Runtime 的平台层：为 Agent 建立版本化身份，使精确
Release 可以被发现和安装，通过受控的 A2A 边界转发托管调用，并在不接管 Agent
内部执行模型的前提下记录完整调用链路。

当使用不同 Agent 框架或自研 Runtime 的多个节点需要在统一契约与安全模型下协作时，
可以使用 NeKiro 管理它们之间的连接、生命周期和组织级信任边界。

## 为什么选择 NeKiro？

[tRPC-Agent-Go](https://github.com/trpc-group/trpc-agent-go) 等 Agent Runtime
负责模型、Prompt、Tool、Planner、Workflow、Memory、RAG 和 Session。
NeKiro 不替代这些能力，而是连接独立实现的 Agent，并管理完整的平台生命周期：

```text
Register -> Discover -> Install -> Invoke -> Record
```

- **运行时无关**：Agent 可以使用 tRPC-Agent-Go、`a2a-go`、其他框架或自研 Runtime。
- **契约优先**：Agent Card、Release、HTTP API、内部 API、A2A Profile、凭证、事件和结果分别进行版本化。
- **Router 托管**：Consumer 到 Provider 以及 Agent-to-Agent 调用统一经过 A2A Router，不向 Consumer 暴露 Provider 直连地址。
- **精确版本路由**：发现与调用保留 Agent 版本、Release ID、Card digest、endpoint provenance 和 audience。
- **Workspace 治理**：发现 Agent 不等于获得调用权限；安装版本和所需权限必须显式授权。
- **可审计链路**：嵌套调用传播 `root_task_id`、`parent_invocation_id` 和 `trace_id`，并写入追加式 metadata Ledger。
- **失败关闭**：拓扑缺失、配置非法、信任校验失败和依赖不可用保持不同错误语义，不使用重试、备用端点或陈旧成功结果掩盖失败。

## 平台生命周期

| 阶段 | 行为 | 所属边界 |
| --- | --- | --- |
| Register | Provider 发布 Agent Card 和不可变 Release，并注册 ready Runtime 实例。 | Catalog Registry + Provider deployment |
| Discover | Consumer 按能力查询已发布 Agent 及其精确版本。 | Gateway + Catalog |
| Install | Workspace 授权一个精确 Agent 版本及所需权限。 | Workspace |
| Invoke | Gateway 将请求交给 A2A Router；Router 解析精确 Release 与 ready 实例。 | Gateway + A2A Router |
| Record | Router 追加记录状态、失败、耗时、Release provenance 以及 task/trace lineage。 | Invocation Ledger |

## 当前已经支持

当前产品 Stack 使用 PostgreSQL、Nacos、生产 Console 和两个独立实现的 Sample
Runtime 验证完整生命周期。Core 也已经为 Router 的 Nacos HTTP 与 gRPC 传输提供
显式私有 CA TLS/mTLS 配置；下一阶段会把 Router 侧安全传输接入真实 Stack 验收。

- 可信 Agent Card 注册与不可变 Release 发布。
- 按能力发现 Agent，并支持公开分享 Agent。
- Workspace 安装与显式权限授权。
- 精确 Release 实例注册和 Nacos 生命周期观察。
- 原子的 initial snapshot/watch handoff，以及空拓扑失败关闭。
- Runtime 下线、实例替换和路由恢复。
- JSON 调用和 Server-Sent Events 流式调用。
- 向 Provider 传播取消信号。
- 双向嵌套 Agent 调用，并保证再次进入 Router。
- PostgreSQL 中可查询的 Invocation 与 trace lineage。
- Router 签发的短期 Agent 调用凭证。
- Router Nacos HTTP/gRPC 私有 CA TLS/mTLS 配置和 Core 测试。
- Provider Nacos TLS/mTLS 注册的产品 E2E。
- Linux 与 Windows Config Center 验证。

Stack 会明确验证：Consumer 不能绕过 Router、已下线实例不会收到新调用、Agent
payload 与密钥不会写入 Ledger。

## 架构

```mermaid
flowchart LR
    User["User / Console / Application"] --> Gateway["Control Plane Gateway"]
    Gateway --> Catalog["Catalog Registry"]
    Gateway --> Workspace["Workspace"]
    Gateway --> Router["A2A Router"]

    Provider["Provider Agent"] -->|"Card + Release"| Gateway
    Provider -->|"Instance lease"| Nacos["Nacos"]
    Nacos -->|"Snapshot + watch"| Router

    Router -->|"Managed A2A call"| Provider
    Provider -->|"Nested managed call"| Router
    Router --> Ledger["Invocation Ledger"]

    Catalog --> Postgres[("PostgreSQL")]
    Workspace --> Postgres
    Ledger --> Postgres
```

部署边界遵守以下规则：

1. Console 和外部应用只能访问 Gateway。
2. Gateway 不直接执行 Agent，而是将托管调用交给 A2A Router。
3. Registry 是 Agent Card 和 Release 永久事实的唯一来源。
4. Router 解析精确 Release 与 ready 实例，但不保存第二份永久 Card。
5. Agent-to-Agent 调用再次经过 Router，并保留 task/trace lineage。
6. Ledger 只记录 metadata 事实，不保存 Agent 输入、输出、凭证或密钥。

完整所有权和信任模型请阅读
[平台方向](docs/architecture/platform-direction.md)与
[Phase 1 架构](docs/architecture/phase-1-spec.md)。

## Samples

[NeKiro-Samples](https://github.com/NeKiro-project/NeKiro-Samples) 提供两个遵循
相同平台契约、但采用不同 Runtime 实现的 Agent：

| Sample | Runtime | 验证内容 |
| --- | --- | --- |
| Runtime A | tRPC-Agent-Go | 基于 Agent 框架的 Runtime、Nacos TLS 注册、Router 托管调用，以及嵌套调用 Runtime B。 |
| Runtime B | 直接使用 `a2a-go` | JSON/SSE/task/cancellation、Nacos mTLS 注册、精确实例身份，以及嵌套调用 Runtime A。 |

Stack 还会启动替换后的 Runtime B 实例，并针对 Provider 注册验证错误 CA、错误 TLS
server name 和缺失 mTLS client identity 等失败场景。

```text
Runtime A -> Router -> Runtime B
Runtime B -> Router -> Runtime A
```

两个 Runtime 都不会获得对方的直接目标地址。

## 快速开始

下面的例子会运行真实的 **Runtime B -> Router -> Runtime A** 链路。Runtime A
先向 Nacos 发布租约，Router 通过初始 snapshot 和持续 watch 观察到该实例。Runtime B
只使用 Agent ID 和 capability 调用 A，不会获得 A 的地址。

```text
Runtime A --lease--> Nacos --snapshot/watch--> Router
Runtime B --Agent ID + capability--> Router --> Runtime A
                                      |
                                      +--> Invocation Ledger
```

生产实现位于 [NeKiro-Samples](https://github.com/NeKiro-project/NeKiro-Samples)。
下面两个完整的 `package main` 程序需要放在该模块内运行，因为 Nacos 部署 adapter
和 endpoint challenge 属于 Samples，而不是 Core。

Runtime A 启动托管 A2A endpoint，注册精确实例，持续维护 lease，并在退出时注销：

```go
package main

import (
    "context"
    "errors"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/NeKiro-project/NeKiro-Samples/internal/challengeproof"
    "github.com/NeKiro-project/NeKiro-Samples/internal/nacosregistration"
    runtimea "github.com/NeKiro-project/NeKiro-Samples/runtime-a"
)

func main() {
    if err := run(); err != nil {
        log.Fatal(err)
    }
}

func run() error {
    config, err := runtimea.LoadConfig(os.LookupEnv)
    if err != nil {
        return err
    }
    registrationConfig, err := nacosregistration.Load(
        os.LookupEnv, "RUNTIME_A", config.AgentID, config.InstanceID,
    )
    if err != nil {
        return err
    }
    registrationClient, err := nacosregistration.NewHTTPClient(registrationConfig)
    if err != nil {
        return err
    }
    registration, err := nacosregistration.New(registrationConfig, registrationClient)
    if err != nil {
        return err
    }

    handler, err := runtimea.NewHandler(config, http.DefaultClient)
    if err != nil {
        return err
    }
    application, err := challengeproof.NewHandler(
        runtimea.NewHTTPHandlerWithReadiness(handler, registration),
        os.LookupEnv,
    )
    if err != nil {
        return err
    }
    if err := registration.Register(context.Background()); err != nil {
        return err
    }

    ctx, stop := signal.NotifyContext(
        context.Background(), os.Interrupt, syscall.SIGTERM,
    )
    defer stop()
    server := &http.Server{Addr: config.ListenAddress, Handler: application}
    serverErrors := make(chan error, 1)
    go func() {
        err := server.ListenAndServe()
        if errors.Is(err, http.ErrServerClosed) {
            err = nil
        }
        serverErrors <- err
    }()
    registrationErrors := make(chan error, 1)
    go func() { registrationErrors <- registration.Run(ctx) }()

    var runErr error
    registrationStopped := false
    select {
    case <-ctx.Done():
    case runErr = <-serverErrors:
    case runErr = <-registrationErrors:
        registrationStopped = true
    }
    stop()

    shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    shutdownErr := server.Shutdown(shutdownContext)
    if !registrationStopped {
        select {
        case registrationErr := <-registrationErrors:
            runErr = errors.Join(runErr, registrationErr)
        case <-shutdownContext.Done():
            runErr = errors.Join(runErr, shutdownContext.Err())
        }
    }
    deregisterErr := registration.Deregister(shutdownContext)
    return errors.Join(runErr, shutdownErr, deregisterErr)
}
```

B 使用同样的方式注册，在入站 A2A endpoint 校验 Router 签发的凭证，并使用 Router
URL 和 Agent token 创建 handler。B 收到 Sample 的 `nested` 请求时，
`NewConfiguredHandler` 使用公共 Agent SDK 按 Agent ID 与 capability 调用 A；B
不会自行解析或直连 A。

```go
package main

import (
    "context"
    "errors"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/NeKiro-project/NeKiro-Samples/internal/challengeproof"
    "github.com/NeKiro-project/NeKiro-Samples/internal/nacosregistration"
    runtimeb "github.com/NeKiro-project/NeKiro-Samples/runtime-b"
    "github.com/NeKiro-project/nekiro-sdk-go/agent/routerauth"
)

func main() {
    if err := run(); err != nil {
        log.Fatal(err)
    }
}

func run() error {
    address, err := runtimeb.ListenAddressFromEnvironment(os.LookupEnv)
    if err != nil {
        return err
    }
    config, err := runtimeb.LoadConfig(os.LookupEnv)
    if err != nil {
        return err
    }
    authenticationConfig, err := routerauth.LoadConfig(os.LookupEnv)
    if err != nil {
        return err
    }
    registrationConfig, err := runtimeb.LoadRegistrationConfig(
        os.LookupEnv, config.AgentID, config.InstanceID,
    )
    if err != nil {
        return err
    }
    registrationClient, err := nacosregistration.NewHTTPClient(registrationConfig)
    if err != nil {
        return err
    }
    registration, err := runtimeb.NewNacosRegistration(
        registrationConfig, registrationClient,
    )
    if err != nil {
        return err
    }

    handler, err := runtimeb.NewConfiguredHandler(config, http.DefaultClient)
    if err != nil {
        return err
    }
    execution, err := runtimeb.NewHTTPHandlerWithAuthAndReadiness(
        handler, authenticationConfig, registration,
    )
    if err != nil {
        return err
    }
    application, err := challengeproof.NewHandler(execution, os.LookupEnv)
    if err != nil {
        return err
    }
    if err := registration.Register(context.Background()); err != nil {
        return err
    }

    ctx, stop := signal.NotifyContext(
        context.Background(), os.Interrupt, syscall.SIGTERM,
    )
    defer stop()
    server := &http.Server{Addr: address, Handler: application}
    serverErrors := make(chan error, 1)
    go func() {
        err := server.ListenAndServe()
        if errors.Is(err, http.ErrServerClosed) {
            err = nil
        }
        serverErrors <- err
    }()
    registrationErrors := make(chan error, 1)
    go func() { registrationErrors <- registration.Run(ctx) }()

    var runErr error
    registrationStopped := false
    select {
    case <-ctx.Done():
    case runErr = <-serverErrors:
    case runErr = <-registrationErrors:
        registrationStopped = true
    }
    stop()

    shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    shutdownErr := server.Shutdown(shutdownContext)
    if !registrationStopped {
        select {
        case registrationErr := <-registrationErrors:
            runErr = errors.Join(runErr, registrationErr)
        case <-shutdownContext.Done():
            runErr = errors.Join(runErr, shutdownContext.Err())
        }
    }
    deregisterErr := registration.Deregister(shutdownContext)
    return errors.Join(runErr, shutdownErr, deregisterErr)
}
```

Handler 只会在 `routerauth` 验证 Router 凭证后提取 `PlatformContext`，再通过
`agentsdk.Client.Invoke` 和 `TargetAgentID: "runtime-a"` 发起调用。完整实现见
[B -> A 调用代码](https://github.com/NeKiro-project/NeKiro-Samples/blob/main/runtime-b/nested.go)。

### 运行完整场景

固定全部组件 revision 的
[NeKiro-Stack](https://github.com/NeKiro-project/NeKiro-Stack) 会补齐上面刻意省略的
可信 Card/Release 发布、Workspace 安装与权限、Router 凭证、PostgreSQL、安全
Nacos、精确组件版本和 Ledger 断言。以下命令需要 Git、Go 1.26+、Docker、Bash
和网络访问：

```bash
git clone https://github.com/NeKiro-project/NeKiro-Stack.git
cd NeKiro-Stack

work_root=$(mktemp -d)
backend_env="$work_root/backend.env"
prepared_env="$work_root/prepared.env"

./scripts/write-ci-env.sh backend "$backend_env" "$(pwd)" nekiro-quickstart
set -a
source "$backend_env"
set +a

./scripts/prepare.sh "$(pwd)/components.json" "$work_root/checkouts" "$prepared_env"
set -a
source "$prepared_env"
set +a

go run ./cmd/nacos-secure-fixture generate "$NEKIRO_E2E_TLS_ROOT"
docker compose --project-name "$NEKIRO_E2E_COMPOSE_PROJECT" \
  --file compose.yaml \
  --file "$NEKIRO_E2E_COMPOSE_OVERRIDE_FILE" \
  --profile router-nacos-secure \
  up --detach --wait --wait-timeout 120
go test -tags=e2e -run '^TestInvokeToRecordAcceptance$' -count=1 ./tests/backend

docker compose --project-name "$NEKIRO_E2E_COMPOSE_PROJECT" \
  --file compose.yaml \
  --file "$NEKIRO_E2E_COMPOSE_OVERRIDE_FILE" \
  --profile router-nacos-secure \
  --profile runtime-registration \
  --profile watch-refresh \
  down --volumes --remove-orphans
```

验收通过表示：A 已注册，Router 发现了 A 对应精确 Release 的实例，B 只通过 Router
到达 A，并且 Ledger 中父子记录共享 `root_task_id` 和 `trace_id`。同一套验收还会
覆盖 A -> B、实例下线、失败关闭路由和替换实例后的恢复。

### 开发 Core

Core 开发需要 Go 1.26 或更高版本：

```bash
git clone https://github.com/NeKiro-project/NeKiro.git
cd NeKiro

go mod download
go build ./...
go test ./...
go test -race ./...
go vet ./...
```

在仓库根目录构建两个 Core 服务镜像：

```bash
docker build --file apps/control-plane/Dockerfile --tag nekiro-control-plane:local .
docker build --file apps/a2a-router/Dockerfile --tag nekiro-a2a-router:local .
```

单独运行 Core 二进制只能验证 Core 开发状态，不能代表产品 E2E 成功。注册、发现、
安装、路由、Sample Agent、Console 行为和 Ledger 记录需要通过 NeKiro-Stack 联合验收。

## 仓库地图

NeKiro 使用独立仓库保持源码所有权与发布边界清晰。

| 仓库 | 职责 |
| --- | --- |
| [NeKiro](https://github.com/NeKiro-project/NeKiro) | Control Plane、A2A Router、契约、Config Center 源语义、服务自有 migration 和 Core 验证。 |
| [NeKiro-Console](https://github.com/NeKiro-project/NeKiro-Console) | 生产 Web Console。 |
| [nekiro-sdk-go](https://github.com/NeKiro-project/nekiro-sdk-go) | 面向应用和 Agent 的公共 Go SDK。 |
| [NeKiro-Samples](https://github.com/NeKiro-project/NeKiro-Samples) | 跨 Runtime Sample Agent 和 Provider 部署接线。 |
| [NeKiro-Stack](https://github.com/NeKiro-project/NeKiro-Stack) | 固定 revision 的多组件装配和产品 backend/browser E2E。 |
| [nekiro-a2a-transport-go](https://github.com/NeKiro-project/nekiro-a2a-transport-go) | 可复用 A2A HTTP/JSON-RPC/SSE transport mechanics。 |

Core required CI 不 checkout 或构建卫星仓源码。Core 合并后，独立 Satellite
Integration workflow 会用精确 Core SHA 调用卫星仓自己维护的 reusable workflow。

## Core 仓库结构

```text
apps/control-plane/   Gateway、Catalog、Workspace、发布与 Dispatch
apps/a2a-router/      A2A 路由、凭证、拓扑、策略扩展点与 Ledger
config_center/        Provider-neutral 字节快照、读取、观察和显式发布
contracts/            JSON Schema、OpenAPI、A2A Profile 与 Go 映射
tests/                Core 契约与服务集成验证
docs/                 架构、契约、ADR、使用和运维文档
```

Catalog、Workspace 和 Ledger migration 与拥有对应 schema 的模块一起维护，并嵌入
对应服务二进制。

## 安全模型

- Secret、生产 endpoint、数据库地址、JWT key、证书与 trust root 没有推测默认值。
- 托管调用不存在 Consumer 到 Provider 的直连路径。
- 安全 Nacos 模式不使用系统根证书或 `InsecureSkipVerify` fallback。
- 不自动重试、切换备用 endpoint、回退旧组件或使用陈旧拓扑返回成功。
- Ledger metadata 不保存 Agent payload、凭证或密钥。
- 跨部署边界不共享内部实现类型，只使用版本化契约。

安全与兼容性决策记录在 [`docs/decisions`](docs/decisions/) 中，包括 Runtime 信任、
Router 签名凭证、实例发现、注册 lease 和 Nacos transport security。

## 开发与验证

PostgreSQL 集成测试需要显式指定名称以 `_test` 结尾的专用数据库：

```bash
export NEKIRO_TEST_DATABASE_URL='postgresql://user:password@127.0.0.1:5432/nekiro_core_test?sslmode=disable'

go test -tags=integration -count=1 ./apps/control-plane/internal/catalog/postgres
go test -tags=integration -count=1 ./apps/control-plane/internal/workspace/postgres
go test -tags=integration -count=1 ./apps/control-plane/internal/workspace/integration
go test -tags=integration -count=1 ./apps/a2a-router/internal/ledger
go test -tags=integration -count=1 ./tests/integration/catalog
```

常用文档：

- [Core 开发](docs/usage/core-development.md)
- [可信发布运维](docs/usage/trusted-publication-operations.md)
- [Config Center Runtime 运维](docs/usage/config-center-runtime-operations.md)
- [外部 Gateway 运维](docs/usage/external-gateway-operations.md)
- [契约兼容性政策](docs/contracts/compatibility.md)
- [统一 RepoWiki](https://nekiro-project.github.io/NeKiro/zh/)

## 项目方向

NeKiro 从最初的“可插拔 Agent”逐步发展为一套 Agent Framework，重点管理独立
节点之间的 link、contract、外部生命周期和组织级信任边界。当前方向是让完整
Provider/Consumer 托管链路更加易部署、安全、可观察且语言无关，同时继续由各个
Runtime owner 负责 Agent 内部智能。

## 参与贡献

欢迎提交 Issue 和 Pull Request。用户可见行为、公共契约、数据模型或核心架构变更
应先建立 Issue；所有权或兼容性决策还应同步更新 ADR。提交前的验证要求见
[Core 开发](docs/usage/core-development.md)。

## 历史与许可证

annotated tag `pre-repository-split-2026-08-04` 保存已验收的 monorepo tree 和
Spec Kit 历史。仓库所有权和迁移原因记录在
[ADR 0009](docs/decisions/0009-core-repository-boundary.md)。

NeKiro 使用 [Apache License 2.0](LICENSE)。
