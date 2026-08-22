import { apiClient } from "../../lib/apiClient";

import type {
  AgentCapabilityBinding,
  BindCapabilityInput,
  CapabilityInstall,
  CreatePromptTemplateInput,
  CreatePromptTemplateResult,
  MarketplaceItem,
  PromptPreview
} from "./types";

export function listMarketplaceItems(itemType?: string, installed = false) {
  const params = new URLSearchParams({ limit: "80" });
  if (itemType) {
    params.set("item_type", itemType);
  }
  if (installed) {
    params.set("installed", "true");
  }
  return apiClient<MarketplaceItem[]>(`/marketplace-items?${params.toString()}`);
}

export function installMarketplaceItem(itemID: string) {
  return apiClient<CapabilityInstall>(`/marketplace-items/${itemID}/install`, {
    method: "POST"
  });
}

export function setCapabilityInstallEnabled(input: { id: string; enabled: boolean }) {
  return apiClient<CapabilityInstall>(
    `/capability-installs/${input.id}/${input.enabled ? "enable" : "disable"}`,
    { method: "POST" }
  );
}

export function bindCapability(input: BindCapabilityInput) {
  return apiClient<AgentCapabilityBinding>("/agent-capability-bindings", {
    method: "POST",
    body: input
  });
}

export function setCapabilityBindingEnabled(input: { id: string; enabled: boolean }) {
  return apiClient<AgentCapabilityBinding>(
    `/agent-capability-bindings/${input.id}/${input.enabled ? "enable" : "disable"}`,
    { method: "POST" }
  );
}

export function createPromptTemplate(input: CreatePromptTemplateInput) {
  return apiClient<CreatePromptTemplateResult>("/system-prompt-templates", {
    method: "POST",
    body: input
  });
}

export function previewPromptTemplate(input: {
  version_id: string;
  variables?: Record<string, string>;
  override?: string | null;
}) {
  return apiClient<PromptPreview>("/system-prompt-templates/preview", {
    method: "POST",
    body: input
  });
}

export function forkPromptTemplate(input: {
  versionID: string;
  name?: string;
  display_name?: string;
  description?: string;
}) {
  return apiClient<CreatePromptTemplateResult>(
    `/system-prompt-template-versions/${input.versionID}/fork`,
    {
      method: "POST",
      body: {
        name: input.name,
        display_name: input.display_name,
        description: input.description
      }
    }
  );
}
