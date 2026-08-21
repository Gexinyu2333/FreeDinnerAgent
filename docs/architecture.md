# FreeDinnerAgent 总体设计架构

## 1. 项目定位

FreeDinnerAgent 是一个具备分层记忆能力的 AI 个人助理。用户可以通过自然语言与助理交流，系统会在合适的时机沉淀用户偏好、对话经历、可复用技能和外部知识，并在后续对话中主动检索相关记忆，为用户提供更个性化、更连续的回复。

## 2. 总体架构

```text
React 前端
  登录 / 对话界面 / 记忆管理 / 任务管理 / 已安排任务 / 渠道连接 / Agent 设置 / Workspace
      |
      | HTTP / JSON
      v
Go 后端
  API / Channel Adapter / Scheduler / Agent Harness / ReAct Loop / MemoryManager / Tool Use / Workspace Sandbox / 数据访问
      |
      +---- LLM API
      +---- Embedding API
      |
      v
PostgreSQL
  用户 / 模型配置 / Agent 配置 / 系统提示词模板 / 会话 / 消息 / 摘要 / 上下文日志 / 四层记忆 / 任务 / 心跳任务 / 渠道事件 / Workspace / 工具调用日志
```

系统分为四层：

- 表现层：React 前端负责用户交互、消息展示和记忆管理。
- 服务层：Go 后端提供 REST API，处理登录鉴权、用户模型配置、系统提示词模板、会话、消息、记忆、任务、心跳任务、Workspace 和外部渠道接入。
- Agent 层：负责上下文组装、Bounded ReAct loop、LLM 输出校验、分层记忆检索、工具调用决策、Workspace 沙箱边界、失败兜底和响应处理。
- 数据层：PostgreSQL 保存用户数据、对话数据、系统提示词模板、四层记忆、任务、心跳任务、Workspace、渠道事件和工具调用日志。

## 3. 前端设计

前端核心页面：

- 对话工作台：左侧会话列表，中间聊天窗口，右侧显示本次使用到的记忆和工具调用结果。
- 对话整理：用户可手动触发“整理当前对话”，将前几轮压缩成摘要，后续上下文加载摘要而不是早期原文。
- 记忆管理：按 Working、Profile、Episodic、Procedural、Semantic 分层查看，支持编辑、归档、删除和共享设置。
- 能力市场：展示 MCP、Skills、Tools、Knowledge Base、Channel Adapter 和 System Prompt Template，支持公共/私有、安装、启用到个人 Agent。
- 任务管理：展示由对话生成的待办事项，支持状态更新。
- 已安排任务：展示每日简报、每周回顾、跟进监控、提醒等心跳任务，支持启停、立即运行和查看运行记录。
- 渠道连接：配置 NapCatQQ、Telegram、Discord、飞书等外部聊天入口，设置私聊/群聊触发策略。
- 知识库页面：上传或录入文档，生成 Semantic Memory，用于 RAG 检索，可选择私有或公共。
- 登录页面：使用用户名和密码登录，进入后只能查看自己的配置、会话、记忆和任务。
- 设置页面：配置 OpenAI 或 Anthropic API Key、默认模型、系统提示词模板、Embedding 成本开关、Workspace 开关、工具开关和隐私选项。

前端核心状态：

- 当前用户信息
- 当前会话
- 消息列表
- 正在生成状态
- 本轮命中的记忆
- 工具调用状态
- 当前用户的 Agent 配置和模型供应商配置
- 当前用户的系统提示词模板、模板版本和变量配置
- 当前用户的心跳任务配置和最近运行状态
- 当前用户的渠道连接、外部会话映射和群聊触发策略
- 当前用户的 Workspace 状态、配额和最近文件/命令事件

## 4. 后端模块设计

建议 Go 后端使用以下模块划分：

