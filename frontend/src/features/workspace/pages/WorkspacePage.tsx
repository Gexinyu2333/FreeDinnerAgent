import { useQueryClient } from "@tanstack/react-query";
import {
  ArrowUp,
  File,
  Folder,
  HardDrive,
  Play,
  RefreshCw,
  Save,
  Shield,
  Terminal,
  UploadCloud
} from "lucide-react";
import { FormEvent, useEffect, useMemo, useState } from "react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";

import { Badge } from "../../../components/ui/Badge";
import { Button } from "../../../components/ui/Button";
import { EmptyState } from "../../../components/ui/EmptyState";
import { ErrorState } from "../../../components/ui/ErrorState";
import { Field } from "../../../components/ui/Field";
import { Input } from "../../../components/ui/Input";
import { LoadingState } from "../../../components/ui/LoadingState";
import { Select } from "../../../components/ui/Select";
import { Textarea } from "../../../components/ui/Textarea";
import { Toast } from "../../../components/ui/Toast";
import { useToast } from "../../../components/ui/ToastProvider";
import { ApiError } from "../../../lib/errors";
import { formatDateTime, formatNumber } from "../../../lib/format";
import {
  useEnableWorkspace,
  useReadWorkspaceFile,
  useRunWorkspaceCommand,
  useUpdateWorkspacePolicy,
  useWorkspaceCommandRuns,
  useWorkspaceFiles,
  useWorkspaceStatus,
  useWriteWorkspaceFile,
  workspaceCommandRunsQueryKey,
  workspaceFilesQueryKey,
  workspaceStatusQueryKey
} from "../hooks";
import type {
  EnableWorkspaceInput,
  UpdateWorkspacePolicyInput,
  WorkspaceCommandRun,
  WorkspaceStatus
} from "../types";

type PolicyFormState = {
  sandbox_type: string;
  network_policy: string;
  network_allowlist: string;
  max_disk_mb: string;
  max_file_count: string;
  max_single_file_mb: string;
  max_command_seconds: string;
  max_stdout_kb: string;
  max_stderr_kb: string;
  cpu_limit: string;
  memory_limit_mb: string;
  idle_after_minutes: string;
  destroy_after_hours: string;
};

type FileFormState = {
  path: string;
  content: string;
};

type CommandFormState = {
  command: string;
  args: string;
  working_dir: string;
  timeout_seconds: string;
};

const defaultPolicyForm: PolicyFormState = {
  sandbox_type: "local_dir",
  network_policy: "disabled",
  network_allowlist: "",
  max_disk_mb: "1024",
  max_file_count: "1000",
  max_single_file_mb: "50",
  max_command_seconds: "30",
  max_stdout_kb: "64",
  max_stderr_kb: "64",
  cpu_limit: "1.0",
  memory_limit_mb: "512",
  idle_after_minutes: "60",
  destroy_after_hours: "24"
};

const defaultFileForm: FileFormState = {
  path: "/notes/workspace-test.md",
  content: "# Workspace test\n\nFreeDinnerAgent can write and read this file.\n"
};

const defaultCommandForm: CommandFormState = {
  command: "ls",
  args: "-la",
  working_dir: "/",
  timeout_seconds: "10"
};

