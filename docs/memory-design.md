# FreeDinnerAgent 记忆系统设计

## 1. 设计目标

FreeDinnerAgent 的记忆系统参考 Hermes Agent 的分层记忆思想，但结合 Web 产品、用户隔离和 PostgreSQL 存储做了工程化改造。

核心目标：

- 分离不同性质的记忆，避免偏好、历史对话、任务流程和外部知识混在一起。
- 按需检索和加载，控制 Prompt Token。
- 支持用户隔离，所有正式记忆都绑定 `user_id`。
- 支持部分记忆共享，Semantic Memory 和 Procedural Memory 可设置为公共资源。
- 支持 Semantic Memory embedding，使用 PostgreSQL + pgvector 实现 RAG 向量检索；是否生成和使用 embedding 由用户配置控制。
- 支持后台 Curator 和 Dreaming 机制，自动压缩历史、合并记忆、沉淀技能、更新向量。

## 2. 四层记忆模型

### 2.1 Working Memory

Working Memory 是会话级工作记忆，代表“当前正在思考的草稿纸”。

存储表：

```text
session_working_memories
```

典型内容：

- 当前任务目标
- 用户本轮临时约束
- 输出格式要求
- 上一步工具调用结果
- 当前任务进度

特点：

- 绑定 `user_id` 和 `conversation_id`。
- 每轮 Prompt 默认加载。
- 数据量严格限制，通常只保留几百 Token。
- 会话结束后可清理或归档，不进入长期记忆召回。

### 2.2 Profile Memory

Profile Memory 存长期用户画像，是用户偏好、习惯、事实和目标的结构化表达。

存储表：

```text
profile_memories
```

记忆类型：

- `preference`：偏好，例如“不喜欢香菜”。
- `habit`：习惯，例如“晚上更适合整理计划”。
- `fact`：稳定事实，例如“正在做 AI 个人助理课题”。
- `relationship`：人物关系。
- `goal`：长期目标。
- `todo`：待办事项。
- `other`：其他。

关键字段：

- `content`：记忆正文。
- `evidence`：原始依据，说明这条记忆从哪里总结而来。
- `confidence`：置信度，避免把不确定内容当成事实。
- `importance`：重要程度，用于召回排序。
- `status`：支持 active、archived、deleted。

记忆类型不直接写死在业务代码中，而是由配置表管理：

```text
memory_type_definitions
```

新增画像记忆类型时，只需要插入新的 `memory_type`、展示名称、说明、抽取提示和检索权重，不需要修改 `profile_memories` 表结构。

### 2.3 Episodic Memory

Episodic Memory 是情景记忆，存储“一轮完整交互经历”，不是简单的聊天消息。

存储表：

```text
episodes
episode_tags
episode_tool_calls
```

每个 episode 包含：

- 用户输入
- Agent 本轮处理摘要
- 最终回复
- 工具调用记录
- 任务类型
- 成功、失败或中断状态
- 标签和重要程度

用途：

- 用户再次提出相似任务时，召回历史处理经验。
- Curator 从多次成功 episode 中提炼程序技能。
- 失败 episode 可作为反例，避免重复错误路径。

检索方式：

- MVP 阶段使用关键词、标签、任务类型和时间过滤。
- `episodes` 保存 embedding，支持按语义相似度召回历史经验。

### 2.4 Procedural Memory

Procedural Memory 是程序/技能记忆，保存可复用的 ReAct 工作流。

存储表：

```text
skills
skill_versions
```

每个 skill 包含：

- 技能名称
- 触发关键词
- 适用场景
- 可见性：`private` 或 `public`
- 权限等级
- ReAct 步骤
- 工具调用序列
- 输出模板
- 异常处理策略
- 使用次数、成功次数和失败次数

生成方式：

- 用户或开发者手动创建。
- Curator 从多个相似且成功的 Episodic Memory 中自动提炼。
- 用户可将通用技能发布为公共技能，供其他用户在相似任务中复用。

加载模式：