```text
backend/
  cmd/server/             # 程序入口
  internal/api/           # HTTP handler 和路由
  internal/auth/          # 用户注册、登录、密码哈希、JWT/session
  internal/agent/         # Agent 编排、Bounded ReAct loop、输出校验和兜底策略
  internal/channel/       # QQ、Telegram、Discord、飞书等外部聊天入口适配器
  internal/scheduler/     # 心跳任务调度、到期扫描、运行记录和补偿执行
  internal/harness/       # Agent Turn、事件流、取消、审批和执行生命周期
  internal/memory/        # MemoryManager、记忆路由、检索、压缩、更新
  internal/marketplace/   # MCP、Skills、Tools、Knowledge Base、系统提示词模板的市场、安装和 Agent 绑定
  internal/prompt/        # 系统提示词模板、版本、变量渲染和 Agent 绑定
  internal/workspace/     # 用户级 Workspace、文件操作、CLI 沙箱和资源配额
  internal/tool/          # 工具注册与执行
  internal/store/         # 数据库访问层
  internal/config/        # 配置加载
  internal/llm/           # LLM 客户端
  internal/provider/      # 用户级 OpenAI / Anthropic 配置读取和 API Key 解密
  internal/embedding/     # Embedding 客户端
  internal/curator/       # 后台记忆复盘、Dreaming、压缩、技能沉淀
```

关键流程：

1. 用户登录后，前端携带访问令牌请求后端。
2. 前端发送用户消息到后端。
3. 后端根据令牌解析 `user_id`，读取该用户默认 Agent 配置和模型供应商配置。
4. 如果用户没有配置 OpenAI 或 Anthropic API Key，返回配置缺失错误。
5. 后端保存用户消息。
6. Harness 创建 `agent_turns`，并通过 `agent_events` 记录执行过程。
7. Memory Router 判断本轮是否需要启用情景记忆、程序记忆或语义知识库。
8. Retrieval Engine 按层检索 Working、Profile、Episodic、Procedural、Semantic Memory。
9. Memory Compressor 对召回片段压缩和截断，控制 Token。
10. Context Builder 加载系统提示词模板版本，渲染变量，并拼入用户覆盖提示词、记忆策略、工具策略、渠道策略和 Workspace 策略。
11. 后端进入 Bounded ReAct loop，按 Reason、Act、Observe、Finalize 推进。
12. LLM 输出先进入 Validator，校验 JSON、工具参数、安全边界、Workspace 边界和最终回复声明。
13. 如果需要工具调用，Tool Router 选择候选工具，Tool Executor 校验参数、处理审批、超时、重试和降级。
14. 如果需要文件或 CLI 操作，Workspace Tool 只能在当前用户启用的 sandbox 内执行。
15. 如果 LLM 输出不稳定，Fallback Manager 先修复，再重试，必要时压缩上下文、禁用异常工具或生成保守兜底回复。
16. 后端保存助手回复、loop step、输出校验、降级事件、工具调用日志、Workspace 事件和本轮使用到的记忆召回记录。
17. Curator 异步复盘本轮交互，更新情景记忆、用户画像记忆、程序技能或语义向量。
18. Dreaming Engine 在空闲或定时阶段离线整理记忆，合并重复记忆、归档低价值记忆、提炼技能候选。
19. 前端展示最终结果和事件流。

心跳任务流程：

1. 用户在“已安排任务”页面选择每日简报、每周回顾、跟进监控或自定义模板。
2. 后端写入 `scheduled_agent_jobs`，保存周期、时区、Prompt 模板、上下文策略和工具权限策略。
3. Scheduler 定期扫描 `next_run_at` 到期的任务。
4. 每次触发写入 `scheduled_agent_job_runs`，并创建一次由系统触发的 Agent Turn。
5. Agent Turn 复用同一套上下文构建、记忆检索、工具调用、输出校验和兜底机制。
6. 运行结果写入会话、运行记录和必要的 Episodic Memory。
7. Scheduler 计算下一次 `next_run_at`，连续失败过多时暂停任务并提示用户处理。

