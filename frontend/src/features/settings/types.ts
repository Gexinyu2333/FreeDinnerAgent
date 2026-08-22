export type ProviderKind = "openai" | "anthropic";

export type ModelProvider = {
  id: string;
  provider: ProviderKind;
  display_name: string;
  chat_base_url: string | null;
  embedding_base_url: string | null;
  default_chat_model: string;
  default_embedding_model: string | null;
  is_default: boolean;
  has_chat_api_key: boolean;
  has_embedding_api_key: boolean;
  status: "active" | "disabled" | "deleted";
};

export type ModelProviderCreateInput = {
  provider: ProviderKind;
  display_name: string;
  chat_base_url?: string | null;
  chat_api_key: string;
  embedding_base_url?: string | null;
  embedding_api_key?: string | null;
  default_chat_model: string;
  default_embedding_model?: string | null;
  is_default: boolean;
};

export type ModelProviderUpdateInput = {
  display_name?: string;
  chat_base_url?: string | null;
  chat_api_key?: string;
  embedding_base_url?: string | null;
  embedding_api_key?: string | null;
  default_chat_model?: string;
  default_embedding_model?: string | null;
  is_default?: boolean;
  status?: "active" | "disabled";
};

export type ToolApprovalPolicy = "never" | "sensitive_only" | "always";
export type ThinkingEffort = "low" | "medium" | "high";

export type LLMFeatureSetting = {
  id: string;
  agent_config_id: string;
  user_id: string;
  feature_key: string;
  enabled: boolean;
  provider_id: string | null;
  model_override: string | null;
  temperature: number | null;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
};

export type LLMFeatureSettingUpdate = {
  feature_key: string;
  enabled: boolean;
  provider_id: string | null;
  model_override: string | null;
  temperature: number | null;
};

export type AgentConfig = {
  id: string;
  user_id: string;
  name: string;
  system_prompt: string;
  default_provider_id: string | null;
  temperature: number;
  thinking_enabled: boolean;
  thinking_effort: ThinkingEffort;
  thinking_budget_tokens: number;
  max_context_tokens: number;
  max_loop_steps: number;
  llm_retry_limit: number;
  fallback_policy: Record<string, unknown>;
  memory_enabled: boolean;
  tool_use_enabled: boolean;
  tool_approval_policy: ToolApprovalPolicy;
  dreaming_enabled: boolean;
  semantic_memory_enabled: boolean;
  embedding_enabled: boolean;
  embedding_cost_policy: Record<string, unknown>;
  llm_feature_settings: LLMFeatureSetting[];
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
};

export type AgentConfigUpdateInput = {
  name: string;
  system_prompt: string;
  default_provider_id: string | null;
  temperature: number;
  thinking_enabled: boolean;
  thinking_effort: ThinkingEffort;
  thinking_budget_tokens: number;
  max_context_tokens: number;
  max_loop_steps: number;
  llm_retry_limit: number;
  memory_enabled: boolean;
  tool_use_enabled: boolean;
  tool_approval_policy: ToolApprovalPolicy;
  dreaming_enabled: boolean;
  semantic_memory_enabled: boolean;
  embedding_enabled: boolean;
  embedding_cost_policy: Record<string, unknown>;
  llm_feature_settings: LLMFeatureSettingUpdate[];
};
