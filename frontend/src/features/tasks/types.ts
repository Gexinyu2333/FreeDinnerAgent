export type ScheduledAgentJob = {
  id: string;
  user_id: string;
  agent_config_id: string | null;
  title: string;
  description: string | null;
  job_type: string;
  schedule_kind: string;
  cron_expr: string | null;
  timezone: string;
  run_at_local_time: string | null;
  weekdays: number[];
  prompt_template: string;
  context_policy: Record<string, unknown>;
  tool_policy: Record<string, unknown>;
  delivery_channel: string;
  visibility: string;
  status: "active" | "paused" | "deleted" | string;
  last_run_at: string | null;
  next_run_at: string | null;
  failure_count: number;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
};

export type ScheduledAgentJobRun = {
  id: string;
  user_id: string;
  scheduled_job_id: string;
  conversation_id: string | null;
  agent_turn_id: string | null;
  status: string;
  trigger_reason: string;
  input_snapshot: Record<string, unknown>;
  output_summary: string | null;
  error_message: string | null;
  scheduled_for: string;
  started_at: string | null;
  finished_at: string | null;
  created_at: string;
};

export type ScheduledJobTemplate = {
  id: string;
  title: string;
  description: string;
  job_type: string;
  schedule_kind: string;
  timezone: string;
  run_at_local_time: string;
  weekdays: number[];
  prompt_template: string;
  context_policy: Record<string, unknown>;
  tool_policy: Record<string, unknown>;
  delivery_channel: string;
};

export type CreateScheduledJobInput = {
  title: string;
  description?: string | null;
  job_type?: string;
  schedule_kind?: string;
  cron_expr?: string | null;
  timezone?: string;
  run_at_local_time?: string | null;
  weekdays?: number[];
  prompt_template: string;
  context_policy?: Record<string, unknown>;
  tool_policy?: Record<string, unknown>;
  delivery_channel?: string;
  visibility?: string;
  metadata?: Record<string, unknown>;
};

export type RunNowResult = {
  job: ScheduledAgentJob;
  run: ScheduledAgentJobRun;
  conversation: {
    id: string;
    title: string;
  };
  message: {
    id: string;
    content: string;
  };
};