多渠道入口流程：

1. 用户在“渠道连接”页面安装 NapCatQQ Channel Adapter。
2. 用户配置 NapCat endpoint、token、secret，并绑定到自己的 Agent。
3. NapCatQQ 收到私聊或群聊事件后上报到 Channel API。
4. 后端写入 `channel_inbox_events`，按 `channel_policies` 判断是否触发 Agent。
5. 如果触发，后端查找或创建 `external_conversations`，映射到本地 `conversations`。
6. 后端写入消息并创建 `agent_turns`。
7. Agent 生成回复后写入 `channel_outbox_messages`。
8. Channel Adapter 调用 NapCatQQ 发送消息，并记录发送结果。

## 5. 记忆系统设计

FreeDinnerAgent 采用类似 Hermes Agent 的分层记忆架构，但将正式存储统一落在 PostgreSQL 中，并通过 `user_id` 和 `conversation_id` 实现用户隔离与会话隔离。

四层记忆：

- Working Memory：会话级工作记忆，保存当前任务目标、临时约束、工具中间结果，每轮 Prompt 必带。
- Profile Memory：长期用户画像记忆，保存偏好、习惯、事实、关系、目标和待办。
- Episodic Memory：情景记忆，保存一轮完整交互轨迹、工具调用、成功失败状态和任务标签。
- Procedural Memory：程序/技能记忆，保存可复用 ReAct 流程、工具序列、输出模板和失败降级策略，可选择私有或公共。
- Semantic Memory：语义知识库，保存外部文档切片，并使用 PostgreSQL + pgvector 保存 embedding；用户可按成本选择是否开启向量检索，可选择私有或公共。
- Dreaming：后台离线记忆整理机制，用于重放历史、合并重复、提炼技能、归档低价值内容。

详细设计见 [记忆系统设计](memory-design.md)。

## 6. 上下文管理与 Token 控制

上下文由五部分组成：

1. 系统提示词：定义助手角色、输出规范、安全边界。
2. Working/Profile Memory：用户当前偏好、临时约束和长期画像。
3. Procedural Memory：匹配到的可复用技能流程。
4. Episodic/Semantic Memory：历史经验和外部知识片段。
5. 短期上下文与工具结果：当前会话最近消息、本轮工具调用结果。

Token 控制策略：

- 最近消息优先保留。
- 过长历史会话生成摘要后再注入上下文。
- 每次只注入最相关的少量分层记忆。
- Procedural Memory 根据任务复杂度选择 light、standard、full 三种加载模式。
- 工具结果先结构化压缩，再交给 LLM。
- `conversation_summaries` 保存被压缩的早期对话摘要。
- `conversation_compression_jobs` 记录自动或手动触发的对话压缩任务。
- `context_build_logs` 和 `context_build_items` 记录本轮 Prompt 组成，用于上下文体检报告。
- `llm_usage_logs` 记录模型调用 Token、耗时和失败状态。

详细设计见 [多轮上下文与 Token 控制设计](context-token-design.md)。

系统提示词不只是一段固定文本，而是可以作为能力市场中的 System Prompt Template 进行管理。用户可以选择系统内置模板、公共模板或自己的私有模板，并锁定具体版本。Context Builder 会加载模板版本、渲染变量、拼入用户自定义覆盖内容，再和记忆、工具、MCP、Skills、渠道策略以及 Workspace 策略组合成本轮上下文。

系统提示词模板属于能力市场的一类能力，详细设计见 [能力市场设计](capability-market-design.md)。

## 7. Agent Loop 与可靠性设计

Agent loop 使用 Bounded ReAct：

```text
Reason -> Validate -> Act -> Observe -> Reason -> ... -> Finalize
```

其中 Reason 阶段只保存面向审计的 `thought_summary`，不保存完整隐藏思维链。每一次用户输入都会创建一条 `agent_turns`，每一步写入 `agent_loop_steps`，LLM 输出校验写入 `llm_output_validations`，修复、重试和降级写入 `agent_fallback_events`。

