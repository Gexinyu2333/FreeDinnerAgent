-- FreeDinnerAgent PostgreSQL initial schema.
-- The memory schema follows a Hermes-like layered memory design.
--
-- Semantic vector search requires pgvector.
-- The default embedding dimension is 1024 for BAAI/bge-m3.
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    username VARCHAR(64) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    display_name VARCHAR(128),
    status VARCHAR(32) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled')),
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash TEXT NOT NULL UNIQUE,
    user_agent TEXT,
    ip_address INET,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_model_providers (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(40) NOT NULL CHECK (provider IN ('openai', 'anthropic')),
    display_name VARCHAR(120) NOT NULL,
    chat_base_url TEXT,
    encrypted_chat_api_key TEXT NOT NULL,
    embedding_base_url TEXT,
    encrypted_embedding_api_key TEXT,
    default_chat_model VARCHAR(120) NOT NULL,
    default_embedding_model VARCHAR(120),
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    status VARCHAR(32) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'disabled', 'deleted')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, provider, display_name)
);

CREATE TABLE IF NOT EXISTS user_agent_configs (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(120) NOT NULL DEFAULT '默认助理',
    system_prompt TEXT NOT NULL DEFAULT '你是一个具备长期记忆能力的个人 AI 助理。',
    default_provider_id UUID REFERENCES user_model_providers(id) ON DELETE SET NULL,
    temperature NUMERIC(3, 2) NOT NULL DEFAULT 0.70 CHECK (temperature >= 0 AND temperature <= 2),
    max_context_tokens INTEGER NOT NULL DEFAULT 12000 CHECK (max_context_tokens > 0),
    max_loop_steps INTEGER NOT NULL DEFAULT 6 CHECK (max_loop_steps > 0),
    llm_retry_limit INTEGER NOT NULL DEFAULT 2 CHECK (llm_retry_limit >= 0),
    fallback_policy JSONB NOT NULL DEFAULT '{"repair_output": true, "reduce_context": true, "ask_clarification": true, "safe_final_answer": true}'::jsonb,
    memory_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    tool_use_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    dreaming_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    semantic_memory_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    embedding_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    embedding_cost_policy JSONB NOT NULL DEFAULT '{"mode": "manual", "max_monthly_tokens": 0, "embed_public_knowledge": false}'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, name)
);

CREATE TABLE IF NOT EXISTS conversations (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(200) NOT NULL DEFAULT '新的对话',
    channel VARCHAR(40) NOT NULL DEFAULT 'web',
    status VARCHAR(32) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived', 'deleted')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(32) NOT NULL CHECK (role IN ('user', 'assistant', 'system', 'tool')),
    content TEXT NOT NULL,
    token_count INTEGER NOT NULL DEFAULT 0,
    is_anchor BOOLEAN NOT NULL DEFAULT FALSE,
    anchor_reason VARCHAR(80),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Conversation summaries support long-context compression and the summary pyramid.
CREATE TABLE IF NOT EXISTS conversation_summaries (
    id UUID PRIMARY KEY,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    summary_type VARCHAR(40) NOT NULL CHECK (summary_type IN ('turn_window', 'session', 'handoff')),
    source_message_start_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    source_message_end_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    source_turn_count INTEGER NOT NULL DEFAULT 0,
    summary TEXT NOT NULL,
    token_count INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'superseded', 'archived')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Compression jobs can be triggered automatically or manually by users.
CREATE TABLE IF NOT EXISTS conversation_compression_jobs (
    id UUID PRIMARY KEY,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    trigger_type VARCHAR(40) NOT NULL CHECK (trigger_type IN ('auto_turn_limit', 'auto_token_threshold', 'manual')),
    source_message_start_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    source_message_end_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    keep_recent_turns INTEGER NOT NULL DEFAULT 8,
    target_summary_type VARCHAR(40) NOT NULL DEFAULT 'turn_window' CHECK (target_summary_type IN ('turn_window', 'session', 'handoff')),
    status VARCHAR(32) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'success', 'failed')),
    summary_id UUID REFERENCES conversation_summaries(id) ON DELETE SET NULL,
    original_token_count INTEGER NOT NULL DEFAULT 0,
    compressed_token_count INTEGER NOT NULL DEFAULT 0,
    error_message TEXT,
    requested_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ
);

-- Agent turns and events model the harness-style lifecycle and streamed execution timeline.
CREATE TABLE IF NOT EXISTS agent_turns (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    assistant_message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    agent_config_id UUID REFERENCES user_agent_configs(id) ON DELETE SET NULL,
    provider_id UUID REFERENCES user_model_providers(id) ON DELETE SET NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'waiting_approval', 'success', 'failed', 'cancelled')),
    cancel_requested BOOLEAN NOT NULL DEFAULT FALSE,
    context_build_id UUID,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS agent_events (
    id UUID PRIMARY KEY,
    turn_id UUID NOT NULL REFERENCES agent_turns(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    event_type VARCHAR(60) NOT NULL CHECK (event_type IN ('turn_started', 'context_built', 'loop_step_started', 'loop_step_finished', 'tool_routed', 'tool_call_started', 'tool_call_finished', 'approval_requested', 'approval_resolved', 'llm_validation_failed', 'fallback_triggered', 'message_delta', 'message_completed', 'turn_failed', 'turn_cancelled')),
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    sequence_no INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (turn_id, sequence_no)
);

-- ReAct loop steps record bounded reasoning/action/observation cycles.
CREATE TABLE IF NOT EXISTS agent_loop_steps (
    id UUID PRIMARY KEY,
    turn_id UUID NOT NULL REFERENCES agent_turns(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    step_no INTEGER NOT NULL,
    step_type VARCHAR(40) NOT NULL CHECK (step_type IN ('reason', 'act', 'observe', 'finalize', 'repair')),
    thought_summary TEXT,
    action_type VARCHAR(40) CHECK (action_type IN ('none', 'tool_call', 'memory_search', 'ask_user', 'final_answer')),
    action_ref_id UUID,
    observation TEXT,
    token_count INTEGER NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'success', 'failed', 'skipped')),
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ,
    UNIQUE (turn_id, step_no)
);

