import { useQueryClient } from "@tanstack/react-query";
import {
  CalendarClock,
  ClipboardList,
  Pause,
  Play,
  RotateCw,
  Trash2
} from "lucide-react";
import { FormEvent, useEffect, useMemo, useState } from "react";
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
  scheduledJobRunsQueryKey,
  scheduledJobsQueryKey,
  useCreateScheduledJob,
  useDeleteScheduledJob,
  usePauseScheduledJob,
  useResumeScheduledJob,
  useRunScheduledJobNow,
  useScheduledJobRun,
  useScheduledJobRuns,
  useScheduledJobTemplates,
  useScheduledJobs
} from "../hooks";
import type {
  CreateScheduledJobInput,
  ScheduledAgentJob,
  ScheduledAgentJobRun,
  ScheduledJobTemplate
} from "../types";

type JobFormState = {
  title: string;
  description: string;
  job_type: string;
  schedule_kind: string;
  cron_expr: string;
  timezone: string;
  run_at_local_time: string;
  weekdays: number[];
  prompt_template: string;
  delivery_channel: string;
  visibility: string;
  context_policy: string;
  tool_policy: string;
};

const initialForm: JobFormState = {
  title: "",
  description: "",
  job_type: "daily_brief",
  schedule_kind: "weekly",
  cron_expr: "",
  timezone: "Asia/Shanghai",
  run_at_local_time: "08:00:00",
  weekdays: [1, 2, 3, 4, 5],
  prompt_template: "",
  delivery_channel: "in_app",
  visibility: "private",
  context_policy:
    '{"include_memory":true,"include_tasks":true,"include_calendar":false,"include_email":false,"max_context_tokens":6000}',
  tool_policy:
    '{"allow_tools":true,"allowed_tools":["list_tasks","search_memory"],"requires_approval_for_write":true}'
};

const weekdayValues = [1, 2, 3, 4, 5, 6, 7];

