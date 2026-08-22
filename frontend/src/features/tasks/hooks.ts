import { useMutation, useQuery } from "@tanstack/react-query";

import {
  createScheduledJob,
  deleteScheduledJob,
  getScheduledJobRun,
  listScheduledJobRuns,
  listScheduledJobTemplates,
  listScheduledJobs,
  pauseScheduledJob,
  resumeScheduledJob,
  runScheduledJobNow
} from "./api";

export const scheduledJobTemplatesQueryKey = ["tasks", "templates"] as const;
export const scheduledJobsQueryKey = ["tasks", "jobs"] as const;
export const scheduledJobRunsQueryKey = ["tasks", "runs"] as const;

export function useScheduledJobTemplates() {
  return useQuery({
    queryKey: scheduledJobTemplatesQueryKey,
    queryFn: listScheduledJobTemplates
  });
}

export function useScheduledJobs(status?: string) {
  return useQuery({
    queryKey: [...scheduledJobsQueryKey, status || "all"],
    queryFn: () => listScheduledJobs(status)
  });
}

export function useCreateScheduledJob() {
  return useMutation({
    mutationFn: createScheduledJob
  });
}

export function usePauseScheduledJob() {
  return useMutation({
    mutationFn: pauseScheduledJob
  });
}

export function useResumeScheduledJob() {
  return useMutation({
    mutationFn: resumeScheduledJob
  });
}

export function useDeleteScheduledJob() {
  return useMutation({
    mutationFn: deleteScheduledJob
  });
}

export function useRunScheduledJobNow() {
  return useMutation({
    mutationFn: runScheduledJobNow
  });
}

export function useScheduledJobRuns(jobID?: string) {
  return useQuery({
    enabled: Boolean(jobID),
    queryKey: [...scheduledJobRunsQueryKey, jobID],
    queryFn: () => listScheduledJobRuns(jobID as string)
  });
}

export function useScheduledJobRun(runID?: string) {
  return useQuery({
    enabled: Boolean(runID),
    queryKey: [...scheduledJobRunsQueryKey, "detail", runID],
    queryFn: () => getScheduledJobRun(runID as string)
  });
}