可靠性策略：

- `max_loop_steps` 控制最大 ReAct 步数，避免无限循环。
- `llm_retry_limit` 控制模型输出失败后的有限重试。
- 输出格式错误时先走 repair，再决定是否重试。
- 上下文过长时压缩早期轮次、降低技能披露级别、减少语义片段。
- 工具失败时根据权限和错误类型选择重试、降级、转草稿或询问用户。
- 最终回复需要校验其声明，避免工具失败却声称成功。

详细设计见 [Agent Loop 与可靠性设计](agent-loop-reliability-design.md)。

## 8. 工具调用设计

工具调用采用 Agent Harness + Tool Registry + Tool Router + Tool Executor 设计。

工具注册表包含：

- 工具名称
- 工具描述
- 参数 JSON Schema
- 执行函数
- 超时时间
- 失败降级策略
- 权限等级和审批策略
- 工具 schema 版本

第一批工具：

- `create_task`：创建待办事项。
- `list_tasks`：查询待办事项。
- `create_scheduled_agent_job`：创建每日简报、每周回顾、跟进监控等心跳任务。
- `list_scheduled_agent_jobs`：查询已安排的心跳任务。
- `run_scheduled_agent_job`：立即运行一次心跳任务。
- `save_profile_memory`：保存用户画像记忆。
- `search_memory`：检索多层记忆。
- `ingest_document`：导入文档并生成语义知识库切片。
- `summarize_text`：总结用户提供的长文本。

工具调用稳定性：

- 参数必须经过 schema 校验。
- 工具执行失败要记录错误日志。
- 可重试错误最多重试一次。
- 不可恢复错误返回降级回复，避免对话中断。

数据库使用 `agent_turns`、`agent_events`、`tool_definitions`、`tool_versions`、`user_tool_settings`、`tool_router_logs`、`tool_call_logs` 和 `tool_approval_requests` 支撑工具注册、用户开关、路由可观测性、审批和调用稳定性。

详细设计见 [工具调用设计](tool-calling-design.md)。

MCP、Skills、Tools、Knowledge Base、Channel Adapter 和 System Prompt Template 通过统一能力市场管理，公共能力所有用户可见，私有能力仅本人可见，用户安装后可绑定到自己的 Agent。系统提示词模板和 Skills 不同：系统提示词模板定义 Agent 的长期身份和边界，Skills 定义具体任务流程。Agent 配置可以绑定某个模板版本，也可以在其上添加用户自己的覆盖提示词。详细设计见 [能力市场设计](capability-market-design.md)。

## 9. 心跳任务与主动助理设计

心跳任务让 Agent 可以在用户授权后主动执行周期性任务，例如：

- 每日简报：工作日 8:00 汇总日程、未完成任务和重要提醒。
- 每周回顾：星期五 16:00 总结本周状态，生成下周计划。
- 跟进监控：工作日 9:00 扫描未完成事项和最近会话，提醒需要关注的内容。
- 定时提醒：在指定时间提醒用户。
- 内容摘要：定期整理某个知识库或主题。
- 社交辅助：定期生成待发送草稿，但不自动发送。

数据库使用 `scheduled_agent_jobs` 保存安排，使用 `scheduled_agent_job_runs` 保存每次运行。心跳任务不绕过 Agent Harness，而是触发一次正常的 Agent Turn，因此仍然受 `max_loop_steps`、输出校验、工具审批、用户隔离和兜底策略约束。

详细设计见 [心跳任务与主动助理设计](scheduled-agent-jobs-design.md)。

## 10. 多渠道入口设计

多渠道入口采用 Channel Adapter 抽象。NapCatQQ 是第一条落地通道，但数据库和后端模块按通用模型设计，后续可以接微信、Telegram、Discord、飞书或自定义 webhook。

核心原则：