- `light`：只加载名称和一句话描述。
- `standard`：加载入参、输出格式和核心步骤。
- `full`：加载完整 ReAct 流程和工具序列。

### 2.5 Semantic Memory

Semantic Memory 是语义知识库，存放外部文档、课程资料、网页内容或用户上传笔记。

存储表：

```text
knowledge_documents
knowledge_chunks
```

设计方式：

1. 用户上传或录入文档。
2. 后端将文档切片。
3. 每个 chunk 保存原文、token 数和 metadata。
4. 如果用户开启 `embedding_enabled`，使用 embedding 模型生成向量。
5. 向量写入 PostgreSQL 的 `pgvector` 字段。
6. 用户提问时，开启 embedding 则按语义相似度召回相关 chunk；未开启则使用关键词、标签和元数据检索。

可见性：

- `private`：仅上传用户可检索。
- `public`：所有用户都可检索，适合课程要求、项目公共文档、通用技术资料。

公共知识库不存放个人隐私和对话轨迹，只用于客观知识检索。

SQL 中默认安装 pgvector 能力：

```sql
CREATE EXTENSION IF NOT EXISTS vector;
```

`knowledge_chunks` 保存向量字段：

```sql
embedding vector(1024)
```

并创建向量索引：

```sql
CREATE INDEX idx_knowledge_chunks_embedding
ON knowledge_chunks USING ivfflat (embedding vector_cosine_ops);
```

默认维度 1024 对应 `BAAI/bge-m3` / `Pro/BAAI/bge-m3`。如果后续切换为其他维度的 embedding 模型，需要通过数据库迁移调整向量字段维度。

成本控制：

- `user_agent_configs.embedding_enabled` 控制是否生成和使用 embedding。
- `user_agent_configs.embedding_cost_policy` 控制生成策略，例如手动、自动、每月 token 上限、是否为公共知识库生成向量。
- 关闭 embedding 时，知识库仍可保存文档切片，只是检索退回关键词、标签和元数据。

## 3. MemoryManager 子模块

后端 `internal/memory` 建议拆成以下组件：

- Memory Router：判断当前问题需要加载哪些记忆层。
- Retrieval Engine：统一执行关键词检索、标签检索和向量检索。
- Memory Compressor：压缩召回片段，控制 Token。
- Memory Persister：统一写入 Working、Profile、Episodic、Procedural、Semantic Memory。
- Curator Connector：把每轮完整交互推送给后台复盘任务。
- Dreaming Engine：在空闲或定时阶段重放历史记忆，做离线合并、提炼、遗忘和技能候选生成。

所有工具和 Agent 逻辑都不直接写记忆表，而是通过 MemoryManager 读写，避免记忆污染。

## 4. 记忆写入时机

记忆写入分为同步写入和异步写入。同步写入保证当前对话立即可见，异步写入交给 Curator 做复盘，不阻塞用户响应。

### 4.1 Working Memory 写入

写入时机：

- 用户提出本轮临时要求，例如“这次回答简洁一点”。
- Agent 进入多步任务，需要保存当前步骤、临时变量或工具中间结果。
- 工具调用返回结果，需要在下一轮继续使用。

写入策略：

- 当前会话立即写入 `session_working_memories`。
- 同一 `conversation_id + memory_key` 使用覆盖更新。
- 设置 `expires_at` 或在会话结束后清理。

### 4.2 Profile Memory 写入

写入时机：

- 用户明确要求记住，例如“你记一下”“以后都这样”。
- 用户表达稳定偏好，例如“我不喜欢香菜”“我喜欢表格输出”。
- 用户说明长期事实、目标、关系或习惯。
- 工具调用产生需要长期跟踪的信息，例如待办或提醒。

写入策略：

- 先由 LLM 或规则抽取候选记忆。
- 使用 `memory_type_definitions.extraction_hint` 判断类型。
- 低置信度候选不直接写入，可进入待确认状态或降低 importance。
- 写入时保存 `evidence` 和 `source_message_id`，便于追溯来源。

