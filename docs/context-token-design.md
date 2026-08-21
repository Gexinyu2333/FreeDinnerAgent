# FreeDinnerAgent 多轮上下文与 Token 控制设计

## 1. 设计目标

多轮对话上下文管理的目标是：在有限 Token 窗口内，让 LLM 同时看到当前任务、用户长期偏好、相关历史经验、可用工具、匹配技能和必要的对话上文。

核心原则：

- 当前用户输入和系统安全约束永远优先。
- Working Memory 和明确用户偏好优先于普通历史消息。
- 工具、技能、记忆都按需加载，不全量塞入 Prompt。
- 对话历史超过轮数或 Token 阈值后自动压缩。
- 压缩保留关键实体、决策、工具结果和未完成事项。

## 2. Prompt 组成

一次 LLM 请求的上下文由 Context Engine 统一组装。

```text
Meta System Prompt
  ↓
User Agent Config
  ↓
Safety / Privacy Rules
  ↓
Working Memory
  ↓
Profile Memory
  ↓
Matched Procedural Skills
  ↓
Retrieved Episodic Memory
  ↓
Retrieved Semantic Memory
  ↓
Tool Definitions
  ↓
Compressed Conversation Summary
  ↓
Recent N Turns
  ↓
Current User Input
```

各部分说明：

- Meta System Prompt：系统级元提示词，定义 Agent 边界、记忆使用规则、工具调用规则和隐私要求。
- User Agent Config：用户自己的 Agent 配置，例如 system prompt、输出风格、温度、最大上下文 Token。
- Safety / Privacy Rules：不能泄露 API Key、不能跨用户读取记忆、公共知识不能覆盖用户私有偏好。
- Working Memory：当前会话目标、临时约束、任务进度和工具中间结果。
- Profile Memory：用户长期偏好、事实、目标、关系和习惯。
- Matched Procedural Skills：命中的 ReAct 技能流程，按 light、standard、full 模式加载。
- Retrieved Episodic Memory：相似历史事件摘要。
- Retrieved Semantic Memory：外部知识库或公共知识库片段。
- Tool Definitions：本轮可用工具名称、描述和参数 schema。
- Compressed Conversation Summary：被压缩的早期对话摘要。
- Recent N Turns：最近 N 轮原始对话。
- Current User Input：当前用户输入，永远完整保留。

## 3. 上下文预算分区

每个用户可以在 `user_agent_configs.max_context_tokens` 中配置最大上下文 Token。Context Engine 会将总预算切成多个区间，避免某一类内容挤占全部窗口。

默认预算建议：

```text
系统与安全提示词        10%
Working/Profile Memory  15%
Procedural Skills       15%
Episodic/Semantic Memory 20%
Tool Definitions        10%
历史对话摘要             10%
最近 N 轮对话            15%
当前用户输入              5%+
```

当前用户输入不是硬性 5%，而是最低优先保留项。如果用户输入很长，系统会先压缩历史、记忆和工具结果，而不是截断当前输入。

## 4. 多轮对话窗口策略

系统同时使用“轮数阈值”和“Token 阈值”。

### 4.1 最近 N 轮保留

默认保留最近 N 轮原始对话，例如：

```text
recent_turn_limit = 8
```

最近 N 轮保留原文，因为它们最可能包含当前任务的上下文、指代、省略和未完成约束。

### 4.2 超出轮数压缩

当会话超过 N 轮时：

1. 取超出部分的早期消息。
2. 提取关键事实、决策、工具调用结果、用户约束和未完成事项。
3. 写入或更新会话摘要。
4. 后续 Prompt 只加载摘要，不加载这些早期原文。

摘要格式建议：

```text
Conversation Summary
- 用户目标：
- 已确认约束：
- 关键事实：
- 已执行工具：
- 工具结果：
- 未完成事项：
- 冲突或待确认点：
```

### 4.3 超出 Token 阈值压缩

即使轮数没有超过 N，如果估算 Token 已超过阈值，也会触发压缩。

触发条件：

```text
estimated_prompt_tokens > max_context_tokens * 0.85
```

