# FreeDinnerAgent

FreeDinnerAgent 是一个面向个人场景的 AI 个人助理全栈 Web 应用。系统通过前端对话界面、Go 后端服务和 PostgreSQL 数据库，为用户提供可记忆、可检索、可调用工具的智能助理体验。

项目目标不是单次问答机器人，而是一个具备分层记忆和工具调用能力的个人 Agent。系统参考 Hermes Agent 的记忆思想，将记忆拆分为 Working、Profile、Episodic、Procedural、Semantic 五类，并通过用户隔离、按需检索和上下文压缩提供更连续的个人助理体验。

## 课题要求

课题名称：AI 个人助理

产出要求：完成一个可运行、可体验的完整 Web 产品，包含前端对话界面、后台服务及必要的数据存储，并提交完整代码和运行说明。

项目重点考虑：

- 记忆系统设计：记忆写入时机、存储结构、检索策略
- 多轮对话上下文管理与 Token 控制
- 工具调用：Function Calling 的注册、路由与稳定性处理
- Agent Loop：Bounded ReAct 推理、执行、观察与最终回复
- 心跳任务：定时提醒、每日简报、每周回顾和跟进监控
- 多渠道入口：当前以 NapCatQQ / OneBot 作为首个外部监听入口；微信、Telegram、Discord、飞书等具体 Adapter 强制暂缓，只保留通用抽象
- LLM 输出不稳定时的兜底、重试与降级机制
- 用户数据隔离与隐私保护
- 后台服务架构与接口设计
- 系统可扩展性：新增工具或记忆类型
- 测试与代码质量

## 技术栈

- 前端：React、TypeScript、Vite
- 后端：Go、Gin、PostgreSQL Driver
- 数据库：PostgreSQL
- AI 能力：LLM API、Embedding API、Tool Use / Function Calling、分层记忆检索
- 部署方式：本地开发优先，后续可扩展为 Docker Compose 部署

## 目录结构

```text
FreeDinnerAgent/
  frontend/               # React 前端项目，提供 Web Chat、Channels 入口和个人助理工作台
  backend/                # Go 后端服务，负责 Agent 编排、接口、工具调用和数据访问
  database/               # PostgreSQL 表结构、迁移脚本、初始化数据
  docs/                   # 总体设计、系统架构、接口设计和开发说明
  LICENSE
  README.md
```

## 核心功能规划

1. 用户与会话
   - 支持用户名、密码登录
   - 支持用户身份隔离
   - 登录后查看自己的 Agent 配置、模型配置、对话记录、记忆和任务
   - 支持多会话对话记录
   - 支持会话标题、创建时间和更新时间

2. AI 对话
   - Web Chat 是用户主动发起的对话，只有用户在当前会话输入 query 时才触发 Agent Loop
   - 用户可配置自己的 OpenAI 或 Anthropic API Key
   - 后端按当前用户的默认 Agent 配置和模型供应商配置调用 LLM
   - 后端组装上下文、相关记忆和可用工具
   - 用户可手动整理当前对话，将早期轮次压缩成摘要
   - 后端使用有最大步数限制的 ReAct loop，记录推理摘要、动作、观察和最终回复
   - LLM 返回自然语言回复或工具调用计划
   - 后端保存用户消息、助手回复和工具调用日志

3. 记忆系统
   - Working Memory：保存当前会话临时状态和约束
   - Profile Memory：保存用户偏好、习惯、事实、关系和目标
   - Episodic Memory：保存完整交互轨迹、工具调用和成功失败状态
   - Procedural Memory：沉淀可复用 ReAct 技能流程，支持私有技能和公共技能
   - Semantic Memory：保存外部文档切片，支持私有知识库和公共知识库；数据库支持 pgvector，用户可按成本选择是否开启 embedding 向量检索
   - Dreaming：后台离线整理记忆，合并重复内容、归档低价值历史、提炼技能候选

4. 工具调用
   - 日程提醒
   - 任务跟踪
   - 内容摘要
   - 信息检索
   - 心跳任务：每日简报、每周回顾、跟进监控、定时提醒
   - 能力市场：MCP、Skills、Tools、Knowledge Base 支持私有/公共和安装到个人 Agent
   - 多渠道入口：先接入 NapCatQQ / OneBot；Channel Adapter 作为独立监听入口，不混入普通 Web Chat 新建对话
   - 每个 Channel connection 默认有一个专用监听/主控会话，用于展示 inbound/outbox、运行日志、审批和人工介入记录
   - 微信、Telegram、Discord、飞书等具体 Adapter 与生产级 sandbox 强隔离项一样，当前 MVP 强制暂缓实现
   - 后续可继续扩展更多工具