### 4.3 Episodic Memory 写入

写入时机：

- 每轮用户消息和助手回复完成后。
- 工具链执行结束后。
- 任务失败、中断或降级时，也要写入情景记忆，用作后续反例。

写入策略：

- 原始聊天仍保存在 `messages`。
- Curator 将一轮完整交互压缩成 `episodes.agent_summary`。
- 工具调用写入 `episode_tool_calls`。
- 根据任务类型写入 `task_type` 和 `episode_tags`。

### 4.4 Procedural Memory 写入

写入时机：

- 多个 Episodic Memory 显示同类任务多次成功。
- 用户手动创建或修改技能。
- Curator 发现稳定可复用的 ReAct 步骤和工具序列。

写入策略：

- 初次生成写入 `skills` 和 `skill_versions`。
- 后续优化新增 `skill_versions`，保留历史版本。
- 使用次数、成功次数和失败次数持续更新。
- 通用技能可以由用户显式发布为 `public`。

### 4.5 Semantic Memory 写入

写入时机：

- 用户上传文档、粘贴资料或录入知识笔记。
- 系统导入课程要求、项目公共文档或通用资料。
- 文档内容发生变化，需要重新切片或重新 embedding。

写入策略：

- 文档元数据写入 `knowledge_documents`。
- 文档切片写入 `knowledge_chunks`。
- MVP 阶段先支持关键词检索。
- 增强阶段通过 Curator 异步生成 embedding。
- 公共知识库必须由用户或管理员显式设置 `visibility = public`。

## 5. 记忆存储结构

当前数据库按记忆层拆表，避免所有记忆混在一个大表里。

```text
session_working_memories     # 会话级工作记忆
memory_type_definitions      # Profile Memory 类型配置
profile_memories             # 长期用户画像记忆
episodes                     # 情景记忆摘要
episode_tags                 # 情景标签
episode_tool_calls           # 情景中的工具调用
skills                       # 程序技能主表
skill_versions               # 技能版本与 ReAct 流程
knowledge_documents          # 语义知识库文档
knowledge_chunks             # 文档切片，保存 embedding vector(1024)
memory_retrieval_logs        # 每轮记忆召回记录
dreaming_sessions            # Dreaming 离线复盘批次
dreaming_insights            # Dreaming 产出的洞察和建议
curator_jobs                 # 后台复盘任务
```

这种结构的好处：

- 不同记忆生命周期不同，可以独立清理和归档。
- 不同记忆检索方式不同，可以独立优化。
- Profile Memory 类型通过配置表扩展。
- Semantic Memory 默认接入 pgvector。
- Procedural Memory 可以支持版本迭代和公共共享。
- Dreaming 可以把分散记忆重新组织成更紧凑、更有用的长期结构。

## 6. 记忆检索策略

检索由 Memory Router 先决定启用哪些层，再由 Retrieval Engine 执行。

### 6.1 路由策略

- 简单闲聊：只加载 Working Memory 和少量 Profile Memory。
- 用户画像相关问题：加载 Working Memory + Profile Memory。
- 重复任务或工具型任务：加载 Working Memory + Procedural Memory + Episodic Memory。
- 文档问答或知识查询：加载 Working Memory + Semantic Memory，必要时补充 Episodic Memory。
- 复杂任务：五层记忆都可启用，但必须经过压缩和 Token 阈值控制。

### 6.2 分层检索策略

Working Memory：

- 按 `conversation_id` 全量加载。
- 限制 token 上限，过期记录不加载。

Profile Memory：

- 按 `user_id`、`memory_type`、`status`、`importance` 检索。
- 使用 `memory_type_definitions.retrieval_weight` 调整排序。
- MVP 阶段可先使用关键词匹配，后续接入 Profile Memory embedding 时可复用 pgvector 能力。

Episodic Memory：

- 按 `user_id`、`task_type`、`episode_tags`、时间和状态过滤。
- 成功 episode 用于参考经验，失败 episode 用于规避错误。
- 使用 episode embedding 做语义召回。

