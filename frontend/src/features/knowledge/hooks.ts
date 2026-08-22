import { useMutation, useQuery } from "@tanstack/react-query";

import {
  ingestKnowledgeDocument,
  listKnowledgeDocuments,
  searchKnowledge
} from "./api";

export const knowledgeDocumentsQueryKey = ["knowledge", "documents"] as const;

export function useKnowledgeDocuments() {
  return useQuery({
    queryKey: knowledgeDocumentsQueryKey,
    queryFn: listKnowledgeDocuments
  });
}

export function useIngestKnowledgeDocument() {
  return useMutation({
    mutationFn: ingestKnowledgeDocument
  });
}

export function useSearchKnowledge() {
  return useMutation({
    mutationFn: ({ q, limit }: { q: string; limit?: number }) => searchKnowledge(q, limit)
  });
}
