import { apiClient } from "../../lib/apiClient";

import type {
  EnableWorkspaceInput,
  RunWorkspaceCommandInput,
  UpdateWorkspacePolicyInput,
  WorkspaceCommandRunsResult,
  WorkspaceFileList,
  WorkspaceReadFileResult,
  WorkspaceRunCommandResult,
  WorkspaceStatus,
  WorkspaceWriteFileResult,
  WriteWorkspaceFileInput
} from "./types";

export function getWorkspaceStatus() {
  return apiClient<WorkspaceStatus>("/me/workspace");
}

export function enableWorkspace(input: EnableWorkspaceInput) {
  return apiClient<WorkspaceStatus>("/me/workspace", {
    method: "POST",
    body: input
  });
}

export function updateWorkspacePolicy(input: UpdateWorkspacePolicyInput) {
  return apiClient<WorkspaceStatus>("/me/workspace", {
    method: "PATCH",
    body: input
  });
}

export function listWorkspaceFiles(path: string) {
  const params = new URLSearchParams({ path });
  return apiClient<WorkspaceFileList>(`/me/workspace/files?${params.toString()}`);
}

export function readWorkspaceFile(path: string) {
  const params = new URLSearchParams({ path });
  return apiClient<WorkspaceReadFileResult>(
    `/me/workspace/files/content?${params.toString()}`
  );
}

export function writeWorkspaceFile(input: WriteWorkspaceFileInput) {
  return apiClient<WorkspaceWriteFileResult>("/me/workspace/files/content", {
    method: "PUT",
    body: input
  });
}

export function runWorkspaceCommand(input: RunWorkspaceCommandInput) {
  return apiClient<WorkspaceRunCommandResult>("/me/workspace/commands", {
    method: "POST",
    body: input
  });
}

export function listWorkspaceCommandRuns(limit = 50) {
  const params = new URLSearchParams({ limit: String(limit) });
  return apiClient<WorkspaceCommandRunsResult>(
    `/me/workspace/commands?${params.toString()}`
  );
}
