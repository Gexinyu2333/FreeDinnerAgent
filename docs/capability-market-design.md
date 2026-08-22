# FreeDinnerAgent 能力市场设计

## 1. 设计目标

FreeDinnerAgent 的能力不只来自内置工具，还可以来自 MCP Server、用户或系统沉淀的 Skills、公共知识库、普通工具、系统提示词模板和外部聊天入口。为了让这些能力可发现、可安装、可启用、可共享，系统设计一个统一的 Capability Market。

核心目标：

- 公共能力所有用户可见，私有能力仅本人可见。
- 用户可以把能力安装到自己的账号。
- 用户可以选择某个 Agent 使用哪些能力。
- MCP、Skills、Tools、Knowledge Base、Channel Adapter、System Prompt Template 使用统一的市场和安装模型。
- Skills 使用渐进式披露，按任务需要加载 light、standard、full 内容，控制 Token。

## 2. 能力类型

市场支持六类能力：

```text
tool                   # 普通内置工具或 HTTP 工具
mcp_server             # MCP Server，提供一组外部工具或资源
skill                  # 可复用工作流 / Procedural Memory
knowledge_base         # Semantic Memory 知识库
channel_adapter        # QQ、Telegram、Discord、飞书等外部聊天入口
system_prompt_template # Agent 系统提示词模板
```

可见性：

- `private`：仅创建者可见。
- `public`：所有用户可见。

注意：公开的是能力定义或公共知识，不自动公开用户的 Profile Memory、Working Memory、Episodic Memory。

## 3. 数据结构

核心表：

```text
marketplace_items           # 市场条目，统一展示 Tool/MCP/Skill/Knowledge/Channel/Prompt
user_capability_installs    # 用户安装了哪些能力
agent_capability_bindings   # 某个 Agent 启用了哪些能力
```

专项表：

```text
tool_definitions            # Tool 定义
tool_versions               # Tool 参数和结果 schema
mcp_server_definitions      # MCP Server 定义
user_mcp_server_settings    # 用户 MCP 配置和健康状态
skills                      # Skill 主表
skill_versions              # Skill 版本
skill_disclosure_sections   # Skill 渐进式披露内容
knowledge_documents         # 知识库文档
knowledge_chunks            # 知识库切片
channel_provider_definitions # Channel Adapter 定义
channel_connections          # 用户的渠道连接实例
channel_policies             # 私聊/群聊触发策略
system_prompt_templates      # 系统提示词模板
system_prompt_template_versions # 系统提示词模板版本
system_prompt_template_variables # 系统提示词变量定义
```

关系：

```text
marketplace_items.ref_id
  -> tool_definitions.id
  -> mcp_server_definitions.id
  -> skills.id
  -> knowledge_documents.id
  -> channel_provider_definitions.id
  -> system_prompt_templates.id

user_capability_installs
  -> 用户安装能力

agent_capability_bindings
  -> 某个 Agent 实际启用能力
```

## 4. 能力市场页面

前端可以做一个“能力市场”页面，包含：

- MCP
- Skills
- Tools
- Knowledge Bases
- Channels
- System Prompts

筛选条件：

- 公共 / 私有
- 已安装 / 未安装
- 类型
- 标签
- 权限等级
- 是否需要配置

典型操作：

- 安装能力
- 卸载能力
- 启用到当前 Agent
- 从当前 Agent 移除
- 发布为公共能力
- 归档公共能力

Channel Adapter 安装后还需要创建连接，例如配置 NapCatQQ endpoint、Telegram Bot Token、飞书事件订阅 secret。能力市场负责“安装入口能力”，渠道连接页负责“绑定具体账号或机器人实例”。

System Prompt Template 安装后可以直接绑定到某个 Agent 配置。Agent 绑定的是具体模板版本，而不是永远跟随最新版本，避免公共模板更新后导致 Agent 行为漂移。

## 5. MCP 接入方式

MCP Server 作为一类能力进入市场。

`mcp_server_definitions` 记录：

- MCP 名称
- 传输方式：`stdio`、`http`、`sse`
- 启动命令或 endpoint
- 需要的环境变量 schema
- 权限等级
- 可见性

用户安装 MCP 后，配置写入：

```text
user_mcp_server_settings
```

其中 `encrypted_env` 保存用户私有配置，例如 token、endpoint 密钥等。

MCP 工具发现流程：

1. 用户安装并启用 MCP Server。
2. 后端根据用户配置启动或连接 MCP Server。
3. 调用 MCP list tools/list resources。
4. 将 MCP 暴露的工具映射为 `tool_definitions.handler_type = mcp`。
5. Tool Router 在当前 Agent 绑定能力中选择可用 MCP 工具。
6. Tool Executor 通过 MCP Client 执行工具。

