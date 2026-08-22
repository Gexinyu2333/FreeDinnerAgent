import { useQueryClient } from "@tanstack/react-query";
import { CopyPlus, Eye, PackageCheck, Store } from "lucide-react";
import { FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";

import { Badge } from "../../../components/ui/Badge";
import { Button } from "../../../components/ui/Button";
import { EmptyState } from "../../../components/ui/EmptyState";
import { Input } from "../../../components/ui/Input";
import { LoadingState } from "../../../components/ui/LoadingState";
import { Select } from "../../../components/ui/Select";
import { Switch } from "../../../components/ui/Switch";
import { Tabs } from "../../../components/ui/Tabs";
import { Textarea } from "../../../components/ui/Textarea";
import { Toast } from "../../../components/ui/Toast";
import { ApiError } from "../../../lib/errors";
import { formatDateTime, formatNumber } from "../../../lib/format";
import { useAgentConfig } from "../../settings/hooks";
import {
  marketplaceItemsQueryKey,
  useBindCapability,
  useCreatePromptTemplate,
  useForkPromptTemplate,
  useInstallMarketplaceItem,
  useMarketplaceItems,
  usePreviewPromptTemplate,
  useSetCapabilityInstallEnabled
} from "../hooks";
import type {
  CapabilityType,
  CreatePromptTemplateInput,
  MarketplaceItem
} from "../types";

type MarketTab = "browse" | "systemPrompt";

const itemTypes: Array<{ value: "" | CapabilityType; key: string }> = [
  { value: "", key: "all" },
  { value: "tool", key: "tool" },
  { value: "mcp_server", key: "mcpServer" },
  { value: "skill", key: "skill" },
  { value: "knowledge_base", key: "knowledgeBase" },
  { value: "channel_adapter", key: "channelAdapter" },
  { value: "system_prompt_template", key: "systemPrompt" }
];

const initialPromptForm: CreatePromptTemplateInput = {
  name: "",
  display_name: "",
  description: "",
  category: "general",
  tags: [],
  visibility: "private",
  content: "",
  change_note: "",
  variables: []
};

export function MarketPage() {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState<MarketTab>("browse");

  return (
    <section className="space-y-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-ink-900">{t("market.title")}</h1>
          <p className="mt-2 text-sm leading-6 text-ink-500">{t("market.description")}</p>
        </div>
        <Tabs
          activeKey={activeTab}
          items={[
            { key: "browse", label: t("market.tabs.browse") },
            { key: "systemPrompt", label: t("market.tabs.systemPrompt") }
          ]}
          onChange={(key) => setActiveTab(key as MarketTab)}
        />
      </div>
      {activeTab === "browse" && <MarketplaceBrowser />}
      {activeTab === "systemPrompt" && <SystemPromptPanel />}
    </section>
  );
}

function MarketplaceBrowser() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [itemType, setItemType] = useState("");
  const [installedOnly, setInstalledOnly] = useState(false);
  const itemsQuery = useMarketplaceItems(itemType || undefined, installedOnly);
  const installMutation = useInstallMarketplaceItem();
  const enableMutation = useSetCapabilityInstallEnabled();
  const bindMutation = useBindCapability();
  const agentConfigQuery = useAgentConfig();
  const items = itemsQuery.data ?? [];
  const mutationError = installMutation.error ?? enableMutation.error ?? bindMutation.error ?? null;

  function refresh() {
    void queryClient.invalidateQueries({ queryKey: marketplaceItemsQueryKey });
  }

  return (
    <div className="space-y-4">
      {mutationError && (
        <Toast
          message={
            mutationError instanceof ApiError
              ? mutationError.message
              : t("market.errors.operationFailed")
          }
          tone="error"
        />
      )}
      {bindMutation.isSuccess && <Toast message={t("market.bound")} tone="success" />}
      <div className="flex flex-col gap-3 rounded-lg border border-ink-200 bg-white p-4 sm:flex-row sm:items-center">
        <Select
          className="w-full sm:w-60"
          onChange={(event) => setItemType(event.target.value)}
          value={itemType}
        >
          {itemTypes.map((item) => (
            <option key={item.key} value={item.value}>
              {t(`market.itemTypes.${item.key}`)}
            </option>
          ))}
        </Select>
        <label className="flex items-center gap-3 text-sm font-medium text-ink-700">
          <Switch checked={installedOnly} onClick={() => setInstalledOnly(!installedOnly)} />
          {t("market.installedOnly")}
        </label>
      </div>

      {itemsQuery.isLoading ? (
        <LoadingState />
      ) : items.length === 0 ? (
        <EmptyState
          description={t("market.empty.description")}
          icon={<Store className="h-8 w-8" />}
          title={t("market.empty.title")}
        />
      ) : (
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
          {items.map((item) => (
            <MarketplaceCard
              agentConfigID={agentConfigQuery.data?.id}
              item={item}
              key={item.id}
              onBind={() => {
                const capabilityRefID =
                  item.item_type === "system_prompt_template"
                    ? item.system_prompt_latest_version_id ?? item.ref_id
                    : item.ref_id;
                bindMutation.mutate(
                  {
                    agent_config_id: agentConfigQuery.data?.id,
                    capability_type: item.item_type,
                    capability_ref_id: capabilityRefID,
                    load_mode: "auto",
                    priority: 0
                  },
                  { onSuccess: refresh }
                );
              }}
              onInstall={() =>
                installMutation.mutate(item.id, {
                  onSuccess: refresh
                })
              }
              onToggleInstall={() => {
                if (!item.viewer_install) {
                  return;
                }
                enableMutation.mutate(
                  {
                    id: item.viewer_install.id,
                    enabled: !item.viewer_install.is_enabled
                  },
                  { onSuccess: refresh }
                );
              }}
              working={
                installMutation.isPending || enableMutation.isPending || bindMutation.isPending
              }
            />
          ))}
        </div>
      )}
    </div>
  );
}