CREATE TABLE IF NOT EXISTS llm_output_validations (
    id UUID PRIMARY KEY,
    turn_id UUID NOT NULL REFERENCES agent_turns(id) ON DELETE CASCADE,
    loop_step_id UUID REFERENCES agent_loop_steps(id) ON DELETE SET NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    validation_type VARCHAR(60) NOT NULL CHECK (validation_type IN ('json_schema', 'tool_call_schema', 'safety_policy', 'final_answer_quality', 'memory_write_policy')),
    status VARCHAR(32) NOT NULL CHECK (status IN ('passed', 'failed', 'repaired')),
    failure_reason TEXT,
    repair_prompt TEXT,
    repaired_output TEXT,
    attempt_no INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS agent_fallback_events (
    id UUID PRIMARY KEY,
    turn_id UUID NOT NULL REFERENCES agent_turns(id) ON DELETE CASCADE,
    loop_step_id UUID REFERENCES agent_loop_steps(id) ON DELETE SET NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    fallback_type VARCHAR(60) NOT NULL CHECK (fallback_type IN ('retry_llm', 'repair_output', 'disable_tool', 'reduce_context', 'provider_fallback', 'ask_clarification', 'safe_final_answer', 'handoff_to_user')),
    reason TEXT NOT NULL,
    action_taken TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Working Memory: small session-level memory loaded on every turn.
CREATE TABLE IF NOT EXISTS session_working_memories (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    memory_key VARCHAR(120) NOT NULL,
    memory_value TEXT NOT NULL,
    category VARCHAR(40) NOT NULL CHECK (category IN ('preference', 'constraint', 'task_state', 'tool_result', 'temporary_context')),
    token_count INTEGER NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (conversation_id, memory_key)
);

-- Configurable profile memory types. Adding a new type only requires inserting a row.
CREATE TABLE IF NOT EXISTS memory_type_definitions (
    memory_type VARCHAR(40) PRIMARY KEY,
    display_name VARCHAR(80) NOT NULL,
    description TEXT NOT NULL,
    extraction_hint TEXT NOT NULL,
    retrieval_weight NUMERIC(4, 3) NOT NULL DEFAULT 1.000 CHECK (retrieval_weight > 0),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Profile Memory: long-term user profile facts and preferences.
CREATE TABLE IF NOT EXISTS profile_memories (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    memory_type VARCHAR(40) NOT NULL REFERENCES memory_type_definitions(memory_type),
    scope VARCHAR(40) NOT NULL DEFAULT 'global' CHECK (scope IN ('global', 'project', 'conversation')),
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    evidence TEXT,
    source_message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    confidence NUMERIC(4, 3) NOT NULL DEFAULT 0.800 CHECK (confidence >= 0 AND confidence <= 1),
    importance INTEGER NOT NULL DEFAULT 3 CHECK (importance BETWEEN 1 AND 5),
    status VARCHAR(32) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived', 'deleted')),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO memory_type_definitions (memory_type, display_name, description, extraction_hint, retrieval_weight)
VALUES
    ('preference', '偏好', '用户长期偏好，例如饮食、沟通风格、工具使用习惯。', '用户表达喜欢、不喜欢、希望以后如何处理时，可抽取为偏好。', 1.200),
    ('habit', '习惯', '用户稳定重复的行为模式。', '用户描述经常、通常、每天、每周会做的事情时，可抽取为习惯。', 1.000),
    ('fact', '事实', '关于用户或其环境的稳定事实。', '用户明确说明身份、项目、地点、设备等稳定事实时，可抽取为事实。', 1.000),
    ('relationship', '关系', '用户重要人物或组织关系。', '用户提到家人、朋友、同事、导师及其关系背景时，可抽取为关系。', 0.900),
    ('goal', '目标', '用户长期目标或阶段性目标。', '用户表达想完成、正在追求、计划达成的目标时，可抽取为目标。', 1.100),
    ('todo', '待办', '需要跟踪或提醒的行动项。', '用户表达要做、提醒、截止时间、任务安排时，可抽取为待办。', 1.300),
    ('other', '其他', '暂时无法归类但可能有长期价值的信息。', '仅当信息有复用价值但不属于其他类型时使用。', 0.600)
ON CONFLICT (memory_type) DO NOTHING;

-- Episodic Memory: durable event-level interaction traces for future recall.
CREATE TABLE IF NOT EXISTS episodes (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    assistant_message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    user_input TEXT NOT NULL,
    agent_summary TEXT NOT NULL,
    final_response TEXT NOT NULL,
    task_type VARCHAR(80),
    status VARCHAR(32) NOT NULL CHECK (status IN ('success', 'failed', 'interrupted')),
    importance INTEGER NOT NULL DEFAULT 3 CHECK (importance BETWEEN 1 AND 5),
    token_count INTEGER NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    embedding vector(1024),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS episode_tags (
    episode_id UUID NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
    tag VARCHAR(80) NOT NULL,
    PRIMARY KEY (episode_id, tag)
);

CREATE TABLE IF NOT EXISTS episode_tool_calls (
    id UUID PRIMARY KEY,
    episode_id UUID NOT NULL REFERENCES episodes(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tool_name VARCHAR(80) NOT NULL,
    arguments JSONB NOT NULL DEFAULT '{}'::jsonb,
    result JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(32) NOT NULL CHECK (status IN ('pending', 'success', 'failed', 'fallback')),
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ
);

-- Procedural Memory: reusable skills distilled from repeated successful episodes.
CREATE TABLE IF NOT EXISTS skills (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(160) NOT NULL,
    description TEXT NOT NULL,
    trigger_keywords TEXT[] NOT NULL DEFAULT '{}',
    scenario TEXT,
    visibility VARCHAR(32) NOT NULL DEFAULT 'private' CHECK (visibility IN ('private', 'public')),
    permission_level VARCHAR(40) NOT NULL DEFAULT 'normal' CHECK (permission_level IN ('readonly', 'normal', 'sensitive')),
    status VARCHAR(32) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived', 'deleted')),
    use_count INTEGER NOT NULL DEFAULT 0,
    success_count INTEGER NOT NULL DEFAULT 0,
    failure_count INTEGER NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    embedding vector(1024),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, name)
);

CREATE TABLE IF NOT EXISTS skill_versions (
    id UUID PRIMARY KEY,
    skill_id UUID NOT NULL REFERENCES skills(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    react_steps TEXT NOT NULL,
    tool_sequence JSONB NOT NULL DEFAULT '[]'::jsonb,
    output_template TEXT,
    fallback_strategy TEXT,
    created_from_episode_id UUID REFERENCES episodes(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (skill_id, version)
);

CREATE TABLE IF NOT EXISTS skill_disclosure_sections (
    id UUID PRIMARY KEY,
    skill_version_id UUID NOT NULL REFERENCES skill_versions(id) ON DELETE CASCADE,
    disclosure_level VARCHAR(40) NOT NULL CHECK (disclosure_level IN ('light', 'standard', 'full')),
    title VARCHAR(160) NOT NULL,
    content TEXT NOT NULL,
    token_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (skill_version_id, disclosure_level)
);

-- Semantic Memory: external documents and chunk-level RAG memory.
CREATE TABLE IF NOT EXISTS knowledge_documents (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(240) NOT NULL,
    source_type VARCHAR(40) NOT NULL CHECK (source_type IN ('upload', 'url', 'note', 'manual')),
    source_uri TEXT,
    visibility VARCHAR(32) NOT NULL DEFAULT 'private' CHECK (visibility IN ('private', 'public')),
    content_hash VARCHAR(128),
    status VARCHAR(32) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived', 'deleted')),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS knowledge_chunks (
    id UUID PRIMARY KEY,
    document_id UUID NOT NULL REFERENCES knowledge_documents(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    visibility VARCHAR(32) NOT NULL DEFAULT 'private' CHECK (visibility IN ('private', 'public')),
    chunk_index INTEGER NOT NULL,
    content TEXT NOT NULL,
    token_count INTEGER NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    embedding vector(1024),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (document_id, chunk_index)
);

CREATE TABLE IF NOT EXISTS tasks (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    profile_memory_id UUID REFERENCES profile_memories(id) ON DELETE SET NULL,
    source_scheduled_job_id UUID,
    title VARCHAR(200) NOT NULL,
    description TEXT,
    due_at TIMESTAMPTZ,
    priority VARCHAR(20) NOT NULL DEFAULT 'normal' CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
    source_type VARCHAR(40) NOT NULL DEFAULT 'manual' CHECK (source_type IN ('manual', 'conversation', 'scheduled_job', 'memory_curator')),
    status VARCHAR(32) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'doing', 'done', 'cancelled')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- User-visible heartbeat/scheduled jobs that proactively trigger the Agent.
CREATE TABLE IF NOT EXISTS scheduled_agent_jobs (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    agent_config_id UUID REFERENCES user_agent_configs(id) ON DELETE SET NULL,
    title VARCHAR(200) NOT NULL,
    description TEXT,
    job_type VARCHAR(60) NOT NULL CHECK (job_type IN ('daily_brief', 'weekly_review', 'follow_up_monitor', 'reminder', 'content_digest', 'social_assist', 'custom')),
    schedule_kind VARCHAR(40) NOT NULL CHECK (schedule_kind IN ('once', 'daily', 'weekly', 'monthly', 'cron')),
    cron_expr VARCHAR(120),
    timezone VARCHAR(80) NOT NULL DEFAULT 'Asia/Shanghai',
    run_at_local_time TIME,
    weekdays INTEGER[] NOT NULL DEFAULT '{}',
    prompt_template TEXT NOT NULL,
    context_policy JSONB NOT NULL DEFAULT '{"include_memory": true, "include_tasks": true, "include_calendar": false, "include_email": false, "max_context_tokens": 6000}'::jsonb,
    tool_policy JSONB NOT NULL DEFAULT '{"allow_tools": true, "requires_approval_for_write": true}'::jsonb,
    delivery_channel VARCHAR(40) NOT NULL DEFAULT 'in_app' CHECK (delivery_channel IN ('in_app', 'email', 'webhook')),
    visibility VARCHAR(32) NOT NULL DEFAULT 'private' CHECK (visibility IN ('private', 'public_template')),
    status VARCHAR(32) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'archived', 'deleted')),
    last_run_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ,
    failure_count INTEGER NOT NULL DEFAULT 0,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS scheduled_agent_job_runs (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    scheduled_job_id UUID NOT NULL REFERENCES scheduled_agent_jobs(id) ON DELETE CASCADE,
    conversation_id UUID REFERENCES conversations(id) ON DELETE SET NULL,
    agent_turn_id UUID REFERENCES agent_turns(id) ON DELETE SET NULL,
    status VARCHAR(32) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'success', 'failed', 'cancelled', 'skipped')),
    trigger_reason VARCHAR(60) NOT NULL DEFAULT 'schedule_due' CHECK (trigger_reason IN ('schedule_due', 'manual_run', 'retry', 'system_recovery')),
    input_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    output_summary TEXT,
    error_message TEXT,
    scheduled_for TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Tool registry stores callable tools and their routing/stability metadata.
CREATE TABLE IF NOT EXISTS tool_definitions (
    id UUID PRIMARY KEY,
    owner_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    name VARCHAR(120) NOT NULL UNIQUE,
    namespace VARCHAR(80) NOT NULL DEFAULT 'default',
    display_name VARCHAR(160) NOT NULL,
    description TEXT NOT NULL,
    category VARCHAR(60) NOT NULL CHECK (category IN ('memory', 'task', 'calendar', 'search', 'document', 'system', 'agent', 'channel', 'social', 'other')),
    handler_type VARCHAR(40) NOT NULL CHECK (handler_type IN ('builtin', 'http', 'mcp', 'agent')),
    handler_ref TEXT NOT NULL,
    visibility VARCHAR(32) NOT NULL DEFAULT 'private' CHECK (visibility IN ('private', 'public')),
    permission_level VARCHAR(40) NOT NULL DEFAULT 'normal' CHECK (permission_level IN ('readonly', 'normal', 'sensitive', 'destructive')),
    requires_approval BOOLEAN NOT NULL DEFAULT FALSE,
    timeout_ms INTEGER NOT NULL DEFAULT 10000 CHECK (timeout_ms > 0),
    max_retries INTEGER NOT NULL DEFAULT 1 CHECK (max_retries >= 0),
    retry_backoff_ms INTEGER NOT NULL DEFAULT 300 CHECK (retry_backoff_ms >= 0),
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tool_versions (
    id UUID PRIMARY KEY,
    tool_id UUID NOT NULL REFERENCES tool_definitions(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    parameter_schema JSONB NOT NULL,
    result_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
    change_note TEXT,
    status VARCHAR(32) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'deprecated', 'deleted')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tool_id, version)
);

CREATE TABLE IF NOT EXISTS user_tool_settings (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tool_id UUID NOT NULL REFERENCES tool_definitions(id) ON DELETE CASCADE,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    approval_policy VARCHAR(40) NOT NULL DEFAULT 'sensitive_only' CHECK (approval_policy IN ('never', 'sensitive_only', 'always')),
    encrypted_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, tool_id)
);

CREATE TABLE IF NOT EXISTS mcp_server_definitions (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(160) NOT NULL,
    display_name VARCHAR(200) NOT NULL,
    description TEXT NOT NULL,
    transport_type VARCHAR(40) NOT NULL CHECK (transport_type IN ('stdio', 'http', 'sse')),
    endpoint TEXT,
    command TEXT,
    args JSONB NOT NULL DEFAULT '[]'::jsonb,
    env_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
    visibility VARCHAR(32) NOT NULL DEFAULT 'private' CHECK (visibility IN ('private', 'public')),
    permission_level VARCHAR(40) NOT NULL DEFAULT 'normal' CHECK (permission_level IN ('readonly', 'normal', 'sensitive', 'destructive')),
    status VARCHAR(32) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived', 'deleted')),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, name)
);

CREATE TABLE IF NOT EXISTS user_mcp_server_settings (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mcp_server_id UUID NOT NULL REFERENCES mcp_server_definitions(id) ON DELETE CASCADE,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    encrypted_env JSONB NOT NULL DEFAULT '{}'::jsonb,
    approval_policy VARCHAR(40) NOT NULL DEFAULT 'sensitive_only' CHECK (approval_policy IN ('never', 'sensitive_only', 'always')),
    last_health_status VARCHAR(40) DEFAULT 'unknown' CHECK (last_health_status IN ('unknown', 'healthy', 'unhealthy')),
    last_checked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, mcp_server_id)
);

-- Channel adapters let external chat platforms become full Agent entry points.
CREATE TABLE IF NOT EXISTS channel_provider_definitions (
    id UUID PRIMARY KEY,
    name VARCHAR(80) NOT NULL UNIQUE,
    display_name VARCHAR(160) NOT NULL,
    description TEXT NOT NULL,
    provider_type VARCHAR(40) NOT NULL CHECK (provider_type IN ('qq', 'wechat', 'telegram', 'discord', 'feishu', 'slack', 'custom')),
    adapter_type VARCHAR(40) NOT NULL CHECK (adapter_type IN ('builtin', 'http_webhook', 'websocket', 'mcp_bridge')),
    inbound_modes TEXT[] NOT NULL DEFAULT '{}',
    outbound_modes TEXT[] NOT NULL DEFAULT '{}',
    config_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
    default_policy JSONB NOT NULL DEFAULT '{}'::jsonb,
    visibility VARCHAR(32) NOT NULL DEFAULT 'public' CHECK (visibility IN ('private', 'public')),
    status VARCHAR(32) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'archived', 'deleted')),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS channel_connections (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider_id UUID NOT NULL REFERENCES channel_provider_definitions(id) ON DELETE CASCADE,
    display_name VARCHAR(160) NOT NULL,
    external_account_id VARCHAR(160),
    external_account_name VARCHAR(200),
    encrypted_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(32) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'revoked', 'deleted')),
    last_health_status VARCHAR(40) DEFAULT 'unknown' CHECK (last_health_status IN ('unknown', 'healthy', 'unhealthy')),
    last_event_at TIMESTAMPTZ,
    last_checked_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, provider_id, display_name)
);

