# FreeDinnerAgent 前端设计与分阶段实现计划

## 1. 目标

前端目标不是做一个展示页，而是做一个可用的个人 Agent 控制台。第一版要让用户能完成完整闭环：

- 注册、登录、保持会话。
- 配置自己的模型 Provider、API Key、Agent 参数、额外 LLM 功能开关。
- 在 Web Chat 中创建对话、发送消息、查看 Agent 回复、工具调用、事件和压缩结果。
- 管理 Memory、Knowledge Base、Dreaming Insight。
- 管理 Tool、Skill、System Prompt 等能力市场。
- 创建和运行心跳任务。
- 管理 Channel Adapter，优先 NapCatQQ / OneBot。
- 管理 Workspace，读写文件、执行受限命令、查看命令历史。

这份计划书用于分阶段开发。后续即使换 Codex 账号或换线程，也可以从“阶段拆分”继续推进。

## 2. 技术选型

建议技术栈：

- React
- TypeScript
- Vite
- React Router
- TanStack Query
- Zustand
- Tailwind CSS
- lucide-react
- react-i18next

选择理由：

- React + Vite：启动快，适合课程项目和快速迭代。
- TanStack Query：适合管理后端 API 的 loading、error、cache、mutation。
- Zustand：只保存少量全局 UI 状态，例如 token、当前 workspace 面板状态。
- Tailwind CSS：初期能快速稳定做出统一界面，避免大量手写 CSS。
- lucide-react：按钮、导航和工具图标统一，符合工具型产品的使用习惯。
- react-i18next：从第一版开始支持中文 / English 切换，避免后期页面写完后再补国际化。

## 3. 产品结构

前端采用“控制台 + 主工作区”的结构。

主要布局：

```text
AppShell
  TopBar
  Sidebar
  MainContent
```

Sidebar 一级导航：

- Chat
- Agent
- Providers
- Memory
- Knowledge
- Market
- Tools
- Tasks
- Channels
- Workspace
- Logs

移动端策略：

- Sidebar 折叠为抽屉。
- Chat 页面优先保留消息区和输入框。
- 配置页改为单列分段表单。

## 4. 路由规划

```text
/login
/register

/app/chat
/app/chat/:conversationId
/app/agent
/app/providers
/app/memory
/app/knowledge
/app/market
/app/tools
/app/tasks
/app/channels
/app/channels/:connectionId
/app/workspace
/app/logs
```

默认跳转：

- 未登录访问 `/app/*`：跳转 `/login`。
- 已登录访问 `/login`：跳转 `/app/chat`。
- `/app`：跳转 `/app/chat`。

## 5. 视觉与交互原则

整体风格：

- 工具型控制台，不做营销页。
- 信息密度适中，支持快速扫描。
- 保持灰白底、清晰边框、少量强调色。
- 卡片只用于列表项、面板和弹窗，不做卡片套卡片。

交互原则：

- 危险操作必须有确认：删除 Provider、销毁 Workspace、拒绝/发送 Channel Outbox。
- Tool 调用审批要明显标识风险等级。
- API Key 输入后不回显明文，只显示 `已配置`。
- 所有异步操作要有 loading、error、empty state。

## 6. 前端目录结构

建议初始化后目录：

```text
frontend/
  src/
    app/
      App.tsx
      router.tsx
      providers.tsx

    components/
      layout/
        AppShell.tsx
        Sidebar.tsx
        TopBar.tsx
      ui/
        Button.tsx
        Input.tsx
        Textarea.tsx
        Select.tsx
        Switch.tsx
        Tabs.tsx
        Dialog.tsx
        Toast.tsx
        EmptyState.tsx
        LoadingState.tsx
        Badge.tsx

    features/
      auth/
        api.ts
        pages/
        types.ts
      chat/
        api.ts
        pages/
        components/
        types.ts
      agent/
      providers/
      memory/
      knowledge/
      market/
      tools/
      tasks/
      channels/
      workspace/
      logs/

    lib/
      apiClient.ts
      authToken.ts
      queryClient.ts
      errors.ts
      format.ts
      i18n.ts

    locales/
      zh-CN.json
      en-US.json

    styles/
      globals.css

    main.tsx
```

