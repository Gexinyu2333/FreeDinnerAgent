import { useMutation, useQuery } from "@tanstack/react-query";

import {
  enableWorkspace,
  getWorkspaceStatus,
  listWorkspaceCommandRuns,
  listWorkspaceFiles,
  readWorkspaceFile,
  runWorkspaceCommand,
  updateWorkspacePolicy,
  writeWorkspaceFile
} from "./api";

export const workspaceStatusQueryKey = ["workspace", "status"] as const;
export const workspaceFilesQueryKey = ["workspace", "files"] as const;
export const workspaceCommandRunsQueryKey = ["workspace", "commands"] as const;

export function useWorkspaceStatus() {
  return useQuery({
    queryKey: workspaceStatusQueryKey,
    queryFn: getWorkspaceStatus,
    retry: false
  });
}

export function useEnableWorkspace() {
  return useMutation({
    mutationFn: enableWorkspace
  });
}

export function useUpdateWorkspacePolicy() {
  return useMutation({
    mutationFn: updateWorkspacePolicy
  });
}

export function useWorkspaceFiles(path: string, enabled: boolean) {
  return useQuery({
    enabled,
    queryKey: [...workspaceFilesQueryKey, path],
    queryFn: () => listWorkspaceFiles(path),
    retry: false
  });
}

export function useReadWorkspaceFile() {
  return useMutation({
    mutationFn: readWorkspaceFile
  });
}

export function useWriteWorkspaceFile() {
  return useMutation({
    mutationFn: writeWorkspaceFile
  });
}

export function useRunWorkspaceCommand() {
  return useMutation({
    mutationFn: runWorkspaceCommand
  });
}

export function useWorkspaceCommandRuns(enabled: boolean) {
  return useQuery({
    enabled,
    queryKey: workspaceCommandRunsQueryKey,
    queryFn: () => listWorkspaceCommandRuns(50),
    retry: false
  });
}