CREATE TABLE IF NOT EXISTS channel_policies (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel_connection_id UUID NOT NULL REFERENCES channel_connections(id) ON DELETE CASCADE,
    scope_type VARCHAR(40) NOT NULL CHECK (scope_type IN ('private_chat', 'group_chat', 'all')),
    external_scope_id VARCHAR(160),
    agent_config_id UUID REFERENCES user_agent_configs(id) ON DELETE SET NULL,
    mode VARCHAR(40) NOT NULL DEFAULT 'mention_only' CHECK (mode IN ('disabled', 'silent_listen', 'mention_only', 'keyword', 'auto_reply')),
    trigger_keywords TEXT[] NOT NULL DEFAULT '{}',
    allow_memory_write BOOLEAN NOT NULL DEFAULT TRUE,
    allow_tool_use BOOLEAN NOT NULL DEFAULT TRUE,
    require_approval_for_outbound BOOLEAN NOT NULL DEFAULT TRUE,
    rate_limit_per_minute INTEGER NOT NULL DEFAULT 6 CHECK (rate_limit_per_minute >= 0),
    quiet_hours JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(32) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'deleted')),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (channel_connection_id, scope_type, external_scope_id)
);

CREATE TABLE IF NOT EXISTS external_conversations (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel_connection_id UUID NOT NULL REFERENCES channel_connections(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    external_conversation_id VARCHAR(160) NOT NULL,
    external_conversation_type VARCHAR(40) NOT NULL CHECK (external_conversation_type IN ('private_chat', 'group_chat', 'channel', 'thread')),
    external_title VARCHAR(200),
    last_message_at TIMESTAMPTZ,
    status VARCHAR(32) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'muted', 'archived', 'deleted')),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (channel_connection_id, external_conversation_id)
);

