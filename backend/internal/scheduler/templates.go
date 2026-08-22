package scheduler

import "encoding/json"

func DefaultTemplates() []JobTemplate {
	return []JobTemplate{
		{
			ID:              "daily_brief",
			Title:           "每日简报",
			Description:     "工作日早上汇总任务、记忆和重要事项。",
			JobType:         "daily_brief",
			ScheduleKind:    "weekly",
			Timezone:        "Asia/Shanghai",
			RunAtLocalTime:  "08:00:00",
			Weekdays:        []int32{1, 2, 3, 4, 5},
			PromptTemplate:  "请根据我的任务、记忆和最近对话生成今天的个人简报。",
			ContextPolicy:   json.RawMessage(`{"include_memory":true,"include_tasks":true,"include_calendar":false,"include_email":false,"max_context_tokens":6000}`),
			ToolPolicy:      json.RawMessage(`{"allow_tools":true,"allowed_tools":["list_tasks","search_memory"],"requires_approval_for_write":true}`),
			DeliveryChannel: "in_app",
		},
		{
			ID:              "weekly_review",
			Title:           "每周回顾",
			Description:     "每周五整理完成事项、阻塞问题和下周计划。",
			JobType:         "weekly_review",
			ScheduleKind:    "weekly",
			Timezone:        "Asia/Shanghai",
			RunAtLocalTime:  "16:00:00",
			Weekdays:        []int32{5},
			PromptTemplate:  "请总结我本周的任务、对话和关键进展，并提炼下周计划。",
			ContextPolicy:   json.RawMessage(`{"include_memory":true,"include_tasks":true,"include_calendar":false,"include_email":false,"max_context_tokens":8000}`),
			ToolPolicy:      json.RawMessage(`{"allow_tools":true,"allowed_tools":["list_tasks","search_memory","save_profile_memory"],"requires_approval_for_write":true}`),
			DeliveryChannel: "in_app",
		},
		{
			ID:              "follow_up_monitor",
			Title:           "跟进监控",
			Description:     "工作日上午检查未跟进事项，只生成建议和草稿。",
			JobType:         "follow_up_monitor",
			ScheduleKind:    "weekly",
			Timezone:        "Asia/Shanghai",
			RunAtLocalTime:  "09:00:00",
			Weekdays:        []int32{1, 2, 3, 4, 5},
			PromptTemplate:  "请检查最近任务和对话中是否存在需要我关注的未跟进事项，给出建议但不要自动创建大量任务。",
			ContextPolicy:   json.RawMessage(`{"include_memory":true,"include_tasks":true,"include_calendar":false,"include_email":false,"max_context_tokens":6000}`),
			ToolPolicy:      json.RawMessage(`{"allow_tools":true,"allowed_tools":["list_tasks","search_memory","create_task"],"requires_approval_for_write":true}`),
			DeliveryChannel: "in_app",
		},
	}
}