export function WorkspacePage() {
  const { t } = useTranslation();
  const toast = useToast();
  const queryClient = useQueryClient();
  const statusQuery = useWorkspaceStatus();
  const workspaceMissing =
    statusQuery.error instanceof ApiError &&
    statusQuery.error.code === "WORKSPACE_NOT_FOUND";
  const workspaceEnabled = Boolean(statusQuery.data) && !workspaceMissing;
  const [currentPath, setCurrentPath] = useState("/");
  const filesQuery = useWorkspaceFiles(currentPath, workspaceEnabled);
  const commandRunsQuery = useWorkspaceCommandRuns(workspaceEnabled);
  const enableMutation = useEnableWorkspace();
  const policyMutation = useUpdateWorkspacePolicy();
  const readMutation = useReadWorkspaceFile();
  const writeMutation = useWriteWorkspaceFile();
  const commandMutation = useRunWorkspaceCommand();
  const [policyForm, setPolicyForm] = useState<PolicyFormState>(defaultPolicyForm);
  const [fileForm, setFileForm] = useState<FileFormState>(defaultFileForm);
  const [commandForm, setCommandForm] = useState<CommandFormState>(defaultCommandForm);

  useEffect(() => {
    if (statusQuery.data) {
      setPolicyForm(policyFormFromStatus(statusQuery.data));
    }
  }, [statusQuery.data]);

  const operationError =
    enableMutation.error ??
    policyMutation.error ??
    readMutation.error ??
    writeMutation.error ??
    commandMutation.error ??
    null;

  const latestRun = commandMutation.data?.run;
  const commandRuns = commandRunsQuery.data?.runs ?? [];
  const files = filesQuery.data?.items ?? [];
  const parentPath = useMemo(() => parentOf(currentPath), [currentPath]);

  function invalidateWorkspace(path = currentPath) {
    void queryClient.invalidateQueries({ queryKey: workspaceStatusQueryKey });
    void queryClient.invalidateQueries({ queryKey: [...workspaceFilesQueryKey, path] });
    void queryClient.invalidateQueries({ queryKey: workspaceCommandRunsQueryKey });
  }

  function handleEnable(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    enableMutation.mutate(policyInput(policyForm) as EnableWorkspaceInput, {
      onSuccess: () => {
        invalidateWorkspace("/");
        toast.notify(t("workspace.enabled"));
      }
    });
  }

  function handlePolicyUpdate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    policyMutation.mutate(policyInput(policyForm), {
      onSuccess: () => {
        invalidateWorkspace();
        toast.notify(t("workspace.policy.saved"));
      }
    });
  }

  function handleReadFile(path = fileForm.path) {
    readMutation.mutate(path, {
      onSuccess: (result) => {
        setFileForm({ path: result.path, content: result.content });
      }
    });
  }

  function handleWriteFile(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    writeMutation.mutate(
      {
        path: fileForm.path.trim(),
        content: fileForm.content
      },
      {
        onSuccess: () => {
          invalidateWorkspace(parentOf(fileForm.path));
          setCurrentPath(parentOf(fileForm.path));
          toast.notify(t("workspace.files.saved"));
        }
      }
    );
  }

  function handleRunCommand(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    commandMutation.mutate(
      {
        command: commandForm.command.trim(),
        args: parseArgs(commandForm.args),
        working_dir: commandForm.working_dir.trim() || "/",
        timeout_seconds: numberValue(commandForm.timeout_seconds, 10)
      },
      {
        onSuccess: () => {
          invalidateWorkspace();
          toast.notify(t("workspace.commands.finished"));
        }
      }
    );
  }

  if (statusQuery.isLoading) {
    return <LoadingState />;
  }

  if (statusQuery.isError && !workspaceMissing) {
    return (
      <ErrorState
        description={
          statusQuery.error instanceof ApiError
            ? statusQuery.error.message
            : t("workspace.errors.operationFailed")
        }
        onRetry={() => void statusQuery.refetch()}
      />
    );
  }

  return (
    <section className="space-y-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-ink-900">{t("workspace.title")}</h1>
          <p className="mt-2 text-sm leading-6 text-ink-500">{t("workspace.description")}</p>
        </div>
        <Button
          icon={<RefreshCw className="h-4 w-4" />}
          onClick={() => invalidateWorkspace()}
          variant="secondary"
        >
          {t("common.refresh")}
        </Button>
      </div>

      {operationError && (
        <Toast
          message={
            operationError instanceof ApiError
              ? operationError.message
              : t("workspace.errors.operationFailed")
          }
          tone="error"
        />
      )}
      {!workspaceEnabled ? (
        <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_420px]">
          <EmptyState
            action={
              <span className="text-xs font-medium text-ink-500">
                {t("workspace.empty.action")}
              </span>
            }
            description={t("workspace.empty.description")}
            icon={<Terminal className="h-8 w-8" />}
            title={t("workspace.empty.title")}
          />
          <PolicyForm
            form={policyForm}
            mode="enable"
            onChange={setPolicyForm}
            onSubmit={handleEnable}
            submitting={enableMutation.isPending}
          />
        </div>
      ) : (
        <>
          <WorkspaceSummary status={statusQuery.data as WorkspaceStatus} />

          <div className="grid gap-5 xl:grid-cols-[390px_minmax(0,1fr)]">
            <PolicyForm
              form={policyForm}
              mode="update"
              onChange={setPolicyForm}
              onSubmit={handlePolicyUpdate}
              submitting={policyMutation.isPending}
            />

            <div className="space-y-5">
              <div className="grid gap-5 2xl:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
                <FileBrowser
                  currentPath={currentPath}
                  files={files}
                  loading={filesQuery.isLoading}
                  onGoParent={() => setCurrentPath(parentPath)}
                  onOpen={(path, type) => {
                    if (type === "directory") {
                      setCurrentPath(path);
                    } else {
                      handleReadFile(path);
                    }
                  }}
                  parentPath={parentPath}
                />
                <FileEditor
                  form={fileForm}
                  onChange={setFileForm}
                  onRead={() => handleReadFile()}
                  onSubmit={handleWriteFile}
                  reading={readMutation.isPending}
                  readResult={readMutation.data}
                  saving={writeMutation.isPending}
                />
              </div>

              <div className="grid gap-5 2xl:grid-cols-[minmax(0,420px)_minmax(0,1fr)]">
                <CommandRunner
                  form={commandForm}
                  latestRun={latestRun}
                  onChange={setCommandForm}
                  onSubmit={handleRunCommand}
                  running={commandMutation.isPending}
                />
                <CommandHistory loading={commandRunsQuery.isLoading} runs={commandRuns} />
              </div>
            </div>
          </div>
        </>
      )}
    </section>
  );
}