CREATE TABLE IF NOT EXISTS channel_inbox_events (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel_connection_id UUID NOT NULL REFERENCES channel_connections(id) ON DELETE CASCADE,
    external_conversation_id UUID REFERENCES external_conversations(id) ON DELETE SET NULL,
    conversation_id UUID REFERENCES conversations(id) ON DELETE SET NULL,
    message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    event_type VARCHAR(60) NOT NULL CHECK (event_type IN ('message_created', 'message_updated', 'message_deleted', 'reaction_added', 'member_joined', 'member_left', 'system')),
    external_event_id VARCHAR(200),
    external_sender_id VARCHAR(160),
    external_sender_name VARCHAR(200),
    raw_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    normalized_text TEXT,
    should_trigger_agent BOOLEAN NOT NULL DEFAULT FALSE,
    trigger_reason VARCHAR(80),
    status VARCHAR(32) NOT NULL DEFAULT 'received' CHECK (status IN ('received', 'ignored', 'queued', 'processed', 'failed')),
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at TIMESTAMPTZ,
    UNIQUE (channel_connection_id, external_event_id)
);

CREATE TABLE IF NOT EXISTS channel_outbox_messages (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    channel_connection_id UUID NOT NULL REFERENCES channel_connections(id) ON DELETE CASCADE,
    external_conversation_id UUID REFERENCES external_conversations(id) ON DELETE SET NULL,
    conversation_id UUID REFERENCES conversations(id) ON DELETE SET NULL,
    agent_turn_id UUID REFERENCES agent_turns(id) ON DELETE SET NULL,
    reply_to_inbox_event_id UUID REFERENCES channel_inbox_events(id) ON DELETE SET NULL,
    message_type VARCHAR(40) NOT NULL DEFAULT 'text' CHECK (message_type IN ('text', 'markdown', 'image', 'file', 'card')),
    content TEXT NOT NULL,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    requires_approval BOOLEAN NOT NULL DEFAULT TRUE,
    status VARCHAR(32) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'sending', 'sent', 'failed', 'cancelled')),
    external_message_id VARCHAR(200),
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    approved_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS marketplace_items (
    id UUID PRIMARY KEY,
    item_type VARCHAR(40) NOT NULL CHECK (item_type IN ('tool', 'mcp_server', 'skill', 'knowledge_base', 'channel_adapter')),
    ref_id UUID NOT NULL,
    owner_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    visibility VARCHAR(32) NOT NULL DEFAULT 'private' CHECK (visibility IN ('private', 'public')),
    title VARCHAR(200) NOT NULL,
    description TEXT NOT NULL,
    category VARCHAR(80) NOT NULL DEFAULT 'general',
    tags TEXT[] NOT NULL DEFAULT '{}',
    install_count INTEGER NOT NULL DEFAULT 0,
    rating NUMERIC(3, 2),
    status VARCHAR(32) NOT NULL DEFAULT 'listed' CHECK (status IN ('listed', 'unlisted', 'archived')),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (item_type, ref_id)
);

