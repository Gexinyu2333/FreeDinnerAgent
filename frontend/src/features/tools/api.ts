import { apiClient } from "../../lib/apiClient";

import type { ToolApprovalRequest, ToolDefinition } from "./types";

export function listTools() {
  return apiClient<ToolDefinition[]>("/tools");
}

export function listToolApprovals(status?: string, limit = 50) {
  const params = new URLSearchParams({ limit: String(limit) });
  if (status) {
    params.set("status", status);
  }
  return apiClient<ToolApprovalRequest[]>(`/tool-approval-requests?${params.toString()}`);
}

export function approveToolApproval(id: string) {
  return apiClient<ToolApprovalRequest>(`/tool-approval-requests/${id}/approve`, {
    method: "POST"
  });
}

export function rejectToolApproval(id: string) {
  return apiClient<ToolApprovalRequest>(`/tool-approval-requests/${id}/reject`, {
    method: "POST"
  });
}
