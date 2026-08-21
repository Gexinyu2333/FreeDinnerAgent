# FreeDinnerAgent Agent Loop 与可靠性设计

## 1. 设计目标

Agent loop 负责把一次用户输入执行成一次完整的 Agent Turn。它不只是“调用一次大模型”，而是一个可中断、可观测、可恢复的执行循环。

核心目标：

- 采用经典 ReAct 思路：Reason、Act、Observe、Finalize。
- 每轮执行有最大步数，避免模型陷入无限工具调用。
- LLM 输出必须经过结构化校验，不能直接信任。
- 工具失败、输出格式错误、上下文过长、模型不可用时都有兜底路径。
- 前端可以展示执行过程，后端可以追踪失败原因。

## 2. 为什么用 ReAct

ReAct 适合本项目，因为 FreeDinnerAgent 同时需要：

- 根据用户输入推断意图。
- 检索记忆或知识库。
- 调用任务、摘要、导入文档等工具。
- 根据工具观察结果继续推理。
- 最后生成面向用户的自然语言回复。

但这里不建议裸用 ReAct。裸 ReAct 容易出现三类问题：

- 一直循环调用工具。
- 输出的 tool call 参数不符合 schema。
- 工具失败后模型继续假装成功。

所以本项目采用“Bounded ReAct + Agent Harness”。

```text
User Message
  ↓
Create Agent Turn
  ↓
Build Context
  ↓
ReAct Loop
  ├─ Reason
  ├─ Validate
  ├─ Act
  ├─ Observe
  └─ Decide Next Step
  ↓
Finalize Answer
  ↓
Persist Messages / Logs / Memories
```

## 3. Agent Harness

Agent Harness 是 loop 外面的执行外壳，参考 OpenAI Codex as a platform 中 platform/harness 的思想：宿主应用负责提供上下文、工具、审批、事件流、持久化和运行边界，模型只负责在这些约束里做决策。

后端建议拆成这些组件：

```text
internal/agent/
  runner.go          # 组织一次 Agent Turn
  loop.go            # Bounded ReAct loop
  validator.go       # LLM 输出校验与修复
  fallback.go        # 重试、降级和兜底
  prompt_builder.go  # 系统提示词、记忆、工具、上下文拼装

internal/harness/
  events.go          # agent_events 事件流
  turn_store.go      # agent_turns 持久化
  cancellation.go    # 取消与超时
```

Harness 管理：

- `agent_turns`：一次用户输入到助手回复的完整生命周期。
- `agent_events`：流式事件，用于前端展示过程。
- `agent_loop_steps`：ReAct 每一步记录。
- `llm_output_validations`：LLM 输出校验结果。
- `agent_fallback_events`：重试、降级和兜底记录。

## 4. Loop 状态机

一次 Agent Turn 的状态：

```text
pending -> running -> success
                  ├-> waiting_approval
                  ├-> failed
                  └-> cancelled
```

ReAct step 的状态：

```text
reason -> act -> observe -> reason -> ... -> finalize
```

每个 step 写入 `agent_loop_steps`：

- `step_no`：第几步。
- `step_type`：`reason`、`act`、`observe`、`finalize`、`repair`。
- `thought_summary`：简短推理摘要，不保存隐藏思维链。
- `action_type`：`tool_call`、`memory_search`、`ask_user`、`final_answer`。
- `action_ref_id`：关联工具调用、检索日志或消息。
- `observation`：工具或检索返回的压缩观察。
- `status`：执行状态。

注意：系统只保存可审计的推理摘要，不保存完整 chain-of-thought。前端展示“我正在检索记忆”“我需要调用任务工具”这种解释性状态即可。

## 5. 单轮执行流程

```text
1. 保存用户消息。
2. 创建 agent_turns，状态 pending。
3. 更新为 running，写入 turn_started 事件。
4. Context Builder 拼装系统提示词、记忆、工具 schema、最近对话和摘要。
5. 写入 context_build_logs 和 context_built 事件。
6. 进入 ReAct loop。
7. LLM 生成下一步动作或最终回复。
8. Validator 校验输出格式、工具参数、安全边界和记忆写入策略。
9. 如果校验失败，进入 repair 或 retry。
10. 如果动作为工具调用，Tool Executor 执行并返回 observation。
11. 如果需要用户审批，turn 状态进入 waiting_approval。
12. 如果达到终止条件，生成最终回复。
13. 保存助手消息、LLM usage、工具日志、记忆召回日志。
14. 更新 agent_turns 为 success 或 failed。
15. 异步触发 Curator / Dreaming。
```