function MarketplaceCard({
  agentConfigID,
  item,
  onBind,
  onInstall,
  onToggleInstall,
  working
}: {
  agentConfigID?: string;
  item: MarketplaceItem;
  onBind: () => void;
  onInstall: () => void;
  onToggleInstall: () => void;
  working: boolean;
}) {
  const { t } = useTranslation();
  const installed = Boolean(item.viewer_install);
  return (
    <article className="rounded-lg border border-ink-200 bg-white p-4 shadow-sm">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <h3 className="truncate text-base font-semibold text-ink-900">{item.title}</h3>
          <p className="mt-1 text-xs text-ink-500">{formatDateTime(item.updated_at)}</p>
        </div>
        <Badge tone={item.visibility === "public" ? "blue" : "neutral"}>
          {item.visibility}
        </Badge>
      </div>
      <p className="mt-3 text-sm leading-6 text-ink-700">{item.description}</p>
      <div className="mt-4 flex flex-wrap gap-2">
        <Badge tone="blue">{item.item_type}</Badge>
        <Badge>{item.category}</Badge>
        <Badge>
          {t("market.installCount", {
            count: item.install_count,
            formattedCount: formatNumber(item.install_count)
          })}
        </Badge>
        {item.rating !== null && <Badge>{t("market.rating", { rating: item.rating })}</Badge>}
      </div>
      {item.tags.length > 0 && (
        <div className="mt-3 flex flex-wrap gap-2">
          {item.tags.map((tag) => (
            <Badge key={tag}>{tag}</Badge>
          ))}
        </div>
      )}
      <div className="mt-4 flex flex-wrap gap-2">
        {!installed ? (
          <Button disabled={working} onClick={onInstall} variant="secondary">
            {t("market.install")}
          </Button>
        ) : (
          <Button disabled={working} onClick={onToggleInstall} variant="secondary">
            {item.viewer_install?.is_enabled ? t("market.disable") : t("market.enable")}
          </Button>
        )}
        <Button disabled={working || !agentConfigID} onClick={onBind}>
          {t("market.bind")}
        </Button>
      </div>
    </article>
  );
}