export function TasksPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const templatesQuery = useScheduledJobTemplates();
  const [statusFilter, setStatusFilter] = useState("");
  const jobsQuery = useScheduledJobs(statusFilter || undefined);
  const createMutation = useCreateScheduledJob();
  const pauseMutation = usePauseScheduledJob();
  const resumeMutation = useResumeScheduledJob();
  const deleteMutation = useDeleteScheduledJob();
  const runNowMutation = useRunScheduledJobNow();
  const [form, setForm] = useState<JobFormState>(initialForm);
  const [selectedJobID, setSelectedJobID] = useState<string | undefined>();
  const [selectedRunID, setSelectedRunID] = useState<string | undefined>();
  const runsQuery = useScheduledJobRuns(selectedJobID);
  const runDetailQuery = useScheduledJobRun(selectedRunID);

  const templates = templatesQuery.data ?? [];
  const jobs = jobsQuery.data ?? [];
  const selectedJob = useMemo(
    () => jobs.find((job) => job.id === selectedJobID),
    [jobs, selectedJobID]
  );
  const runs = runsQuery.data ?? [];
  const mutationError =
    createMutation.error ??
    pauseMutation.error ??
    resumeMutation.error ??
    deleteMutation.error ??
    runNowMutation.error ??
    null;

  useEffect(() => {
    if (!selectedJobID && jobs.length > 0) {
      setSelectedJobID(jobs[0].id);
    }
    if (selectedJobID && !jobs.find((job) => job.id === selectedJobID)) {
      setSelectedJobID(jobs[0]?.id);
    }
  }, [jobs, selectedJobID]);

  function applyTemplate(template: ScheduledJobTemplate) {
    setForm({
      title: template.title,
      description: template.description,
      job_type: template.job_type,
      schedule_kind: template.schedule_kind,
      cron_expr: "",
      timezone: template.timezone,
      run_at_local_time: template.run_at_local_time,
      weekdays: template.weekdays,
      prompt_template: template.prompt_template,
      delivery_channel: template.delivery_channel,
      visibility: "private",
      context_policy: JSON.stringify(template.context_policy, null, 2),
      tool_policy: JSON.stringify(template.tool_policy, null, 2)
    });
  }

  function refresh(jobID?: string) {
    void queryClient.invalidateQueries({ queryKey: scheduledJobsQueryKey });
    if (jobID) {
      void queryClient.invalidateQueries({ queryKey: scheduledJobRunsQueryKey });
      setSelectedJobID(jobID);
    }
  }

  function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const input: CreateScheduledJobInput = {
      title: form.title.trim(),
      description: form.description.trim() || null,
      job_type: form.job_type,
      schedule_kind: form.schedule_kind,
      cron_expr: form.cron_expr.trim() || null,
      timezone: form.timezone.trim() || "Asia/Shanghai",
      run_at_local_time: form.run_at_local_time.trim() || null,
      weekdays: form.weekdays,
      prompt_template: form.prompt_template.trim(),
      delivery_channel: form.delivery_channel,
      visibility: form.visibility,
      context_policy: parseJSONObject(form.context_policy),
      tool_policy: parseJSONObject(form.tool_policy)
    };
    createMutation.mutate(input, {
      onSuccess: (job) => {
        refresh(job.id);
        setForm(initialForm);
      }
    });
  }

  return (
    <section className="space-y-5">
      <div>
        <h1 className="text-2xl font-semibold text-ink-900">{t("tasks.title")}</h1>
        <p className="mt-2 text-sm leading-6 text-ink-500">{t("tasks.description")}</p>
      </div>

      {mutationError && (
        <Toast
          message={
            mutationError instanceof ApiError
              ? mutationError.message
              : t("tasks.errors.operationFailed")
          }
          tone="error"
        />
      )}
      {createMutation.isSuccess && <Toast message={t("tasks.created")} tone="success" />}
      {runNowMutation.data && (
        <Toast
          message={t("tasks.runNowQueued", { status: runNowMutation.data.run.status })}
          tone="success"
        />
      )}

      <div className="grid gap-5 xl:grid-cols-[430px_minmax(0,1fr)]">
        <div className="space-y-4">
          <TemplatePanel
            loading={templatesQuery.isLoading}
            onApply={applyTemplate}
            templates={templates}
          />
          <CreateJobForm
            form={form}
            onChange={setForm}
            onSubmit={handleCreate}
            submitting={createMutation.isPending}
          />
        </div>

        <div className="space-y-4">
          <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <h2 className="text-base font-semibold text-ink-900">{t("tasks.jobs.title")}</h2>
            <Select
              className="w-full sm:w-44"
              onChange={(event) => setStatusFilter(event.target.value)}
              value={statusFilter}
            >
              <option value="">{t("tasks.status.all")}</option>
              <option value="active">{t("tasks.status.active")}</option>
              <option value="paused">{t("tasks.status.paused")}</option>
            </Select>
          </div>

          {jobsQuery.isLoading ? (
            <LoadingState />
          ) : jobs.length === 0 ? (
            <EmptyState
              description={t("tasks.jobs.empty.description")}
              icon={<CalendarClock className="h-8 w-8" />}
              title={t("tasks.jobs.empty.title")}
            />
          ) : (
            <div className="grid gap-3 lg:grid-cols-2">
              {jobs.map((job) => (
                <JobCard
                  active={job.id === selectedJobID}
                  job={job}
                  key={job.id}
                  onDelete={() => {
                    if (!window.confirm(t("tasks.deleteConfirm", { title: job.title }))) {
                      return;
                    }
                    deleteMutation.mutate(job.id, {
                      onSuccess: () => refresh()
                    });
                  }}
                  onPause={() =>
                    pauseMutation.mutate(job.id, {
                      onSuccess: () => refresh(job.id)
                    })
                  }
                  onResume={() =>
                    resumeMutation.mutate(job.id, {
                      onSuccess: () => refresh(job.id)
                    })
                  }
                  onRunNow={() =>
                    runNowMutation.mutate(job.id, {
                      onSuccess: (result) => {
                        refresh(job.id);
                        setSelectedRunID(result.run.id);
                      }
                    })
                  }
                  onSelect={() => setSelectedJobID(job.id)}
                  working={
                    pauseMutation.isPending ||
                    resumeMutation.isPending ||
                    deleteMutation.isPending ||
                    runNowMutation.isPending
                  }
                />
              ))}
            </div>
          )}

          <RunsPanel
            job={selectedJob}
            loading={runsQuery.isLoading}
            onSelectRun={setSelectedRunID}
            runDetail={runDetailQuery.data}
            runs={runs}
            selectedRunID={selectedRunID}
          />
        </div>
      </div>
    </section>
  );
}