压缩顺序：

1. 压缩工具结果。
2. 压缩 Semantic Memory 片段。
3. 降级 Procedural Skill 加载模式，例如 full -> standard -> light。
4. 压缩 Episodic Memory。
5. 压缩更早的对话轮次。
6. 减少召回记忆 top-k。

不优先压缩：

- 当前用户输入。
- Meta System Prompt。
- 安全规则。
- 当前任务的 Working Memory。
- 用户明确要求保留的偏好或约束。

### 4.4 手动压缩当前对话

除了自动压缩，前端可以提供“整理当前对话”按钮，让用户手动压缩当前会话的前几轮。

适用场景：

- 用户感觉当前对话太长，希望提升后续回复速度。
- 当前任务已经进入新阶段，早期探索过程不再需要逐轮保留。
- 用户准备继续一个长期会话，希望先把前文整理成摘要。

默认行为：

1. 保留最近 N 轮原始对话，例如最近 8 轮。
2. 将更早的消息压缩成 `conversation_summaries`。
3. 自动识别并保护 Anchor Messages。
4. 创建一条 `conversation_compression_jobs` 记录，方便前端展示进度。
5. 压缩成功后，后续 Prompt 使用摘要替代早期原文。

用户可选参数：

```text
keep_recent_turns = 8
target_summary_type = turn_window
```

安全边界：

- 不删除原始 `messages`，只是改变上下文加载方式。
- 锚点消息进入结构化摘要，保留来源 message id。
- 压缩结果可在前端查看。
- 用户不满意可以重新触发压缩，旧摘要标记为 `superseded`。

## 5. 摘要金字塔

为了避免反复压缩导致信息丢失，系统采用摘要金字塔。

```text
Raw Messages
  ↓ 每 8 轮压缩
Turn Summary
  ↓ 每 5 个 Turn Summary 合并
Session Summary
  ↓ 会话结束后复盘
Episodic Memory
  ↓ Dreaming 离线整理
Profile / Procedural Memory
```

好处：

- 短期细节保留在最近 N 轮。
- 中期上下文保留在会话摘要。
- 长期价值沉淀为 Episodic、Profile 或 Procedural Memory。
- Dreaming 可以继续清理重复摘要和低价值历史。

## 6. 锚点消息保护

不是所有旧消息都应该压缩。系统会标记 Anchor Messages，即使较早也优先保留或高质量摘要。

锚点消息包括：

- 用户明确说“记住”“以后都这样”的消息。
- 包含关键需求变更的消息。
- 包含外部约束的消息，例如截止时间、格式要求、权限限制。
- 重要工具调用结果。
- 用户纠正 Agent 的消息。
- 涉及安全、隐私或 API Key 的消息。

锚点不会被简单丢弃，只会被结构化压缩，并保留来源 `message_id`。

数据库字段：

```text
messages.is_anchor
messages.anchor_reason
```

## 7. 工具与技能的 Token 控制

### 7.1 Tool Definitions

工具定义不全量加载。Tool Router 根据当前任务选择候选工具。

工具加载策略：

- 简单聊天：不加载工具。
- 任务管理：加载 task 相关工具。
- 记忆管理：加载 memory 相关工具。
- 文档问答：加载 semantic memory / document 工具。

每个工具定义只保留：

- 工具名
- 简短描述
- 必需参数 schema
- 关键错误处理规则

### 7.2 Tool Results

工具结果先结构化压缩，再注入上下文。

压缩格式：

```text
Tool Result
- tool:
- status:
- key_result:
- important_fields:
- error:
- next_action:
```

长文本工具结果不直接塞进 Prompt，而是写入 Semantic Memory 或临时 Working Memory，再只注入摘要和引用。

### 7.3 Procedural Skill

技能使用三级加载：

- light：名称 + 适用场景 + 一句话流程。
- standard：输入、输出、核心步骤、风险点。
- full：完整 ReAct 步骤、工具序列和异常分支。

当 Token 紧张时，优先从 full 降级到 standard，再降级到 light。

## 8. 记忆注入控制

