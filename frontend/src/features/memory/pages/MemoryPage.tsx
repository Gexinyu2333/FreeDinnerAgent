import { useQueryClient } from "@tanstack/react-query";
import { Brain, Check, RotateCcw, Search, Sparkles, X } from "lucide-react";
import { FormEvent, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";

import { Badge } from "../../../components/ui/Badge";
import { Button } from "../../../components/ui/Button";
import { EmptyState } from "../../../components/ui/EmptyState";
import { Input } from "../../../components/ui/Input";
import { LoadingState } from "../../../components/ui/LoadingState";
import { Select } from "../../../components/ui/Select";
import { Tabs } from "../../../components/ui/Tabs";
import { Textarea } from "../../../components/ui/Textarea";
import { Toast } from "../../../components/ui/Toast";
import { ApiError } from "../../../lib/errors";
import { formatDateTime } from "../../../lib/format";
import {
  dreamingInsightsQueryKey,
  profileMemoriesQueryKey,
  useApplyDreamingInsight,
  useCreateProfileMemory,
  useDreamingInsights,
  useMemoryContext,
  useMemoryTypes,
  useProfileMemories,
  useRejectDreamingInsight,
  useSearchProfileMemories
} from "../hooks";
import type { CreateProfileMemoryInput, MemoryChunk, ProfileMemory } from "../types";

type MemoryTab = "profile" | "context" | "dreaming";

const initialForm: CreateProfileMemoryInput = {
  memory_type: "preference",
  scope: "global",
  title: "",
  content: "",
  evidence: "",
  confidence: 0.8,
  importance: 3
};

export function MemoryPage() {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState<MemoryTab>("profile");

  return (
    <section className="space-y-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-ink-900">{t("memory.title")}</h1>
          <p className="mt-2 text-sm leading-6 text-ink-500">{t("memory.description")}</p>
        </div>
        <Tabs
          activeKey={activeTab}
          items={[
            { key: "profile", label: t("memory.tabs.profile") },
            { key: "context", label: t("memory.tabs.context") },
            { key: "dreaming", label: t("memory.tabs.dreaming") }
          ]}
          onChange={(key) => setActiveTab(key as MemoryTab)}
        />
      </div>

      {activeTab === "profile" && <ProfileMemoryPanel />}
      {activeTab === "context" && <MemoryContextPanel />}
      {activeTab === "dreaming" && <DreamingInsightsPanel />}
    </section>
  );
}

function ProfileMemoryPanel() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const typesQuery = useMemoryTypes();
  const [typeFilter, setTypeFilter] = useState("");
  const memoriesQuery = useProfileMemories(typeFilter || undefined);
  const createMutation = useCreateProfileMemory();
  const searchMutation = useSearchProfileMemories();
  const [form, setForm] = useState<CreateProfileMemoryInput>(initialForm);
  const [searchQuery, setSearchQuery] = useState("");

  const memoryTypes = typesQuery.data ?? [];
  const profileMemories = memoriesQuery.data ?? [];
  const searchResults = searchMutation.data?.memories ?? [];
  const visibleMemories = searchMutation.data ? searchResults : profileMemories;

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    createMutation.mutate(
      {
        ...form,
        title: form.title.trim(),
        content: form.content.trim(),
        evidence: form.evidence?.trim() || null,
        confidence: Number(form.confidence),
        importance: Number(form.importance)
      },
      {
        onSuccess: () => {
          setForm({ ...initialForm, memory_type: form.memory_type });
          void queryClient.invalidateQueries({ queryKey: profileMemoriesQueryKey });
        }
      }
    );
  }

  function handleSearch(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!searchQuery.trim()) {
      searchMutation.reset();
      return;
    }
    searchMutation.mutate({ q: searchQuery.trim(), limit: 10 });
  }

  const mutationError = createMutation.error ?? searchMutation.error ?? null;

  return (
    <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_420px]">
      <div className="space-y-4">
        {mutationError && (
          <Toast
            message={
              mutationError instanceof ApiError
                ? mutationError.message
                : t("memory.errors.operationFailed")
            }
            tone="error"
          />
        )}
        {createMutation.isSuccess && (
          <Toast message={t("memory.profile.created")} tone="success" />
        )}

        <form className="flex flex-col gap-3 rounded-lg border border-ink-200 bg-white p-4 sm:flex-row" onSubmit={handleSearch}>
          <Input
            onChange={(event) => setSearchQuery(event.target.value)}
            placeholder={t("memory.profile.searchPlaceholder")}
            value={searchQuery}
          />
          <Button
            disabled={searchMutation.isPending}
            icon={<Search className="h-4 w-4" />}
            type="submit"
          >
            {t("common.search")}
          </Button>
          {searchMutation.data && (
            <Button onClick={() => searchMutation.reset()} variant="ghost">
              {t("common.reset")}
            </Button>
          )}
        </form>

        <div className="flex items-center gap-3">
          <Select
            className="w-56"
            onChange={(event) => setTypeFilter(event.target.value)}
            value={typeFilter}
          >
            <option value="">{t("memory.profile.allTypes")}</option>
            {memoryTypes.map((type) => (
              <option key={type.memory_type} value={type.memory_type}>
                {type.display_name}
              </option>
            ))}
          </Select>
          {searchMutation.data && (
            <Badge tone="blue">
              {t("memory.profile.searchMode", { mode: searchMutation.data.mode })}
            </Badge>
          )}
        </div>

        {memoriesQuery.isLoading || typesQuery.isLoading ? (
          <LoadingState />
        ) : visibleMemories.length === 0 ? (
          <EmptyState
            description={t("memory.profile.empty.description")}
            icon={<Brain className="h-8 w-8" />}
            title={t("memory.profile.empty.title")}
          />
        ) : (
          <div className="grid gap-3 md:grid-cols-2">
            {visibleMemories.map((memory) => (
              <ProfileMemoryCard key={memory.id} memory={memory} />
            ))}
          </div>
        )}
      </div>

      <form className="h-fit rounded-lg border border-ink-200 bg-white p-5 shadow-soft" onSubmit={handleSubmit}>
        <h2 className="text-base font-semibold text-ink-900">
          {t("memory.profile.formTitle")}
        </h2>
        <div className="mt-5 space-y-4">
          <Field label={t("memory.profile.fields.memoryType")}>
            <Select
              onChange={(event) => setForm({ ...form, memory_type: event.target.value })}
              value={form.memory_type}
            >
              {memoryTypes.map((type) => (
                <option key={type.memory_type} value={type.memory_type}>
                  {type.display_name}
                </option>
              ))}
              {memoryTypes.length === 0 && <option value="preference">preference</option>}
            </Select>
          </Field>
          <Field label={t("memory.profile.fields.scope")}>
            <Select
              onChange={(event) => setForm({ ...form, scope: event.target.value })}
              value={form.scope}
            >
              <option value="global">{t("memory.scope.global")}</option>
              <option value="conversation">{t("memory.scope.conversation")}</option>
              <option value="project">{t("memory.scope.project")}</option>
            </Select>
          </Field>
          <Field label={t("memory.profile.fields.title")}>
            <Input
              onChange={(event) => setForm({ ...form, title: event.target.value })}
              required
              value={form.title}
            />
          </Field>
          <Field label={t("memory.profile.fields.content")}>
            <Textarea
              onChange={(event) => setForm({ ...form, content: event.target.value })}
              required
              value={form.content}
            />
          </Field>
          <Field label={t("memory.profile.fields.evidence")}>
            <Textarea
              onChange={(event) => setForm({ ...form, evidence: event.target.value })}
              value={form.evidence ?? ""}
            />
          </Field>
          <div className="grid gap-3 sm:grid-cols-2">
            <Field label={t("memory.profile.fields.confidence")}>
              <Input
                max={1}
                min={0}
                onChange={(event) =>
                  setForm({ ...form, confidence: Number(event.target.value) })
                }
                step={0.05}
                type="number"
                value={form.confidence}
              />
            </Field>
            <Field label={t("memory.profile.fields.importance")}>
              <Input
                max={5}
                min={1}
                onChange={(event) =>
                  setForm({ ...form, importance: Number(event.target.value) })
                }
                step={1}
                type="number"
                value={form.importance}
              />
            </Field>
          </div>
          <Button disabled={createMutation.isPending} type="submit">
            {createMutation.isPending ? t("memory.profile.creating") : t("memory.profile.create")}
          </Button>
        </div>
      </form>
    </div>
  );
}

