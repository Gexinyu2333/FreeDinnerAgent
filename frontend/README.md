# Frontend

本目录用于存放 FreeDinnerAgent 的 React 前端项目。

计划技术栈：

- React
- TypeScript
- Vite
- CSS Modules 或 Tailwind CSS

主要页面规划：

- Web Chat 对话页面：展示用户主动发起的多轮聊天、工具调用结果和记忆引用
- Channels 入口页面：选择 QQ/NapCatQQ 等外部入口，管理连接、监听策略、inbox、outbox、审批和专用监听会话
- 记忆管理页面：查看、编辑、删除个人记忆
- 任务页面：展示由助理识别或创建的待办事项
- 设置页面：配置模型、隐私策略和工具开关

设计边界：

- Web Chat 和 Channel Adapter 不共用同一个“新建对话”入口。
- Web Chat 由用户输入 query 主动触发 Agent Loop。
- Channel Adapter 由外部消息监听触发 Agent Loop，每个连接默认有一个专用监听/主控会话。
- 当前 MVP 只把 NapCatQQ / OneBot 作为可验证入口；微信、Telegram、Discord、飞书等具体 Adapter 强制暂缓实现。

后续初始化命令建议：

```bash
npm create vite@latest . -- --template react-ts
npm install
npm run dev
```