约定：

- `features/<domain>/api.ts` 只放该领域请求函数。
- `features/<domain>/types.ts` 放后端 DTO 类型。
- 页面组件放 `pages/`。
- 可复用业务组件放 `components/`。
- 通用 UI 组件放 `components/ui/`。
- 所有前端固定文案走 i18n key，不在组件中直接散落中文或英文。

## 7. API Client 约定

统一响应格式：

```ts
type ApiResponse<T> = {
  data: T | null;
  error: null | {
    code: string;
    message: string;
  };
};
```

`apiClient` 负责：

- 自动拼接 `/api/v1`。
- 自动带上 `Authorization: Bearer <token>`。
- 统一解析 `{ data, error }`。
- 遇到 401 清理 token 并跳转登录页。
- 将后端错误转成前端可展示错误。

开发期代理：

```ts
// vite.config.ts
server: {
  proxy: {
    "/api": "http://localhost:8080"
  }
}
```

## 8. 页面设计

### 8.1 登录 / 注册

页面：

- `/login`
- `/register`

接口：

- `POST /api/v1/auth/login`
- `POST /api/v1/auth/register`
- `GET /api/v1/me`

功能：

- 用户名、密码登录。
- 注册后直接保存 token 并进入 `/app/chat`。
- 登录失败展示后端错误。
- token 存储到 localStorage。

### 8.2 Chat

页面：

- `/app/chat`
- `/app/chat/:conversationId`

接口：

- `POST /api/v1/conversations`
- `GET /api/v1/conversations`
- `GET /api/v1/conversations/:conversation_id/messages`
- `POST /api/v1/conversations/:conversation_id/messages`
- `POST /api/v1/conversations/:conversation_id/compress`
- `GET /api/v1/conversations/:conversation_id/agent-events`
- `GET /api/v1/conversations/:conversation_id/agent-turns/:turn_id`
- `GET /api/v1/conversations/:conversation_id/agent-turns/:turn_id/loop-steps`
- `GET /api/v1/conversations/:conversation_id/tool-calls`

界面结构：

- 左侧：会话列表、创建会话。
- 中间：消息流。
- 右侧：Agent Inspector，可切换事件、工具调用、上下文压缩、记忆引用。
- 底部：输入框、发送按钮。

第一版 Chat 行为：

- 发送消息后刷新消息列表。
- 暂不做流式输出，先用 loading 状态。
- Agent events 可手动刷新或定时轮询。
- 手动压缩当前会话前几轮。

### 8.3 Agent 配置

页面：

- `/app/agent`

接口：

- `GET /api/v1/me/agent-config`
- `PATCH /api/v1/me/agent-config`
- `GET /api/v1/me/model-providers`

表单分区：

- 基础：名称、系统提示词、默认 Provider。
- 生成参数：temperature、thinking enabled、thinking effort、thinking budget tokens。
- 上下文：max context tokens、max loop steps。
- 记忆：memory enabled、semantic memory enabled、dreaming enabled。
- 工具：tool use enabled、tool approval policy。
- Embedding：embedding enabled、embedding cost policy。
- 额外 LLM 功能：auto compression、dreaming、curator，每项可选择 provider、model override、temperature。

### 8.4 Providers

页面：

- `/app/providers`

接口：

- `GET /api/v1/me/model-providers`
- `POST /api/v1/me/model-providers`
- `PATCH /api/v1/me/model-providers/:provider_id`
- `DELETE /api/v1/me/model-providers/:provider_id`

功能：

- 新增 OpenAI-compatible / Anthropic-compatible Provider。
- 分别配置 chat base URL、embedding base URL、chat API key、embedding API key。
- 设置默认 chat model、默认 embedding model。
- 显示是否已配置 API Key。
- 删除前确认。

### 8.5 Memory

页面：

- `/app/memory`

接口：

- `GET /api/v1/memory-types`
- `POST /api/v1/profile-memories`
- `GET /api/v1/profile-memories`
- `GET /api/v1/profile-memory-search`
- `GET /api/v1/memory-context`
- `GET /api/v1/dreaming-insights`
- `POST /api/v1/dreaming-insights/:insight_id/apply`
- `POST /api/v1/dreaming-insights/:insight_id/reject`