## 6. 终止条件

Loop 必须有明确终止条件：

- LLM 输出 `final_answer`。
- 达到 `user_agent_configs.max_loop_steps`。
- 用户取消。
- 工具审批被拒绝。
- 连续校验失败超过 `llm_retry_limit`。
- 当前 token 预算不足，且压缩后仍无法继续。
- 命中不可恢复错误。

达到最大步数时，不直接报错。优先生成一个保守总结：

```text
我已经完成了可确认的部分，但剩余步骤需要更多信息或稍后重试。
```

## 7. LLM 输出校验

LLM 输出分成两类：

- 最终回复：自然语言 answer。
- 动作请求：结构化 action，例如 tool call、memory write、ask user。

所有结构化输出都必须校验：

- JSON 是否可解析。
- action type 是否在允许范围内。
- tool name 是否存在并启用。
- tool arguments 是否符合 JSON Schema。
- 是否越权访问其他用户资源。
- memory write 是否满足写入策略。
- final answer 是否含有明显错误状态，例如工具失败却声称成功。

校验结果写入：

```text
llm_output_validations
```

## 8. 修复、重试与降级

### 8.1 修复优先

如果只是格式问题，先走 repair，不重新跑完整上下文。

典型情况：

- JSON 多了注释。
- tool 参数字段名错误。
- 枚举值不合法。
- final answer 缺少必要字段。

修复 prompt 只包含：

- 原始错误输出。
- schema 错误信息。
- 期望输出格式。

修复结果仍然必须再次校验。

### 8.2 有限重试

如果修复失败，再按 `llm_retry_limit` 重试模型调用。

可重试：

- 临时网络错误。
- provider 5xx。
- rate limit，且退避后可恢复。
- 模型输出结构持续不合格。

不可重试：

- 用户没有配置 API Key。
- 当前工具无权限。
- 审批被拒绝。
- 参数表达了危险或越权操作。

### 8.3 上下文降级

如果 token 超限：

1. 压缩最早对话轮次。
2. 降低 Procedural Skill 披露级别，从 full 到 standard，再到 light。
3. 减少 Semantic Memory chunk 数。
4. 压缩工具 observation。
5. 仍然超限时，要求用户缩小问题范围。

### 8.4 工具降级

工具失败后：

- 只读检索工具失败：改用已有上下文回答，并说明无法完成实时检索。
- 写入类工具失败：生成草稿，不假装已经写入成功。
- 外部 MCP 工具失败：尝试同类内置工具或让用户稍后重试。
- 高风险工具失败：停止执行，要求用户确认下一步。

### 8.5 模型降级

如果用户配置了多个 provider，可以按用户配置降级：

```text
主模型失败 -> 同 provider 备用模型 -> 备用 provider -> 安全回复
```

降级必须记录到 `agent_fallback_events`，并且不能跨用户读取 provider 配置。

## 9. 兜底策略顺序

推荐统一策略：

```text
validate
  ↓ fail
repair_output
  ↓ fail
retry_llm
  ↓ fail
reduce_context
  ↓ fail
provider_fallback
  ↓ fail
ask_clarification
  ↓ fail
safe_final_answer
```

写入：

```text
agent_fallback_events
```

兜底类型包括：

- `retry_llm`
- `repair_output`
- `disable_tool`
- `reduce_context`
- `provider_fallback`
- `ask_clarification`
- `safe_final_answer`
- `handoff_to_user`

## 10. Agent Loop 伪代码

```go
func RunTurn(ctx context.Context, input UserMessage) (*AssistantMessage, error) {
    turn := harness.CreateTurn(input)
    defer harness.FinishTurn(turn)

    contextPack := contextBuilder.Build(input)
    harness.Emit(turn, "context_built", contextPack.Summary())

    for stepNo := 1; stepNo <= config.MaxLoopSteps; stepNo++ {
        step := harness.StartLoopStep(turn, stepNo)

        output, err := llm.Generate(ctx, contextPack)
        if err != nil {
            recovered := fallback.HandleLLMError(ctx, turn, step, err)
            if recovered != nil {
                contextPack = recovered.ContextPack
                continue
            }
            return fallback.SafeFinalAnswer(turn, err), nil
        }

        validation := validator.Validate(output)
        if !validation.Passed {
            repaired, ok := validator.TryRepair(ctx, output, validation)
            if ok {
                output = repaired
            } else if fallback.CanRetry(turn) {
                fallback.Record(turn, step, "retry_llm", validation.Reason)
                continue
            } else {
                return fallback.SafeFinalAnswer(turn, validation.Err), nil
            }
        }

        switch output.Action.Type {
        case "tool_call":
            observation := toolExecutor.Execute(ctx, output.Action)
            contextPack.AddObservation(observation)
        case "memory_search":
            observation := memory.Search(ctx, output.Action)
            contextPack.AddObservation(observation)
        case "ask_user":
            return messages.Clarification(output.Question), nil
        case "final_answer":
            return messages.Assistant(output.Answer), nil
        }
    }

    return fallback.MaxStepFinalAnswer(turn), nil
}
```

