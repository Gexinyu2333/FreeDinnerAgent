import { useMutation, useQuery } from "@tanstack/react-query";

import {
  bindCapability,
  createPromptTemplate,
  forkPromptTemplate,
  installMarketplaceItem,
  listMarketplaceItems,
  previewPromptTemplate,
  setCapabilityInstallEnabled
} from "./api";

export const marketplaceItemsQueryKey = ["market", "items"] as const;

export function useMarketplaceItems(itemType?: string, installed = false) {
  return useQuery({
    queryKey: [...marketplaceItemsQueryKey, itemType || "all", installed],
    queryFn: () => listMarketplaceItems(itemType, installed)
  });
}

export function useInstallMarketplaceItem() {
  return useMutation({
    mutationFn: installMarketplaceItem
  });
}

export function useSetCapabilityInstallEnabled() {
  return useMutation({
    mutationFn: setCapabilityInstallEnabled
  });
}

export function useBindCapability() {
  return useMutation({
    mutationFn: bindCapability
  });
}

export function useCreatePromptTemplate() {
  return useMutation({
    mutationFn: createPromptTemplate
  });
}

export function usePreviewPromptTemplate() {
  return useMutation({
    mutationFn: previewPromptTemplate
  });
}

export function useForkPromptTemplate() {
  return useMutation({
    mutationFn: forkPromptTemplate
  });
}