MCP 的好处：

- 外部系统能力不需要写死在后端。
- 一个 MCP Server 可以提供多个工具。
- 用户可以选择是否把某个 MCP 应用于自己的 Agent。

## 6. Skills 接入方式

Skills 对应 Procedural Memory，是可复用的任务流程。

来源：

- 用户手写。
- Dreaming 从成功 Episodic Memory 中提炼。
- 公共市场安装。

Skills 不一定直接执行代码，它更像“做事说明书 + 工具序列 + 输出模板”。如果 Skill 里声明了工具序列，Tool Router 会优先启用这些工具。

Skill 结构：

```text
skills
skill_versions
skill_disclosure_sections
```

## 7. Skills 渐进式披露

为了控制 Token，Skills 不一次性全量注入 Prompt，而是渐进式披露。

### light

加载内容：

- 技能名称
- 一句话描述
- 适用场景
- 触发关键词

使用时机：

- 路由阶段
- 候选技能很多时

### standard

加载内容：

- 输入要求
- 输出格式
- 核心步骤
- 关键工具列表
- 风险点

使用时机：

- 技能被选中，但任务不复杂
- Token 预算中等

### full

加载内容：

- 完整 ReAct 步骤
- 分支判断
- 工具调用序列
- 异常处理
- 输出模板

使用时机：

- 复杂任务
- 用户明确要求按某个技能执行
- 当前 Token 预算充足

`agent_capability_bindings.load_mode` 可以设置为：

- `auto`：系统按任务复杂度和 Token 预算决定。
- `light`
- `standard`
- `full`

## 8. System Prompt Template 接入方式

System Prompt Template 定义 Agent 的长期身份、表达风格、工作边界和安全约束。它不应该只是一段保存在 `user_agent_configs.system_prompt` 里的普通文本，而应该像 MCP、Skills、Knowledge Base 一样，沉淀成可复用、可分享、可版本化的市场能力。

核心目标：

- 用户可以创建自己的私有系统提示词模板。
- 用户可以发布公开模板，供其他用户安装或 fork。
- 用户可以把模板绑定到自己的 Agent 配置。
- 模板需要版本化，避免公共模板更新后悄悄改变用户 Agent 行为。
- 模板可以声明变量、适用场景、风险边界和推荐能力组合。
- Context Builder 在构建上下文时，根据 Agent 配置加载当前模板版本。

来源：

- 系统内置模板。
- 用户私有模板。
- 公共市场安装。
- 用户 fork 公共模板后修改。

它和 Skill 的区别：

- System Prompt Template：定义 Agent 的长期身份、风格、规则和总边界。
- Skill：定义某类任务的可复用流程、工具序列和输出模板。

一次 Agent Turn 中，系统提示词模板通常位于上下文最前面，而 Skill 只在当前任务命中时按渐进式披露加载。

可见性：

- `private`：仅创建者可见。
- `public`：所有用户可见。

生命周期：

- `draft`：草稿，只能创建者使用。
- `published`：已发布，可被安装或绑定。
- `archived`：归档，不再推荐新用户安装，但已绑定的 Agent 可以继续使用旧版本。
- `deleted`：逻辑删除。

公开模板推荐支持 fork：

```text
公共模板
  -> 用户安装
  -> 用户 fork 为自己的私有模板
  -> 用户修改变量、风格或安全规则
```

模板结构：

```text
system_prompt_templates
system_prompt_template_versions
system_prompt_template_variables
```

`system_prompt_templates` 保存模板主信息：

- `id`
- `owner_user_id`：为空表示系统内置模板。
- `name`
- `display_name`
- `description`
- `category`
- `tags`
- `visibility`
- `status`
- `latest_version`
- `created_at`
- `updated_at`

`system_prompt_template_versions` 保存具体版本：

- `id`
- `template_id`
- `version`
- `content`
- `change_note`
- `recommended_model_family`
- `recommended_capabilities`
- `safety_policy`
- `token_estimate`
- `status`
- `created_at`

`system_prompt_template_variables` 保存变量定义：

- `id`
- `template_version_id`
- `name`
- `display_name`
- `description`
- `default_value`
- `required`
- `value_type`

能力市场关联：

```text
marketplace_items.item_type = 'system_prompt_template'
marketplace_items.ref_id -> system_prompt_templates.id
```

Agent 配置绑定：

```text
user_agent_configs.system_prompt_template_id
user_agent_configs.system_prompt_template_version_id
user_agent_configs.system_prompt_variables
user_agent_configs.custom_system_prompt_override
```

其中：

- `system_prompt_template_id`：当前 Agent 使用哪个模板。
- `system_prompt_template_version_id`：锁定哪个版本。
- `system_prompt_variables`：用户填入的变量值。
- `custom_system_prompt_override`：用户自己的覆盖内容，可为空。