function WorkspaceSummary({ status }: { status: WorkspaceStatus }) {
  const { t } = useTranslation();
  const workspace = status.workspace;
  const quota = status.quota;
  return (
    <div className="grid gap-4 md:grid-cols-3">
      <SummaryCard
        icon={<Shield className="h-5 w-5" />}
        label={t("workspace.summary.sandbox")}
        value={workspace.sandbox_type}
        detail={t("workspace.summary.network", { policy: workspace.network_policy })}
      />
      <SummaryCard
        icon={<HardDrive className="h-5 w-5" />}
        label={t("workspace.summary.disk")}
        value={`${formatBytes(quota.used_disk_bytes)} / ${formatBytes(workspace.max_disk_bytes)}`}
        detail={t("workspace.summary.files", {
          fileCount: formatNumber(quota.file_count),
          fileLimit: formatNumber(workspace.max_file_count)
        })}
      />
      <SummaryCard
        icon={<Terminal className="h-5 w-5" />}
        label={t("workspace.summary.commands")}
        value={formatNumber(quota.command_count)}
        detail={t("workspace.summary.timeout", {
          seconds: workspace.max_command_seconds
        })}
      />
    </div>
  );
}

function SummaryCard({
  detail,
  icon,
  label,
  value
}: {
  detail: string;
  icon: ReactNode;
  label: string;
  value: string;
}) {
  return (
    <div className="rounded-lg border border-ink-200 bg-white p-4">
      <div className="flex items-center gap-2 text-sm font-medium text-ink-500">
        <span className="text-ocean-600">{icon}</span>
        {label}
      </div>
      <div className="mt-3 text-lg font-semibold text-ink-900">{value}</div>
      <div className="mt-1 text-xs text-ink-500">{detail}</div>
    </div>
  );
}

