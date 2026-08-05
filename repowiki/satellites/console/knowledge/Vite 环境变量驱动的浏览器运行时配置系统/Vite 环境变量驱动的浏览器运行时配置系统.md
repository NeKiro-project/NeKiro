<div class="satellite-source-note">Read-only mirror of <a href="https://github.com/NeKiro-project/NeKiro-Console/blob/5e577d86825e2ff80434e752342990b9b313d947/.qoder/repowiki/knowledge/zh/Vite%20%E7%8E%AF%E5%A2%83%E5%8F%98%E9%87%8F%E9%A9%B1%E5%8A%A8%E7%9A%84%E6%B5%8F%E8%A7%88%E5%99%A8%E8%BF%90%E8%A1%8C%E6%97%B6%E9%85%8D%E7%BD%AE%E7%B3%BB%E7%BB%9F/Vite%20%E7%8E%AF%E5%A2%83%E5%8F%98%E9%87%8F%E9%A9%B1%E5%8A%A8%E7%9A%84%E6%B5%8F%E8%A7%88%E5%99%A8%E8%BF%90%E8%A1%8C%E6%97%B6%E9%85%8D%E7%BD%AE%E7%B3%BB%E7%BB%9F.md"><code>NeKiro-project/NeKiro-Console/.qoder/repowiki/knowledge/zh/Vite 环境变量驱动的浏览器运行时配置系统/Vite 环境变量驱动的浏览器运行时配置系统.md</code></a> at <code>5e577d86825e2ff80434e752342990b9b313d947</code>. The canonical document remains in the satellite repository; edit it there and refresh this snapshot.</div>

---
kind: configuration_system
name: Vite 环境变量驱动的浏览器运行时配置系统
category: configuration_system
scope:
    - '**'
source_files:
    - .env.example
    - vite.config.ts
    - src/App.tsx
    - src/api/nekiro.ts
    - package.json
---

## 系统概述
本项目采用 Vite 原生 `import.meta.env` 机制作为前端运行时配置系统，通过 `.env*` 文件注入以 `VITE_` 为前缀的环境变量，在构建期被内联到打包产物中。所有 NeKiro Control Plane 的连接参数、认证凭据和默认工作区均通过此方式提供。

## 关键文件与包
- `.env.example` — 环境变量模板，声明全部可配置项及用途说明
- `vite.config.ts` — 构建期 HMR/watch 行为受 `DISABLE_HMR` 控制（非业务配置）
- `src/App.tsx` — 唯一读取 `import.meta.env.VITE_NEKIRO_*` 的入口，集中构造 `NekiroApiClient`
- `src/api/nekiro.ts` — API 客户端，对缺失 `baseUrl` 返回 `CONFIGURATION_ERROR` 错误码
- `package.json` — 依赖 `dotenv`（当前未在服务端使用），脚本定义 `dev/build/preview` 三阶段

## 架构与约定
1. **变量命名规范**：所有暴露给浏览器的配置必须以 `VITE_` 开头，由 Vite 自动注入；其余变量仅用于 Node 侧（如 `GEMINI_API_KEY`、`APP_URL`）。
2. **配置分层**：
   - 连接层：`VITE_NEKIRO_API_BASE_URL`（必填）、`VITE_NEKIRO_TOKEN`（可选 Bearer）
   - 身份层：`VITE_NEKIRO_OWNER_ID`、`VITE_NEKIRO_OWNER_NAME`（注册 Agent 时作为 owner）
   - 初始化层：`VITE_NEKIRO_DEFAULT_WORKSPACE_ID`（启动后自动加载对应 Workspace）
3. **读取位置集中化**：仅在 `App.tsx` 一处通过 `useMemo` 创建 `NekiroApiClient`，组件仅消费 props，避免散落式 `import.meta.env` 调用。
4. **安全约束**：文档与代码注释反复强调 token 不得写入源码或 localStorage，仅随构建产物静态注入。
5. **构建期常量内联**：由于是纯前端应用，不存在运行时动态加载配置的能力；不同环境通过不同的构建命令或 CI 注入不同 `.env` 实现切换。
6. **开发辅助**：`vite.config.ts` 中的 `DISABLE_HMR` 属于开发体验开关，不属于业务配置范畴。

## 开发者应遵循的规则
- 新增浏览器可见配置必须加 `VITE_` 前缀并同步更新 `.env.example` 注释。
- 禁止在组件内部直接访问 `import.meta.env`，统一在顶层模块聚合后以 props/state 传递。
- 敏感信息（token、密钥）永远不提交进版本库，仅保留占位示例。
- 若某配置缺失导致功能不可用，应在 UI 显式提示（参考 Settings Overlay 中对 base URL 的展示）。
- 服务端逻辑如需读取非 `VITE_` 变量，应通过独立的 Node 配置模块管理，不要混入前端构建产物。