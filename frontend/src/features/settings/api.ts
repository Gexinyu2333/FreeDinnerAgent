import { apiClient } from "../../lib/apiClient";

import type {
  AgentConfig,
  AgentConfigUpdateInput,
  ModelProvider,
  ModelProviderCreateInput,
  ModelProviderUpdateInput
} from "./types";

export function getAgentConfig() {
  return apiClient<AgentConfig>("/me/agent-config");
}

export function updateAgentConfig(input: AgentConfigUpdateInput) {
  return apiClient<AgentConfig>("/me/agent-config", {
    method: "PATCH",
    body: input
  });
}

export function listModelProviders() {
  return apiClient<ModelProvider[]>("/me/model-providers");
}

export function createModelProvider(input: ModelProviderCreateInput) {
  return apiClient<ModelProvider>("/me/model-providers", {
    method: "POST",
    body: input
  });
}

export function updateModelProvider(id: string, input: ModelProviderUpdateInput) {
  return apiClient<ModelProvider>(`/me/model-providers/${id}`, {
    method: "PATCH",
    body: input
  });
}

export function deleteModelProvider(id: string) {
  return apiClient<{ deleted: boolean }>(`/me/model-providers/${id}`, {
    method: "DELETE"
  });
}