Context Builder 加载规则：

1. 读取当前用户的默认 Agent 配置。
2. 如果配置绑定了 `system_prompt_template_version_id`，加载该版本内容。
3. 根据 `system_prompt_variables` 替换模板变量。
4. 如果存在 `custom_system_prompt_override`，按配置策略追加或覆盖模板内容。
5. 拼入安全边界、工具策略、记忆策略、MCP/Skill 选择、渠道策略和 Workspace 策略。
6. 写入 `context_build_items`，记录本轮使用了哪个模板版本。

建议拼接顺序：

```text
Meta System Policy
  -> System Prompt Template
  -> User Custom Override
  -> Memory Policy
  -> Tool / MCP / Skill Policy
  -> Channel / Workspace Policy
```

这样系统层规则、用户模板、记忆和工具边界各有位置，不会混在一坨文本里。

模板可以声明变量，例如：

```text
{agent_name}
{user_display_name}
{language}
{tone}
{memory_policy}
{workspace_policy}
{preferred_output_style}
```

变量必须有类型：

- `string`
- `number`
- `boolean`
- `enum`
- `json`

前端根据变量定义渲染配置表单，用户不需要直接改模板原文。

模板必须版本化。公共模板发布新版本后，不自动改变已经绑定旧版本的 Agent；用户需要手动升级。

版本规则：

- 新建模板先产生 `v1`。
- 编辑已发布版本时，不直接修改原版本，而是创建新版本。
- Agent 默认锁定具体版本。
- 用户可以手动升级到新版本。
- Context Build 日志记录实际使用的版本。

公开模板可能诱导模型泄露隐私、绕过工具审批或访问不该访问的 workspace，因此需要基本审核。

发布公共模板时应检查：

- 是否包含越权指令。
- 是否要求忽略系统安全策略。
- 是否默认开启高风险工具。
- 是否要求自动发送社交消息。
- 是否诱导读取其他用户数据。

公共模板只能表达“角色和工作方式”，不能绕过：

- 用户隔离。
- API Key 隔离。
- Tool approval。
- Channel outbox 审批。
- Workspace sandbox 边界。

前端可以在能力市场中展示 System Prompts，也可以在 Agent 设置页提供快捷入口。

市场页面：

- 浏览公共模板。
- 查看模板详情和版本。
- 安装模板。
- Fork 模板。
- 发布自己的模板。

Agent 设置页：

- 选择系统提示词模板。
- 选择版本。
- 填写变量。
- 添加个人覆盖内容。
- 预览最终系统提示词。

预览很重要。用户需要看到最终进入上下文的系统提示词片段，而不是只看到模板名字。

MVP 第一版可以先做：

1. 新增系统提示词模板和版本表。
2. 支持私有/公共模板。
3. Agent 配置绑定模板版本。
4. Context Builder 加载模板内容。
5. 前端可创建、选择、预览模板。
6. 创建模板时执行规则版安全扫描，并把 `auto_approved` 审核记录写入版本的 `safety_policy`。

高级项：

- 独立管理员工作台和人工审核角色体系。
- A/B 测试。
- 变量的跨字段联动、条件展示等高级校验。

## 9. Agent 如何使用市场能力

一次对话中，Context Engine 和 Tool Router 只看当前 Agent 已绑定的能力。

流程：

1. 读取当前用户的 `user_agent_configs`。
2. 加载 Agent 绑定的系统提示词模板版本。
3. 查询 `agent_capability_bindings`。
4. 过滤用户已安装且启用的能力。
5. 加载私有能力 + 公共能力。
6. MCP Server 提供工具候选。
7. Skills 提供流程候选和工具序列。
8. Knowledge Base 提供 Semantic Memory 候选。
9. Tool Router 选择本轮真正需要注入 Prompt 的工具 schema。
10. Context Engine 按渐进式披露加载 Skill 内容。

这样避免“安装了很多能力就全部进 Prompt”的问题。

## 10. 公共与私有发布规则

默认规则：

- 用户新建能力默认 `private`。
- 用户显式发布后才变为 `public`。
- Profile Memory、Working Memory、Episodic Memory 不允许直接发布。
- Skill 可以发布为公共能力，但需要去除个人信息。
- Knowledge Base 可以发布为公共能力，但不应包含隐私内容。
- MCP Server 可以发布定义，但用户密钥永远保存在自己的 `user_mcp_server_settings.encrypted_env`。
- System Prompt Template 可以发布为公共能力，但不能包含越权指令、隐私信息或绕过审批的规则。

## 11. 新增能力扩展流程

新增 MCP：

