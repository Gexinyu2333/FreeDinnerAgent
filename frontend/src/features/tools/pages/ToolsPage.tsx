import { useQueryClient } from "@tanstack/react-query";
import { Check, Hammer, ShieldAlert, X } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";

import { Badge } from "../../../components/ui/Badge";
import { Button } from "../../../components/ui/Button";
import { EmptyState } from "../../../components/ui/EmptyState";
import { LoadingState } from "../../../components/ui/LoadingState";
import { Select } from "../../../components/ui/Select";
import { Toast } from "../../../components/ui/Toast";
import { ApiError } from "../../../lib/errors";
import { formatDateTime } from "../../../lib/format";
import {
  toolApprovalsQueryKey,
  useApproveToolApproval,
  useRejectToolApproval,
  useToolApprovals,
  useTools
} from "../hooks";
import type { ToolApprovalRequest, ToolDefinition } from "../types";

export function ToolsPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const toolsQuery = useTools();
  const [approvalStatus, setApprovalStatus] = useState("pending");
  const approvalsQuery = useToolApprovals(approvalStatus || undefined);
  const approveMutation = useApproveToolApproval();
  const rejectMutation = useRejectToolApproval();
  const tools = toolsQuery.data ?? [];
  const approvals = approvalsQuery.data ?? [];
  const mutationError = approveMutation.error ?? rejectMutation.error ?? null;

  function refreshApprovals() {
    void queryClient.invalidateQueries({ queryKey: toolApprovalsQueryKey });
  }

  return (
    <section className="space-y-5">
      <div>
        <h1 className="text-2xl font-semibold text-ink-900">{t("tools.title")}</h1>
        <p className="mt-2 text-sm leading-6 text-ink-500">{t("tools.description")}</p>
      </div>

      {mutationError && (
        <Toast
          message={
            mutationError instanceof ApiError
              ? mutationError.message
              : t("tools.errors.operationFailed")
          }
          tone="error"
        />
      )}

      <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_430px]">
        <div className="space-y-3">
          <h2 className="text-base font-semibold text-ink-900">{t("tools.registry.title")}</h2>
          {toolsQuery.isLoading ? (
            <LoadingState />
          ) : tools.length === 0 ? (
            <EmptyState
              description={t("tools.registry.empty.description")}
              icon={<Hammer className="h-8 w-8" />}
              title={t("tools.registry.empty.title")}
            />
          ) : (
            <div className="grid gap-3 md:grid-cols-2">
              {tools.map((tool) => (
                <ToolCard key={tool.id} tool={tool} />
              ))}
            </div>
          )}
        </div>

        <aside className="space-y-3">
          <div className="flex items-center justify-between gap-3">
            <h2 className="text-base font-semibold text-ink-900">
              {t("tools.approvals.title")}
            </h2>
            <Select
              className="w-40"
              onChange={(event) => setApprovalStatus(event.target.value)}
              value={approvalStatus}
            >
              <option value="pending">{t("tools.approvals.status.pending")}</option>
              <option value="approved">{t("tools.approvals.status.approved")}</option>
              <option value="rejected">{t("tools.approvals.status.rejected")}</option>
              <option value="">{t("tools.approvals.status.all")}</option>
            </Select>
          </div>
          {approvalsQuery.isLoading ? (
            <LoadingState />
          ) : approvals.length === 0 ? (
            <EmptyState
              description={t("tools.approvals.empty.description")}
              icon={<ShieldAlert className="h-8 w-8" />}
              title={t("tools.approvals.empty.title")}
            />
          ) : (
            <div className="space-y-3">
              {approvals.map((approval) => (
                <ApprovalCard
                  approval={approval}
                  key={approval.id}
                  onApprove={() =>
                    approveMutation.mutate(approval.id, { onSuccess: refreshApprovals })
                  }
                  onReject={() =>
                    rejectMutation.mutate(approval.id, { onSuccess: refreshApprovals })
                  }
                  resolving={approveMutation.isPending || rejectMutation.isPending}
                />
              ))}
            </div>
          )}
        </aside>
      </div>
    </section>
  );
}

function ToolCard({ tool }: { tool: ToolDefinition }) {
  const { t } = useTranslation();
  return (
    <article className="rounded-lg border border-ink-200 bg-white p-4 shadow-sm">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <h3 className="truncate text-base font-semibold text-ink-900">
            {tool.display_name}
          </h3>
          <p className="mt-1 text-xs text-ink-500">
            {tool.namespace}.{tool.name}
          </p>
        </div>
        <Badge tone={tool.is_enabled ? "green" : "amber"}>
          {tool.is_enabled ? t("common.enabled") : t("common.disabled")}
        </Badge>
      </div>
      <p className="mt-3 text-sm leading-6 text-ink-700">{tool.description}</p>
      <div className="mt-4 flex flex-wrap gap-2">
        <Badge tone="blue">{tool.category}</Badge>
        <Badge>{tool.handler_type}</Badge>
        <Badge>{tool.permission_level}</Badge>
        {tool.requires_approval && (
          <Badge tone="amber">{t("tools.registry.requiresApproval")}</Badge>
        )}
        {tool.active_version && (
          <Badge>{t("tools.registry.version", { version: tool.active_version })}</Badge>
        )}
      </div>
    </article>
  );
}

function ApprovalCard({
  approval,
  onApprove,
  onReject,
  resolving
}: {
  approval: ToolApprovalRequest;
  onApprove: () => void;
  onReject: () => void;
  resolving: boolean;
}) {
  const { t } = useTranslation();
  return (
    <article className="rounded-lg border border-ink-200 bg-white p-4 shadow-sm">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h3 className="text-base font-semibold text-ink-900">{approval.risk_level}</h3>
          <p className="mt-1 text-xs text-ink-500">{formatDateTime(approval.created_at)}</p>
        </div>
        <Badge tone={approval.status === "pending" ? "amber" : "green"}>
          {approval.status}
        </Badge>
      </div>
      <p className="mt-3 text-sm leading-6 text-ink-700">{approval.approval_reason}</p>
      <pre className="mt-3 max-h-44 overflow-auto rounded-md bg-ink-900 p-3 text-xs leading-5 text-white">
        {JSON.stringify(approval.proposed_arguments, null, 2)}
      </pre>
      {approval.status === "pending" && (
        <div className="mt-4 flex gap-2">
          <Button
            disabled={resolving}
            icon={<Check className="h-4 w-4" />}
            onClick={onApprove}
          >
            {t("tools.approvals.approve")}
          </Button>
          <Button
            disabled={resolving}
            icon={<X className="h-4 w-4" />}
            onClick={onReject}
            variant="secondary"
          >
            {t("tools.approvals.reject")}
          </Button>
        </div>
      )}
    </article>
  );
}