CREATE TABLE IF NOT EXISTS user_capability_installs (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    marketplace_item_id UUID REFERENCES marketplace_items(id) ON DELETE SET NULL,
    capability_type VARCHAR(40) NOT NULL CHECK (capability_type IN ('tool', 'mcp_server', 'skill', 'knowledge_base', 'channel_adapter')),
    capability_ref_id UUID NOT NULL,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    install_source VARCHAR(40) NOT NULL DEFAULT 'marketplace' CHECK (install_source IN ('marketplace', 'private', 'system')),
    installed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, capability_type, capability_ref_id)
);

CREATE TABLE IF NOT EXISTS agent_capability_bindings (
    id UUID PRIMARY KEY,
    agent_config_id UUID NOT NULL REFERENCES user_agent_configs(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    capability_type VARCHAR(40) NOT NULL CHECK (capability_type IN ('tool', 'mcp_server', 'skill', 'knowledge_base', 'channel_adapter')),
    capability_ref_id UUID NOT NULL,
    is_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    load_mode VARCHAR(40) NOT NULL DEFAULT 'auto' CHECK (load_mode IN ('auto', 'light', 'standard', 'full')),
    priority INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (agent_config_id, capability_type, capability_ref_id)
);

CREATE TABLE IF NOT EXISTS tool_router_logs (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    query TEXT NOT NULL,
    candidate_tools JSONB NOT NULL DEFAULT '[]'::jsonb,
    selected_tools JSONB NOT NULL DEFAULT '[]'::jsonb,
    route_reason TEXT,
    risk_level VARCHAR(40) NOT NULL DEFAULT 'normal' CHECK (risk_level IN ('low', 'normal', 'sensitive', 'destructive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS tool_call_logs (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    context_build_id UUID,
    tool_id UUID REFERENCES tool_definitions(id) ON DELETE SET NULL,
    tool_name VARCHAR(120) NOT NULL,
    tool_version INTEGER,
    idempotency_key VARCHAR(160),
    arguments JSONB NOT NULL DEFAULT '{}'::jsonb,
    validated_arguments JSONB NOT NULL DEFAULT '{}'::jsonb,
    result JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(32) NOT NULL CHECK (status IN ('pending', 'running', 'success', 'failed', 'timeout', 'fallback', 'cancelled')),
    error_type VARCHAR(80),
    error_message TEXT,
    attempt_count INTEGER NOT NULL DEFAULT 0,
    duration_ms INTEGER,
    requires_approval BOOLEAN NOT NULL DEFAULT FALSE,
    approved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS tool_approval_requests (
    id UUID PRIMARY KEY,
    tool_call_id UUID NOT NULL REFERENCES tool_call_logs(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    turn_id UUID REFERENCES agent_turns(id) ON DELETE SET NULL,
    approval_reason TEXT NOT NULL,
    risk_level VARCHAR(40) NOT NULL CHECK (risk_level IN ('normal', 'sensitive', 'destructive')),
    proposed_arguments JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(32) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'expired')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ
);

-- User-level Workspace Sandbox stores per-user file and CLI execution boundaries.
CREATE TABLE IF NOT EXISTS user_workspaces (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(32) NOT NULL DEFAULT 'disabled' CHECK (status IN ('disabled', 'provisioning', 'active', 'idle', 'archived', 'destroying', 'destroyed', 'suspended')),
    root_path TEXT NOT NULL,
    sandbox_type VARCHAR(40) NOT NULL DEFAULT 'local_dir' CHECK (sandbox_type IN ('local_dir', 'docker', 'podman', 'nsjail', 'firecracker')),
    network_policy VARCHAR(40) NOT NULL DEFAULT 'disabled' CHECK (network_policy IN ('disabled', 'allowlist', 'full')),
    network_allowlist TEXT[] NOT NULL DEFAULT '{}',
    max_disk_bytes BIGINT NOT NULL DEFAULT 1073741824 CHECK (max_disk_bytes > 0),
    max_file_count INTEGER NOT NULL DEFAULT 5000 CHECK (max_file_count > 0),
    max_single_file_bytes BIGINT NOT NULL DEFAULT 52428800 CHECK (max_single_file_bytes > 0),
    max_command_seconds INTEGER NOT NULL DEFAULT 30 CHECK (max_command_seconds > 0),
    max_stdout_bytes INTEGER NOT NULL DEFAULT 262144 CHECK (max_stdout_bytes > 0),
    max_stderr_bytes INTEGER NOT NULL DEFAULT 262144 CHECK (max_stderr_bytes > 0),
    cpu_limit VARCHAR(80),
    memory_limit_bytes BIGINT CHECK (memory_limit_bytes IS NULL OR memory_limit_bytes > 0),
    last_active_at TIMESTAMPTZ,
    idle_after_seconds INTEGER NOT NULL DEFAULT 604800 CHECK (idle_after_seconds > 0),
    destroy_after_seconds INTEGER NOT NULL DEFAULT 2592000 CHECK (destroy_after_seconds > 0),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id),
    UNIQUE (root_path)
);

CREATE TABLE IF NOT EXISTS workspace_files (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES user_workspaces(id) ON DELETE CASCADE,
    relative_path TEXT NOT NULL CHECK (
        relative_path <> ''
        AND relative_path NOT LIKE '/%'
        AND relative_path NOT LIKE '../%'
        AND relative_path NOT LIKE '%/../%'
    ),
    file_type VARCHAR(40) NOT NULL CHECK (file_type IN ('file', 'directory', 'artifact')),
    size_bytes BIGINT NOT NULL DEFAULT 0 CHECK (size_bytes >= 0),
    content_hash VARCHAR(128),
    mime_type VARCHAR(120),
    created_by VARCHAR(40) NOT NULL DEFAULT 'agent' CHECK (created_by IN ('user', 'agent', 'tool', 'system')),
    status VARCHAR(32) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'deleted', 'archived')),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (workspace_id, relative_path)
);

CREATE TABLE IF NOT EXISTS workspace_command_runs (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES user_workspaces(id) ON DELETE CASCADE,
    conversation_id UUID REFERENCES conversations(id) ON DELETE SET NULL,
    agent_turn_id UUID REFERENCES agent_turns(id) ON DELETE SET NULL,
    tool_call_id UUID REFERENCES tool_call_logs(id) ON DELETE SET NULL,
    command VARCHAR(160) NOT NULL,
    args JSONB NOT NULL DEFAULT '[]'::jsonb,
    working_dir TEXT NOT NULL DEFAULT '.',
    network_policy VARCHAR(40) NOT NULL DEFAULT 'disabled' CHECK (network_policy IN ('disabled', 'allowlist', 'full')),
    status VARCHAR(32) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'success', 'failed', 'timeout', 'cancelled', 'blocked')),
    exit_code INTEGER,
    stdout_preview TEXT,
    stderr_preview TEXT,
    stdout_truncated BOOLEAN NOT NULL DEFAULT FALSE,
    stderr_truncated BOOLEAN NOT NULL DEFAULT FALSE,
    duration_ms INTEGER CHECK (duration_ms IS NULL OR duration_ms >= 0),
    error_message TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS workspace_events (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES user_workspaces(id) ON DELETE CASCADE,
    event_type VARCHAR(60) NOT NULL CHECK (event_type IN ('created', 'enabled', 'disabled', 'provisioned', 'idle', 'archived', 'destroying', 'destroyed', 'suspended', 'resumed', 'file_created', 'file_read', 'file_updated', 'file_deleted', 'command_started', 'command_finished', 'quota_exceeded', 'policy_changed', 'security_blocked')),
    actor_type VARCHAR(40) NOT NULL DEFAULT 'system' CHECK (actor_type IN ('user', 'agent', 'tool', 'system')),
    actor_id UUID,
    file_id UUID REFERENCES workspace_files(id) ON DELETE SET NULL,
    command_run_id UUID REFERENCES workspace_command_runs(id) ON DELETE SET NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS workspace_quota_snapshots (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id UUID NOT NULL REFERENCES user_workspaces(id) ON DELETE CASCADE,
    used_disk_bytes BIGINT NOT NULL DEFAULT 0 CHECK (used_disk_bytes >= 0),
    file_count INTEGER NOT NULL DEFAULT 0 CHECK (file_count >= 0),
    command_count INTEGER NOT NULL DEFAULT 0 CHECK (command_count >= 0),
    active_process_count INTEGER NOT NULL DEFAULT 0 CHECK (active_process_count >= 0),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Records what memory was retrieved for each turn. Useful for debugging and UX transparency.
CREATE TABLE IF NOT EXISTS memory_retrieval_logs (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    memory_layer VARCHAR(40) NOT NULL CHECK (memory_layer IN ('working', 'profile', 'episodic', 'procedural', 'semantic')),
    memory_ref_id UUID NOT NULL,
    score NUMERIC(8, 6),
    token_count INTEGER NOT NULL DEFAULT 0,
    load_mode VARCHAR(40) NOT NULL DEFAULT 'standard' CHECK (load_mode IN ('light', 'standard', 'full')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Records how a prompt context was assembled. This powers the context health report.
CREATE TABLE IF NOT EXISTS context_build_logs (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    agent_config_id UUID REFERENCES user_agent_configs(id) ON DELETE SET NULL,
    provider_id UUID REFERENCES user_model_providers(id) ON DELETE SET NULL,
    max_context_tokens INTEGER NOT NULL,
    estimated_prompt_tokens INTEGER NOT NULL DEFAULT 0,
    system_tokens INTEGER NOT NULL DEFAULT 0,
    memory_tokens INTEGER NOT NULL DEFAULT 0,
    skill_tokens INTEGER NOT NULL DEFAULT 0,
    tool_tokens INTEGER NOT NULL DEFAULT 0,
    summary_tokens INTEGER NOT NULL DEFAULT 0,
    recent_message_tokens INTEGER NOT NULL DEFAULT 0,
    current_input_tokens INTEGER NOT NULL DEFAULT 0,
    recent_turn_count INTEGER NOT NULL DEFAULT 0,
    compressed_turn_count INTEGER NOT NULL DEFAULT 0,
    truncated_item_count INTEGER NOT NULL DEFAULT 0,
    compression_strategy VARCHAR(80),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS context_build_items (
    id UUID PRIMARY KEY,
    context_build_id UUID NOT NULL REFERENCES context_build_logs(id) ON DELETE CASCADE,
    item_type VARCHAR(40) NOT NULL CHECK (item_type IN ('system', 'agent_config', 'safety', 'working_memory', 'profile_memory', 'procedural_skill', 'episodic_memory', 'semantic_memory', 'tool_definition', 'tool_result', 'summary', 'recent_message', 'current_input')),
    ref_id UUID,
    title VARCHAR(200),
    token_count INTEGER NOT NULL DEFAULT 0,
    load_mode VARCHAR(40) NOT NULL DEFAULT 'standard' CHECK (load_mode IN ('light', 'standard', 'full', 'summary')),
    was_compressed BOOLEAN NOT NULL DEFAULT FALSE,
    was_truncated BOOLEAN NOT NULL DEFAULT FALSE,
    priority INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS llm_usage_logs (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    conversation_id UUID REFERENCES conversations(id) ON DELETE SET NULL,
    message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    context_build_id UUID REFERENCES context_build_logs(id) ON DELETE SET NULL,
    provider_id UUID REFERENCES user_model_providers(id) ON DELETE SET NULL,
    provider VARCHAR(40) NOT NULL CHECK (provider IN ('openai', 'anthropic')),
    model VARCHAR(120) NOT NULL,
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    latency_ms INTEGER,
    status VARCHAR(32) NOT NULL CHECK (status IN ('success', 'failed', 'fallback')),
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Dreaming sessions replay and reorganize memories offline, similar to sleep consolidation.
CREATE TABLE IF NOT EXISTS dreaming_sessions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    trigger_type VARCHAR(40) NOT NULL CHECK (trigger_type IN ('scheduled', 'idle', 'manual', 'threshold')),
    scope VARCHAR(40) NOT NULL DEFAULT 'user' CHECK (scope IN ('user', 'conversation', 'project', 'public')),
    status VARCHAR(32) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'success', 'failed')),
    input_summary TEXT,
    output_summary TEXT,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS dreaming_insights (
    id UUID PRIMARY KEY,
    dreaming_session_id UUID NOT NULL REFERENCES dreaming_sessions(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    insight_type VARCHAR(40) NOT NULL CHECK (insight_type IN ('merge', 'promote', 'archive', 'skill_candidate', 'profile_update', 'semantic_link')),
    source_layer VARCHAR(40) NOT NULL CHECK (source_layer IN ('profile', 'episodic', 'procedural', 'semantic')),
    source_ref_ids UUID[] NOT NULL DEFAULT '{}',
    target_layer VARCHAR(40) CHECK (target_layer IN ('profile', 'episodic', 'procedural', 'semantic')),
    target_ref_id UUID,
    content TEXT NOT NULL,
    confidence NUMERIC(4, 3) NOT NULL DEFAULT 0.800 CHECK (confidence >= 0 AND confidence <= 1),
    status VARCHAR(32) NOT NULL DEFAULT 'proposed' CHECK (status IN ('proposed', 'applied', 'rejected')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    applied_at TIMESTAMPTZ
);

-- Background curator jobs for summarization, dreaming, skill distillation, cleanup, and re-embedding.
CREATE TABLE IF NOT EXISTS curator_jobs (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    job_type VARCHAR(60) NOT NULL CHECK (job_type IN ('episode_summary', 'dreaming', 'memory_consolidation', 'skill_distillation', 'memory_cleanup', 'document_embedding', 'memory_embedding')),
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(32) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'success', 'failed')),
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_conversations_user_id_status ON conversations(user_id, status);
CREATE INDEX IF NOT EXISTS idx_user_sessions_user_expires ON user_sessions(user_id, expires_at);
CREATE INDEX IF NOT EXISTS idx_user_model_providers_user_status ON user_model_providers(user_id, status);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_model_providers_one_default
    ON user_model_providers(user_id)
    WHERE is_default = TRUE AND status = 'active';
CREATE INDEX IF NOT EXISTS idx_user_agent_configs_user ON user_agent_configs(user_id);
CREATE INDEX IF NOT EXISTS idx_messages_conversation_id_created_at ON messages(conversation_id, created_at);
CREATE INDEX IF NOT EXISTS idx_messages_conversation_anchor ON messages(conversation_id, is_anchor);
CREATE INDEX IF NOT EXISTS idx_conversation_summaries_conversation_status ON conversation_summaries(conversation_id, status);
CREATE INDEX IF NOT EXISTS idx_conversation_compression_jobs_conversation_status ON conversation_compression_jobs(conversation_id, status);
CREATE INDEX IF NOT EXISTS idx_agent_turns_conversation_status ON agent_turns(conversation_id, status);
CREATE INDEX IF NOT EXISTS idx_agent_events_turn_sequence ON agent_events(turn_id, sequence_no);
CREATE INDEX IF NOT EXISTS idx_agent_loop_steps_turn_step ON agent_loop_steps(turn_id, step_no);
CREATE INDEX IF NOT EXISTS idx_llm_output_validations_turn ON llm_output_validations(turn_id);
CREATE INDEX IF NOT EXISTS idx_agent_fallback_events_turn ON agent_fallback_events(turn_id);
CREATE INDEX IF NOT EXISTS idx_working_memory_conversation ON session_working_memories(conversation_id, category);
CREATE INDEX IF NOT EXISTS idx_memory_type_definitions_active ON memory_type_definitions(is_active);
CREATE INDEX IF NOT EXISTS idx_profile_memories_user_type_status ON profile_memories(user_id, memory_type, status);
CREATE INDEX IF NOT EXISTS idx_episodes_user_task_status ON episodes(user_id, task_type, status);
CREATE INDEX IF NOT EXISTS idx_episodes_created_at ON episodes(created_at);
CREATE INDEX IF NOT EXISTS idx_episode_tags_tag ON episode_tags(tag);
CREATE INDEX IF NOT EXISTS idx_episode_tool_calls_episode ON episode_tool_calls(episode_id);
CREATE INDEX IF NOT EXISTS idx_skills_user_visibility_status ON skills(user_id, visibility, status);
CREATE INDEX IF NOT EXISTS idx_skills_trigger_keywords ON skills USING GIN (trigger_keywords);
CREATE INDEX IF NOT EXISTS idx_skill_disclosure_sections_version_level ON skill_disclosure_sections(skill_version_id, disclosure_level);
CREATE INDEX IF NOT EXISTS idx_knowledge_documents_user_visibility_status ON knowledge_documents(user_id, visibility, status);
CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_document ON knowledge_chunks(document_id, chunk_index);
CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_user_visibility ON knowledge_chunks(user_id, visibility);
CREATE INDEX IF NOT EXISTS idx_tasks_user_id_status_due_at ON tasks(user_id, status, due_at);
CREATE INDEX IF NOT EXISTS idx_scheduled_agent_jobs_user_status_next_run ON scheduled_agent_jobs(user_id, status, next_run_at);
CREATE INDEX IF NOT EXISTS idx_scheduled_agent_job_runs_job_created ON scheduled_agent_job_runs(scheduled_job_id, created_at);
CREATE INDEX IF NOT EXISTS idx_scheduled_agent_job_runs_user_status ON scheduled_agent_job_runs(user_id, status);
CREATE INDEX IF NOT EXISTS idx_tool_definitions_namespace_enabled ON tool_definitions(namespace, is_enabled);
CREATE INDEX IF NOT EXISTS idx_tool_definitions_category_enabled ON tool_definitions(category, is_enabled);
CREATE INDEX IF NOT EXISTS idx_tool_definitions_owner_visibility ON tool_definitions(owner_user_id, visibility);
CREATE INDEX IF NOT EXISTS idx_tool_versions_tool_status ON tool_versions(tool_id, status);
CREATE INDEX IF NOT EXISTS idx_user_tool_settings_user_enabled ON user_tool_settings(user_id, is_enabled);
CREATE INDEX IF NOT EXISTS idx_mcp_server_definitions_visibility_status ON mcp_server_definitions(visibility, status);
CREATE INDEX IF NOT EXISTS idx_user_mcp_server_settings_user_enabled ON user_mcp_server_settings(user_id, is_enabled);
CREATE INDEX IF NOT EXISTS idx_channel_provider_definitions_type_status ON channel_provider_definitions(provider_type, status);
CREATE INDEX IF NOT EXISTS idx_channel_connections_user_status ON channel_connections(user_id, status);
CREATE INDEX IF NOT EXISTS idx_channel_policies_connection_scope ON channel_policies(channel_connection_id, scope_type, external_scope_id);
CREATE INDEX IF NOT EXISTS idx_external_conversations_channel_external ON external_conversations(channel_connection_id, external_conversation_id);
CREATE INDEX IF NOT EXISTS idx_channel_inbox_events_connection_status ON channel_inbox_events(channel_connection_id, status, received_at);
CREATE INDEX IF NOT EXISTS idx_channel_inbox_events_conversation ON channel_inbox_events(conversation_id, received_at);
CREATE INDEX IF NOT EXISTS idx_channel_outbox_messages_connection_status ON channel_outbox_messages(channel_connection_id, status, created_at);
CREATE INDEX IF NOT EXISTS idx_marketplace_items_type_visibility_status ON marketplace_items(item_type, visibility, status);
CREATE INDEX IF NOT EXISTS idx_marketplace_items_tags ON marketplace_items USING GIN (tags);
CREATE INDEX IF NOT EXISTS idx_user_capability_installs_user_enabled ON user_capability_installs(user_id, is_enabled);
CREATE INDEX IF NOT EXISTS idx_agent_capability_bindings_agent_enabled ON agent_capability_bindings(agent_config_id, is_enabled);
CREATE INDEX IF NOT EXISTS idx_tool_router_logs_message ON tool_router_logs(message_id);
CREATE INDEX IF NOT EXISTS idx_tool_call_logs_conversation ON tool_call_logs(conversation_id, created_at);
CREATE INDEX IF NOT EXISTS idx_tool_call_logs_user_status ON tool_call_logs(user_id, status);
CREATE INDEX IF NOT EXISTS idx_tool_approval_requests_user_status ON tool_approval_requests(user_id, status);
CREATE INDEX IF NOT EXISTS idx_user_workspaces_user_status ON user_workspaces(user_id, status);
CREATE INDEX IF NOT EXISTS idx_user_workspaces_status_last_active ON user_workspaces(status, last_active_at);
CREATE INDEX IF NOT EXISTS idx_workspace_files_workspace_path ON workspace_files(workspace_id, relative_path);
CREATE INDEX IF NOT EXISTS idx_workspace_files_user_status_updated ON workspace_files(user_id, status, updated_at);
CREATE INDEX IF NOT EXISTS idx_workspace_command_runs_workspace_created ON workspace_command_runs(workspace_id, created_at);
CREATE INDEX IF NOT EXISTS idx_workspace_command_runs_user_status ON workspace_command_runs(user_id, status, created_at);
CREATE INDEX IF NOT EXISTS idx_workspace_events_workspace_created ON workspace_events(workspace_id, created_at);
CREATE INDEX IF NOT EXISTS idx_workspace_events_user_type_created ON workspace_events(user_id, event_type, created_at);
CREATE INDEX IF NOT EXISTS idx_workspace_quota_snapshots_workspace_created ON workspace_quota_snapshots(workspace_id, created_at);
CREATE INDEX IF NOT EXISTS idx_memory_retrieval_logs_message ON memory_retrieval_logs(message_id);
CREATE INDEX IF NOT EXISTS idx_context_build_logs_message ON context_build_logs(message_id);
CREATE INDEX IF NOT EXISTS idx_context_build_items_build ON context_build_items(context_build_id);
CREATE INDEX IF NOT EXISTS idx_llm_usage_logs_user_created ON llm_usage_logs(user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_dreaming_sessions_user_status ON dreaming_sessions(user_id, status);
CREATE INDEX IF NOT EXISTS idx_dreaming_insights_session ON dreaming_insights(dreaming_session_id);
CREATE INDEX IF NOT EXISTS idx_dreaming_insights_user_status ON dreaming_insights(user_id, status);
CREATE INDEX IF NOT EXISTS idx_curator_jobs_user_status ON curator_jobs(user_id, status);

CREATE INDEX IF NOT EXISTS idx_episodes_embedding ON episodes USING ivfflat (embedding vector_cosine_ops);
CREATE INDEX IF NOT EXISTS idx_skills_embedding ON skills USING ivfflat (embedding vector_cosine_ops);
CREATE INDEX IF NOT EXISTS idx_knowledge_chunks_embedding ON knowledge_chunks USING ivfflat (embedding vector_cosine_ops);