function TemplatePanel({
  loading,
  onApply,
  templates
}: {
  loading: boolean;
  onApply: (template: ScheduledJobTemplate) => void;
  templates: ScheduledJobTemplate[];
}) {
  const { t } = useTranslation();
  return (
    <section className="rounded-lg border border-ink-200 bg-white p-5 shadow-sm">
      <h2 className="text-base font-semibold text-ink-900">{t("tasks.templates.title")}</h2>
      {loading ? (
        <LoadingState />
      ) : (
        <div className="mt-4 space-y-3">
          {templates.map((template) => (
            <article className="rounded-md border border-ink-200 p-3" key={template.id}>
              <div className="flex items-start justify-between gap-3">
                <div>
                  <h3 className="text-sm font-semibold text-ink-900">{template.title}</h3>
                  <p className="mt-1 text-sm leading-6 text-ink-500">{template.description}</p>
                </div>
                <Button onClick={() => onApply(template)} variant="secondary">
                  {t("tasks.templates.use")}
                </Button>
              </div>
              <div className="mt-3 flex flex-wrap gap-2">
                <Badge tone="blue">{template.job_type}</Badge>
                <Badge>{template.run_at_local_time}</Badge>
                <Badge>{formatWeekdays(template.weekdays)}</Badge>
              </div>
            </article>
          ))}
        </div>
      )}
    </section>
  );
}

function CreateJobForm({
  form,
  onChange,
  onSubmit,
  submitting
}: {
  form: JobFormState;
  onChange: (form: JobFormState) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  submitting: boolean;
}) {
  const { t } = useTranslation();
  return (
    <form className="rounded-lg border border-ink-200 bg-white p-5 shadow-sm" onSubmit={onSubmit}>
      <h2 className="text-base font-semibold text-ink-900">{t("tasks.form.title")}</h2>
      <div className="mt-5 space-y-4">
        <Field label={t("tasks.fields.title")}>
          <Input
            onChange={(event) => onChange({ ...form, title: event.target.value })}
            required
            value={form.title}
          />
        </Field>
        <Field label={t("tasks.fields.description")}>
          <Textarea
            onChange={(event) => onChange({ ...form, description: event.target.value })}
            value={form.description}
          />
        </Field>
        <div className="grid gap-3 sm:grid-cols-2">
          <Field label={t("tasks.fields.jobType")}>
            <Input
              onChange={(event) => onChange({ ...form, job_type: event.target.value })}
              value={form.job_type}
            />
          </Field>
          <Field label={t("tasks.fields.scheduleKind")}>
            <Select
              onChange={(event) => onChange({ ...form, schedule_kind: event.target.value })}
              value={form.schedule_kind}
            >
              <option value="weekly">{t("tasks.schedule.weekly")}</option>
              <option value="daily">{t("tasks.schedule.daily")}</option>
              <option value="cron">{t("tasks.schedule.cron")}</option>
            </Select>
          </Field>
          <Field label={t("tasks.fields.localTime")}>
            <Input
              onChange={(event) =>
                onChange({ ...form, run_at_local_time: event.target.value })
              }
              placeholder="08:00:00"
              value={form.run_at_local_time}
            />
          </Field>
          <Field label={t("tasks.fields.timezone")}>
            <Input
              onChange={(event) => onChange({ ...form, timezone: event.target.value })}
              value={form.timezone}
            />
          </Field>
        </div>
        <Field label={t("tasks.fields.weekdays")}>
          <div className="grid grid-cols-7 gap-2">
            {weekdayValues.map((weekday) => {
              const checked = form.weekdays.includes(weekday);
              return (
                <button
                  className={[
                    "h-9 rounded-md border text-sm font-medium transition",
                    checked
                      ? "border-ink-900 bg-ink-900 text-white"
                      : "border-ink-200 bg-white text-ink-600 hover:bg-ink-50"
                  ].join(" ")}
                  key={weekday}
                  onClick={() =>
                    onChange({
                      ...form,
                      weekdays: checked
                        ? form.weekdays.filter((item) => item !== weekday)
                        : [...form.weekdays, weekday].sort()
                    })
                  }
                  type="button"
                >
                  {t(`tasks.weekdays.${weekday}`)}
                </button>
              );
            })}
          </div>
        </Field>
        <Field label={t("tasks.fields.cron")}>
          <Input
            onChange={(event) => onChange({ ...form, cron_expr: event.target.value })}
            value={form.cron_expr}
          />
        </Field>
        <Field label={t("tasks.fields.prompt")}>
          <Textarea
            className="min-h-36"
            onChange={(event) => onChange({ ...form, prompt_template: event.target.value })}
            required
            value={form.prompt_template}
          />
        </Field>
        <div className="grid gap-3 sm:grid-cols-2">
          <Field label={t("tasks.fields.deliveryChannel")}>
            <Select
              onChange={(event) => onChange({ ...form, delivery_channel: event.target.value })}
              value={form.delivery_channel}
            >
              <option value="in_app">{t("tasks.delivery.inApp")}</option>
              <option value="channel">{t("tasks.delivery.channel")}</option>
            </Select>
          </Field>
          <Field label={t("tasks.fields.visibility")}>
            <Select
              onChange={(event) => onChange({ ...form, visibility: event.target.value })}
              value={form.visibility}
            >
              <option value="private">{t("tasks.visibility.private")}</option>
              <option value="public">{t("tasks.visibility.public")}</option>
            </Select>
          </Field>
        </div>
        <Field label={t("tasks.fields.contextPolicy")}>
          <Textarea
            className="font-mono"
            onChange={(event) => onChange({ ...form, context_policy: event.target.value })}
            value={form.context_policy}
          />
        </Field>
        <Field label={t("tasks.fields.toolPolicy")}>
          <Textarea
            className="font-mono"
            onChange={(event) => onChange({ ...form, tool_policy: event.target.value })}
            value={form.tool_policy}
          />
        </Field>
        <Button disabled={submitting} type="submit">
          {submitting ? t("tasks.form.creating") : t("tasks.form.create")}
        </Button>
      </div>
    </form>
  );
}

