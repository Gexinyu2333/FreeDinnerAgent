# Frontend

本目录是 FreeDinnerAgent 的 React 前端，使用 Vite + React + TypeScript 实现个人 Agent 控制台。`docs/frontend-design-plan.md` 中的 F1-F10 已完成，后续主要是体验打磨和高级项扩展。

## 技术栈

- React 19、TypeScript、Vite
- React Router
- TanStack Query
- Tailwind CSS
- lucide-react
- react-i18next

## 当前页面

- Auth：注册、登录、退出、登录态守卫
- Web Chat：会话列表、消息列表、发送消息、Agent 回复展示
- Providers：用户级 OpenAI-compatible / Anthropic-compatible provider 配置，支持 chat、embedding 和 extra LLM feature
- Agent Config：系统提示词、模型参数、thinking、temperature、embedding、工具审批和 feature provider 选择
- Memory：Profile Memory、上下文预览、Dreaming insight 查看、应用、拒绝
- Knowledge：文档写入、切片、关键词/向量检索结果
- Market：能力市场、安装、启用、Agent 绑定、System Prompt Template 创建、预览、fork
- Tools：工具审批请求和处理
- Tasks：普通任务、心跳任务、立即运行和运行记录
- Channels：NapCat / OneBot 连接、endpoint、监听策略、inbox、outbox、审批和发送状态
- Workspace：启用 workspace、隔离策略、文件列表、文件读写、受限命令和命令历史
- Logs：开发占位页，后续可扩展为运行日志和审计中心

Channels 在空连接列表中提供“创建连接”按钮；单个群聊策略可选择“@ 或关键词触发”，命中任一条件即可回复。关键词支持中英文逗号和换行分隔。

Web Chat 和 Channel Adapter 是两个入口：Web Chat 由用户在当前会话主动输入触发 Agent Loop；Channel Adapter 由外部消息监听触发，每个连接默认绑定一个专用监听/主控会话。

## 本地运行

```bash
npm install
npm run dev
```

默认连接本地后端：

```text
http://localhost:8080
```

后端地址可通过 Vite 环境变量覆盖：

```bash
VITE_API_BASE_URL=http://localhost:8080 npm run dev
```

## 构建检查

```bash
npm run build
```

本轮已验证 `npm run build` 通过。当前构建有 Vite chunk size warning，不影响运行；后续可以用路由级懒加载继续优化包体积。

## Step F10 验收清单

- 登录 / 注册：可以进入应用壳，退出后回到登录页。
- Provider：可以新增、编辑、删除用户级模型供应商。
- Agent Config：可以保存模型、temperature、thinking、embedding、工具审批和 feature provider 配置。
- Chat：可以创建会话、发送消息、看到回复和错误提示。
- Memory：可以新增 Profile Memory、搜索上下文、处理 Dreaming insight。
- Knowledge：可以写入文档并检索 chunk。
- Task：可以创建任务、心跳任务、立即运行并查看运行记录。
- Workspace：可以启用 workspace、读写文件、列目录、执行白名单命令。
- Channels：可以配置 NapCat / OneBot endpoint、策略、inbox、outbox 和显式发送。
- 移动端：窄屏下通过顶部菜单打开侧边导航，主要表单和列表不遮挡。
- i18n：顶部语言开关可以在中文和英文之间切换。

## NapCat

NapCat 连接的 URL 类配置统一保存为 endpoint：`message_api`、`event_stream`、`webhook_callback`。部署和调试说明见 [../backend/NAPCAT.md](../backend/NAPCAT.md)。

## Channel 回归测试

启动前端后，在已有 Playwright 和 Chrome 的环境运行 `FRONTEND_URL=http://127.0.0.1:5174 node tests/channels-smoke.mjs`。可通过 `PLAYWRIGHT_MODULE` 指定已有 Playwright 模块路径，`BROWSER_CHANNEL` 指定浏览器。测试拦截全部 API，不修改真实数据；覆盖桌面/手机空列表创建、组合触发策略保存与编辑、零限频和关闭策略。