5. 稳定性处理
   - 工具参数校验
   - LLM 输出解析失败时先修复，再按用户配置重试
   - Agent loop 达到最大步数时生成保守兜底回复
   - 上下文超限时自动压缩早期轮次并减少注入记忆
   - 工具执行失败时返回可理解的降级回复
   - 记录 loop step、输出校验、降级事件和工具调用状态，方便排查

## 部署与运行要求

本项目计划使用以下环境运行：

- Node.js 20+
- npm 10+ 或 pnpm 9+
- Go 1.26+
- PostgreSQL 17+ 推荐，macOS Homebrew 环境下 pgvector 默认适配 PostgreSQL 17/18
- pgvector，用于 Semantic Memory / RAG 向量检索
- 用户自备 OpenAI 或 Anthropic API Key

后端需要配置环境变量：

```bash
APP_ENV=development
SERVER_PORT=8080
DATABASE_URL=postgres://freedinner:freedinner@localhost:5432/freedinner_agent?sslmode=disable
JWT_SECRET=change_me_to_a_long_random_string
API_KEY_ENCRYPTION_KEY=32_byte_base64_or_hex_key
```

OpenAI、Anthropic 的 API Key 不作为全局环境变量配置，而是在用户登录后进入设置页，由用户自行添加并保存到自己的模型供应商配置中。后端加密存储 API Key，接口不返回明文。

## 本地运行方式

当前仓库先完成项目结构与设计文档。完整代码实现后，计划按以下步骤运行：

1. 安装 PostgreSQL 与 pgvector

macOS Homebrew 示例：

```bash
brew install postgresql@17
brew install pgvector
brew services start postgresql@17
```

如果你已经安装并启动 PostgreSQL，只需要确认 pgvector 已安装即可。

2. 初始化数据库

```bash
createuser -P freedinner
createdb -O freedinner freedinner_agent
psql -d freedinner_agent -c "CREATE EXTENSION IF NOT EXISTS vector;"
psql -U freedinner -d freedinner_agent -f database/init.sql
```

`database/init.sql` 会执行 `CREATE EXTENSION IF NOT EXISTS vector;`，并创建 `episodes`、`skills`、`knowledge_chunks` 的 `embedding vector(1024)` 字段和向量索引。默认维度对应 `BAAI/bge-m3` / `Pro/BAAI/bge-m3`。数据库层默认具备向量能力，但用户是否生成和使用 embedding 由个人 Agent 配置中的 `embedding_enabled` 和 `embedding_cost_policy` 控制。

如果初始化时提示 `CREATE EXTENSION vector` 权限不足，说明当前执行用户不是 PostgreSQL 管理员。先用本机 PostgreSQL 管理员账号执行：

```bash
psql -d freedinner_agent -c "CREATE EXTENSION IF NOT EXISTS vector;"
psql -U freedinner -d freedinner_agent -f database/init.sql
```

验证表结构：

```bash
psql -U freedinner -d freedinner_agent -c "\dt"
psql -U freedinner -d freedinner_agent -c "\dx"
```

3. 启动后端

```bash
cd backend
go mod tidy
go run ./cmd/server
```

4. 启动前端

```bash
cd frontend
npm install
npm run dev
```

5. 浏览器访问

```text
http://localhost:5173
```

## 设计文档

- [总体设计架构](docs/architecture.md)
- [记忆系统设计](docs/memory-design.md)
- [多轮上下文与 Token 控制设计](docs/context-token-design.md)
- [Agent Loop 与可靠性设计](docs/agent-loop-reliability-design.md)
- [工具调用设计](docs/tool-calling-design.md)
- [心跳任务与主动助理设计](docs/scheduled-agent-jobs-design.md)
- [多渠道入口设计](docs/channel-adapter-design.md)
- [能力市场设计](docs/capability-market-design.md)
- [用户级 Workspace Sandbox 设计](docs/workspace-sandbox-design.md)
- [接口设计草案](docs/api-design.md)

## 当前阶段

当前版本已完成项目目录规划、Hermes-like 分层记忆数据库草案和系统设计说明。下一阶段将初始化 React 前端和 Go 后端工程，并实现最小可运行版本：

- 用户发送消息
- 用户名密码登录
- 用户配置自己的 OpenAI 或 Anthropic API Key
- 查看自己的 Agent 配置和对话记录
- 后端生成 AI 回复
- 保存会话消息
- 创建和查看心跳任务：每日简报、每周回顾、跟进监控
- 支持心跳任务立即运行并写入运行记录
- 接入 NapCatQQ 私聊和群聊 @ 触发，Channel 入口与普通 Web Chat 入口分离
- 微信、Telegram、Discord、飞书等具体 Adapter 暂缓实现，只保留通用 Channel Adapter 抽象
- 写入和检索 Profile Memory
- 保存 Episodic Memory
- 支持 Semantic Memory 文档切片
- 通过 `memory_type_definitions` 扩展新的画像记忆类型
