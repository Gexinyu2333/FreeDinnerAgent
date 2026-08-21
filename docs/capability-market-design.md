# FreeDinnerAgent 能力市场设计

## 1. 设计目标

FreeDinnerAgent 的能力不只来自内置工具，还可以来自 MCP Server、用户或系统沉淀的 Skills、公共知识库、普通工具和外部聊天入口。为了让这些能力可发现、可安装、可启用、可共享，系统设计一个统一的 Capability Market。

核心目标：

- 公共能力所有用户可见，私有能力仅本人可见。
- 用户可以把能力安装到自己的账号。
- 用户可以选择某个 Agent 使用哪些能力。
- MCP、Skills、Tools、Knowledge Base、Channel Adapter 使用统一的市场和安装模型。
- Skills 使用渐进式披露，按任务需要加载 light、standard、full 内容，控制 Token。

## 2. 能力类型

市场支持五类能力：

```text
tool            # 普通内置工具或 HTTP 工具
mcp_server      # MCP Server，提供一组外部工具或资源
skill           # 可复用工作流 / Procedural Memory
knowledge_base  # Semantic Memory 知识库
channel_adapter # QQ、Telegram、Discord、飞书等外部聊天入口
```

可见性：

- `private`：仅创建者可见。
- `public`：所有用户可见。

注意：公开的是能力定义或公共知识，不自动公开用户的 Profile Memory、Working Memory、Episodic Memory。

## 3. 数据结构

核心表：

```text
marketplace_items           # 市场条目，统一展示 Tool/MCP/Skill/Knowledge/Channel
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
```

关系：

```text
marketplace_items.ref_id
  -> tool_definitions.id
  -> mcp_server_definitions.id
  -> skills.id
  -> knowledge_documents.id
  -> channel_provider_definitions.id

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

## 8. Agent 如何使用市场能力

一次对话中，Context Engine 和 Tool Router 只看当前 Agent 已绑定的能力。

流程：

1. 读取当前用户的 `user_agent_configs`。
2. 查询 `agent_capability_bindings`。
3. 过滤用户已安装且启用的能力。
4. 加载私有能力 + 公共能力。
5. MCP Server 提供工具候选。
6. Skills 提供流程候选和工具序列。
7. Knowledge Base 提供 Semantic Memory 候选。
8. Tool Router 选择本轮真正需要注入 Prompt 的工具 schema。
9. Context Engine 按渐进式披露加载 Skill 内容。

这样避免“安装了很多能力就全部进 Prompt”的问题。

## 9. 公共与私有发布规则

默认规则：

- 用户新建能力默认 `private`。
- 用户显式发布后才变为 `public`。
- Profile Memory、Working Memory、Episodic Memory 不允许直接发布。
- Skill 可以发布为公共能力，但需要去除个人信息。
- Knowledge Base 可以发布为公共能力，但不应包含隐私内容。
- MCP Server 可以发布定义，但用户密钥永远保存在自己的 `user_mcp_server_settings.encrypted_env`。

## 10. 新增能力扩展流程

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

## 11. 小巧思：能力推荐

系统可以根据用户当前对话推荐能力：

- 用户经常问文档问题，推荐 Semantic Memory 能力。
- 用户多次做相似流程，推荐安装或生成 Skill。
- 用户请求外部系统操作，推荐相关 MCP。

推荐来源可以写入 `dreaming_insights`，由用户确认后安装。
