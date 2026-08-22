import { useQueryClient } from "@tanstack/react-query";
import { Save } from "lucide-react";
import { FormEvent, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";

import { Button } from "../../../components/ui/Button";
import { Input } from "../../../components/ui/Input";
import { LoadingState } from "../../../components/ui/LoadingState";
import { Select } from "../../../components/ui/Select";
import { Switch } from "../../../components/ui/Switch";
import { Textarea } from "../../../components/ui/Textarea";
import { Toast } from "../../../components/ui/Toast";
import { ApiError } from "../../../lib/errors";
import {
  agentConfigQueryKey,
  useAgentConfig,
  useModelProviders,
  useUpdateAgentConfig
} from "../hooks";
import type {
  AgentConfig,
  LLMFeatureSettingUpdate,
  ThinkingEffort,
  ToolApprovalPolicy
} from "../types";

type AgentFormState = {
  name: string;
  system_prompt: string;
  default_provider_id: string;
  temperature: number;
  thinking_enabled: boolean;
  thinking_effort: ThinkingEffort;
  thinking_budget_tokens: number;
  max_context_tokens: number;
  max_loop_steps: number;
  llm_retry_limit: number;
  memory_enabled: boolean;
  tool_use_enabled: boolean;
  tool_approval_policy: ToolApprovalPolicy;
  dreaming_enabled: boolean;
  semantic_memory_enabled: boolean;
  embedding_enabled: boolean;
  embedding_monthly_tokens: number;
  embed_public_knowledge: boolean;
  llm_feature_settings: LLMFeatureSettingUpdate[];
};

export function AgentConfigPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const configQuery = useAgentConfig();
  const providersQuery = useModelProviders();
  const updateMutation = useUpdateAgentConfig();
  const [form, setForm] = useState<AgentFormState | null>(null);

  useEffect(() => {
    if (configQuery.data) {
      setForm(toForm(configQuery.data));
    }
  }, [configQuery.data]);

  if (configQuery.isLoading || !form) {
    return <LoadingState />;
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!form) {
      return;
    }
    updateMutation.mutate(
      {
        name: form.name,
        system_prompt: form.system_prompt,
        default_provider_id: form.default_provider_id || null,
        temperature: Number(form.temperature),
        thinking_enabled: form.thinking_enabled,
        thinking_effort: form.thinking_effort,
        thinking_budget_tokens: Number(form.thinking_budget_tokens),
        max_context_tokens: Number(form.max_context_tokens),
        max_loop_steps: Number(form.max_loop_steps),
        llm_retry_limit: Number(form.llm_retry_limit),
        memory_enabled: form.memory_enabled,
        tool_use_enabled: form.tool_use_enabled,
        tool_approval_policy: form.tool_approval_policy,
        dreaming_enabled: form.dreaming_enabled,
        semantic_memory_enabled: form.semantic_memory_enabled,
        embedding_enabled: form.embedding_enabled,
        embedding_cost_policy: {
          mode: "manual",
          max_monthly_tokens: Number(form.embedding_monthly_tokens),
          embed_public_knowledge: form.embed_public_knowledge
        },
        llm_feature_settings: form.llm_feature_settings.map((setting) => ({
          feature_key: setting.feature_key,
          enabled: setting.enabled,
          provider_id: setting.provider_id || null,
          model_override: setting.model_override || null,
          temperature: setting.temperature
        }))
      },
      {
        onSuccess: (updated) => {
          queryClient.setQueryData(agentConfigQueryKey, updated);
          setForm(toForm(updated));
        }
      }
    );
  }

  const providers = providersQuery.data ?? [];
  const errorMessage =
    updateMutation.error instanceof ApiError
      ? updateMutation.error.message
      : updateMutation.error
        ? t("agent.errors.saveFailed")
        : null;

  return (
    <form className="space-y-5" onSubmit={handleSubmit}>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-ink-900">{t("agent.title")}</h1>
          <p className="mt-2 text-sm leading-6 text-ink-500">{t("agent.description")}</p>
        </div>
        <Button
          disabled={updateMutation.isPending}
          icon={<Save className="h-4 w-4" />}
          type="submit"
        >
          {updateMutation.isPending ? t("agent.saving") : t("common.save")}
        </Button>
      </div>

      {errorMessage && <Toast message={errorMessage} tone="error" />}
      {updateMutation.isSuccess && <Toast message={t("agent.saved")} tone="success" />}

      <Section title={t("agent.sections.basic")}>
        <Field label={t("agent.fields.name")}>
          <Input
            onChange={(event) => setForm({ ...form, name: event.target.value })}
            required
            value={form.name}
          />
        </Field>
        <Field label={t("agent.fields.defaultProvider")}>
          <Select
            onChange={(event) =>
              setForm({ ...form, default_provider_id: event.target.value })
            }
            value={form.default_provider_id}
          >
            <option value="">{t("agent.none")}</option>
            {providers.map((provider) => (
              <option key={provider.id} value={provider.id}>
                {provider.display_name} · {provider.default_chat_model}
              </option>
            ))}
          </Select>
        </Field>
        <Field label={t("agent.fields.systemPrompt")}>
          <Textarea
            onChange={(event) =>
              setForm({ ...form, system_prompt: event.target.value })
            }
            value={form.system_prompt}
          />
        </Field>
      </Section>

      <Section title={t("agent.sections.model")}>
        <NumberField
          label={t("agent.fields.temperature")}
          max={2}
          min={0}
          onChange={(value) => setForm({ ...form, temperature: value })}
          step={0.1}
          value={form.temperature}
        />
        <ToggleField
          checked={form.thinking_enabled}
          label={t("agent.fields.thinkingEnabled")}
          onChange={(checked) => setForm({ ...form, thinking_enabled: checked })}
        />
        <Field label={t("agent.fields.thinkingEffort")}>
          <Select
            onChange={(event) =>
              setForm({ ...form, thinking_effort: event.target.value as ThinkingEffort })
            }
            value={form.thinking_effort}
          >
            <option value="low">{t("agent.thinking.low")}</option>
            <option value="medium">{t("agent.thinking.medium")}</option>
            <option value="high">{t("agent.thinking.high")}</option>
          </Select>
        </Field>
        <NumberField
          label={t("agent.fields.thinkingBudgetTokens")}
          min={0}
          onChange={(value) => setForm({ ...form, thinking_budget_tokens: value })}
          step={100}
          value={form.thinking_budget_tokens}
        />
        <NumberField
          label={t("agent.fields.maxContextTokens")}
          min={1}
          onChange={(value) => setForm({ ...form, max_context_tokens: value })}
          step={1000}
          value={form.max_context_tokens}
        />
        <NumberField
          label={t("agent.fields.maxLoopSteps")}
          min={1}
          onChange={(value) => setForm({ ...form, max_loop_steps: value })}
          step={1}
          value={form.max_loop_steps}
        />
        <NumberField
          label={t("agent.fields.llmRetryLimit")}
          min={0}
          onChange={(value) => setForm({ ...form, llm_retry_limit: value })}
          step={1}
          value={form.llm_retry_limit}
        />
      </Section>

      <Section title={t("agent.sections.capabilities")}>
        <ToggleField
          checked={form.memory_enabled}
          label={t("agent.fields.memoryEnabled")}
          onChange={(checked) => setForm({ ...form, memory_enabled: checked })}
        />
        <ToggleField
          checked={form.semantic_memory_enabled}
          label={t("agent.fields.semanticMemoryEnabled")}
          onChange={(checked) =>
            setForm({ ...form, semantic_memory_enabled: checked })
          }
        />
        <ToggleField
          checked={form.dreaming_enabled}
          label={t("agent.fields.dreamingEnabled")}
          onChange={(checked) => setForm({ ...form, dreaming_enabled: checked })}
        />
        <ToggleField
          checked={form.tool_use_enabled}
          label={t("agent.fields.toolUseEnabled")}
          onChange={(checked) => setForm({ ...form, tool_use_enabled: checked })}
        />
        <Field label={t("agent.fields.toolApprovalPolicy")}>
          <Select
            onChange={(event) =>
              setForm({
                ...form,
                tool_approval_policy: event.target.value as ToolApprovalPolicy
              })
            }
            value={form.tool_approval_policy}
          >
            <option value="never">{t("agent.toolApproval.never")}</option>
            <option value="sensitive_only">{t("agent.toolApproval.sensitiveOnly")}</option>
            <option value="always">{t("agent.toolApproval.always")}</option>
          </Select>
        </Field>
      </Section>

      <Section title={t("agent.sections.embedding")}>
        <ToggleField
          checked={form.embedding_enabled}
          label={t("agent.fields.embeddingEnabled")}
          onChange={(checked) => setForm({ ...form, embedding_enabled: checked })}
        />
        <NumberField
          label={t("agent.fields.embeddingMonthlyTokens")}
          min={0}
          onChange={(value) => setForm({ ...form, embedding_monthly_tokens: value })}
          step={1000}
          value={form.embedding_monthly_tokens}
        />
        <ToggleField
          checked={form.embed_public_knowledge}
          label={t("agent.fields.embedPublicKnowledge")}
          onChange={(checked) =>
            setForm({ ...form, embed_public_knowledge: checked })
          }
        />
      </Section>

      <Section title={t("agent.sections.features")}>
        <div className="space-y-3">
          {form.llm_feature_settings.map((setting, index) => (
            <div
              className="grid gap-3 rounded-md border border-ink-200 p-3 lg:grid-cols-[1.4fr_1fr_1fr_120px]"
              key={setting.feature_key}
            >
              <ToggleField
                checked={setting.enabled}
                label={t(`agent.features.${setting.feature_key}`, {
                  defaultValue: setting.feature_key
                })}
                onChange={(checked) =>
                  updateFeature(index, { ...setting, enabled: checked })
                }
              />
              <Field label={t("agent.fields.featureProvider")}>
                <Select
                  onChange={(event) =>
                    updateFeature(index, {
                      ...setting,
                      provider_id: event.target.value || null
                    })
                  }
                  value={setting.provider_id ?? ""}
                >
                  <option value="">{t("agent.defaultChatProvider")}</option>
                  {providers.map((provider) => (
                    <option key={provider.id} value={provider.id}>
                      {provider.display_name}
                    </option>
                  ))}
                </Select>
              </Field>
              <Field label={t("agent.fields.modelOverride")}>
                <Input
                  onChange={(event) =>
                    updateFeature(index, {
                      ...setting,
                      model_override: event.target.value
                    })
                  }
                  value={setting.model_override ?? ""}
                />
              </Field>
              <NumberField
                label={t("agent.fields.featureTemperature")}
                max={2}
                min={0}
                onChange={(value) => updateFeature(index, { ...setting, temperature: value })}
                step={0.1}
                value={setting.temperature ?? 0}
              />
            </div>
          ))}
        </div>
      </Section>
    </form>
  );

  function updateFeature(index: number, setting: LLMFeatureSettingUpdate) {
    setForm((current) =>
      current
        ? {
            ...current,
            llm_feature_settings: current.llm_feature_settings.map((item, itemIndex) =>
              itemIndex === index ? setting : item
            )
          }
        : current
    );
  }
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="rounded-lg border border-ink-200 bg-white p-5 shadow-sm">
      <h2 className="text-base font-semibold text-ink-900">{title}</h2>
      <div className="mt-4 grid gap-4 md:grid-cols-2">{children}</div>
    </section>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <label className="block space-y-2">
      <span className="text-sm font-medium text-ink-700">{label}</span>
      {children}
    </label>
  );
}

