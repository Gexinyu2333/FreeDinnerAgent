# FreeDinnerAgent 心跳任务与主动助理设计

## 1. 设计目标

心跳任务让 FreeDinnerAgent 不只是被动聊天，而是可以在用户授权后主动执行周期性任务，例如每日简报、每周回顾、跟进监控、定时提醒、内容摘要和社交辅助。

它对应产品里的“已安排的任务”页面：

```text
每日简报   工作日 8:00
每周回顾   星期五 16:00
跟进监控   工作日 9:00
```

## 2. 概念区分

系统里有三类容易混淆的“任务”：

- `tasks`：用户要完成的待办事项，例如“今天下午提交报告”。
- `scheduled_agent_jobs`：会主动触发 Agent 的心跳任务，例如“每天 8 点生成简报”。
- `curator_jobs`：系统内部后台任务，例如记忆整理、Dreaming、文档 embedding。

这三类不要混在一张表里。用户待办强调完成状态；心跳任务强调触发计划和执行结果；后台任务强调系统维护。

## 3. 核心流程

```text
用户创建心跳任务
  ↓
写入 scheduled_agent_jobs
  ↓
Scheduler 扫描 next_run_at 到期任务
  ↓
创建 scheduled_agent_job_runs
  ↓
创建或复用 conversation
  ↓
拉起一次 Agent Turn
  ↓
Context Builder 按任务策略加载记忆、待办、知识库和工具
  ↓
Bounded ReAct Loop 执行
  ↓
结果写入会话、运行记录和必要记忆
  ↓
计算下一次 next_run_at
```

心跳任务本质上是“定时生成一条系统触发的用户意图”，然后复用已有 Agent loop。

## 4. 数据库存储

核心表：

```text
scheduled_agent_jobs
scheduled_agent_job_runs
```

`scheduled_agent_jobs` 保存安排本身：

- `job_type`：`daily_brief`、`weekly_review`、`follow_up_monitor`、`reminder`、`content_digest`、`social_assist`、`custom`。
- `schedule_kind`：`once`、`daily`、`weekly`、`monthly`、`cron`。
- `cron_expr`：复杂周期表达式。
- `timezone`：用户时区。
- `run_at_local_time`：本地触发时间。
- `weekdays`：周几触发。
- `prompt_template`：触发时给 Agent 的任务说明。
- `context_policy`：本任务允许加载哪些上下文。
- `tool_policy`：本任务允许使用哪些工具、是否需要审批。
- `delivery_channel`：站内通知、邮件或 webhook。
- `next_run_at`：下一次触发时间。

`scheduled_agent_job_runs` 保存每次运行：

- `scheduled_job_id`：属于哪个心跳任务。
- `conversation_id`：结果写入哪个会话。
- `agent_turn_id`：对应哪次 Agent 执行。
- `status`：pending、running、success、failed、cancelled、skipped。
- `input_snapshot`：本次执行时使用的输入快照。
- `output_summary`：结果摘要。
- `scheduled_for`：计划触发时间。

## 5. 第一批心跳任务模板

### 5.1 每日简报

```text
工作日 8:00
```

目标：

- 汇总今天日程。
- 汇总未完成任务。
- 提醒即将到期事项。
- 可选读取知识库或外部工具。

默认上下文：

- Profile Memory：工作偏好、关注项目。
- Working Memory：近期目标。
- Tasks：open/doing 且 due_at 接近的任务。
- Calendar MCP：用户启用后才加载。

默认工具：

- `list_tasks`
- `search_memory`
- `search_semantic_memory`
- 可选 `calendar_list_events`

### 5.2 每周回顾

```text
星期五 16:00
```

目标：

- 总结本周完成事项。
- 标记阻塞问题。
- 提炼下周计划。
- 生成一条 Episodic Memory 或 Procedural Skill 候选。

默认上下文：

- 最近一周 conversations 摘要。
- 本周完成和取消的 tasks。
- 高重要度 Episodic Memory。

默认工具：

- `list_tasks`
- `search_memory`
- `summarize_text`
- `save_profile_memory`

### 5.3 跟进监控

```text
工作日 9:00
```

目标：