function SystemPromptPanel() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const createMutation = useCreatePromptTemplate();
  const previewMutation = usePreviewPromptTemplate();
  const forkMutation = useForkPromptTemplate();
  const bindMutation = useBindCapability();
  const agentConfigQuery = useAgentConfig();
  const [form, setForm] = useState<CreatePromptTemplateInput>(initialPromptForm);
  const [tagsText, setTagsText] = useState("");
  const [variablesText, setVariablesText] = useState("[]");
  const [previewVersionID, setPreviewVersionID] = useState("");
  const [previewVariables, setPreviewVariables] = useState("{}");
  const [previewOverride, setPreviewOverride] = useState("");
  const [forkName, setForkName] = useState("");
  const mutationError =
    createMutation.error ?? previewMutation.error ?? forkMutation.error ?? bindMutation.error ?? null;

  function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const variables = parseJSON(variablesText, []);
    createMutation.mutate(
      {
        ...form,
        tags: splitTags(tagsText),
        variables,
        change_note: form.change_note || null
      },
      {
        onSuccess: (result) => {
          setPreviewVersionID(result.version.id);
          void queryClient.invalidateQueries({ queryKey: marketplaceItemsQueryKey });
        }
      }
    );
  }

  function handlePreview(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    previewMutation.mutate({
      version_id: previewVersionID.trim(),
      variables: parseJSON(previewVariables, {}),
      override: previewOverride.trim() || null
    });
  }

  function handleFork() {
    forkMutation.mutate(
      {
        versionID: previewVersionID.trim(),
        name: forkName.trim() || undefined,
        display_name: forkName.trim() || undefined
      },
      {
        onSuccess: (result) => {
          setPreviewVersionID(result.version.id);
          void queryClient.invalidateQueries({ queryKey: marketplaceItemsQueryKey });
        }
      }
    );
  }

  function handleBindVersion() {
    bindMutation.mutate({
      agent_config_id: agentConfigQuery.data?.id,
      capability_type: "system_prompt_template",
      capability_ref_id: previewVersionID.trim(),
      load_mode: "auto",
      priority: 100
    });
  }

  return (
    <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_430px]">
      <div className="space-y-4">
        {mutationError && (
          <Toast
            message={
              mutationError instanceof ApiError
                ? mutationError.message
                : t("market.errors.operationFailed")
            }
            tone="error"
          />
        )}
        {createMutation.isSuccess && (
          <Toast message={t("market.prompt.created")} tone="success" />
        )}
        {bindMutation.isSuccess && <Toast message={t("market.bound")} tone="success" />}
        <form className="rounded-lg border border-ink-200 bg-white p-5 shadow-sm" onSubmit={handleCreate}>
          <div className="flex items-center gap-2">
            <PackageCheck className="h-5 w-5 text-ocean-600" />
            <h2 className="text-base font-semibold text-ink-900">
              {t("market.prompt.createTitle")}
            </h2>
          </div>
          <div className="mt-5 grid gap-4 md:grid-cols-2">
            <Field label={t("market.prompt.fields.name")}>
              <Input
                onChange={(event) => setForm({ ...form, name: event.target.value })}
                required
                value={form.name}
              />
            </Field>
            <Field label={t("market.prompt.fields.displayName")}>
              <Input
                onChange={(event) => setForm({ ...form, display_name: event.target.value })}
                required
                value={form.display_name}
              />
            </Field>
            <Field label={t("market.prompt.fields.category")}>
              <Input
                onChange={(event) => setForm({ ...form, category: event.target.value })}
                value={form.category}
              />
            </Field>
            <Field label={t("market.prompt.fields.visibility")}>
              <Select
                onChange={(event) => setForm({ ...form, visibility: event.target.value })}
                value={form.visibility}
              >
                <option value="private">{t("market.visibility.private")}</option>
                <option value="public">{t("market.visibility.public")}</option>
              </Select>
            </Field>
          </div>
          <div className="mt-4 space-y-4">
            <Field label={t("market.prompt.fields.description")}>
              <Textarea
                onChange={(event) => setForm({ ...form, description: event.target.value })}
                required
                value={form.description}
              />
            </Field>
            <Field label={t("market.prompt.fields.content")}>
              <Textarea
                className="min-h-56"
                onChange={(event) => setForm({ ...form, content: event.target.value })}
                required
                value={form.content}
              />
            </Field>
            <Field label={t("market.prompt.fields.tags")}>
              <Input onChange={(event) => setTagsText(event.target.value)} value={tagsText} />
            </Field>
            <Field label={t("market.prompt.fields.variables")}>
              <Textarea
                className="font-mono"
                onChange={(event) => setVariablesText(event.target.value)}
                value={variablesText}
              />
            </Field>
            <Button disabled={createMutation.isPending} type="submit">
              {createMutation.isPending ? t("market.prompt.creating") : t("market.prompt.create")}
            </Button>
          </div>
        </form>
      </div>

      <aside className="space-y-4">
        <form className="rounded-lg border border-ink-200 bg-white p-5 shadow-sm" onSubmit={handlePreview}>
          <div className="flex items-center gap-2">
            <Eye className="h-5 w-5 text-ocean-600" />
            <h2 className="text-base font-semibold text-ink-900">
              {t("market.prompt.previewTitle")}
            </h2>
          </div>
          <div className="mt-5 space-y-4">
            <Field label={t("market.prompt.fields.versionId")}>
              <Input
                onChange={(event) => setPreviewVersionID(event.target.value)}
                required
                value={previewVersionID}
              />
            </Field>
            <Field label={t("market.prompt.fields.previewVariables")}>
              <Textarea
                className="font-mono"
                onChange={(event) => setPreviewVariables(event.target.value)}
                value={previewVariables}
              />
            </Field>
            <Field label={t("market.prompt.fields.override")}>
              <Textarea
                onChange={(event) => setPreviewOverride(event.target.value)}
                value={previewOverride}
              />
            </Field>
            <Button
              disabled={previewMutation.isPending}
              icon={<Eye className="h-4 w-4" />}
              type="submit"
            >
              {t("market.prompt.preview")}
            </Button>
          </div>
        </form>

        {previewMutation.data && (
          <article className="rounded-lg border border-ink-200 bg-white p-5 shadow-sm">
            <div className="flex items-start justify-between gap-3">
              <div>
                <h3 className="text-base font-semibold text-ink-900">
                  {previewMutation.data.template.display_name}
                </h3>
                <p className="mt-1 text-xs text-ink-500">
                  {t("market.prompt.tokens", { count: previewMutation.data.tokens })}
                </p>
              </div>
              <Badge tone="blue">
                v{previewMutation.data.version.version}
              </Badge>
            </div>
            <pre className="mt-4 max-h-80 overflow-auto whitespace-pre-wrap rounded-md bg-ink-900 p-3 text-xs leading-5 text-white">
              {previewMutation.data.content}
            </pre>
            <div className="mt-4 space-y-3">
              <Field label={t("market.prompt.fields.forkName")}>
                <Input onChange={(event) => setForkName(event.target.value)} value={forkName} />
              </Field>
              <div className="flex flex-wrap gap-2">
                <Button
                  disabled={forkMutation.isPending}
                  icon={<CopyPlus className="h-4 w-4" />}
                  onClick={handleFork}
                  variant="secondary"
                >
                  {t("market.prompt.fork")}
                </Button>
                <Button
                  disabled={bindMutation.isPending || !agentConfigQuery.data?.id}
                  onClick={handleBindVersion}
                >
                  {t("market.prompt.bindVersion")}
                </Button>
              </div>
            </div>
          </article>
        )}
      </aside>
    </div>
  );
}

function Field({ children, label }: { children: React.ReactNode; label: string }) {
  return (
    <label className="block space-y-1.5">
      <span className="text-sm font-medium text-ink-700">{label}</span>
      {children}
    </label>
  );
}

function splitTags(value: string) {
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}

function parseJSON<T>(value: string, fallback: T): T {
  try {
    return JSON.parse(value) as T;
  } catch {
    return fallback;
  }
}