function PolicyForm({
  form,
  mode,
  onChange,
  onSubmit,
  submitting
}: {
  form: PolicyFormState;
  mode: "enable" | "update";
  onChange: (form: PolicyFormState) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  submitting: boolean;
}) {
  const { t } = useTranslation();
  const set = (key: keyof PolicyFormState, value: string) =>
    onChange({ ...form, [key]: value });

  return (
    <form className="rounded-lg border border-ink-200 bg-white p-4" onSubmit={onSubmit}>
      <div className="flex items-center gap-2">
        <Shield className="h-5 w-5 text-ocean-600" />
        <h2 className="text-base font-semibold text-ink-900">
          {mode === "enable" ? t("workspace.policy.enableTitle") : t("workspace.policy.title")}
        </h2>
      </div>

      <div className="mt-4 grid gap-3 sm:grid-cols-2 xl:grid-cols-1 2xl:grid-cols-2">
        <Field label={t("workspace.fields.sandboxType")}>
          <Select
            className="w-full"
            onChange={(event) => set("sandbox_type", event.target.value)}
            value={form.sandbox_type}
          >
            <option value="local_dir">local_dir</option>
            <option value="docker">docker</option>
            <option value="podman">podman</option>
            <option value="nsjail">nsjail</option>
          </Select>
        </Field>
        <Field label={t("workspace.fields.networkPolicy")}>
          <Select
            className="w-full"
            onChange={(event) => set("network_policy", event.target.value)}
            value={form.network_policy}
          >
            <option value="disabled">{t("workspace.network.disabled")}</option>
            <option value="allowlist">{t("workspace.network.allowlist")}</option>
            <option value="open">{t("workspace.network.open")}</option>
          </Select>
        </Field>
        <Field label={t("workspace.fields.maxDiskMB")}>
          <Input
            min="1"
            onChange={(event) => set("max_disk_mb", event.target.value)}
            type="number"
            value={form.max_disk_mb}
          />
        </Field>
        <Field label={t("workspace.fields.maxFileCount")}>
          <Input
            min="1"
            onChange={(event) => set("max_file_count", event.target.value)}
            type="number"
            value={form.max_file_count}
          />
        </Field>
        <Field label={t("workspace.fields.maxSingleFileMB")}>
          <Input
            min="1"
            onChange={(event) => set("max_single_file_mb", event.target.value)}
            type="number"
            value={form.max_single_file_mb}
          />
        </Field>
        <Field label={t("workspace.fields.maxCommandSeconds")}>
          <Input
            min="1"
            onChange={(event) => set("max_command_seconds", event.target.value)}
            type="number"
            value={form.max_command_seconds}
          />
        </Field>
        <Field label={t("workspace.fields.maxStdoutKB")}>
          <Input
            min="1"
            onChange={(event) => set("max_stdout_kb", event.target.value)}
            type="number"
            value={form.max_stdout_kb}
          />
        </Field>
        <Field label={t("workspace.fields.maxStderrKB")}>
          <Input
            min="1"
            onChange={(event) => set("max_stderr_kb", event.target.value)}
            type="number"
            value={form.max_stderr_kb}
          />
        </Field>
        <Field label={t("workspace.fields.cpuLimit")}>
          <Input
            onChange={(event) => set("cpu_limit", event.target.value)}
            value={form.cpu_limit}
          />
        </Field>
        <Field label={t("workspace.fields.memoryMB")}>
          <Input
            min="0"
            onChange={(event) => set("memory_limit_mb", event.target.value)}
            type="number"
            value={form.memory_limit_mb}
          />
        </Field>
        <Field label={t("workspace.fields.idleMinutes")}>
          <Input
            min="0"
            onChange={(event) => set("idle_after_minutes", event.target.value)}
            type="number"
            value={form.idle_after_minutes}
          />
        </Field>
        <Field label={t("workspace.fields.destroyHours")}>
          <Input
            min="0"
            onChange={(event) => set("destroy_after_hours", event.target.value)}
            type="number"
            value={form.destroy_after_hours}
          />
        </Field>
      </div>

      <Field className="mt-3" label={t("workspace.fields.networkAllowlist")}>
        <Input
          onChange={(event) => set("network_allowlist", event.target.value)}
          placeholder="api.example.com, 10.0.0.12"
          value={form.network_allowlist}
        />
      </Field>

      <Button className="mt-4 w-full" disabled={submitting} icon={<Save className="h-4 w-4" />} type="submit">
        {submitting
          ? t("common.saving")
          : mode === "enable"
            ? t("workspace.policy.enable")
            : t("workspace.policy.save")}
      </Button>
    </form>
  );
}