Procedural Memory：

- 检索当前用户私有技能和公共技能。
- 先匹配 `trigger_keywords`，再按场景和成功率排序。
- 根据复杂度选择 `light`、`standard`、`full` 加载模式。

Semantic Memory：

- 检索当前用户私有知识库和公共知识库。
- 使用关键词检索 + pgvector embedding top-k。
- 检索时必须先限定可见范围，再做相似度召回。

### 6.3 统一排序与裁剪

每条召回结果统一转成 Memory Chunk：

```text
layer
ref_id
content
score
token_count
visibility
source
```

排序因素：

- 相似度分数
- 记忆层级优先级
- 重要程度
- 成功率或失败标记
- 最近访问时间
- 用户私有资源优先于公共资源

裁剪策略：

- 先去重，再压缩。
- 超出 Token 阈值时，优先丢弃低分、低重要度、公共且泛化的信息。
- Working Memory 和明确用户偏好优先保留。

## 7. 新增记忆类型与扩展方式

### 7.1 新增 Profile Memory 类型

例如要新增 `health` 类型：

```sql
INSERT INTO memory_type_definitions (
    memory_type,
    display_name,
    description,
    extraction_hint,
    retrieval_weight
) VALUES (
    'health',
    '健康偏好',
    '用户与健康、运动、饮食限制相关的长期信息。',
    '用户表达过敏、忌口、运动习惯、健康目标时，可抽取为健康偏好。',
    1.200
);
```

随后：

- 前端记忆筛选项从 `memory_type_definitions` 获取。
- Memory Extractor 根据 `extraction_hint` 识别新类型。
- Retrieval Engine 根据 `retrieval_weight` 参与排序。
- 不需要修改 `profile_memories` 表结构。

### 7.2 新增记忆层

如果未来要新增新的记忆层，例如 Project Memory：

1. 新增独立数据表，例如 `project_memories`。
2. 在 Memory Router 中增加路由规则。
3. 在 Retrieval Engine 中实现该层检索器。
4. 在 Memory Compressor 中定义压缩格式。
5. 在前端记忆管理页面增加展示入口。
6. 如需记录召回日志，扩展 `memory_retrieval_logs.memory_layer`。

新增记忆层比新增 Profile Memory 类型更重，一般只有当生命周期、检索方式或权限模型明显不同时才需要新增层。

## 8. Dreaming 记忆机制

当前系统不仅有 Curator 的单轮复盘，还设计了 Dreaming 机制。Curator 更像“每轮对话后的整理员”，Dreaming 更像“离线睡眠复盘”：在用户空闲、定时任务或记忆膨胀到阈值后，后台主动重放一批历史记忆，进行合并、抽象、遗忘和技能提炼。

### 8.1 Dreaming 触发时机

Dreaming 不阻塞用户对话，只在后台执行。

触发方式：

- `scheduled`：每天或每周定时运行。
- `idle`：用户一段时间没有继续对话时运行。
- `manual`：用户点击“整理我的记忆”主动触发。
- `threshold`：某类记忆数量、Token 总量或重复率超过阈值后触发。

### 8.2 Dreaming 处理对象

Dreaming 主要读取：

- 最近一段时间的 Episodic Memory。
- 重复或冲突的 Profile Memory。
- 成功率较高或较低的 Procedural Memory。
- 被频繁召回或长期未使用的 Semantic Memory。

默认只处理当前用户的私有记忆。公共技能和公共知识库的 Dreaming 需要管理员或显式授权，避免用户私有信息污染公共资源。

### 8.3 Dreaming 产出类型

Dreaming 不一定直接修改记忆。它先生成 `dreaming_insights`，再由系统规则或用户确认后应用。

产出类型：

- `merge`：发现重复记忆，建议合并。
- `promote`：发现高价值 episode，建议提升为 Profile Memory 或 Procedural Memory。
- `archive`：发现长期无用或低价值记忆，建议归档。
- `skill_candidate`：发现可复用流程，建议生成新技能。
- `profile_update`：发现用户画像需要更新或修正。
- `semantic_link`：发现知识库文档与某类任务强相关，建立关联。

