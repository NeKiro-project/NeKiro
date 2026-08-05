<div class="satellite-source-note">Read-only mirror of <a href="https://github.com/NeKiro-project/NeKiro-Console/blob/5e577d86825e2ff80434e752342990b9b313d947/.qoder/repowiki/knowledge/zh/Tailwind%20v4%20%2B%20%E6%9A%97%E8%89%B2%E7%8E%BB%E7%92%83%E6%8B%9F%E6%80%81%E8%AE%BE%E8%AE%A1%E7%B3%BB%E7%BB%9F/Tailwind%20v4%20%2B%20%E6%9A%97%E8%89%B2%E7%8E%BB%E7%92%83%E6%8B%9F%E6%80%81%E8%AE%BE%E8%AE%A1%E7%B3%BB%E7%BB%9F.md"><code>NeKiro-project/NeKiro-Console/.qoder/repowiki/knowledge/zh/Tailwind v4 + 暗色玻璃拟态设计系统/Tailwind v4 + 暗色玻璃拟态设计系统.md</code></a> at <code>5e577d86825e2ff80434e752342990b9b313d947</code>. The canonical document remains in the satellite repository; edit it there and refresh this snapshot.</div>

---
kind: frontend_style
name: Tailwind v4 + 暗色玻璃拟态设计系统
category: frontend_style
scope:
    - '**'
source_files:
    - src/index.css
    - vite.config.ts
    - package.json
    - src/components/Sidebar.tsx
    - src/components/Header.tsx
---

## 样式体系概览
本项目采用 React + Vite + Tailwind CSS v4 构建，整体视觉风格为深色科技风，通过 CSS @theme 变量集中定义品牌色板与半透明层级，配合 backdrop-filter 实现玻璃拟态效果。

## 核心文件与依赖
- src/index.css：全局样式入口，导入 Tailwind、声明 @theme 设计令牌、定义 mesh 背景动画、玻璃拟态覆盖与自定义滚动条
- vite.config.ts：注册 @tailwindcss/vite 插件，并配置 @ 路径别名指向项目根目录
- package.json：依赖 tailwindcss@^4.1.14、@tailwindcss/vite@^4.1.14、lucide-react（图标库）、motion（动画）

## 设计令牌（Design Tokens）
所有颜色以 --color-brand-* 前缀的 CSS 变量形式在 @theme 中声明，遵循 Material Design 3 语义命名：表面层级（brand-surface / brand-lowest / brand-container / brand-high 等）、前景文本（brand-on-surface / brand-on-surface-variant）、描边与轮廓（brand-outline / brand-outline-variant）、功能色（brand-primary #818cf8、brand-secondary、brand-tertiary、brand-error 及其 on-/container 变体）、背景（brand-bg 深蓝半透明）。组件内直接使用 Tailwind 类名引用这些令牌，如 bg-brand-lowest、text-brand-primary、border-brand-outline-variant，无需额外配置文件。

## 视觉风格约定
- 背景：固定定位的 mesh 渐变 blob（4 个），使用 mix-blend-mode: screen 与 filter: blur(120px) 产生流动光晕；页面主体 background-color: #020617 作为底色
- 玻璃拟态：侧边栏 aside#sidebar、顶部 header 以及 .bg-brand-low/container/surface/lowest 等容器统一添加 backdrop-filter: blur(20~30px) 与半透明边框
- 字体：主字体 Inter，辅以 -apple-system, BlinkMacSystemFont, sans-serif 回退；组件内大量使用 font-mono-label / font-mono-code 营造终端感
- 动效：blob 浮动使用 @keyframes float1~4 配合 alternate ease-in-out；导航项 hover 使用 transition-all duration-150 与 active:scale-98 微交互
- 滚动条：自定义 ::-webkit-scrollbar，宽度 6px，轨道与滑块均为半透明灰白，hover 时加深

## 组件级样式模式
- 侧边栏：固定左侧 w-60 h-screen，使用 bg-brand-lowest border-r border-brand-outline-variant，导航项按 active 状态切换 text-brand-primary bg-brand-secondary-container/30 border-r-2 border-brand-primary
- 头部：固定顶部 left-60 right-0，搜索框与 workspace 输入框统一使用 bg-brand-container border border-brand-outline-variant/60 rounded 包裹，按钮尺寸紧凑（text-[9.5px] ~ text-[11px]）
- 图标：全部来自 lucide-react，通过 size={} 控制大小，active 状态用 text-brand-primary 高亮

## 开发者规范
1. 颜色一律走 --color-brand-* 令牌，禁止在组件中硬编码十六进制值
2. 布局间距使用 Tailwind 原子类（gap-3、px-6、py-4 等），避免手写 margin/padding
3. 玻璃拟态元素统一加 backdrop-filter: blur(...)，并在 index.css 的覆盖段中维护，不要在组件里重复写
4. 图标统一从 lucide-react 引入，不自行 SVG 或图片资源
5. 字号保持「终端密度」：正文 10~12px，标签 9~11px，标题不超过 14px，优先使用 font-mono-* 系列
6. 响应式策略：当前以桌面端为主，仅通过 hidden xl:flex 等断点控制错误提示显示，移动端适配尚未完善