- 外部平台接入统一落到 `channel_provider_definitions`、`channel_connections`、`external_conversations`、`channel_inbox_events`、`channel_outbox_messages` 和 `channel_policies`。
- Agent 内部只面对统一的 `conversations` 和 `messages`，不关心消息来自 QQ 还是飞书。
- Channel Adapter 负责收消息和发消息；MCP/Tool 负责平台上的额外动作，例如查群成员、取最近消息、发送富文本。
- 群聊默认 `mention_only`，私聊可自动回复。
- 社交辅助默认先生成草稿或要求审批，避免自动发送敏感内容。

详细设计见 [多渠道入口设计](channel-adapter-design.md)。

## 11. 用户级 Workspace Sandbox 设计

Workspace Sandbox 让 Agent 可以在用户授权的独立工作区内处理文件、运行代码和生成产物。每个用户可以选择是否启用 workspace；启用后后端为其创建独立目录或容器，并按用户配置限制磁盘、文件数、命令超时、输出大小、网络访问和生命周期。

核心原则：

- Agent 只能读写当前用户 workspace 内的文件。
- CLI 操作必须通过受控 workspace tool 执行，不能裸跑宿主机命令。
- 网络默认关闭，需要用户显式开启或配置白名单。
- 所有文件操作、命令执行、配额超限和销毁操作都写审计日志。
- 长时间不活跃的 workspace 可以进入 idle、archived 或 destroyed 状态。
- MVP 可以从本地目录隔离开始，生产环境建议升级到 Docker/Podman/nsjail/Firecracker 等更强 sandbox。

详细设计见 [用户级 Workspace Sandbox 设计](workspace-sandbox-design.md)。

## 12. 数据隔离与隐私保护

- 所有核心表都保存 `user_id`。
- 后端所有查询必须按 `user_id` 过滤。
- 用户密码只保存哈希，不保存明文。
- 用户 API Key 加密后存储在 `user_model_providers.encrypted_api_key`，接口永远不返回明文。
- LLM 调用时只读取当前用户自己的默认供应商配置，支持 OpenAI 和 Anthropic。
- 工作记忆额外绑定 `conversation_id`，确保临时上下文不会串会话。
- Profile、Working、Episodic Memory 默认私有，只召回当前用户数据。
- Procedural、Semantic Memory 支持 `private/public` 可见性；检索时召回当前用户私有资源和全局公共资源。
- 公共技能或公共知识库发布前需要用户显式确认。
- 心跳任务默认私有，只能使用当前用户启用的模型供应商、工具、MCP 和知识库。
- 心跳任务的写入类工具默认需要审批，社交辅助默认只能生成草稿。
- 渠道连接默认私有，外部账号、token、secret 加密保存。
- 群聊消息默认只在 @Agent 或命中白名单策略时触发 Agent。
- 外发社交消息进入 `channel_outbox_messages`，高风险消息需要审批或草稿确认。
- 系统提示词模板默认私有，公开模板发布前需要显式确认；Agent 配置绑定具体模板版本，避免公共模板更新导致行为漂移。
- Workspace 默认关闭；开启后只能访问当前用户 workspace，文件路径、命令、网络和配额都需要校验。
- Workspace CLI 执行必须记录审计日志，生产环境必须使用容器或更强 sandbox 隔离。
- Dreaming 默认只处理当前用户私有记忆，高风险修改先生成建议，不直接应用。
- 工具只能通过 MemoryManager 请求读写记忆，不能直接写数据库。
- 日志中避免记录完整敏感内容。
- 用户可删除记忆和会话。
- 未来可增加本地加密或字段级加密。

## 13. 可扩展性

新增记忆层或记忆类型：

- 新增 Profile Memory 类型时，向 `memory_type_definitions` 插入配置即可。
- 前端记忆筛选项从配置表读取，不写死类型列表。
- Memory Extractor 使用类型配置里的 `extraction_hint` 抽取新类型。
- Retrieval Engine 使用类型配置里的 `retrieval_weight` 参与排序。
- 新增完整记忆层时，再新增独立表、路由规则、检索器和前端展示入口。