function FileBrowser({
  currentPath,
  files,
  loading,
  onGoParent,
  onOpen,
  parentPath
}: {
  currentPath: string;
  files: { name: string; path: string; type: string; size_bytes: number; last_modified: string }[];
  loading: boolean;
  onGoParent: () => void;
  onOpen: (path: string, type: string) => void;
  parentPath: string;
}) {
  const { t } = useTranslation();
  return (
    <div className="rounded-lg border border-ink-200 bg-white">
      <div className="flex items-center justify-between border-b border-ink-100 px-4 py-3">
        <div>
          <h2 className="text-base font-semibold text-ink-900">{t("workspace.files.title")}</h2>
          <p className="mt-1 break-all text-xs text-ink-500">{currentPath}</p>
        </div>
        <Button
          disabled={currentPath === "/"}
          icon={<ArrowUp className="h-4 w-4" />}
          onClick={onGoParent}
          variant="secondary"
        >
          {parentPath}
        </Button>
      </div>
      <div className="max-h-[440px] overflow-auto p-2">
        {loading ? (
          <LoadingState />
        ) : files.length === 0 ? (
          <EmptyState
            description={t("workspace.files.empty.description")}
            icon={<Folder className="h-8 w-8" />}
            title={t("workspace.files.empty.title")}
          />
        ) : (
          <div className="divide-y divide-ink-100">
            {files.map((file) => {
              const Icon = file.type === "directory" ? Folder : File;
              return (
                <button
                  className="flex w-full items-center gap-3 px-2 py-3 text-left transition hover:bg-ink-50"
                  key={file.path}
                  onClick={() => onOpen(file.path, file.type)}
                  type="button"
                >
                  <Icon className="h-5 w-5 shrink-0 text-ocean-600" />
                  <div className="min-w-0 flex-1">
                    <div className="truncate text-sm font-medium text-ink-900">{file.name}</div>
                    <div className="mt-1 text-xs text-ink-500">
                      {file.type === "directory" ? t("workspace.files.directory") : formatBytes(file.size_bytes)}
                      {" · "}
                      {formatDateTime(file.last_modified)}
                    </div>
                  </div>
                  <Badge tone={file.type === "directory" ? "blue" : "neutral"}>{file.type}</Badge>
                </button>
              );
            })}
          </div>
        )}
      </div>
    </div>
  );
}

function FileEditor({
  form,
  onChange,
  onRead,
  onSubmit,
  reading,
  readResult,
  saving
}: {
  form: FileFormState;
  onChange: (form: FileFormState) => void;
  onRead: () => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  reading: boolean;
  readResult?: { size_bytes: number; mime_type: string; content_hash: string };
  saving: boolean;
}) {
  const { t } = useTranslation();
  return (
    <form className="rounded-lg border border-ink-200 bg-white p-4" onSubmit={onSubmit}>
      <div className="flex items-center gap-2">
        <UploadCloud className="h-5 w-5 text-ocean-600" />
        <h2 className="text-base font-semibold text-ink-900">{t("workspace.files.editor")}</h2>
      </div>
      <Field className="mt-4" label={t("workspace.fields.path")}>
        <Input
          onChange={(event) => onChange({ ...form, path: event.target.value })}
          value={form.path}
        />
      </Field>
      <Field className="mt-3" label={t("workspace.fields.content")}>
        <Textarea
          className="min-h-72 font-mono"
          onChange={(event) => onChange({ ...form, content: event.target.value })}
          value={form.content}
        />
      </Field>
      {readResult && (
        <div className="mt-3 rounded-md bg-ink-50 px-3 py-2 text-xs text-ink-500">
          {formatBytes(readResult.size_bytes)} · {readResult.mime_type} ·{" "}
          <span className="break-all">{readResult.content_hash}</span>
        </div>
      )}
      <div className="mt-4 flex flex-col gap-2 sm:flex-row">
        <Button
          className="flex-1"
          disabled={reading}
          onClick={onRead}
          type="button"
          variant="secondary"
        >
          {reading ? t("workspace.files.reading") : t("workspace.files.read")}
        </Button>
        <Button
          className="flex-1"
          disabled={saving}
          icon={<Save className="h-4 w-4" />}
          type="submit"
        >
          {saving ? t("common.saving") : t("workspace.files.write")}
        </Button>
      </div>
    </form>
  );
}