function JobCard({
  active,
  job,
  onDelete,
  onPause,
  onResume,
  onRunNow,
  onSelect,
  working
}: {
  active: boolean;
  job: ScheduledAgentJob;
  onDelete: () => void;
  onPause: () => void;
  onResume: () => void;
  onRunNow: () => void;
  onSelect: () => void;
  working: boolean;
}) {
  const { t } = useTranslation();
  return (
    <article
      className={[
        "rounded-lg border bg-white p-4 shadow-sm transition",
        active ? "border-ocean-500 ring-2 ring-ocean-500/15" : "border-ink-200"
      ].join(" ")}
    >
      <button className="block w-full text-left" onClick={onSelect} type="button">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <h3 className="truncate text-base font-semibold text-ink-900">{job.title}</h3>
            <p className="mt-1 text-xs text-ink-500">
              {job.next_run_at
                ? t("tasks.jobs.nextRun", { time: formatDateTime(job.next_run_at) })
                : t("tasks.jobs.noNextRun")}
            </p>
          </div>
          <Badge tone={job.status === "active" ? "green" : "amber"}>{job.status}</Badge>
        </div>
        {job.description && (
          <p className="mt-3 text-sm leading-6 text-ink-700">{job.description}</p>
        )}
        <div className="mt-4 flex flex-wrap gap-2">
          <Badge tone="blue">{job.job_type}</Badge>
          <Badge>{job.schedule_kind}</Badge>
          {job.run_at_local_time && <Badge>{job.run_at_local_time}</Badge>}
          <Badge>{formatWeekdays(job.weekdays)}</Badge>
          {job.failure_count > 0 && (
            <Badge tone="amber">
              {t("tasks.jobs.failures", { count: job.failure_count })}
            </Badge>
          )}
        </div>
      </button>
      <div className="mt-4 flex flex-wrap gap-2">
        <Button
          disabled={working || job.status !== "active"}
          icon={<RotateCw className="h-4 w-4" />}
          onClick={onRunNow}
        >
          {t("tasks.actions.runNow")}
        </Button>
        {job.status === "active" ? (
          <Button
            disabled={working}
            icon={<Pause className="h-4 w-4" />}
            onClick={onPause}
            variant="secondary"
          >
            {t("tasks.actions.pause")}
          </Button>
        ) : (
          <Button
            disabled={working}
            icon={<Play className="h-4 w-4" />}
            onClick={onResume}
            variant="secondary"
          >
            {t("tasks.actions.resume")}
          </Button>
        )}
        <Button disabled={working} icon={<Trash2 className="h-4 w-4" />} onClick={onDelete} variant="ghost">
          {t("common.delete")}
        </Button>
      </div>
    </article>
  );
}

