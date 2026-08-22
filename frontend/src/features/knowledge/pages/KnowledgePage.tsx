import { useQueryClient } from "@tanstack/react-query";
import { Database, FilePlus2, Search } from "lucide-react";
import { FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";

import { Badge } from "../../../components/ui/Badge";
import { Button } from "../../../components/ui/Button";
import { EmptyState } from "../../../components/ui/EmptyState";
import { Input } from "../../../components/ui/Input";
import { LoadingState } from "../../../components/ui/LoadingState";
import { Select } from "../../../components/ui/Select";
import { Textarea } from "../../../components/ui/Textarea";
import { Toast } from "../../../components/ui/Toast";
import { ApiError } from "../../../lib/errors";
import { formatDateTime } from "../../../lib/format";
import {
  knowledgeDocumentsQueryKey,
  useIngestKnowledgeDocument,
  useKnowledgeDocuments,
  useSearchKnowledge
} from "../hooks";
import type { KnowledgeDocument } from "../types";

type KnowledgeFormState = {
  title: string;
  content: string;
  source_type: string;
  source_uri: string;
  visibility: string;
};

const initialForm: KnowledgeFormState = {
  title: "",
  content: "",
  source_type: "manual",
  source_uri: "",
  visibility: "private"
};

export function KnowledgePage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const documentsQuery = useKnowledgeDocuments();
  const ingestMutation = useIngestKnowledgeDocument();
  const searchMutation = useSearchKnowledge();
  const [form, setForm] = useState<KnowledgeFormState>(initialForm);
  const [searchQuery, setSearchQuery] = useState("");
  const documents = documentsQuery.data ?? [];
  const searchChunks = searchMutation.data?.chunks ?? [];
  const mutationError = ingestMutation.error ?? searchMutation.error ?? null;

  function handleIngest(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    ingestMutation.mutate(
      {
        title: form.title.trim(),
        content: form.content.trim(),
        source_type: form.source_type,
        source_uri: form.source_uri.trim() || null,
        visibility: form.visibility
      },
      {
        onSuccess: () => {
          setForm(initialForm);
          void queryClient.invalidateQueries({ queryKey: knowledgeDocumentsQueryKey });
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

  return (
    <section className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_420px]">
      <div className="space-y-4">
        <div>
          <h1 className="text-2xl font-semibold text-ink-900">{t("knowledge.title")}</h1>
          <p className="mt-2 text-sm leading-6 text-ink-500">
            {t("knowledge.description")}
          </p>
        </div>

        {mutationError && (
          <Toast
            message={
              mutationError instanceof ApiError
                ? mutationError.message
                : t("knowledge.errors.operationFailed")
            }
            tone="error"
          />
        )}
        {ingestMutation.data && (
          <Toast
            message={t("knowledge.ingest.created", {
              chunks: ingestMutation.data.chunks.length,
              status: ingestMutation.data.embedding_status
            })}
            tone="success"
          />
        )}

        <form className="flex flex-col gap-3 rounded-lg border border-ink-200 bg-white p-4 sm:flex-row" onSubmit={handleSearch}>
          <Input
            onChange={(event) => setSearchQuery(event.target.value)}
            placeholder={t("knowledge.search.placeholder")}
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

        {searchMutation.data && (
          <div className="space-y-3">
            <div className="flex items-center gap-2">
              <h2 className="text-base font-semibold text-ink-900">
                {t("knowledge.search.results")}
              </h2>
              <Badge tone={searchMutation.data.mode === "vector" ? "green" : "blue"}>
                {searchMutation.data.mode}
              </Badge>
            </div>
            {searchChunks.length === 0 ? (
              <EmptyState
                description={t("knowledge.search.empty.description")}
                title={t("knowledge.search.empty.title")}
              />
            ) : (
              <div className="grid gap-3 md:grid-cols-2">
                {searchChunks.map((chunk) => (
                  <article className="rounded-lg border border-ink-200 bg-white p-4 shadow-sm" key={chunk.id}>
                    <div className="flex flex-wrap gap-2">
                      <Badge tone="blue">{chunk.document_title ?? t("knowledge.unknownDocument")}</Badge>
                      <Badge>{t("knowledge.chunkIndex", { index: chunk.chunk_index + 1 })}</Badge>
                      <Badge>{t("knowledge.tokens", { count: chunk.token_count })}</Badge>
                      {chunk.has_embedding && <Badge tone="green">embedding</Badge>}
                      {typeof chunk.similarity === "number" && (
                        <Badge>{t("knowledge.similarity", { value: chunk.similarity.toFixed(2) })}</Badge>
                      )}
                    </div>
                    <p className="mt-3 text-sm leading-6 text-ink-700">{chunk.content}</p>
                  </article>
                ))}
              </div>
            )}
          </div>
        )}

        <div className="space-y-3">
          <h2 className="text-base font-semibold text-ink-900">
            {t("knowledge.documents.title")}
          </h2>
          {documentsQuery.isLoading ? (
            <LoadingState />
          ) : documents.length === 0 ? (
            <EmptyState
              description={t("knowledge.documents.empty.description")}
              icon={<Database className="h-8 w-8" />}
              title={t("knowledge.documents.empty.title")}
            />
          ) : (
            <div className="grid gap-3 md:grid-cols-2">
              {documents.map((document) => (
                <KnowledgeDocumentCard document={document} key={document.id} />
              ))}
            </div>
          )}
        </div>
      </div>

      <form className="h-fit rounded-lg border border-ink-200 bg-white p-5 shadow-soft" onSubmit={handleIngest}>
        <div className="flex items-center gap-2">
          <FilePlus2 className="h-5 w-5 text-ocean-600" />
          <h2 className="text-base font-semibold text-ink-900">
            {t("knowledge.ingest.formTitle")}
          </h2>
        </div>
        <div className="mt-5 space-y-4">
          <Field label={t("knowledge.fields.title")}>
            <Input
              onChange={(event) => setForm({ ...form, title: event.target.value })}
              required
              value={form.title}
            />
          </Field>
          <Field label={t("knowledge.fields.content")}>
            <Textarea
              className="min-h-56"
              onChange={(event) => setForm({ ...form, content: event.target.value })}
              required
              value={form.content}
            />
          </Field>
          <div className="grid gap-3 sm:grid-cols-2">
            <Field label={t("knowledge.fields.sourceType")}>
              <Select
                onChange={(event) => setForm({ ...form, source_type: event.target.value })}
                value={form.source_type}
              >
                <option value="manual">{t("knowledge.sourceTypes.manual")}</option>
                <option value="note">{t("knowledge.sourceTypes.note")}</option>
                <option value="url">{t("knowledge.sourceTypes.url")}</option>
                <option value="upload">{t("knowledge.sourceTypes.upload")}</option>
              </Select>
            </Field>
            <Field label={t("knowledge.fields.visibility")}>
              <Select
                onChange={(event) => setForm({ ...form, visibility: event.target.value })}
                value={form.visibility}
              >
                <option value="private">{t("knowledge.visibility.private")}</option>
                <option value="public">{t("knowledge.visibility.public")}</option>
              </Select>
            </Field>
          </div>
          <Field label={t("knowledge.fields.sourceURI")}>
            <Input
              onChange={(event) => setForm({ ...form, source_uri: event.target.value })}
              value={form.source_uri}
            />
          </Field>
          <Button disabled={ingestMutation.isPending} type="submit">
            {ingestMutation.isPending ? t("knowledge.ingest.creating") : t("knowledge.ingest.create")}
          </Button>
        </div>
      </form>
    </section>
  );
}

function KnowledgeDocumentCard({ document }: { document: KnowledgeDocument }) {
  return (
    <article className="rounded-lg border border-ink-200 bg-white p-4 shadow-sm">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <h3 className="truncate text-base font-semibold text-ink-900">{document.title}</h3>
          <p className="mt-1 text-xs text-ink-500">{formatDateTime(document.updated_at)}</p>
        </div>
        <Badge tone={document.status === "active" ? "green" : "amber"}>
          {document.status}
        </Badge>
      </div>
      <div className="mt-4 flex flex-wrap gap-2">
        <Badge tone="blue">{document.source_type}</Badge>
        <Badge>{document.visibility}</Badge>
        {document.source_uri && <Badge>{document.source_uri}</Badge>}
      </div>
    </article>
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
