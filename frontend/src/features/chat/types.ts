export type Conversation = {
  id: string;
  user_id: string;
  title: string;
  channel: string;
  status: string;
  created_at: string;
  updated_at: string;
};

export type MessageRole = "system" | "user" | "assistant" | "tool";

export type Message = {
  id: string;
  conversation_id: string;
  user_id: string;
  role: MessageRole;
  content: string;
  token_count: number;
  is_anchor: boolean;
  anchor_reason: string | null;
  metadata: Record<string, unknown>;
  created_at: string;
};

export type CreateConversationInput = {
  title?: string;
};

export type SendMessageInput = {
  content: string;
};

export type SendMessageResult = {
  turn_id: string | null;
  user_message: Message;
  assistant_message: Message;
  used_memories: unknown[];
  tool_calls: unknown[];
};
