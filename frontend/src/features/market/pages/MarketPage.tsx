import { useQueryClient } from "@tanstack/react-query";
import { CopyPlus, Eye, PackageCheck, Store } from "lucide-react";
import { FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";

import { Badge } from "../../../components/ui/Badge";
import { Button } from "../../../components/ui/Button";
import { EmptyState } from "../../../components/ui/EmptyState";
import { Field } from "../../../components/ui/Field";
import { Input } from "../../../components/ui/Input";
import { LoadingState } from "../../../components/ui/LoadingState";
import { Select } from "../../../components/ui/Select";
import { Switch } from "../../../components/ui/Switch";
import { Tabs } from "../../../components/ui/Tabs";
import { Textarea } from "../../../components/ui/Textarea";
import { Toast } from "../../../components/ui/Toast";
import { useToast } from "../../../components/ui/ToastProvider";
import { ApiError } from "../../../lib/errors";
import { formatDateTime, formatNumber } from "../../../lib/format";
import { useAgentConfig } from "../../settings/hooks";
import {
  marketplaceItemsQueryKey,
  useBindCapability,
  useCreateMCPServer,
  useCreatePromptTemplate,
  useCreateSkill,
  useForkPromptTemplate,
  useInstallMarketplaceItem,
  useMarketplaceItems,
  usePreviewPromptTemplate,
  useSetCapabilityInstallEnabled
} from "../hooks";
import type {
  CapabilityType,
  CreateMCPServerInput,
  CreatePromptTemplateInput,
  CreateSkillInput,
  MarketplaceItem
} from "../types";

type MarketTab = "browse" | "systemPrompt" | "skill" | "mcp";

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

const initialSkillForm: CreateSkillInput = {
  name: "",
  description: "",
  keywords: [],
  react_steps: "",
  output_template: "",
  visibility: "private",
  category: "skill",
  tags: []
};

const initialMCPForm: CreateMCPServerInput = {
  name: "",
  display_name: "",
  description: "",
  transport_type: "http",
  endpoint: "",
  command: "",
  args: [],
  env_schema: {},
  visibility: "private",
  permission_level: "normal",
  category: "mcp",
  tags: []
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
            { key: "systemPrompt", label: t("market.tabs.systemPrompt") },
            { key: "skill", label: t("market.tabs.skill") },
            { key: "mcp", label: t("market.tabs.mcp") }
          ]}
          onChange={(key) => setActiveTab(key as MarketTab)}
        />
      </div>
      {activeTab === "browse" && <MarketplaceBrowser />}
      {activeTab === "systemPrompt" && <SystemPromptPanel />}
      {activeTab === "skill" && <SkillPanel />}
      {activeTab === "mcp" && <MCPPanel />}
    </section>
  );
}

function MarketplaceBrowser() {
  const { t } = useTranslation();
  const toast = useToast();
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
                  {
                    onSuccess: () => {
                      refresh();
                      toast.notify(t("market.bound"));
                    }
                  }
                );
              }}
              onInstall={() =>
                installMutation.mutate(item.id, {
                  onSuccess: () => {
                    refresh();
                    toast.notify(t("common.saved"));
                  }
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
                  {
                    onSuccess: () => {
                      refresh();
                      toast.notify(t("common.saved"));
                    }
                  }
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
  const toast = useToast();
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
          toast.notify(t("market.prompt.created"));
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
          toast.notify(t("common.created"));
        }
      }
    );
  }

  function handleBindVersion() {
    bindMutation.mutate(
      {
        agent_config_id: agentConfigQuery.data?.id,
        capability_type: "system_prompt_template",
        capability_ref_id: previewVersionID.trim(),
        load_mode: "auto",
        priority: 100
      },
      {
        onSuccess: () => toast.notify(t("market.bound"))
      }
    );
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

function SkillPanel() {
  const { t } = useTranslation();
  const toast = useToast();
  const queryClient = useQueryClient();
  const createMutation = useCreateSkill();
  const [form, setForm] = useState<CreateSkillInput>(initialSkillForm);
  const [keywordsText, setKeywordsText] = useState("");
  const [tagsText, setTagsText] = useState("");

  function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    createMutation.mutate(
      {
        ...form,
        keywords: splitTags(keywordsText),
        tags: splitTags(tagsText),
        output_template: form.output_template?.trim() || null
      },
      {
        onSuccess: () => {
          setForm(initialSkillForm);
          setKeywordsText("");
          setTagsText("");
          void queryClient.invalidateQueries({ queryKey: marketplaceItemsQueryKey });
          toast.notify(t("market.skill.created"));
        }
      }
    );
  }

  return (
    <form className="space-y-4 rounded-lg border border-ink-200 bg-white p-5 shadow-sm" onSubmit={handleCreate}>
      {createMutation.error && (
        <Toast
          message={createMutation.error instanceof ApiError ? createMutation.error.message : t("market.errors.operationFailed")}
          tone="error"
        />
      )}
      <div className="flex items-center gap-2">
        <PackageCheck className="h-5 w-5 text-ocean-600" />
        <h2 className="text-base font-semibold text-ink-900">{t("market.skill.createTitle")}</h2>
      </div>
      <div className="grid gap-4 md:grid-cols-2">
        <Field label={t("market.skill.fields.name")}>
          <Input onChange={(event) => setForm({ ...form, name: event.target.value })} required value={form.name} />
        </Field>
        <Field label={t("market.skill.fields.visibility")}>
          <Select onChange={(event) => setForm({ ...form, visibility: event.target.value })} value={form.visibility}>
            <option value="private">{t("market.visibility.private")}</option>
            <option value="public">{t("market.visibility.public")}</option>
          </Select>
        </Field>
      </div>
      <Field label={t("market.skill.fields.description")}>
        <Textarea onChange={(event) => setForm({ ...form, description: event.target.value })} required value={form.description} />
      </Field>
      <Field label={t("market.skill.fields.reactSteps")}>
        <Textarea className="min-h-44" onChange={(event) => setForm({ ...form, react_steps: event.target.value })} required value={form.react_steps} />
      </Field>
      <Field label={t("market.skill.fields.outputTemplate")}>
        <Textarea onChange={(event) => setForm({ ...form, output_template: event.target.value })} value={form.output_template ?? ""} />
      </Field>
      <div className="grid gap-4 md:grid-cols-2">
        <Field label={t("market.skill.fields.keywords")}>
          <Input onChange={(event) => setKeywordsText(event.target.value)} value={keywordsText} />
        </Field>
        <Field label={t("market.skill.fields.tags")}>
          <Input onChange={(event) => setTagsText(event.target.value)} value={tagsText} />
        </Field>
      </div>
      <Button disabled={createMutation.isPending} type="submit">
        {createMutation.isPending ? t("common.saving") : t("market.skill.create")}
      </Button>
    </form>
  );
}

