import { useQueryClient } from "@tanstack/react-query";
import { KeyRound, Plus, Trash2 } from "lucide-react";
import { FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";

import { Badge } from "../../../components/ui/Badge";
import { Button } from "../../../components/ui/Button";
import { EmptyState } from "../../../components/ui/EmptyState";
import { Input } from "../../../components/ui/Input";
import { LoadingState } from "../../../components/ui/LoadingState";
import { Select } from "../../../components/ui/Select";
import { Switch } from "../../../components/ui/Switch";
import { Toast } from "../../../components/ui/Toast";
import { ApiError } from "../../../lib/errors";
import {
  modelProvidersQueryKey,
  useCreateModelProvider,
  useDeleteModelProvider,
  useModelProviders,
  useUpdateModelProvider
} from "../hooks";
import type { ModelProvider, ProviderKind } from "../types";

type ProviderFormState = {
  provider: ProviderKind;
  display_name: string;
  chat_base_url: string;
  chat_api_key: string;
  embedding_base_url: string;
  embedding_api_key: string;
  default_chat_model: string;
  default_embedding_model: string;
  is_default: boolean;
};

const emptyForm: ProviderFormState = {
  provider: "openai",
  display_name: "",
  chat_base_url: "",
  chat_api_key: "",
  embedding_base_url: "",
  embedding_api_key: "",
  default_chat_model: "",
  default_embedding_model: "",
  is_default: true
};

export function ProvidersPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const providersQuery = useModelProviders();
  const createMutation = useCreateModelProvider();
  const updateMutation = useUpdateModelProvider();
  const deleteMutation = useDeleteModelProvider();
  const [form, setForm] = useState<ProviderFormState>(emptyForm);
  const [editingID, setEditingID] = useState<string | null>(null);

  const providers = providersQuery.data ?? [];
  const editingProvider = providers.find((provider) => provider.id === editingID);
  const mutationError =
    createMutation.error ?? updateMutation.error ?? deleteMutation.error ?? null;

  function resetForm() {
    setForm(emptyForm);
    setEditingID(null);
  }

  function editProvider(provider: ModelProvider) {
    setEditingID(provider.id);
    setForm({
      provider: provider.provider,
      display_name: provider.display_name,
      chat_base_url: provider.chat_base_url ?? "",
      chat_api_key: "",
      embedding_base_url: provider.embedding_base_url ?? "",
      embedding_api_key: "",
      default_chat_model: provider.default_chat_model,
      default_embedding_model: provider.default_embedding_model ?? "",
      is_default: provider.is_default
    });
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (editingProvider) {
      updateMutation.mutate(
        {
          id: editingProvider.id,
          input: {
            display_name: form.display_name,
            chat_base_url: nullable(form.chat_base_url),
            ...(form.chat_api_key ? { chat_api_key: form.chat_api_key } : {}),
            embedding_base_url: nullable(form.embedding_base_url),
            ...(form.embedding_api_key
              ? { embedding_api_key: form.embedding_api_key }
              : {}),
            default_chat_model: form.default_chat_model,
            default_embedding_model: nullable(form.default_embedding_model),
            is_default: form.is_default
          }
        },
        {
          onSuccess: () => {
            void queryClient.invalidateQueries({ queryKey: modelProvidersQueryKey });
            resetForm();
          }
        }
      );
      return;
    }

    createMutation.mutate(
      {
        provider: form.provider,
        display_name: form.display_name,
        chat_base_url: nullable(form.chat_base_url),
        chat_api_key: form.chat_api_key,
        embedding_base_url: nullable(form.embedding_base_url),
        embedding_api_key: nullable(form.embedding_api_key),
        default_chat_model: form.default_chat_model,
        default_embedding_model: nullable(form.default_embedding_model),
        is_default: form.is_default
      },
      {
        onSuccess: () => {
          void queryClient.invalidateQueries({ queryKey: modelProvidersQueryKey });
          resetForm();
        }
      }
    );
  }

  function deleteProvider(provider: ModelProvider) {
    if (!window.confirm(t("providers.deleteConfirm", { name: provider.display_name }))) {
      return;
    }
    deleteMutation.mutate(provider.id, {
      onSuccess: () => {
        void queryClient.invalidateQueries({ queryKey: modelProvidersQueryKey });
        if (editingID === provider.id) {
          resetForm();
        }
      }
    });
  }

  return (
    <section className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_420px]">
      <div className="space-y-4">
        <div>
          <h1 className="text-2xl font-semibold text-ink-900">
            {t("providers.title")}
          </h1>
          <p className="mt-2 text-sm leading-6 text-ink-500">
            {t("providers.description")}
          </p>
        </div>

        {mutationError && (
          <Toast
            message={
              mutationError instanceof ApiError
                ? mutationError.message
                : t("providers.errors.mutationFailed")
            }
            tone="error"
          />
        )}

        {providersQuery.isLoading ? (
          <LoadingState />
        ) : providers.length === 0 ? (
          <EmptyState
            description={t("providers.empty.description")}
            icon={<KeyRound className="h-8 w-8" />}
            title={t("providers.empty.title")}
          />
        ) : (
          <div className="grid gap-3 md:grid-cols-2">
            {providers.map((provider) => (
              <article
                className="rounded-lg border border-ink-200 bg-white p-4 shadow-sm"
                key={provider.id}
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <h2 className="truncate text-base font-semibold text-ink-900">
                      {provider.display_name}
                    </h2>
                    <p className="mt-1 text-sm text-ink-500">
                      {provider.provider} · {provider.default_chat_model}
                    </p>
                  </div>
                  <div className="flex gap-1">
                    {provider.is_default && <Badge tone="green">{t("providers.default")}</Badge>}
                    <Badge tone={provider.status === "active" ? "blue" : "amber"}>
                      {provider.status}
                    </Badge>
                  </div>
                </div>

                <dl className="mt-4 grid gap-2 text-sm">
                  <ProviderMeta label={t("providers.chatBaseURL")} value={provider.chat_base_url} />
                  <ProviderMeta
                    label={t("providers.embeddingBaseURL")}
                    value={provider.embedding_base_url}
                  />
                  <ProviderMeta
                    label={t("providers.embeddingModel")}
                    value={provider.default_embedding_model}
                  />
                  <ProviderMeta
                    label={t("providers.apiKeys")}
                    value={[
                      provider.has_chat_api_key ? t("providers.chatKeyConfigured") : t("providers.chatKeyMissing"),
                      provider.has_embedding_api_key
                        ? t("providers.embeddingKeyConfigured")
                        : t("providers.embeddingKeyMissing")
                    ].join(" / ")}
                  />
                </dl>

                <div className="mt-4 flex gap-2">
                  <Button onClick={() => editProvider(provider)} variant="secondary">
                    {t("common.edit")}
                  </Button>
                  <Button
                    icon={<Trash2 className="h-4 w-4" />}
                    onClick={() => deleteProvider(provider)}
                    variant="ghost"
                  >
                    {t("common.delete")}
                  </Button>
                </div>
              </article>
            ))}
          </div>
        )}
      </div>

      <form
        className="h-fit rounded-lg border border-ink-200 bg-white p-5 shadow-soft"
        onSubmit={handleSubmit}
      >
        <div className="flex items-center justify-between gap-3">
          <h2 className="text-base font-semibold text-ink-900">
            {editingProvider ? t("providers.form.editTitle") : t("providers.form.createTitle")}
          </h2>
          {editingProvider && (
            <Button onClick={resetForm} variant="ghost">
              {t("common.cancel")}
            </Button>
          )}
        </div>

        <div className="mt-5 space-y-4">
          <Field label={t("providers.provider")}>
            <Select
              disabled={Boolean(editingProvider)}
              onChange={(event) =>
                setForm({ ...form, provider: event.target.value as ProviderKind })
              }
              value={form.provider}
            >
              <option value="openai">OpenAI-compatible</option>
              <option value="anthropic">Anthropic-compatible</option>
            </Select>
          </Field>
          <Field label={t("providers.displayName")}>
            <Input
              onChange={(event) => setForm({ ...form, display_name: event.target.value })}
              required
              value={form.display_name}
            />
          </Field>
          <Field label={t("providers.chatBaseURL")}>
            <Input
              onChange={(event) => setForm({ ...form, chat_base_url: event.target.value })}
              placeholder="https://api.openai.com/v1"
              value={form.chat_base_url}
            />
          </Field>
          <Field label={t("providers.chatAPIKey")}>
            <Input
              onChange={(event) => setForm({ ...form, chat_api_key: event.target.value })}
              placeholder={editingProvider ? t("providers.keepExistingKey") : "sk-..."}
              required={!editingProvider}
              type="password"
              value={form.chat_api_key}
            />
          </Field>
          <Field label={t("providers.chatModel")}>
            <Input
              onChange={(event) =>
                setForm({ ...form, default_chat_model: event.target.value })
              }
              required
              value={form.default_chat_model}
            />
          </Field>
          <Field label={t("providers.embeddingBaseURL")}>
            <Input
              onChange={(event) =>
                setForm({ ...form, embedding_base_url: event.target.value })
              }
              placeholder="https://api.siliconflow.cn/v1"
              value={form.embedding_base_url}
            />
          </Field>
          <Field label={t("providers.embeddingAPIKey")}>
            <Input
              onChange={(event) =>
                setForm({ ...form, embedding_api_key: event.target.value })
              }
              placeholder={editingProvider ? t("providers.keepExistingKey") : "sk-..."}
              type="password"
              value={form.embedding_api_key}
            />
          </Field>
          <Field label={t("providers.embeddingModel")}>
            <Input
              onChange={(event) =>
                setForm({ ...form, default_embedding_model: event.target.value })
              }
              placeholder="Pro/BAAI/bge-m3"
              value={form.default_embedding_model}
            />
          </Field>
          <label className="flex items-center justify-between rounded-md border border-ink-200 px-3 py-2">
            <span className="text-sm font-medium text-ink-700">{t("providers.isDefault")}</span>
            <Switch
              checked={form.is_default}
              onClick={() => setForm({ ...form, is_default: !form.is_default })}
            />
          </label>
        </div>

        <Button
          className="mt-5 w-full"
          disabled={createMutation.isPending || updateMutation.isPending}
          icon={<Plus className="h-4 w-4" />}
          type="submit"
        >
          {editingProvider ? t("common.save") : t("providers.form.create")}
        </Button>
      </form>
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

function ProviderMeta({ label, value }: { label: string; value?: string | null }) {
  return (
    <div>
      <dt className="text-xs text-ink-500">{label}</dt>
      <dd className="mt-0.5 truncate text-ink-700">{value || "-"}</dd>
    </div>
  );
}

function nullable(value: string) {
  const trimmed = value.trim();
  return trimmed === "" ? null : trimmed;
}
