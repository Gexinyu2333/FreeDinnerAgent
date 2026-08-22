import { useMutation, useQuery } from "@tanstack/react-query";

import {
  approveToolApproval,
  listToolApprovals,
  listTools,
  rejectToolApproval
} from "./api";

export const toolsQueryKey = ["tools", "definitions"] as const;
export const toolApprovalsQueryKey = ["tools", "approvals"] as const;

export function useTools() {
  return useQuery({
    queryKey: toolsQueryKey,
    queryFn: listTools
  });
}

export function useToolApprovals(status?: string) {
  return useQuery({
    queryKey: [...toolApprovalsQueryKey, status || "all"],
    queryFn: () => listToolApprovals(status)
  });
}

export function useApproveToolApproval() {
  return useMutation({
    mutationFn: approveToolApproval
  });
}

export function useRejectToolApproval() {
  return useMutation({
    mutationFn: rejectToolApproval
  });
}
