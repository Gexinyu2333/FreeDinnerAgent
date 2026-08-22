import { apiClient } from "../../lib/apiClient";

import type {
  CreateScheduledJobInput,
  RunNowResult,
  ScheduledAgentJob,
  ScheduledAgentJobRun,
  ScheduledJobTemplate
} from "./types";

export function listScheduledJobTemplates() {
  return apiClient<ScheduledJobTemplate[]>("/scheduled-agent-job-templates");
}

export function createScheduledJob(input: CreateScheduledJobInput) {
  return apiClient<ScheduledAgentJob>("/scheduled-agent-jobs", {
    method: "POST",
    body: input
  });
}

export function listScheduledJobs(status?: string) {
  const params = new URLSearchParams({ limit: "80" });
  if (status) {
    params.set("status", status);
  }
  return apiClient<ScheduledAgentJob[]>(`/scheduled-agent-jobs?${params.toString()}`);
}

export function pauseScheduledJob(id: string) {
  return apiClient<ScheduledAgentJob>(`/scheduled-agent-jobs/${id}/pause`, {
    method: "POST"
  });
}

export function resumeScheduledJob(id: string) {
  return apiClient<ScheduledAgentJob>(`/scheduled-agent-jobs/${id}/resume`, {
    method: "POST"
  });
}

export function deleteScheduledJob(id: string) {
  return apiClient<ScheduledAgentJob>(`/scheduled-agent-jobs/${id}`, {
    method: "DELETE"
  });
}

export function runScheduledJobNow(id: string) {
  return apiClient<RunNowResult>(`/scheduled-agent-jobs/${id}/run-now`, {
    method: "POST"
  });
}

export function listScheduledJobRuns(jobID: string) {
  return apiClient<ScheduledAgentJobRun[]>(`/scheduled-agent-jobs/${jobID}/runs?limit=50`);
}

export function getScheduledJobRun(runID: string) {
  return apiClient<ScheduledAgentJobRun>(`/scheduled-agent-job-runs/${runID}`);
}
