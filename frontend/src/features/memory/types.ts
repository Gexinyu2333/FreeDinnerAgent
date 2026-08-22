export type MemoryTypeDefinition = {
  memory_type: string;
  display_name: string;
  description: string;
  extraction_hint: string;
  retrieval_weight: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
};

export type ProfileMemory = {
  id: string;
  user_id: string;
  memory_type: string;
  scope: string;
  title: string;
  content: string;
  evidence: string | null;
  source_message_id: string | null;
  confidence: number;
  importance: number;
  status: string;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
};

export type CreateProfileMemoryInput = {
  memory_type: string;
  scope?: string;
  title: string;
  content: string;
  evidence?: string | null;
  confidence?: number;
  importance?: number;
};

export type ProfileMemorySearchResult = {
  mode: "keyword";
  memories: ProfileMemory[];
};

export type MemoryChunk = {
  layer: string;
  ref_id: string;
  content: string;
  score: number;
  token_count: number;
  visibility: string;
  source: string;
  load_mode: string;
  metadata: Record<string, unknown>;
};

export type MemoryContextResult = {
  chunks: MemoryChunk[];
  token_count: number;
  used_layers: string[];
  skipped: string[];
  semantic_mode?: string;
};

export type MemoryContextInput = {
  conversation_id: string;
  q: string;
  max_memory_tokens?: number;
  working?: boolean;
  profile?: boolean;
  semantic?: boolean;
  log?: boolean;
};

export type DreamingInsight = {
  id: string;
  dreaming_session_id: string;
  user_id: string;
  insight_type: string;
  source_layer: string;
  source_ref_ids: string[];
  target_layer: string | null;
  target_ref_id: string | null;
  content: string;
  confidence: number;
  status: string;
  created_at: string;
  applied_at: string | null;
};

export type DreamingApplyResult = {
  insight: DreamingInsight;
  profile_memory?: ProfileMemory;
  skill?: unknown;
  curator_job?: unknown;
};