Tabs：

- Profile Memory
- Memory Context
- Dreaming Insights

功能：

- 手动新增画像记忆。
- 按关键词搜索画像记忆。
- 查看当前 query 会注入哪些 memory context。
- 应用或拒绝 dreaming insight。

### 8.6 Knowledge

页面：

- `/app/knowledge`

接口：

- `POST /api/v1/knowledge-documents`
- `GET /api/v1/knowledge-documents`
- `GET /api/v1/knowledge-search`

功能：

- 粘贴文本创建知识文档。
- 选择 visibility：private / public。
- 显示 embedding status。
- 搜索知识库，展示 keyword 或 vector 模式。

### 8.7 Market

页面：

- `/app/market`

接口：

- `GET /api/v1/marketplace-items`
- `POST /api/v1/marketplace-items/:item_id/install`
- `POST /api/v1/marketplace-items/:item_id/rate`
- `POST /api/v1/capability-installs/:install_id/enable`
- `POST /api/v1/capability-installs/:install_id/disable`
- `POST /api/v1/agent-capability-bindings`
- `POST /api/v1/agent-capability-bindings/:binding_id/enable`
- `POST /api/v1/agent-capability-bindings/:binding_id/disable`
- `POST /api/v1/system-prompt-templates`
- `POST /api/v1/system-prompt-templates/preview`
- `POST /api/v1/system-prompt-template-versions/:version_id/fork`

Tabs：

- Tools
- Skills
- MCP
- Channel Adapters
- System Prompts

功能：

- 浏览公有和私有能力。
- 安装/启用/禁用能力。
- 绑定能力到当前 Agent。
- 创建、预览、fork 系统提示词模板。

### 8.8 Tools

页面：

- `/app/tools`

接口：

- `GET /api/v1/tools`
- `POST /api/v1/tools/:tool_name/call`
- `GET /api/v1/tool-calls/:tool_call_id`
- `POST /api/v1/tool-approval-requests/:approval_id/approve`
- `POST /api/v1/tool-approval-requests/:approval_id/reject`

功能：

- 查看 Tool Registry。
- 显示 namespace、permission level、approval requirement。
- 手动测试工具调用。
- 查看等待审批的工具请求。

### 8.9 Tasks

页面：

- `/app/tasks`

接口：

- `POST /api/v1/scheduled-agent-jobs`
- `GET /api/v1/scheduled-agent-jobs`
- `GET /api/v1/scheduled-agent-job-templates`
- `PATCH /api/v1/scheduled-agent-jobs/:job_id`
- `POST /api/v1/scheduled-agent-jobs/:job_id/pause`
- `POST /api/v1/scheduled-agent-jobs/:job_id/resume`
- `DELETE /api/v1/scheduled-agent-jobs/:job_id`
- `GET /api/v1/scheduled-agent-jobs/:job_id/runs`
- `POST /api/v1/scheduled-agent-jobs/:job_id/run-now`
- `GET /api/v1/scheduled-agent-job-runs/:run_id`

功能：

- 使用模板创建每日简报、每周回顾、跟进监控。
- 自定义 schedule。
- 暂停、恢复、删除。
- run now。
- 查看运行记录。

### 8.10 Channels

页面：

- `/app/channels`
- `/app/channels/:connectionId`

接口：

- `GET /api/v1/channel-providers`
- `POST /api/v1/me/channel-connections`
- `GET /api/v1/me/channel-connections`
- `PATCH /api/v1/me/channel-connections/:connection_id/policies`
- `GET /api/v1/me/channel-connections/:connection_id/external-conversations`
- `GET /api/v1/me/channel-connections/:connection_id/inbox-events`
- `GET /api/v1/me/channel-connections/:connection_id/outbox-messages`
- `POST /api/v1/channel-outbox-messages/:outbox_id/approve`
- `POST /api/v1/channel-outbox-messages/:outbox_id/cancel`
- `POST /api/v1/channel-outbox-messages/:outbox_id/send`

设计重点：

- Channel 是独立入口，不混入 Web Chat 的新建会话。
- 每个 adapter connection 默认绑定一个专用监听会话。
- NapCatQQ / OneBot 是第一版唯一重点。