function MCPPanel() {
  const { t } = useTranslation();
  const toast = useToast();
  const queryClient = useQueryClient();
  const createMutation = useCreateMCPServer();
  const [form, setForm] = useState<CreateMCPServerInput>(initialMCPForm);
  const [argsText, setArgsText] = useState("");
  const [envSchemaText, setEnvSchemaText] = useState("{}");
  const [tagsText, setTagsText] = useState("");

  function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    createMutation.mutate(
      {
        ...form,
        endpoint: form.endpoint?.trim() || null,
        command: form.command?.trim() || null,
        args: splitTags(argsText),
        env_schema: parseJSON(envSchemaText, {}),
        tags: splitTags(tagsText)
      },
      {
        onSuccess: () => {
          setForm(initialMCPForm);
          setArgsText("");
          setEnvSchemaText("{}");
          setTagsText("");
          void queryClient.invalidateQueries({ queryKey: marketplaceItemsQueryKey });
          toast.notify(t("market.mcp.created"));
        }
      }
    );
  }

  return (
    <form className="space-y-4 rounded-lg border border-ink-200 bg-white p-5 shadow-sm" onSubmit={handleCreate}>
      {createMutation.error && (
        <Toast
          message={createMutation.error instanceof ApiError ? createMutation.error.message : t("market.errors.operationFailed")}
          tone="error"
        />
      )}
      <div className="flex items-center gap-2">
        <PackageCheck className="h-5 w-5 text-ocean-600" />
        <h2 className="text-base font-semibold text-ink-900">{t("market.mcp.createTitle")}</h2>
      </div>
      <div className="grid gap-4 md:grid-cols-2">
        <Field label={t("market.mcp.fields.name")}>
          <Input onChange={(event) => setForm({ ...form, name: event.target.value })} required value={form.name} />
        </Field>
        <Field label={t("market.mcp.fields.displayName")}>
          <Input onChange={(event) => setForm({ ...form, display_name: event.target.value })} required value={form.display_name} />
        </Field>
        <Field label={t("market.mcp.fields.transport")}>
          <Select onChange={(event) => setForm({ ...form, transport_type: event.target.value })} value={form.transport_type}>
            <option value="http">HTTP</option>
            <option value="sse">SSE</option>
            <option value="stdio">stdio</option>
          </Select>
        </Field>
        <Field label={t("market.mcp.fields.permission")}>
          <Select onChange={(event) => setForm({ ...form, permission_level: event.target.value })} value={form.permission_level}>
            <option value="readonly">readonly</option>
            <option value="normal">normal</option>
            <option value="sensitive">sensitive</option>
            <option value="destructive">destructive</option>
          </Select>
        </Field>
        <Field label={t("market.mcp.fields.visibility")}>
          <Select onChange={(event) => setForm({ ...form, visibility: event.target.value })} value={form.visibility}>
            <option value="private">{t("market.visibility.private")}</option>
            <option value="public">{t("market.visibility.public")}</option>
          </Select>
        </Field>
        <Field label={t("market.mcp.fields.endpoint")}>
          <Input onChange={(event) => setForm({ ...form, endpoint: event.target.value })} value={form.endpoint ?? ""} />
        </Field>
        <Field label={t("market.mcp.fields.command")}>
          <Input onChange={(event) => setForm({ ...form, command: event.target.value })} value={form.command ?? ""} />
        </Field>
        <Field label={t("market.mcp.fields.args")}>
          <Input onChange={(event) => setArgsText(event.target.value)} value={argsText} />
        </Field>
      </div>
      <Field label={t("market.mcp.fields.description")}>
        <Textarea onChange={(event) => setForm({ ...form, description: event.target.value })} required value={form.description} />
      </Field>
      <Field label={t("market.mcp.fields.envSchema")}>
        <Textarea className="font-mono" onChange={(event) => setEnvSchemaText(event.target.value)} value={envSchemaText} />
      </Field>
      <Field label={t("market.mcp.fields.tags")}>
        <Input onChange={(event) => setTagsText(event.target.value)} value={tagsText} />
      </Field>
      <Button disabled={createMutation.isPending} type="submit">
        {createMutation.isPending ? t("common.saving") : t("market.mcp.create")}
      </Button>
    </form>
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