function RunsPanel({
  job,
  loading,
  onSelectRun,
  runDetail,
  runs,
  selectedRunID
}: {
  job?: ScheduledAgentJob;
  loading: boolean;
  onSelectRun: (runID: string) => void;
  runDetail?: ScheduledAgentJobRun;
  runs: ScheduledAgentJobRun[];
  selectedRunID?: string;
}) {
  const { t } = useTranslation();
  return (
    <section className="rounded-lg border border-ink-200 bg-white p-5 shadow-sm">
      <h2 className="text-base font-semibold text-ink-900">
        {job ? t("tasks.runs.titleForJob", { title: job.title }) : t("tasks.runs.title")}
      </h2>
      {!job ? (
        <EmptyState
          description={t("tasks.runs.noJob.description")}
          icon={<ClipboardList className="h-8 w-8" />}
          title={t("tasks.runs.noJob.title")}
        />
      ) : loading ? (
        <LoadingState />
      ) : runs.length === 0 ? (
        <EmptyState
          description={t("tasks.runs.empty.description")}
          title={t("tasks.runs.empty.title")}
        />
      ) : (
        <div className="mt-4 grid gap-4 xl:grid-cols-[minmax(0,1fr)_minmax(320px,420px)]">
          <div className="space-y-3">
            {runs.map((run) => (
              <button
                className={[
                  "block w-full rounded-md border p-3 text-left transition",
                  run.id === selectedRunID
                    ? "border-ocean-500 bg-ocean-500/5"
                    : "border-ink-200 hover:bg-ink-50"
                ].join(" ")}
                key={run.id}
                onClick={() => onSelectRun(run.id)}
                type="button"
              >
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <div className="text-sm font-semibold text-ink-900">
                      {formatDateTime(run.created_at)}
                    </div>
                    <div className="mt-1 text-xs text-ink-500">{run.trigger_reason}</div>
                  </div>
                  <Badge tone={run.status === "success" ? "green" : "amber"}>
                    {run.status}
                  </Badge>
                </div>
                {run.output_summary && (
                  <p className="mt-2 line-clamp-2 text-sm leading-6 text-ink-600">
                    {run.output_summary}
                  </p>
                )}
                {run.error_message && (
                  <p className="mt-2 line-clamp-2 text-sm leading-6 text-red-700">
                    {run.error_message}
                  </p>
                )}
              </button>
            ))}
          </div>
          <RunDetail run={runDetail} />
        </div>
      )}
    </section>
  );
}

function RunDetail({ run }: { run?: ScheduledAgentJobRun }) {
  const { t } = useTranslation();
  if (!run) {
    return (
      <div className="rounded-md border border-dashed border-ink-200 p-5 text-sm text-ink-500">
        {t("tasks.runs.selectRun")}
      </div>
    );
  }
  return (
    <article className="rounded-md border border-ink-200 p-4">
      <div className="flex items-start justify-between gap-3">
        <h3 className="text-sm font-semibold text-ink-900">{run.id}</h3>
        <Badge tone={run.status === "success" ? "green" : "amber"}>{run.status}</Badge>
      </div>
      <dl className="mt-4 grid gap-2 text-sm">
        <Meta label={t("tasks.runs.fields.scheduledFor")} value={formatDateTime(run.scheduled_for)} />
        <Meta
          label={t("tasks.runs.fields.startedAt")}
          value={run.started_at ? formatDateTime(run.started_at) : "-"}
        />
        <Meta
          label={t("tasks.runs.fields.finishedAt")}
          value={run.finished_at ? formatDateTime(run.finished_at) : "-"}
        />
        <Meta label={t("tasks.runs.fields.conversation")} value={run.conversation_id ?? "-"} />
      </dl>
      {run.output_summary && (
        <p className="mt-4 rounded-md bg-mint-500/10 p-3 text-sm leading-6 text-mint-700">
          {run.output_summary}
        </p>
      )}
      {run.error_message && (
        <p className="mt-4 rounded-md bg-red-500/10 p-3 text-sm leading-6 text-red-700">
          {run.error_message}
        </p>
      )}
      <pre className="mt-4 max-h-72 overflow-auto rounded-md bg-ink-900 p-3 text-xs leading-5 text-white">
        {JSON.stringify(run.input_snapshot, null, 2)}
      </pre>
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

function Meta({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid grid-cols-[120px_minmax(0,1fr)] gap-3">
      <dt className="text-ink-500">{label}</dt>
      <dd className="min-w-0 break-words text-ink-800">{value}</dd>
    </div>
  );
}

function parseJSONObject(value: string): Record<string, unknown> {
  try {
    const parsed = JSON.parse(value) as unknown;
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>;
    }
  } catch {
    // Keep the form forgiving; backend defaults cover malformed optional policies.
  }
  return {};
}

function formatWeekdays(values: number[]) {
  if (!values || values.length === 0) {
    return "-";
  }
  return values.join(",");
}
