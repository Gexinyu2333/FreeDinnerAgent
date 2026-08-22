export type ChannelProviderDefinition = {
  id: string;
  name: string;
  display_name: string;
  description: string;
  provider_type: string;
  adapter_type: string;
  inbound_modes: string[];
  outbound_modes: string[];
  config_schema: Record<string, unknown>;
  default_policy: Record<string, unknown>;
  visibility: string;
  status: string;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
};

export type PublicChannelConnection = {
  id: string;
  user_id: string;
  provider_id: string;
  display_name: string;
  external_account_id: string | null;
  external_account_name: string | null;
  has_config: boolean;
  status: string;
  last_health_status: string | null;
  last_event_at: string | null;
  last_checked_at: string | null;
  created_at: string;
  updated_at: string;
};

export type ChannelPolicy = {
  id: string;
  user_id: string;
  channel_connection_id: string;
  scope_type: string;
  external_scope_id: string | null;
  mode: string;
  trigger_keywords: string[];
  allow_memory_write: boolean;
  allow_tool_use: boolean;
  require_approval_for_outbound: boolean;
  rate_limit_per_minute: number;
  quiet_hours: Record<string, unknown>;
  status: string;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
};

export type ExternalConversation = {
  id: string;
  user_id: string;
  channel_connection_id: string;
  conversation_id: string;
  external_conversation_id: string;
  external_conversation_type: string;
  external_title: string | null;
  last_message_at: string | null;
  status: string;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
};

export type ChannelInboxEvent = {
  id: string;
  user_id: string;
  channel_connection_id: string;
  external_conversation_id: string | null;
  conversation_id: string | null;
  message_id: string | null;
  event_type: string;
  external_event_id: string | null;
  external_sender_id: string | null;
  external_sender_name: string | null;
  raw_payload: Record<string, unknown>;
  normalized_text: string | null;
  should_trigger_agent: boolean;
  trigger_reason: string | null;
  status: string;
  received_at: string;
  processed_at: string | null;
};

export type ChannelOutboxMessage = {
  id: string;
  user_id: string;
  channel_connection_id: string;
  external_conversation_id: string | null;
  conversation_id: string | null;
  agent_turn_id: string | null;
  reply_to_inbox_event_id: string | null;
  message_type: string;
  content: string;
  payload: Record<string, unknown>;
  requires_approval: boolean;
  status: string;
  external_message_id: string | null;
  error_message: string | null;
  created_at: string;
  approved_at: string | null;
  sent_at: string | null;
};

export type CreateChannelConnectionInput = {
  provider_id: string;
  display_name: string;
  external_account_id?: string | null;
  external_account_name?: string | null;
  config?: Record<string, unknown>;
};

export type UpsertChannelPolicyInput = {
  connection_id: string;
  scope_type: string;
  external_scope_id?: string | null;
  mode: string;
  trigger_keywords?: string[];
  allow_memory_write?: boolean;
  allow_tool_use?: boolean;
  require_approval_for_outbound?: boolean;
  rate_limit_per_minute?: number;
  rate_limit_policy?: Record<string, unknown>;
};
