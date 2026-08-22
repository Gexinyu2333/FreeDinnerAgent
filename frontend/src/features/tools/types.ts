export type ToolDefinition = {
  id: string;
  owner_user_id: string | null;
  name: string;
  namespace: string;
  display_name: string;
  description: string;
  category: string;
  handler_type: string;
  handler_ref: string;
  visibility: string;
  permission_level: string;
  requires_approval: boolean;
  timeout_ms: number;
  max_retries: number;
  retry_backoff_ms: number;
  is_enabled: boolean;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
  active_version?: number;
  parameter_schema?: Record<string, unknown>;
  result_schema?: Record<string, unknown>;
};

export type ToolApprovalRequest = {
  id: string;
  tool_call_id: string;
  user_id: string;
  conversation_id: string;
  turn_id: string | null;
  approval_reason: string;
  risk_level: "normal" | "sensitive" | "destructive";
  proposed_arguments: Record<string, unknown>;
  status: "pending" | "approved" | "rejected" | "expired";
  created_at: string;
  resolved_at: string | null;
};