### 8.4 Dreaming 数据结构

存储表：

```text
dreaming_sessions
dreaming_insights
```

`dreaming_sessions` 记录一次离线复盘批次，包括触发方式、范围、状态、输入摘要和输出摘要。

`dreaming_insights` 记录该批次产生的具体洞察，包括来源记忆、目标记忆、建议内容、置信度和应用状态。

### 8.5 Dreaming 流程

1. 创建 `curator_jobs`，类型为 `dreaming` 或 `memory_consolidation`。
2. 创建 `dreaming_sessions`，记录本轮复盘范围。
3. Retrieval Engine 召回待复盘记忆。
4. Memory Compressor 压缩输入，去除闲聊和重复片段。
5. LLM 或规则引擎生成候选洞察。
6. 将候选写入 `dreaming_insights`，状态为 `proposed`。
7. 对低风险操作自动应用，例如归档重复 episode。
8. 对高风险操作等待用户确认，例如修改用户画像或发布公共技能。
9. 更新相关记忆的 `status`、`importance`、技能版本或 embedding 任务。

### 8.6 Dreaming 与普通检索的区别

普通检索发生在用户提问时，目标是“为当前问题找上下文”。

Dreaming 发生在后台，目标是“让记忆库本身变得更好”：

- 减少重复记忆。
- 提升重要记忆。
- 归档低价值历史。
- 从历史成功经验中沉淀技能。
- 发现记忆之间的隐含关联。

### 8.7 Dreaming 安全边界

- Dreaming 默认只处理当前用户私有记忆。
- 不把 Profile Memory 或 Episodic Memory 自动发布为公共资源。
- 涉及删除、公共发布、敏感信息改写的 insight 必须确认后应用。
- `dreaming_insights` 保留来源引用，方便用户追溯为什么系统提出该建议。
- 所有 Dreaming 结果都受 `user_id` 隔离。

## 9. 一轮对话的记忆流水线

1. 用户发送消息。
2. 后端保存 `messages`。
3. Memory Router 判断任务类型。
4. 加载当前会话 Working Memory。
5. 检索相关 Profile Memory。
6. 匹配可能可用的 Procedural Memory。
7. 检索相似 Episodic Memory。
8. 如问题涉及文档或知识，检索 Semantic Memory。
9. Memory Compressor 压缩和截断召回片段。
10. Context Engine 按优先级组装 Prompt。
11. LLM 推理并可能触发工具调用。
12. 保存最终消息、工具调用和 memory retrieval log。
13. Curator 异步生成 episode 摘要、抽取画像记忆、沉淀技能或更新 embedding。

## 10. Prompt 加载优先级

Prompt 中的记忆优先级：

1. Working Memory：当前约束和任务状态。
2. Profile Memory：长期用户偏好和用户画像。
3. Procedural Memory：可复用技能流程。
4. Episodic Memory：历史相似经历。
5. Semantic Memory：外部知识库参考资料。

这样可以避免外部知识覆盖用户偏好，也避免历史闲聊影响当前任务。

## 11. 用户隔离与安全

隔离策略：

- 所有记忆表都包含 `user_id`。
- Working Memory 额外绑定 `conversation_id`。
- Profile Memory、Working Memory、Episodic Memory 默认只允许当前用户访问。
- Semantic Memory 的 document 和 chunk 绑定 `user_id`，同时支持 `visibility = public`。
- Procedural Memory 的 skill 绑定 `user_id`，同时支持 `visibility = public`。
- 后端查询私有记忆时必须始终带 `user_id` 条件。
- 检索公共技能和公共知识库时，只允许读取 `visibility = public` 且 `status = active` 的记录。
- Memory Retrieval Log 记录每轮召回来源，方便排查越权或误召回。

安全策略：

