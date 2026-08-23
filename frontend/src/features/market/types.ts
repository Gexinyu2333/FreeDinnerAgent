export type CapabilityType =
  | "tool"
  | "mcp_server"
  | "skill"
  | "knowledge_base"
  | "channel_adapter"
  | "system_prompt_template";

export type CapabilityInstall = {
  id: string;
  user_id: string;
  marketplace_item_id: string | null;
  capability_type: CapabilityType;
  capability_ref_id: string;
  is_enabled: boolean;
  install_source: string;
  installed_at: string;
  updated_at: string;
};

export type MarketplaceItem = {
  id: string;
  item_type: CapabilityType;
  ref_id: string;
  owner_user_id: string | null;
  visibility: string;
  title: string;
  description: string;
  category: string;
  tags: string[];
  install_count: number;
  rating: number | null;
  status: string;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
  viewer_install?: CapabilityInstall;
  system_prompt_latest_version_id?: string;
};

export type AgentCapabilityBinding = {
  id: string;
  agent_config_id: string;
  user_id: string;
  capability_type: CapabilityType;
  capability_ref_id: string;
  is_enabled: boolean;
  load_mode: string;
  priority: number;
  created_at: string;
  updated_at: string;
};

export type PromptVariableInput = {
  name: string;
  display_name?: string;
  description?: string | null;
  default_value?: string | null;
  required?: boolean;
  value_type?: string;
  allowed_values?: string[];
};

export type CreatePromptTemplateInput = {
  name: string;
  display_name: string;
  description: string;
  category?: string;
  tags?: string[];
  visibility?: string;
  content: string;
  change_note?: string | null;
  variables?: PromptVariableInput[];
};

export type SystemPromptTemplate = {
  id: string;
  owner_user_id: string | null;
  name: string;
  display_name: string;
  description: string;
  category: string;
  tags: string[];
  visibility: string;
  status: string;
  latest_version: number;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
};

export type SystemPromptTemplateVersion = {
  id: string;
  template_id: string;
  version: number;
  content: string;
  change_note: string | null;
  recommended_model_family: string | null;
  recommended_capabilities: unknown[];
  safety_policy: Record<string, unknown>;
  token_estimate: number;
  status: string;
  created_at: string;
};

export type SystemPromptVariable = {
  id: string;
  template_version_id: string;
  name: string;
  display_name: string;
  description: string | null;
  default_value: string | null;
  required: boolean;
  value_type: string;
  metadata: Record<string, unknown>;
  created_at: string;
};

export type CreatePromptTemplateResult = {
  template: SystemPromptTemplate;
  version: SystemPromptTemplateVersion;
  marketplace_item: MarketplaceItem;
};

export type Skill = {
  id: string;
  user_id: string;
  name: string;
  description: string;
  trigger_keywords: string[];
  visibility: string;
  permission_level: string;
  status: string;
  created_at: string;
  updated_at: string;
};

export type SkillVersion = {
  id: string;
  skill_id: string;
  version: number;
  react_steps: string;
  output_template: string | null;
  created_at: string;
};

export type CreateSkillInput = {
  name: string;
  description: string;
  keywords?: string[];
  react_steps: string;
  output_template?: string | null;
  visibility?: string;
  category?: string;
  tags?: string[];
};

export type CreateSkillResult = {
  skill: Skill;
  version: SkillVersion;
  marketplace_item: MarketplaceItem;
};

export type MCPServerDefinition = {
  id: string;
  user_id: string | null;
  name: string;
  display_name: string;
  description: string;
  transport_type: string;
  endpoint: string | null;
  command: string | null;
  visibility: string;
  permission_level: string;
  status: string;
  created_at: string;
  updated_at: string;
};

export type CreateMCPServerInput = {
  name: string;
  display_name: string;
  description: string;
  transport_type?: string;
  endpoint?: string | null;
  command?: string | null;
  args?: string[];
  env_schema?: Record<string, unknown>;
  visibility?: string;
  permission_level?: string;
  category?: string;
  tags?: string[];
};

export type CreateMCPServerResult = {
  server: MCPServerDefinition;
  marketplace_item: MarketplaceItem;
};

export type PromptPreview = {
  template: SystemPromptTemplate;
  version: SystemPromptTemplateVersion;
  variables: SystemPromptVariable[];
  content: string;
  tokens: number;
};

export type BindCapabilityInput = {
  agent_config_id?: string;
  capability_type: CapabilityType;
  capability_ref_id: string;
  load_mode?: string;
  priority?: number;
};