function CommandRunner({
  form,
  latestRun,
  onChange,
  onSubmit,
  running
}: {
  form: CommandFormState;
  latestRun?: WorkspaceCommandRun;
  onChange: (form: CommandFormState) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  running: boolean;
}) {
  const { t } = useTranslation();
  return (
    <form className="rounded-lg border border-ink-200 bg-white p-4" onSubmit={onSubmit}>
      <div className="flex items-center gap-2">
        <Terminal className="h-5 w-5 text-ocean-600" />
        <h2 className="text-base font-semibold text-ink-900">{t("workspace.commands.title")}</h2>
      </div>
      <div className="mt-4 grid gap-3 sm:grid-cols-2 2xl:grid-cols-1">
        <Field label={t("workspace.fields.command")}>
          <Input
            onChange={(event) => onChange({ ...form, command: event.target.value })}
            value={form.command}
          />
        </Field>
        <Field label={t("workspace.fields.args")}>
          <Input
            onChange={(event) => onChange({ ...form, args: event.target.value })}
            value={form.args}
          />
        </Field>
        <Field label={t("workspace.fields.workingDir")}>
          <Input
            onChange={(event) => onChange({ ...form, working_dir: event.target.value })}
            value={form.working_dir}
          />
        </Field>
        <Field label={t("workspace.fields.timeout")}>
          <Input
            min="1"
            onChange={(event) => onChange({ ...form, timeout_seconds: event.target.value })}
            type="number"
            value={form.timeout_seconds}
          />
        </Field>
      </div>
      <Button className="mt-4 w-full" disabled={running} icon={<Play className="h-4 w-4" />} type="submit">
        {running ? t("workspace.commands.running") : t("workspace.commands.run")}
      </Button>
      {latestRun && <CommandOutput className="mt-4" run={latestRun} />}
    </form>
  );
}

function CommandHistory({
  loading,
  runs
}: {
  loading: boolean;
  runs: WorkspaceCommandRun[];
}) {
  const { t } = useTranslation();
  return (
    <div className="rounded-lg border border-ink-200 bg-white">
      <div className="border-b border-ink-100 px-4 py-3">
        <h2 className="text-base font-semibold text-ink-900">{t("workspace.commands.history")}</h2>
      </div>
      <div className="max-h-[520px] overflow-auto p-3">
        {loading ? (
          <LoadingState />
        ) : runs.length === 0 ? (
          <EmptyState
            description={t("workspace.commands.empty.description")}
            icon={<Terminal className="h-8 w-8" />}
            title={t("workspace.commands.empty.title")}
          />
        ) : (
          <div className="space-y-3">
            {runs.map((run) => (
              <CommandOutput key={run.id} run={run} />
            ))}
          </div>
        )}
      </div>
    </div>
  );
}

function CommandOutput({
  className = "",
  run
}: {
  className?: string;
  run: WorkspaceCommandRun;
}) {
  const args = formatArgs(run.args);
  return (
    <div className={`rounded-md border border-ink-200 bg-ink-50 p-3 ${className}`}>
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div className="min-w-0 font-mono text-sm font-semibold text-ink-900">
          <span>{run.command}</span>
          {args && <span className="text-ink-500"> {args}</span>}
        </div>
        <Badge tone={run.status === "success" ? "green" : run.status === "blocked" ? "amber" : "neutral"}>
          {run.status}
        </Badge>
      </div>
      <div className="mt-2 text-xs text-ink-500">
        {run.working_dir} · {formatDateTime(run.created_at)}
        {run.duration_ms !== null ? ` · ${run.duration_ms}ms` : ""}
        {run.exit_code !== null ? ` · exit ${run.exit_code}` : ""}
      </div>
      {run.error_message && (
        <pre className="mt-3 overflow-auto whitespace-pre-wrap rounded bg-red-50 p-2 text-xs text-red-700">
          {run.error_message}
        </pre>
      )}
      {run.stdout_preview && (
        <pre className="mt-3 max-h-48 overflow-auto whitespace-pre-wrap rounded bg-white p-2 text-xs text-ink-800">
          {run.stdout_preview}
          {run.stdout_truncated ? "\n..." : ""}
        </pre>
      )}
      {run.stderr_preview && (
        <pre className="mt-3 max-h-48 overflow-auto whitespace-pre-wrap rounded bg-amber-50 p-2 text-xs text-amber-700">
          {run.stderr_preview}
          {run.stderr_truncated ? "\n..." : ""}
        </pre>
      )}
    </div>
  );
}

