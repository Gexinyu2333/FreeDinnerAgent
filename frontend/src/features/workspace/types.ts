export type UserWorkspace = {
  id: string;
  user_id: string;
  status: string;
  root_path: string;
  sandbox_type: string;
  network_policy: string;
  network_allowlist: string[];
  max_disk_bytes: number;
  max_file_count: number;
  max_single_file_bytes: number;
  max_command_seconds: number;
  max_stdout_bytes: number;
  max_stderr_bytes: number;
  cpu_limit: string | null;
  memory_limit_bytes: number | null;
  last_active_at: string | null;
  idle_after_seconds: number;
  destroy_after_seconds: number;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
};

export type WorkspaceQuotaSnapshot = {
  id: string;
  user_id: string;
  workspace_id: string;
  used_disk_bytes: number;
  file_count: number;
  command_count: number;
  active_process_count: number;
  metadata: Record<string, unknown>;
  created_at: string;
};

export type WorkspaceStatus = {
  workspace: UserWorkspace;
  quota: WorkspaceQuotaSnapshot;
};

export type WorkspaceFileEntry = {
  name: string;
  path: string;
  type: "directory" | "file" | string;
  size_bytes: number;
  last_modified: string;
};

export type WorkspaceFileList = {
  path: string;
  items: WorkspaceFileEntry[];
};

export type WorkspaceReadFileResult = {
  path: string;
  content: string;
  size_bytes: number;
  content_hash: string;
  mime_type: string;
};

export type WorkspaceFileRecord = {
  id: string;
  user_id: string;
  workspace_id: string;
  relative_path: string;
  file_type: string;
  size_bytes: number;
  content_hash: string | null;
  mime_type: string | null;
  created_by: string;
  status: string;
  metadata: Record<string, unknown>;
  created_at: string;
  updated_at: string;
};

export type WorkspaceWriteFileResult = {
  file: WorkspaceFileRecord;
};

export type WorkspaceCommandRun = {
  id: string;
  user_id: string;
  workspace_id: string;
  conversation_id: string | null;
  agent_turn_id: string | null;
  tool_call_id: string | null;
  command: string;
  args: unknown;
  working_dir: string;
  network_policy: string;
  status: string;
  exit_code: number | null;
  stdout_preview: string | null;
  stderr_preview: string | null;
  stdout_truncated: boolean;
  stderr_truncated: boolean;
  duration_ms: number | null;
  error_message: string | null;
  metadata: Record<string, unknown>;
  created_at: string;
  started_at: string | null;
  finished_at: string | null;
};

export type WorkspaceRunCommandResult = {
  run: WorkspaceCommandRun;
};

export type WorkspaceCommandRunsResult = {
  runs: WorkspaceCommandRun[];
};

export type EnableWorkspaceInput = {
  sandbox_type: string;
  network_policy: string;
  network_allowlist: string[];
  max_disk_bytes: number;
  max_file_count: number;
  max_single_file_bytes: number;
  max_command_seconds: number;
  max_stdout_bytes: number;
  max_stderr_bytes: number;
  cpu_limit: string | null;
  memory_limit_bytes: number | null;
  idle_after_seconds: number;
  destroy_after_seconds: number;
};

export type UpdateWorkspacePolicyInput = Partial<EnableWorkspaceInput>;

export type WriteWorkspaceFileInput = {
  path: string;
  content: string;
};

export type RunWorkspaceCommandInput = {
  command: string;
  args: string[];
  working_dir: string;
  timeout_seconds: number;
};