function ProfileMemoryCard({ memory }: { memory: ProfileMemory }) {
  const { t } = useTranslation();
  return (
    <article className="rounded-lg border border-ink-200 bg-white p-4 shadow-sm">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <h2 className="truncate text-base font-semibold text-ink-900">{memory.title}</h2>
          <p className="mt-1 text-xs text-ink-500">{formatDateTime(memory.updated_at)}</p>
        </div>
        <Badge tone={memory.status === "active" ? "green" : "amber"}>
          {memory.status}
        </Badge>
      </div>
      <p className="mt-3 text-sm leading-6 text-ink-700">{memory.content}</p>
      {memory.evidence && (
        <p className="mt-3 rounded-md bg-ink-50 px-3 py-2 text-xs leading-5 text-ink-500">
          {memory.evidence}
        </p>
      )}
      <div className="mt-4 flex flex-wrap gap-2">
        <Badge tone="blue">{memory.memory_type}</Badge>
        <Badge>{memory.scope}</Badge>
        <Badge>{t("memory.profile.confidence", { value: memory.confidence.toFixed(2) })}</Badge>
        <Badge>{t("memory.profile.importance", { value: memory.importance })}</Badge>
      </div>
    </article>
  );
}

function MemoryContextPanel() {
  const { t } = useTranslation();
  const contextMutation = useMemoryContext();
  const [conversationID, setConversationID] = useState("");
  const [query, setQuery] = useState("");
  const [includeSemantic, setIncludeSemantic] = useState(true);

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    contextMutation.mutate({
      conversation_id: conversationID.trim(),
      q: query.trim(),
      semantic: includeSemantic
    });
  }

  const chunks = contextMutation.data?.chunks ?? [];
  const chunksByLayer = useMemo(() => groupByLayer(chunks), [chunks]);

  return (
    <div className="space-y-4">
      {contextMutation.error && (
        <Toast
          message={
            contextMutation.error instanceof ApiError
              ? contextMutation.error.message
              : t("memory.errors.operationFailed")
          }
          tone="error"
        />
      )}
      <form className="rounded-lg border border-ink-200 bg-white p-5 shadow-sm" onSubmit={handleSubmit}>
        <div className="grid gap-4 lg:grid-cols-[280px_minmax(0,1fr)_170px]">
          <Field label={t("memory.context.fields.conversationId")}>
            <Input
              onChange={(event) => setConversationID(event.target.value)}
              required
              value={conversationID}
            />
          </Field>
          <Field label={t("memory.context.fields.query")}>
            <Input
              onChange={(event) => setQuery(event.target.value)}
              required
              value={query}
            />
          </Field>
          <Field label={t("memory.context.fields.semantic")}>
            <Select
              onChange={(event) => setIncludeSemantic(event.target.value === "true")}
              value={String(includeSemantic)}
            >
              <option value="true">{t("common.enabled")}</option>
              <option value="false">{t("common.disabled")}</option>
            </Select>
          </Field>
        </div>
        <Button
          className="mt-4"
          disabled={contextMutation.isPending}
          icon={<RotateCcw className="h-4 w-4" />}
          type="submit"
        >
          {t("memory.context.retrieve")}
        </Button>
      </form>

      {contextMutation.isPending ? (
        <LoadingState />
      ) : contextMutation.data ? (
        <div className="space-y-4">
          <div className="flex flex-wrap gap-2">
            <Badge tone="blue">
              {t("memory.context.tokenCount", { count: contextMutation.data.token_count })}
            </Badge>
            {contextMutation.data.used_layers.map((layer) => (
              <Badge key={layer} tone="green">
                {layer}
              </Badge>
            ))}
            {contextMutation.data.semantic_mode && (
              <Badge>{t("memory.context.semanticMode", { mode: contextMutation.data.semantic_mode })}</Badge>
            )}
          </div>
          {chunks.length === 0 ? (
            <EmptyState
              description={t("memory.context.empty.description")}
              title={t("memory.context.empty.title")}
            />
          ) : (
            Object.entries(chunksByLayer).map(([layer, items]) => (
              <div className="space-y-3" key={layer}>
                <h2 className="text-sm font-semibold uppercase tracking-wide text-ink-500">
                  {layer}
                </h2>
                <div className="grid gap-3 md:grid-cols-2">
                  {items.map((chunk) => (
                    <MemoryChunkCard chunk={chunk} key={`${chunk.layer}-${chunk.ref_id}`} />
                  ))}
                </div>
              </div>
            ))
          )}
        </div>
      ) : null}
    </div>
  );
}