分层记忆不是召回多少就注入多少。每一层都有 top-k 和 Token 上限。

建议默认值：

```text
working_memory.max_tokens = 800
profile_memory.top_k = 8
episodic_memory.top_k = 5
procedural_memory.top_k = 3
semantic_memory.top_k = 5
memory_total_budget_ratio = 0.35
```

如果总记忆 Token 超过预算：

1. 保留 Working Memory。
2. 保留高重要度 Profile Memory。
3. 保留直接命中的 Procedural Skill。
4. 对 Semantic Memory 片段做摘要。
5. 丢弃低分 Episodic Memory。
6. 公共知识优先于私有用户偏好被裁剪。

## 9. Token 估算与记录

后端需要在关键表中记录或估算 Token：

- `messages.token_count`
- `session_working_memories.token_count`
- `episodes.token_count`
- `knowledge_chunks.token_count`
- `memory_retrieval_logs.token_count`
- `conversation_summaries.token_count`

Token 估算可以分两阶段：

- MVP：用字符数粗略估算，例如中文约 1.5 到 2 字符一个 token，英文约 4 字符一个 token。
- 增强：接入模型对应 tokenizer，按真实模型计算。

每次请求后记录：

```text
input_tokens
output_tokens
memory_tokens
tool_tokens
summary_tokens
truncated_items
```

这些数据写入：

```text
context_build_logs      # 本轮上下文总体预算、压缩和截断情况
context_build_items     # 本轮 Prompt 具体拼入了哪些内容
llm_usage_logs          # 模型、供应商、输入输出 Token、耗时和状态
```

会话压缩摘要写入：

```text
conversation_summaries
conversation_compression_jobs
```

摘要类型：

- `turn_window`：固定窗口摘要，例如每 8 轮压缩一次。
- `session`：会话级总摘要。
- `handoff`：用于后续新会话或跨会话衔接的摘要。

## 10. 小巧思：上下文体检报告

为了让系统更可解释，前端可以展示本轮“上下文体检报告”。

示例：

```text
本轮上下文
- 最近对话：8 轮
- 使用记忆：6 条
- 使用技能：论文总结流程
- 使用知识库：课程要求文档 3 段
- 已压缩历史：12 轮
- 预计输入 Token：9200 / 12000
```

这个功能有两个价值：

- 用户能理解 Agent 为什么这样回答。
- 开发时能排查 Token 超限、记忆误召回和技能加载过重的问题。

该报告可以直接从 `context_build_logs` 和 `context_build_items` 生成，不需要重新解析 Prompt。

## 11. 小巧思：事实变更检测

当用户说出和旧记忆冲突的信息时，不直接覆盖旧记忆，而是生成待确认变更。

示例：

旧记忆：

```text
用户喜欢简洁回答。
```

新输入：

```text
这次你详细讲，我想看完整推理。
```

处理：

- 当前会话写入 Working Memory：本次回答详细。
- 不删除长期 Profile Memory：用户喜欢简洁回答。
- 如果用户说“以后都详细讲”，再更新 Profile Memory。

这能避免临时要求污染长期偏好。

## 12. 小巧思：遗忘预算

系统不只控制“放进多少上下文”，也控制“保留多少低价值历史”。

Dreaming 可以周期性计算记忆价值：

```text
memory_value = recall_count * usefulness_score * importance - age_penalty
```

低价值记忆进入 `dreaming_insights` 的 `archive` 建议，不直接删除。这样系统会越来越轻，但用户仍有机会恢复或拒绝归档。

## 13. 失败与降级策略

上下文管理失败时不能让对话崩掉。

降级顺序：

1. 禁用 Semantic Memory。
2. 技能加载降级到 light。
3. 减少 Episodic Memory top-k。
4. 压缩工具结果。
5. 只保留最近 4 轮对话。
6. 返回提示：本轮上下文较长，已自动压缩部分历史。

如果用户输入本身超过模型限制：

- 先提示用户内容过长。
- 支持将长文本作为文档写入 Semantic Memory。
- 再基于文档摘要进行问答。