- 敏感内容可在 `metadata` 中标记敏感级别。
- 工具不能直接写记忆，必须通过 MemoryManager。
- 删除接口应支持按记忆层删除、按会话删除、按时间段删除。
- 日志中避免输出完整密钥、账号、隐私文本。
- 用户发布公共技能或公共知识库前，需要经过显式确认。
- Dreaming 产生的高风险修改建议先进入 `proposed` 状态，不直接应用。

检索范围示例：

```sql
-- Semantic Memory 召回范围：当前用户私有知识 + 全局公共知识
WHERE status = 'active'
  AND (
    user_id = :current_user_id
    OR visibility = 'public'
  )
```

## 12. Semantic Memory Embedding 策略

Semantic Memory 最适合做 embedding，因为它存的是外部知识切片，天然适合 RAG。

推荐策略：

- 文档导入后，如果用户开启 embedding，则异步生成 embedding，不阻塞用户。
- chunk 粒度控制在 300 到 800 tokens。
- 检索时先限定可见范围，再做向量 top-k，避免私有知识被跨用户召回。
- 检索结果进入 Prompt 前做摘要或截断。
- 文档更新后重新生成 content hash，只对变化 chunk 重算 embedding。
- 公共知识库的 embedding 可复用，减少每个用户重复向量化的成本。

MVP 阶段建议实现：

- `knowledge_documents`
- `knowledge_chunks`
- embedding 成本开关
- 开启 embedding 时生成并写入 `embedding vector(1024)`
- 开启 embedding 时使用 pgvector top-k 召回
- 简单关键词检索

增强阶段再实现：

- 混合检索
- 重排序
- 公共知识库 embedding 复用

## 13. 和普通记忆表的区别

普通单表记忆只能回答“用户有什么偏好”。分层记忆能回答更多问题：

- 当前任务做到哪一步：Working Memory。
- 用户长期喜欢什么：Profile Memory。
- 以前类似问题怎么处理：Episodic Memory。
- 这类任务有没有成熟流程：Procedural Memory。
- 外部资料里相关内容是什么：Semantic Memory。
- 哪些知识或技能可多人复用：Public Semantic/Procedural Memory。

这也是 FreeDinnerAgent 区别于普通聊天机器人的核心设计亮点。

## 14. 当前后端 MVP 实现范围

当前已落地：

- 数据库已包含 Working、Profile、Episodic、Procedural、Semantic、Dreaming、Curator 相关表结构。
- `memory_type_definitions` 已支持新增 Profile Memory 类型，不需要修改 `profile_memories` 表结构。
- Profile Memory 已支持类型列表、手动写入、列表和关键词检索。
- Working Memory 已支持按 `conversation_id + memory_key` 覆盖写入和按会话加载。
- Semantic Memory 已支持文档写入、切片、公共/私有可见性、关键词检索、embedding 开关和 pgvector 召回。
- 新增 `internal/memory.Manager`，统一输出 `Memory Chunk`，负责基础 Memory Router、Working/Profile/Semantic 检索、去重排序、Token 裁剪和 `memory_retrieval_logs` 写入。
- 新增 `GET /api/v1/memory-context`，用于预览当前问题会召回哪些记忆块，方便调试 Prompt 上下文。

当前后端进展：

- Agent 对话流程已自动调用 MemoryManager 构建 Prompt，支持 working/profile/semantic memory 注入。
- 成功 Agent Turn 会自动写入 `episodes`，并创建 `curator_jobs.episode_summary`。
- Procedural Memory/Skills 已支持 light disclosure 匹配和 Context Builder 注入。
- Dreaming 已有规则版执行器，可创建 `dreaming_sessions` 和 `dreaming_insights`。

当前仍留给后续 Curator 阶段：

- episode embedding 和相似 episode 向量检索尚未接入。
- 从 episodes 自动沉淀新 `skills/skill_versions/skill_disclosure_sections` 尚未实现。
- Dreaming insight 的应用流程、用户确认和自动合并尚未实现。
- Semantic Memory embedding 当前是同步生成的 MVP；后续应改成 curator 异步任务，只对变更 chunk 重算。