function MemoryChunkCard({ chunk }: { chunk: MemoryChunk }) {
  const { t } = useTranslation();
  return (
    <article className="rounded-lg border border-ink-200 bg-white p-4 shadow-sm">
      <div className="flex flex-wrap gap-2">
        <Badge tone="blue">{chunk.source}</Badge>
        <Badge>{chunk.load_mode}</Badge>
        <Badge>{t("memory.context.score", { value: chunk.score.toFixed(2) })}</Badge>
        <Badge>{t("memory.context.tokens", { count: chunk.token_count })}</Badge>
      </div>
      <p className="mt-3 text-sm leading-6 text-ink-700">{chunk.content}</p>
    </article>
  );
}

function DreamingInsightsPanel() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [status, setStatus] = useState("pending");
  const insightsQuery = useDreamingInsights(status || undefined);
  const applyMutation = useApplyDreamingInsight();
  const rejectMutation = useRejectDreamingInsight();
  const insights = insightsQuery.data ?? [];

  function refresh() {
    void queryClient.invalidateQueries({ queryKey: dreamingInsightsQueryKey });
    void queryClient.invalidateQueries({ queryKey: profileMemoriesQueryKey });
  }

  const mutationError = applyMutation.error ?? rejectMutation.error ?? null;

  return (
    <div className="space-y-4">
      {mutationError && (
        <Toast
          message={
            mutationError instanceof ApiError
              ? mutationError.message
              : t("memory.errors.operationFailed")
          }
          tone="error"
        />
      )}
      <div className="flex items-center gap-3">
        <Select className="w-48" onChange={(event) => setStatus(event.target.value)} value={status}>
          <option value="pending">{t("memory.dreaming.status.pending")}</option>
          <option value="applied">{t("memory.dreaming.status.applied")}</option>
          <option value="rejected">{t("memory.dreaming.status.rejected")}</option>
          <option value="">{t("memory.dreaming.status.all")}</option>
        </Select>
      </div>
      {insightsQuery.isLoading ? (
        <LoadingState />
      ) : insights.length === 0 ? (
        <EmptyState
          description={t("memory.dreaming.empty.description")}
          icon={<Sparkles className="h-8 w-8" />}
          title={t("memory.dreaming.empty.title")}
        />
      ) : (
        <div className="grid gap-3 md:grid-cols-2">
          {insights.map((insight) => (
            <article className="rounded-lg border border-ink-200 bg-white p-4 shadow-sm" key={insight.id}>
              <div className="flex items-start justify-between gap-3">
                <div>
                  <h2 className="text-base font-semibold text-ink-900">{insight.insight_type}</h2>
                  <p className="mt-1 text-xs text-ink-500">{formatDateTime(insight.created_at)}</p>
                </div>
                <Badge tone={insight.status === "pending" ? "amber" : "green"}>
                  {insight.status}
                </Badge>
              </div>
              <p className="mt-3 text-sm leading-6 text-ink-700">{insight.content}</p>
              <div className="mt-4 flex flex-wrap gap-2">
                <Badge>{insight.source_layer}</Badge>
                {insight.target_layer && <Badge tone="blue">{insight.target_layer}</Badge>}
                <Badge>
                  {t("memory.dreaming.confidence", { value: insight.confidence.toFixed(2) })}
                </Badge>
              </div>
              {insight.status === "pending" && (
                <div className="mt-4 flex gap-2">
                  <Button
                    disabled={applyMutation.isPending}
                    icon={<Check className="h-4 w-4" />}
                    onClick={() =>
                      applyMutation.mutate(insight.id, {
                        onSuccess: refresh
                      })
                    }
                  >
                    {t("memory.dreaming.apply")}
                  </Button>
                  <Button
                    disabled={rejectMutation.isPending}
                    icon={<X className="h-4 w-4" />}
                    onClick={() =>
                      rejectMutation.mutate(insight.id, {
                        onSuccess: refresh
                      })
                    }
                    variant="secondary"
                  >
                    {t("memory.dreaming.reject")}
                  </Button>
                </div>
              )}
            </article>
          ))}
        </div>
      )}
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

function groupByLayer(chunks: MemoryChunk[]) {
  return chunks.reduce<Record<string, MemoryChunk[]>>((groups, chunk) => {
    groups[chunk.layer] = [...(groups[chunk.layer] ?? []), chunk];
    return groups;
  }, {});
}
