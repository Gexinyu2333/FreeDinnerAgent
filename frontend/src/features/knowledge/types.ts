export type KnowledgeDocument = {
  id: string;
  user_id: string;
  title: string;
  source_type: string;
  source_uri: string | null;
  visibility: string;
  content_hash: string;
  status: string;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
};

export type KnowledgeChunk = {
  id: string;
  document_id: string;
  user_id: string;
  visibility: string;
  chunk_index: number;
  content: string;
  token_count: number;
  metadata: Record<string, unknown>;
  has_embedding: boolean;
  similarity?: number | null;
  created_at: string;
  document_title?: string | null;
  document_source?: string | null;
};

export type KnowledgeIngestInput = {
  title: string;
  content: string;
  source_type?: string;
  source_uri?: string | null;
  visibility?: string;
};

export type KnowledgeIngestResult = {
  document: KnowledgeDocument;
  chunks: KnowledgeChunk[];
  embedding_status: string;
  embedding_error?: string;
};

export type KnowledgeSearchResult = {
  mode: "keyword" | "vector";
  chunks: KnowledgeChunk[];
};
