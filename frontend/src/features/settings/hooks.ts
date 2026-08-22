import { useMutation, useQuery } from "@tanstack/react-query";

import {
  createModelProvider,
  deleteModelProvider,
  getAgentConfig,
  listModelProviders,
  updateAgentConfig,
  updateModelProvider
} from "./api";
import type { ModelProviderUpdateInput } from "./types";

export const agentConfigQueryKey = ["settings", "agent-config"] as const;
export const modelProvidersQueryKey = ["settings", "model-providers"] as const;

export function useAgentConfig() {
  return useQuery({
    queryKey: agentConfigQueryKey,
    queryFn: getAgentConfig
  });
}

export function useUpdateAgentConfig() {
  return useMutation({
    mutationFn: updateAgentConfig
  });
}

export function useModelProviders() {
  return useQuery({
    queryKey: modelProvidersQueryKey,
    queryFn: listModelProviders
  });
}

export function useCreateModelProvider() {
  return useMutation({
    mutationFn: createModelProvider
  });
}

export function useUpdateModelProvider() {
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: ModelProviderUpdateInput }) =>
      updateModelProvider(id, input)
  });
}

export function useDeleteModelProvider() {
  return useMutation({
    mutationFn: deleteModelProvider
  });
}
