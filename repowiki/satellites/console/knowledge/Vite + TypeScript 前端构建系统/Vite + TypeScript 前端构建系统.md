<div class="satellite-source-note">Read-only mirror of <a href="https://github.com/NeKiro-project/NeKiro-Console/blob/5e577d86825e2ff80434e752342990b9b313d947/.qoder/repowiki/knowledge/zh/Vite%20%2B%20TypeScript%20%E5%89%8D%E7%AB%AF%E6%9E%84%E5%BB%BA%E7%B3%BB%E7%BB%9F/Vite%20%2B%20TypeScript%20%E5%89%8D%E7%AB%AF%E6%9E%84%E5%BB%BA%E7%B3%BB%E7%BB%9F.md"><code>NeKiro-project/NeKiro-Console/.qoder/repowiki/knowledge/zh/Vite + TypeScript 前端构建系统/Vite + TypeScript 前端构建系统.md</code></a> at <code>5e577d86825e2ff80434e752342990b9b313d947</code>. The canonical document remains in the satellite repository; edit it there and refresh this snapshot.</div>

---
kind: build_system
name: Vite + TypeScript 前端构建系统
category: build_system
scope:
    - '**'
source_files:
    - package.json
    - vite.config.ts
    - tsconfig.json
    - index.html
---

本项目采用 Vite 6 作为核心构建工具，配合 React 19、TypeScript 5.8 和 Tailwind CSS v4 构成完整的前端工程化方案。

## 构建系统与工具链

- 构建器：Vite 6（基于 esbuild 0.25 进行预构建）
- 运行时框架：React 19 + react-dom，使用 react-jsx 模式
- 类型系统：TypeScript 5.8，目标 ES2022，模块解析为 bundler 模式
- 样式方案：Tailwind CSS v4 + @tailwindcss/vite 插件
- 动画库：motion 12
- 测试：tsx --test（原生 Node.js 测试运行器）
- 开发服务器：Vite dev server，默认端口 3000，监听 0.0.0.0

## 关键脚本与命令

- npm run dev：启动开发服务器，支持 HMR（可通过 DISABLE_HMR=true 环境变量关闭）
- npm run build：生产构建，输出到 dist/ 目录
- npm run preview：本地预览生产构建产物
- npm run clean：清理 dist 和 server.js
- npm run lint：TypeScript 类型检查（不生成文件）
- npm run test：使用 tsx 运行 src 下的 *.test.ts 测试文件

## 构建配置要点

- 路径别名：@/* 映射到项目根目录，便于绝对路径导入
- HMR 控制：通过 DISABLE_HMR 环境变量动态开关热更新，适配 AI Studio 的 agent 编辑场景
- 模块格式：type 为 module，全项目使用 ESM
- JSX 处理：react-jsx 自动注入模式，无需显式 import React
- 类型检查：noEmit: true，仅做类型校验，由 Vite 负责编译输出
- 入口文件：index.html 中通过 script type="module" src="/src/main.tsx" 直接挂载

## 依赖管理策略

- 所有依赖均声明在 package.json 中，无 lock 文件提交
- 同时包含运行时依赖（react、express、dotenv 等）和开发依赖（vite、typescript、tailwindcss 等）
- express 和 dotenv 表明项目可能提供轻量后端服务用于代理或预览

## 未覆盖的构建环节

仓库中未发现 Dockerfile、docker-compose.yml、Makefile、CI/CD 配置文件、发布脚本或版本管理脚本。当前构建流程停留在本地开发阶段，尚未集成容器化、自动化部署或制品发布流程。