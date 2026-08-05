<div class="satellite-source-note">Read-only mirror of <a href="https://github.com/NeKiro-project/NeKiro-Console/blob/5e577d86825e2ff80434e752342990b9b313d947/.qoder/repowiki/knowledge/zh/%E5%89%8D%E7%AB%AF%E9%94%99%E8%AF%AF%E5%A4%84%E7%90%86%EF%BC%9ANekiroApiError%20%E4%B8%8E%20PlatformErrorView%20%E7%BB%9F%E4%B8%80%E6%A8%A1%E5%9E%8B/%E5%89%8D%E7%AB%AF%E9%94%99%E8%AF%AF%E5%A4%84%E7%90%86%EF%BC%9ANekiroApiError%20%E4%B8%8E%20PlatformErrorView%20%E7%BB%9F%E4%B8%80%E6%A8%A1%E5%9E%8B.md"><code>NeKiro-project/NeKiro-Console/.qoder/repowiki/knowledge/zh/前端错误处理：NekiroApiError 与 PlatformErrorView 统一模型/前端错误处理：NekiroApiError 与 PlatformErrorView 统一模型.md</code></a> at <code>5e577d86825e2ff80434e752342990b9b313d947</code>. The canonical document remains in the satellite repository; edit it there and refresh this snapshot.</div>

---
kind: error_handling
name: 前端错误处理：NekiroApiError 与 PlatformErrorView 统一模型
category: error_handling
scope:
    - '**'
source_files:
    - src/api/nekiro.ts
    - src/types.ts
    - src/App.tsx
---

本仓库采用「单一异常类型 + 平台错误视图」的前端错误处理体系，集中在 src/api/nekiro.ts 中实现，并在 App.tsx 的每个异步操作处统一捕获、归一化并渲染。

## 1. 使用的系统与模式
- 自定义异常类：NekiroApiError extends Error，携带 HTTP status、业务 code、traceId，并提供 toView() 转换为 UI 层可消费的 PlatformErrorView。
- 统一错误视图类型：PlatformErrorView（定义于 types.ts）作为跨组件传递的错误载体，包含 status | code | message | traceId。
- API 客户端集中封装：NekiroApiClient.request<T> 是唯一的网络入口，负责把原始 fetch 响应或网络异常统一包装为 NekiroApiError。
- 顶层 try/catch + toPlatformErrorView：App.tsx 中每个异步函数用 try/catch 包裹，调用 toPlatformErrorView(error, fallbackMessage) 将任意 unknown 错误归一化为 PlatformErrorView，再写入对应 useState。

## 2. 关键文件与位置
- src/api/nekiro.ts
  - NekiroApiError 类（第 106–127 行）
  - NekiroApiClient.request 统一请求/错误转换（第 235–276 行）
  - toPlatformErrorView 归一化工具（第 375–384 行）
  - 参数校验辅助函数 readText / readIdentifier / ensureUnique / isSemver 等抛出 Error（第 402–440 行）
- src/types.ts
  - PlatformErrorView 接口（第 44–49 行）
- src/App.tsx
  - 每个数据加载/写操作的 try/catch + setXxxError(toPlatformErrorView(...)) 模式（如第 43–50、56–68、78–86、109–118、169–176、182–189 行）

## 3. 架构与约定
- 错误来源分层
  - 配置错误：baseUrl 为空时抛 NekiroApiError(0, ..., 'CONFIGURATION_ERROR')。
  - 网络错误：fetch 抛错时捕获并转 NekiroApiError(0, ..., 'NETWORK_ERROR')。
  - HTTP 错误：!response.ok 时解析后端 PlatformErrorPayload（支持 {code,message,traceId} 及嵌套 {error: {...}}），提取 x-nek-trace-id 响应头作为 fallback。
  - 响应体错误：非 204 且 JSON 解析失败 → INVALID_RESPONSE。
  - 客户端参数校验：在 buildAgentCard 及相关 helper 中直接 throw new Error('...')，由上层 toPlatformErrorView 兜底转为 CLIENT_ERROR。
- 错误传播路径
  NekiroApiClient → NekiroApiError → App.tsx catch → toPlatformErrorView → PlatformErrorView state → Tab 组件以 errorBox 形式展示。
- 无全局中间件/拦截器：未使用 axios 或 fetch 拦截器，所有错误处理显式写在调用点，保持 MVP 阶段简单透明。
- 无 panic/recover：纯浏览器环境，不使用 throw 控制流以外的机制。

## 4. 开发者应遵循的规则
1. 不要直接向上冒泡 Error：需要用户可见的错误一律通过 NekiroApiClient 发起；若需构造业务错误，优先使用 new NekiroApiError(status, message, code, traceId)。
2. catch 后统一走 toPlatformErrorView：所有 catch (error) 分支必须调用 toPlatformErrorView(error, '友好提示')，禁止自行拼接 UI 字符串。
3. 区分状态码与业务码：HTTP status 用于网络层判断，code（如 CONFIGURATION_ERROR / NETWORK_ERROR / INVALID_RESPONSE / CLIENT_ERROR）用于 UI 分类展示。
4. traceId 透传：从后端 PlatformErrorPayload.error.traceId 或 x-nek-trace-id 头部获取，保留到 PlatformErrorView.traceId，便于联调定位。
5. 参数校验错误归类为 CLIENT_ERROR：在 buildAgentCard 等输入构造函数中 throw new Error(...) 即可，无需手动包装。
6. 避免在组件内 throw：UI 层只消费 PlatformErrorView，不要在 React 组件中直接 throw 导致整棵子树崩溃。