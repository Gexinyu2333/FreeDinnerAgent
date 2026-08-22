# Frontend

本目录存放 FreeDinnerAgent 的 React 前端项目，当前已经初始化为 Vite + React + TypeScript 工程。

详细设计与分阶段实现计划见：

- [../docs/frontend-design-plan.md](../docs/frontend-design-plan.md)

技术栈：

- React
- TypeScript
- Vite
- React Router
- TanStack Query
- Zustand
- Tailwind CSS
- lucide-react
- react-i18next

当前页面与能力：

- 登录与应用壳：提供基础登录态、侧边导航、中文/英文切换。
- Web Chat 对话页面：展示用户主动发起的多轮聊天。
- Settings 页面：配置用户级模型供应商、Agent 参数、thinking、temperature、embedding 和额外 LLM feature。
- Channels 页面：选择 QQ/NapCat 等外部入口，管理连接、endpoint、监听策略、inbox、outbox、审批和发送状态。
- 其它能力页面仍按 [../docs/frontend-design-plan.md](../docs/frontend-design-plan.md) 分阶段补齐。

设计边界：

- Web Chat 和 Channel Adapter 不共用同一个“新建对话”入口。
- Web Chat 由用户输入 query 主动触发 Agent Loop。
- Channel Adapter 由外部消息监听触发 Agent Loop，每个连接默认有一个专用监听/主控会话。
- 当前 MVP 只把 NapCat / OneBot 作为可验证入口；微信、Telegram、Discord、飞书等具体 Adapter 归入高级项。
- NapCat 连接的 URL 类配置统一保存为 endpoint：`message_api`、`event_stream`、`webhook_callback`。本机调试时 webhook URL 建议使用 `?token=` 形式，详细说明见 [../backend/NAPCAT.md](../backend/NAPCAT.md)。

本地运行：

```bash
npm install
npm run dev
```

构建检查：

```bash
npm run build
```