## 11. 前端展示

前端可以通过事件流展示轻量执行过程：

```text
正在整理上下文
正在检索相关记忆
正在选择工具
正在调用 create_task
工具执行完成
正在生成回复
```

如果触发兜底：

```text
工具暂时不可用，已切换为草稿回复
上下文过长，已自动压缩早期内容
需要你确认后才能继续执行
```

这些状态来自 `agent_events`，详细排查来自 `agent_loop_steps`、`llm_output_validations` 和 `agent_fallback_events`。

## 12. 和记忆系统的关系

Agent loop 不直接写长期记忆，而是生成候选记忆事件：

- 用户明确表达偏好：候选 Profile Memory。
- 一轮任务完成：候选 Episodic Memory。
- 多次成功流程重复出现：候选 Procedural Skill。
- 长文档导入：候选 Semantic Memory。

MemoryManager 或 Curator 决定是否写入、合并、归档或等待用户确认。这样可以避免模型在一次不稳定输出里直接污染长期记忆。

## 13. 小巧思

### 13.1 Answer Contract

对关键场景要求模型先输出内部结构，再由后端渲染用户可见文本：

```json
{
  "answer_type": "task_created",
  "user_visible_text": "已帮你创建任务。",
  "claims": [
    {
      "type": "tool_success",
      "tool_call_id": "uuid"
    }
  ]
}
```

后端校验 `claims` 是否真实存在，避免模型把失败工具说成成功。

如果工具链没有得到必要证据，模型必须返回结构化失败，而不是编造结果：

```json
{
  "answer_type": "structured_failure",
  "user_visible_text": "我暂时不能确认任务是否创建成功。",
  "missing_evidence": ["tool_success:create_task"],
  "next_action": "retry_or_show_draft"
}
```

### 13.2 Confidence Gate

记忆写入、技能沉淀、公共知识库发布前增加置信度门槛。低置信度内容只作为建议，不自动生效。

### 13.3 Tool Dry Run

对敏感工具先执行 dry run，生成影响预览，用户确认后再真正执行。

### 13.4 Shadow Validator

用更便宜的模型或规则校验器检查最终回复是否引用了不存在的工具结果、是否越权、是否遗漏失败说明。

## 14. 数据库支撑

相关表：

- `user_agent_configs.max_loop_steps`：单轮最大 ReAct 步数。
- `user_agent_configs.llm_retry_limit`：LLM 输出失败重试上限。
- `user_agent_configs.fallback_policy`：用户级兜底策略配置。
- `agent_turns`：一次 Agent 执行生命周期。
- `agent_events`：前端事件流。
- `agent_loop_steps`：Reason、Act、Observe、Finalize 记录。
- `llm_output_validations`：输出校验与修复记录。
- `agent_fallback_events`：重试、降级与兜底记录。
- `tool_call_logs`：工具调用状态。
- `llm_usage_logs`：模型调用成本、耗时和错误。

## 15. MVP 建议

第一版不用做得太重，建议先实现：

1. `max_loop_steps = 6` 的 Bounded ReAct。
2. 工具调用 JSON Schema 校验。
3. 格式错误 repair 一次。
4. LLM 调用失败 retry 两次。
5. 工具 timeout 和幂等。
6. 最大步数兜底回复。
7. `agent_events` 流式展示过程。

后续再加 provider fallback、shadow validator、dry run 和更细的 answer contract。

## 16. 参考资料

- [OpenAI Developers: Codex as a platform](https://developers.openai.com/blog/codex-as-a-platform)
- [OpenAI API: Model guidance](https://developers.openai.com/api/docs/guides/latest-model)
- [OpenAI Agents SDK: Tools](https://openai.github.io/openai-agents-python/tools/)