1. 写入 `mcp_server_definitions`。
2. 如需上架，写入 `marketplace_items`。
3. 用户安装后写入 `user_capability_installs`。
4. 用户配置私有 env 后写入 `user_mcp_server_settings`。
5. 用户选择应用到 Agent 后写入 `agent_capability_bindings`。

新增 Skill：

1. 写入 `skills`。
2. 写入 `skill_versions`。
3. 写入 `skill_disclosure_sections` 的 light/standard/full 内容。
4. 如需上架，写入 `marketplace_items`。
5. 用户安装并绑定到 Agent。

新增公共知识库：

1. 写入 `knowledge_documents`，`visibility = public`。
2. 切片写入 `knowledge_chunks`。
3. 可选生成 embedding。
4. 写入 `marketplace_items`。
5. 用户安装并绑定到 Agent。

## 12. 小巧思：能力推荐

系统可以根据用户当前对话推荐能力：

- 用户经常问文档问题，推荐 Semantic Memory 能力。
- 用户多次做相似流程，推荐安装或生成 Skill。
- 用户请求外部系统操作，推荐相关 MCP。

推荐来源可以写入 `dreaming_insights`，由用户确认后安装。

## 13. 当前后端 MVP 实现范围

当前已落地：

- 数据库已包含 `marketplace_items`、`user_capability_installs`、`agent_capability_bindings`。
- 市场能力类型已覆盖 `tool`、`mcp_server`、`skill`、`knowledge_base`、`channel_adapter`、`system_prompt_template`。
- 新增 `system_prompt_templates`、`system_prompt_template_versions`、`system_prompt_template_variables`，支持系统提示词模板版本化。
- 新增市场 API：
  - `GET /api/v1/marketplace-items`
  - `POST /api/v1/marketplace-items/{item_id}/install`
  - `POST /api/v1/marketplace-items/{item_id}/rate`
  - `POST /api/v1/capability-installs/{install_id}/enable|disable`
  - `POST /api/v1/agent-capability-bindings`
  - `POST /api/v1/agent-capability-bindings/{binding_id}/enable|disable`
- 新增系统提示词模板 API：
  - `POST /api/v1/system-prompt-templates`
  - `POST /api/v1/system-prompt-templates/preview`
  - `POST /api/v1/system-prompt-template-versions/{version_id}/fork`
- Agent 绑定 `system_prompt_template` 时，`agent_capability_bindings.capability_ref_id` 锁定具体 `system_prompt_template_versions.id`，避免公共模板更新导致行为漂移。
- `llm.Service` 已在构建上下文前解析 Agent 绑定的系统提示词模板版本；如果没有绑定，则继续使用 `user_agent_configs.system_prompt`。
- 系统提示词模板预览支持 `{variable}` 变量替换、用户自定义 override、required 校验、number/boolean/enum/json 类型校验；创建模板时会自动从内容中提取基础变量定义。
- 市场条目已支持安装量回算和用户评分，评分会写入 `marketplace_item_reviews` 并回算 `marketplace_items.rating`。
- 系统提示词模板创建时会做规则版安全扫描，拦截绕过审批、泄露 API Key、跨用户读取等危险指令；通过扫描的版本会把 `review_status = auto_approved` 写入 `safety_policy`，便于审计。
- 公共模板版本支持 fork 为当前用户的私有模板。

当前已落地：

- MCP Server runtime 可从 MCP definition metadata 与用户启用设置中发现 tool specs，并在启动时同步为 Tool Registry 中的 MCP tool；Tool Executor 支持通过 HTTP MCP bridge 调用 `tools/call`。
- Skills 的自动匹配和 `skill_disclosure_sections` light/standard/full 渐进式披露已接入 Context Builder；基于 episode 的规则版自动 skill 沉淀已接入。
- Tool 与 Channel Adapter 的自动上架已统一封装到启动同步流程：内置工具和 NapCatQQ Channel Provider 会写入 `marketplace_items`。
- Tool Router 已优先按默认 Agent 的 `agent_capability_bindings` 过滤 tool 候选；如果该 Agent 暂未绑定任何 tool，则回退到用户可见工具，方便开发期逐步接入能力市场。

高级项：

- 独立管理员审核工作台和更完整的公共模板安全扫描不属于当前个人助理 MVP，后续可以在引入角色/权限系统后扩展。
- LLM/embedding Skill Router 属于高成本增强；当前使用规则匹配和 light/standard/full 渐进披露，已满足完整流程闭环。
- MCP stdio 进程生命周期、resources/prompts 枚举和复杂权限映射归入高级项；当前运行时以 HTTP MCP bridge 作为可执行闭环。
- Knowledge Base 的自动上架需要独立“发布/共享知识库”事件，目前只完成文档写入、私有/公共可见性和检索；市场化发布工作台归入高级项。