function NumberField({
  label,
  value,
  min,
  max,
  step,
  onChange
}: {
  label: string;
  value: number;
  min?: number;
  max?: number;
  step?: number;
  onChange: (value: number) => void;
}) {
  return (
    <Field label={label}>
      <Input
        max={max}
        min={min}
        onChange={(event) => onChange(Number(event.target.value))}
        step={step}
        type="number"
        value={value}
      />
    </Field>
  );
}

function ToggleField({
  label,
  checked,
  onChange
}: {
  label: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
}) {
  return (
    <label className="flex items-center justify-between rounded-md border border-ink-200 px-3 py-2">
      <span className="text-sm font-medium text-ink-700">{label}</span>
      <Switch checked={checked} onClick={() => onChange(!checked)} />
    </label>
  );
}

function toForm(config: AgentConfig): AgentFormState {
  const embeddingPolicy = config.embedding_cost_policy ?? {};
  return {
    name: config.name,
    system_prompt: config.system_prompt,
    default_provider_id: config.default_provider_id ?? "",
    temperature: config.temperature,
    thinking_enabled: config.thinking_enabled,
    thinking_effort: config.thinking_effort,
    thinking_budget_tokens: config.thinking_budget_tokens,
    max_context_tokens: config.max_context_tokens,
    max_loop_steps: config.max_loop_steps,
    llm_retry_limit: config.llm_retry_limit,
    memory_enabled: config.memory_enabled,
    tool_use_enabled: config.tool_use_enabled,
    tool_approval_policy: config.tool_approval_policy,
    dreaming_enabled: config.dreaming_enabled,
    semantic_memory_enabled: config.semantic_memory_enabled,
    embedding_enabled: config.embedding_enabled,
    embedding_monthly_tokens:
      typeof embeddingPolicy.max_monthly_tokens === "number"
        ? embeddingPolicy.max_monthly_tokens
        : 0,
    embed_public_knowledge: Boolean(embeddingPolicy.embed_public_knowledge),
    llm_feature_settings: config.llm_feature_settings.map((setting) => ({
      feature_key: setting.feature_key,
      enabled: setting.enabled,
      provider_id: setting.provider_id,
      model_override: setting.model_override,
      temperature: setting.temperature
    }))
  };
}
