import { apiClient } from "../../lib/apiClient";

import type {
  KnowledgeDocument,
  KnowledgeIngestInput,
  KnowledgeIngestResult,
  KnowledgeSearchResult
} from "./types";

export function ingestKnowledgeDocument(input: KnowledgeIngestInput) {
  return apiClient<KnowledgeIngestResult>("/knowledge-documents", {
    method: "POST",
    body: input
  });
}

export function listKnowledgeDocuments() {
  return apiClient<KnowledgeDocument[]>("/knowledge-documents");
}

export function searchKnowledge(q: string, limit = 8) {
  const params = new URLSearchParams({ q, limit: String(limit) });
  return apiClient<KnowledgeSearchResult>(`/knowledge-search?${params.toString()}`);
}
