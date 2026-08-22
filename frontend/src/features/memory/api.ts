import { apiClient } from "../../lib/apiClient";

import type {
  CreateProfileMemoryInput,
  DreamingApplyResult,
  DreamingInsight,
  MemoryContextInput,
  MemoryContextResult,
  MemoryTypeDefinition,
  ProfileMemory,
  ProfileMemorySearchResult
} from "./types";

export function listMemoryTypes() {
  return apiClient<MemoryTypeDefinition[]>("/memory-types");
}

export function createProfileMemory(input: CreateProfileMemoryInput) {
  return apiClient<ProfileMemory>("/profile-memories", {
    method: "POST",
    body: input
  });
}

export function listProfileMemories(memoryType?: string) {
  const params = new URLSearchParams();
  if (memoryType) {
    params.set("memory_type", memoryType);
  }
  return apiClient<ProfileMemory[]>(
    `/profile-memories${params.size ? `?${params.toString()}` : ""}`
  );
}

export function searchProfileMemories(q: string, limit = 8) {
  const params = new URLSearchParams({ q, limit: String(limit) });
  return apiClient<ProfileMemorySearchResult>(`/profile-memory-search?${params.toString()}`);
}

export function getMemoryContext(input: MemoryContextInput) {
  const params = new URLSearchParams({
    conversation_id: input.conversation_id,
    q: input.q,
    max_memory_tokens: String(input.max_memory_tokens ?? 900),
    working: String(input.working ?? true),
    profile: String(input.profile ?? true),
    semantic: String(input.semantic ?? true),
    log: String(input.log ?? false)
  });
  return apiClient<MemoryContextResult>(`/memory-context?${params.toString()}`);
}

export function listDreamingInsights(status?: string, limit = 20) {
  const params = new URLSearchParams({ limit: String(limit) });
  if (status) {
    params.set("status", status);
  }
  return apiClient<DreamingInsight[]>(`/dreaming-insights?${params.toString()}`);
}

export function applyDreamingInsight(insightID: string) {
  return apiClient<DreamingApplyResult>(`/dreaming-insights/${insightID}/apply`, {
    method: "POST"
  });
}

export function rejectDreamingInsight(insightID: string) {
  return apiClient<DreamingInsight>(`/dreaming-insights/${insightID}/reject`, {
    method: "POST"
  });
}