新增工具：

- 在工具注册表中注册工具。
- 定义参数 schema。
- 实现执行函数。
- 添加测试用例。
- 为工具声明超时、重试、审批和降级策略。
- 将工具失败样例加入 Agent loop 测试，确保不会卡死或假装成功。

新增心跳任务模板：

- 在前端新增模板卡片。
- 在后端定义 `job_type`、默认 `prompt_template`、`context_policy` 和 `tool_policy`。
- 如果需要新工具，先通过工具注册表接入。
- 增加运行记录测试、错过触发补偿测试和连续失败暂停测试。

新增外部聊天入口：

- 向 `channel_provider_definitions` 注册 provider。
- 实现一个 `ChannelAdapter`，负责校验事件、规范化消息、发送消息和健康检查。
- 在前端渠道连接页增加配置表单。
- 在能力市场中发布 `channel_adapter`。
- 增加私聊、群聊 @、限频、失败重试和外发审批测试。

新增系统提示词模板：

- 新增 `system_prompt_templates` 和 `system_prompt_template_versions`。
- 在能力市场中发布 `system_prompt_template`。
- Agent 配置绑定具体模板版本，避免公共模板更新导致行为漂移。
- Context Builder 渲染模板变量，并写入 `context_build_items` 便于审计。
- 增加模板 fork、版本升级、变量缺失和越权提示词检测测试。

新增 Workspace Sandbox 能力：

- 新增 workspace 配置、文件索引、命令执行和审计事件表。
- 在 Tool Registry 注册 workspace 文件和 CLI 工具。
- MVP 阶段使用本地目录隔离，生产阶段升级到 Docker/Podman/nsjail/Firecracker。
- 增加路径逃逸、超时、输出截断、网络策略、磁盘配额和不活跃销毁测试。

更换模型：

- 用户在设置页新增或更新 OpenAI、Anthropic 模型供应商配置。
- 每个用户可以设置自己的默认 chat model 和 embedding model。
- 每个用户可以通过 `embedding_enabled` 控制是否生成和使用 embedding，避免默认产生额外费用。
- 后端通过统一 Provider Client 屏蔽 OpenAI 与 Anthropic 的接口差异。
- 后端需要校验 Agent 配置引用的 provider 属于当前用户，避免跨用户使用模型配置。

更换向量检索方案：

- MVP 阶段数据库使用 PostgreSQL + pgvector 支持 Semantic Memory embedding，默认向量维度为 1024。
- 如果用户关闭 embedding，则 Retrieval Engine 退回关键词、标签和元数据检索。
- 如果文档规模增长，可切换到 Qdrant、Chroma 或 Pinecone。
- 后端通过统一 Retrieval Engine 屏蔽向量库差异。

## 14. 测试规划

- 后端单元测试：Memory Router、记忆检索、工具参数校验、上下文组装。
- Agent loop 测试：最大步数终止、格式修复、LLM 重试、工具失败降级、审批中断恢复。
- 后端接口测试：会话、消息、记忆、任务 API。
- Scheduler 测试：心跳任务创建、下一次运行时间计算、立即运行、连续失败暂停。
- Channel 测试：NapCatQQ 入站事件去重、群聊触发策略、外部会话映射、外发消息审批和失败重试。
- Prompt Market 测试：模板版本锁定、变量渲染、公共模板 fork、Agent 配置绑定和上下文注入顺序。
- Workspace Sandbox 测试：路径逃逸拦截、用户目录隔离、CLI 超时、stdout/stderr 截断、网络策略和配额限制。
- 前端组件测试：消息列表、输入框、记忆面板。
- 端到端测试：发送消息、生成回复、保存画像记忆、检索情景记忆、命中程序技能、引用语义知识库。
