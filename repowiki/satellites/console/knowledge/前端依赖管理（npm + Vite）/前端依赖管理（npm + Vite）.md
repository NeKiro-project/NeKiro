<div class="satellite-source-note">Read-only mirror of <a href="https://github.com/NeKiro-project/NeKiro-Console/blob/5e577d86825e2ff80434e752342990b9b313d947/.qoder/repowiki/knowledge/zh/%E5%89%8D%E7%AB%AF%E4%BE%9D%E8%B5%96%E7%AE%A1%E7%90%86%EF%BC%88npm%20%2B%20Vite%EF%BC%89/%E5%89%8D%E7%AB%AF%E4%BE%9D%E8%B5%96%E7%AE%A1%E7%90%86%EF%BC%88npm%20%2B%20Vite%EF%BC%89.md"><code>NeKiro-project/NeKiro-Console/.qoder/repowiki/knowledge/zh/前端依赖管理（npm + Vite）/前端依赖管理（npm + Vite）.md</code></a> at <code>5e577d86825e2ff80434e752342990b9b313d947</code>. The canonical document remains in the satellite repository; edit it there and refresh this snapshot.</div>

---
kind: dependency_management
name: 前端依赖管理（npm + Vite）
category: dependency_management
scope:
    - '**'
source_files:
    - package.json
    - vite.config.ts
    - .env.example
---

本项目采用 npm 作为包管理器，基于 React + Vite + TypeScript 的前端工程。依赖声明集中在 `package.json` 中，分为运行时依赖与开发依赖两类：

- **运行时依赖**：React 19、Vite 6、Express（内嵌轻量服务端）、Tailwind CSS v4、Lucide 图标库、Google GenAI SDK、Framer Motion 等。
- **开发依赖**：TypeScript ~5.8、@types/node、tsx（用于运行测试脚本）、esbuild、autoprefixer、vite 插件等。

关键约定与现状：
1. **无锁文件**：仓库未提交 `package-lock.json` / `pnpm-lock.yaml` / `yarn.lock`，每次安装均从 registry 拉取最新满足语义化版本的包，存在可重现性风险。
2. **版本策略**：所有依赖使用 `^` 或 `~` 前缀，允许小版本/补丁自动升级；`typescript` 显式锁定到 `~5.8.2`，其余多为 `^` 范围。
3. **私有源/代理**：未发现 `.npmrc`、`.yarnrc` 或 `package.json` 中的 `registry`/`@scope:registry` 配置，默认使用官方 npm registry；也未见 vendoring 或本地镜像策略。
4. **构建期别名**：通过 `vite.config.ts` 的 `resolve.alias` 将 `@` 映射到项目根目录，统一模块导入路径。
5. **环境变量注入**：通过 `.env.example` 定义 `VITE_*` 前缀变量，由 Vite 在构建时注入到客户端代码，用于连接 NeKiro Control Plane API。
6. **模块解析模式**：`tsconfig.json` 使用 `"moduleResolution": "bundler"`，适配现代打包器行为，无需额外 `node_modules` 类型兼容配置。
7. **脚本约定**：`dev`/`build`/`preview`/`lint`/`test` 等标准 npm scripts 覆盖完整开发流程，`test` 使用 `tsx --test` 直接执行 TS 测试文件。

开发者应遵循的规则：
- 新增依赖后务必提交生成的锁文件（建议启用 `npm install --package-lock-only` 并纳入版本控制），以保证团队与环境一致性。
- 区分 `dependencies` 与 `devDependencies`，仅将构建产物实际运行的包放入前者。
- 对需要严格复现的关键工具（如 TypeScript、Vite 插件）优先使用 `~` 锁定次版本，避免破坏性更新。
- 如需接入私有 npm registry 或镜像，应在仓库根添加 `.npmrc` 并在 CI 中同步配置，而非硬编码到脚本中。