- 检查最近任务和会话里是否有未跟进事项。
- 发现需要用户关注的任务。
- 不确定时生成建议，不直接创建大量待办。

默认上下文：

- 最近几天对话摘要。
- open/doing tasks。
- todo 类型 Profile Memory。

默认工具：

- `list_tasks`
- `search_memory`
- `create_task`

写入类工具默认需要用户确认，避免 Agent 自动制造过多任务。

## 6. 前端设计

新增页面：已安排的任务。

页面区域：

- 搜索框：搜索已安排任务。
- 建议：展示系统推荐模板，如每日简报、每周回顾、跟进监控。
- 我的安排：展示用户已启用的心跳任务。
- 运行记录：查看最近执行结果、失败原因和关联会话。

心跳任务卡片字段：

- 图标
- 标题
- 周期
- 时间
- 简短说明
- 状态开关
- 下一次运行时间
- 立即运行按钮

创建流程：

```text
选择模板
  ↓
设置周期、时间、上下文范围、工具权限
  ↓
预览 Agent 将执行什么
  ↓
保存
```

## 7. Agent 如何执行

心跳任务触发时，系统构造一条内部消息：

```json
{
  "source": "scheduled_agent_job",
  "job_type": "daily_brief",
  "instruction": "请生成今天的个人简报。",
  "scheduled_for": "2026-08-21T08:00:00+08:00"
}
```

这条消息不会伪装成用户手打消息，而是以 `system` 或专门的 `scheduled` 来源进入 Agent Harness。

Context Builder 根据 `context_policy` 加载：

- 用户 Agent 配置。
- 任务模板 prompt。
- 相关记忆。
- 相关待办。
- 用户已授权的 MCP、Skills、Tools。
- 最近摘要。

然后进入正常的 Bounded ReAct loop。

## 8. 权限与安全

心跳任务是主动执行，所以权限要比普通聊天更谨慎。

建议规则：

- 默认只允许只读工具。
- 写入类工具需要 `requires_approval_for_write = true`。
- 外部 MCP 默认需要用户启用并授权。
- 社交辅助类任务不能自动发送消息，只能生成草稿。
- 删除、支付、外部发布等破坏性操作不允许定时自动执行。
- 每次运行都写入 `scheduled_agent_job_runs` 和 `agent_events`。

这样能避免“用户睡觉时 Agent 自动做了不该做的事”。

## 9. 失败处理

失败分几类：

- LLM 失败：按 Agent reliability 策略 repair、retry、fallback。
- 工具失败：返回部分结果并记录失败工具。
- 时间错过：创建 `trigger_reason = system_recovery` 的补偿运行，或跳过过期任务。
- 连续失败：增加 `failure_count`，达到阈值后暂停任务。
- 上下文不足：生成澄清问题，等待用户补充配置。

推荐策略：

```text
失败 1 次：记录并下次继续
连续失败 3 次：标记为需要用户关注
连续失败 5 次：自动暂停
```

## 10. 和记忆系统的关系

心跳任务可以读记忆，但写记忆要走 MemoryManager。

典型写入：

- 每日简报生成后，保存当天关键状态到 Episodic Memory。
- 每周回顾生成后，提炼用户目标、习惯和长期偏好。
- 跟进监控发现重复流程后，生成 Procedural Skill 候选。
- 用户修改模板后，更新 Profile Memory 中的偏好。

## 11. 能力市场扩展

心跳任务模板也可以成为能力市场的一类模板。

MVP 可以先不新增 marketplace 类型，使用 `scheduled_agent_jobs.visibility = public_template` 表示公共模板。后续如果要完整市场化，可以把 `marketplace_items.item_type` 扩展为：

```text
scheduled_job_template
```

这样用户可以安装别人分享的“日报模板”“健身复盘模板”“论文阅读摘要模板”，再绑定到自己的 Agent。

## 12. MVP 建议

第一版建议实现三种模板：

1. 每日简报：读取任务和记忆，生成站内简报。
2. 每周回顾：总结一周任务和对话摘要。
3. 跟进监控：扫描未完成任务和最近对话，生成建议。

暂时不接真实邮箱和日历，先把外部工具接口留在 `tool_policy` 和 MCP 设计里。这样工程量可控，但产品形态已经完整。