function policyInput(form: PolicyFormState): UpdateWorkspacePolicyInput {
  return {
    sandbox_type: form.sandbox_type,
    network_policy: form.network_policy,
    network_allowlist: form.network_allowlist
      .split(",")
      .map((item) => item.trim())
      .filter(Boolean),
    max_disk_bytes: numberValue(form.max_disk_mb, 1024) * 1024 * 1024,
    max_file_count: numberValue(form.max_file_count, 1000),
    max_single_file_bytes: numberValue(form.max_single_file_mb, 50) * 1024 * 1024,
    max_command_seconds: numberValue(form.max_command_seconds, 30),
    max_stdout_bytes: numberValue(form.max_stdout_kb, 64) * 1024,
    max_stderr_bytes: numberValue(form.max_stderr_kb, 64) * 1024,
    cpu_limit: form.cpu_limit.trim() || null,
    memory_limit_bytes: numberValue(form.memory_limit_mb, 0) > 0
      ? numberValue(form.memory_limit_mb, 512) * 1024 * 1024
      : null,
    idle_after_seconds: numberValue(form.idle_after_minutes, 60) * 60,
    destroy_after_seconds: numberValue(form.destroy_after_hours, 24) * 60 * 60
  };
}

function policyFormFromStatus(status: WorkspaceStatus): PolicyFormState {
  const workspace = status.workspace;
  return {
    sandbox_type: workspace.sandbox_type,
    network_policy: workspace.network_policy,
    network_allowlist: workspace.network_allowlist.join(", "),
    max_disk_mb: String(Math.round(workspace.max_disk_bytes / 1024 / 1024)),
    max_file_count: String(workspace.max_file_count),
    max_single_file_mb: String(Math.round(workspace.max_single_file_bytes / 1024 / 1024)),
    max_command_seconds: String(workspace.max_command_seconds),
    max_stdout_kb: String(Math.round(workspace.max_stdout_bytes / 1024)),
    max_stderr_kb: String(Math.round(workspace.max_stderr_bytes / 1024)),
    cpu_limit: workspace.cpu_limit ?? "",
    memory_limit_mb:
      workspace.memory_limit_bytes === null
        ? ""
        : String(Math.round(workspace.memory_limit_bytes / 1024 / 1024)),
    idle_after_minutes: String(Math.round(workspace.idle_after_seconds / 60)),
    destroy_after_hours: String(Math.round(workspace.destroy_after_seconds / 60 / 60))
  };
}

function parseArgs(value: string) {
  return value
    .split(/\s+/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function formatArgs(value: unknown) {
  if (Array.isArray(value)) {
    return value.join(" ");
  }
  if (typeof value === "string") {
    return value;
  }
  return "";
}

function parentOf(path: string) {
  const normalized = normalizePath(path);
  if (normalized === "/") {
    return "/";
  }
  const parts = normalized.split("/").filter(Boolean);
  parts.pop();
  return parts.length === 0 ? "/" : `/${parts.join("/")}`;
}

function normalizePath(path: string) {
  const trimmed = path.trim();
  if (!trimmed || trimmed === ".") {
    return "/";
  }
  return trimmed.startsWith("/") ? trimmed : `/${trimmed}`;
}

function numberValue(value: string, fallback: number) {
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed >= 0 ? parsed : fallback;
}

function formatBytes(value: number) {
  if (value < 1024) {
    return `${value} B`;
  }
  if (value < 1024 * 1024) {
    return `${(value / 1024).toFixed(1)} KB`;
  }
  if (value < 1024 * 1024 * 1024) {
    return `${(value / 1024 / 1024).toFixed(1)} MB`;
  }
  return `${(value / 1024 / 1024 / 1024).toFixed(1)} GB`;
}