功能：

- 创建 QQ 连接。
- 配置 webhook / access token / endpoint。
- 配置私聊、群聊、@ 触发、quiet hours、rate limits。
- 查看 inbox events。
- 查看 outbox drafts。
- 手动 approve / cancel / send。

### 8.11 Workspace

页面：

- `/app/workspace`

接口：

- `GET /api/v1/me/workspace`
- `POST /api/v1/me/workspace`
- `PATCH /api/v1/me/workspace`
- `DELETE /api/v1/me/workspace`
- `GET /api/v1/me/workspace/files`
- `GET /api/v1/me/workspace/files/content`
- `PUT /api/v1/me/workspace/files/content`
- `POST /api/v1/me/workspace/commands`
- `GET /api/v1/me/workspace/commands`

功能：

- 启用 / 销毁 workspace。
- 查看路径、配额、策略。
- 文件树。
- 文件内容查看和编辑。
- 执行受限命令。
- 查看命令运行历史。

第一版限制：

- 不做真实终端模拟器。
- 命令输出用普通文本块展示。
- 强隔离属于高级项，前端只展示策略和警告。

### 8.12 Logs

页面：

- `/app/logs`

第一版可以只做聚合入口：

- 最近 Agent events。
- 最近 Tool calls。
- 最近 Scheduled job runs。
- 最近 Channel inbox/outbox。

如果后端没有统一 logs 接口，可以在第一版隐藏该页，等其它页能跑通后再做。

## 9. 状态管理

LocalStorage：

- `access_token`
- `refresh_token`
- `last_conversation_id`

Zustand：

- 当前用户。
- Sidebar 折叠状态。
- 当前主题。

TanStack Query：

- 所有 API 数据。
- mutation 后通过 invalidateQueries 刷新。

## 10. 错误与空状态

统一错误展示：

- 表单错误：字段附近展示。
- 页面请求错误：页面顶部 inline alert。
- mutation 错误：toast。
- 401：清理 token 并跳转登录。

空状态：

- 没有 conversation：显示创建会话按钮。
- 没有 provider：提示先配置模型。
- 没有 memory：提示可以手动添加或通过对话积累。
- 没有 channel connection：提示创建 NapCatQQ 连接。
- workspace 未启用：展示启用按钮和风险说明。

## 11. 国际化

第一版就支持中文 / English 切换。

实现建议：

- 使用 `react-i18next`。
- 默认语言为 `zh-CN`。
- 支持 `zh-CN` 和 `en-US`。
- 用户选择语言后写入 localStorage，例如 `freedinner.locale`。
- TopBar 右侧提供语言切换菜单。
- Sidebar、按钮、表单 label、placeholder、empty state、loading state、toast 固定文案都走翻译 key。
- 后端返回的 `error.message` 第一版可以原样展示；前端自己定义的错误说明走 i18n。
- 日期、时间、数字格式化封装在 `lib/format.ts`，根据当前语言选择格式。

文件建议：

```text
src/lib/i18n.ts
src/locales/zh-CN.json
src/locales/en-US.json
```

翻译 key 命名建议：

```text
nav.chat
nav.agent
auth.login.title
auth.login.submit
chat.input.placeholder
common.save
common.cancel
common.loading
common.empty
```

## 12. 分阶段实现计划

### Step F1：前端工程初始化

目标：

- 初始化 Vite React TS。
- 安装依赖。
- 配置 Tailwind、React Router、TanStack Query、react-i18next。
- 建立 AppShell、Sidebar、TopBar、基础 UI 组件。
- 建立 `src/lib/i18n.ts`、`src/locales/zh-CN.json`、`src/locales/en-US.json`。
- TopBar 提供中文 / English 切换，语言选择保存到 localStorage。
- Sidebar、基础按钮、空状态、loading state 使用 i18n key。
- 配置 `/api` 代理。

验收：

- `npm run dev` 能启动。
- `/login` 和 `/app/chat` 页面能打开。
- 未登录访问 `/app/chat` 会跳到 `/login`。
- TopBar 可以切换中文 / English，刷新后仍保留选择。

### Step F2：Auth MVP

目标：

