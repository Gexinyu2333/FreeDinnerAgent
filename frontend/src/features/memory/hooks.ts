import { useMutation, useQuery } from "@tanstack/react-query";

import {
  applyDreamingInsight,
  createProfileMemory,
  getMemoryContext,
  listDreamingInsights,
  listMemoryTypes,
  listProfileMemories,
  rejectDreamingInsight,
  searchProfileMemories
} from "./api";
import type { MemoryContextInput } from "./types";

export const memoryTypesQueryKey = ["memory", "types"] as const;
export const profileMemoriesQueryKey = ["memory", "profile"] as const;
export const dreamingInsightsQueryKey = ["memory", "dreaming-insights"] as const;

export function useMemoryTypes() {
  return useQuery({
    queryKey: memoryTypesQueryKey,
    queryFn: listMemoryTypes
  });
}

export function useProfileMemories(memoryType?: string) {
  return useQuery({
    queryKey: [...profileMemoriesQueryKey, memoryType || "all"],
    queryFn: () => listProfileMemories(memoryType)
  });
}

export function useCreateProfileMemory() {
  return useMutation({
    mutationFn: createProfileMemory
  });
}

export function useSearchProfileMemories() {
  return useMutation({
    mutationFn: ({ q, limit }: { q: string; limit?: number }) =>
      searchProfileMemories(q, limit)
  });
}

export function useMemoryContext() {
  return useMutation({
    mutationFn: (input: MemoryContextInput) => getMemoryContext(input)
  });
}

export function useDreamingInsights(status?: string) {
  return useQuery({
    queryKey: [...dreamingInsightsQueryKey, status || "all"],
    queryFn: () => listDreamingInsights(status)
  });
}

export function useApplyDreamingInsight() {
  return useMutation({
    mutationFn: applyDreamingInsight
  });
}

export function useRejectDreamingInsight() {
  return useMutation({
    mutationFn: rejectDreamingInsight
  });
}