- 实现登录、注册、当前用户接口。
- token 保存和自动带 Authorization。
- 登录后进入 `/app/chat`。
- 退出登录。

验收：

- 可用测试账号登录。
- 刷新页面后仍保持登录。
- token 失效时回登录页。

### Step F3：Chat MVP

目标：

- 会话列表。
- 创建会话。
- 消息列表。
- 发送消息。
- 显示 assistant 回复。

验收：

- 用户可以从空状态创建会话并发送第一条消息。
- 后端返回消息后页面刷新展示。

### Step F4：Provider + Agent Config

目标：

- Provider 列表、新增、编辑、删除。
- Agent 配置读取和保存。
- thinking、temperature、tool approval policy、embedding 开关、feature provider policy。

验收：

- 用户能在前端配置 LongCat chat provider 和 SiliconFlow embedding provider。
- Agent 配置保存后刷新仍然存在。

### Step F5：Memory + Knowledge

目标：

- Profile Memory 增加、列表、搜索。
- Memory Context 查看。
- Dreaming Insight 应用/拒绝。
- Knowledge 文档创建、列表、搜索。

验收：

- 可以手动添加记忆。
- 可以上传文本知识并搜索。
- embedding 开关打开时能看到 vector 模式；关闭时 keyword 模式。

### Step F6：Tools + Market

目标：

- Tool 列表。
- Tool 调用审批列表。
- 手动 approve / reject。
- Market 浏览、安装、启用、绑定。
- System Prompt 创建、预览、fork。

验收：

- 用户能看到内置工具。
- 用户能通过市场绑定工具或系统提示词到 Agent。

### Step F7：Scheduled Jobs

目标：

- 模板列表。
- 创建心跳任务。
- 任务列表。
- 暂停、恢复、删除、run now。
- 运行记录详情。

验收：

- 能创建每日简报并手动运行。
- 运行记录可见。

### Step F8：Channels

目标：

- Channel provider 列表。
- 创建 NapCatQQ 连接。
- 查看连接列表。
- 配置 policy。
- 查看 inbox/outbox。
- approve/cancel/send outbox。

验收：

- 前端能完成 QQ 连接配置。
- 能看到 webhook 收到的 inbox event。
- 能处理 outbox 草稿。

### Step F9：Workspace

目标：

- 启用 workspace。
- 查看状态和策略。
- 文件列表、读文件、写文件。
- 执行受限命令。
- 查看命令历史。

验收：

- 用户能在前端创建 workspace。
- Agent能调用写入一个文件并读取。
- Agent能执行一个处理文件等的命令，用户允许命令执行（三级tool_call的执行的设定）并看到输出。

### Step F10：前端整理与验收

目标：

- 统一 loading、empty、error、toast。
- 统一表单样式。
- 移动端基本可用。
- 补 README。

验收：

- `npm run build` 成功。
- 用浏览器手动跑通核心链路：登录、provider、agent config、chat、memory、knowledge、task、workspace。

## 13. 推荐每轮 /goal 写法

第一轮：

```text
/goal 按 docs/frontend-design-plan.md 的 Step F1 初始化前端工程，完成 Vite React TS、路由、AppShell、基础 UI、API client、react-i18next 中英文切换和 /api 代理，并验证 npm run dev 和 npm run build。
```

第二轮：

```text
/goal 按 docs/frontend-design-plan.md 的 Step F2 实现 Auth MVP，包括登录、注册、当前用户、token 保存、鉴权路由和退出登录，并验证可与后端接口联通。
```

第三轮：

```text
/goal 按 docs/frontend-design-plan.md 的 Step F3 实现 Chat MVP，包括会话列表、创建会话、消息列表、发送消息和 assistant 回复展示。
```

后续依次把 Step 编号替换成 F4、F5、F6、F7、F8、F9、F10。

## 14. 需要暂缓的前端内容

这些内容不影响第一版前端闭环：

- 真正流式输出。
- 复杂图形化 Agent Trace。
- Monaco Editor。
- 真终端模拟器。
- 多主题系统。
- 离线缓存。
- 复杂权限管理后台。

第一版先把“能配置、能聊天、能看记忆、能用工具、能创建任务、能接 QQ、能用 workspace”跑通。
