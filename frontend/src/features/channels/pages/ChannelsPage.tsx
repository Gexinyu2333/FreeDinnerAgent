import { useQueryClient } from "@tanstack/react-query";
import {
  Check,
  Edit3,
  MessageSquareMore,
  PlugZap,
  RotateCcw,
  RefreshCw,
  Send,
  ShieldCheck,
  Trash2,
  X
} from "lucide-react";
import { FormEvent, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate, useParams } from "react-router-dom";

import { Badge } from "../../../components/ui/Badge";
import { Button } from "../../../components/ui/Button";
import { Dialog } from "../../../components/ui/Dialog";
import { EmptyState } from "../../../components/ui/EmptyState";
import { Input } from "../../../components/ui/Input";
import { LoadingState } from "../../../components/ui/LoadingState";
import { Select } from "../../../components/ui/Select";
import { SecretInput } from "../../../components/ui/SecretInput";
import { Switch } from "../../../components/ui/Switch";
import { Tabs } from "../../../components/ui/Tabs";
import { Textarea } from "../../../components/ui/Textarea";
import { Toast } from "../../../components/ui/Toast";
import { useToast } from "../../../components/ui/ToastProvider";
import { ApiError } from "../../../lib/errors";
import { formatDateTime } from "../../../lib/format";
import {
  channelConnectionsQueryKey,
  channelExternalConversationMessagesQueryKey,
  channelExternalConversationsQueryKey,
  channelInboxQueryKey,
  channelOutboxQueryKey,
  channelPoliciesQueryKey,
  useApproveOutboxMessage,
  useCancelOutboxMessage,
  useChannelConnections,
  useChannelPolicies,
  useChannelProviders,
  useCreateChannelConnection,
  useCreateOutboxDraft,
  useDeleteChannelConnection,
  useDeleteChannelPolicy,
  useExternalConversationMessages,
  useExternalConversations,
  useInboxEvents,
  useOutboxMessages,
  useSendOutboxMessage,
  useUpdateChannelConnection,
  useUpsertChannelPolicy
} from "../hooks";
import type { Message } from "../../chat/types";
import type {
  ChannelInboxEvent,
  ChannelOutboxMessage,
  ChannelPolicy,
  ChannelProviderEndpointField,
  ChannelProviderDefinition,
  ChannelProviderFormField,
  ChannelProviderFormMetadata,
  ExternalConversation,
  PublicChannelConnection
} from "../types";

type ConnectionFormState = {
  provider_id: string;
  display_name: string;
  external_account_id: string;
  external_account_name: string;
  endpoint_values: Record<string, string>;
  secret_values: Record<string, string>;
  config: string;
};

type PolicyFormState = {
  scope_type: string;
  external_scope_id: string;
  mode: string;
  trigger_keywords: string;
  allow_memory_write: boolean;
  allow_tool_use: boolean;
  require_approval_for_outbound: boolean;
  rate_limit_per_minute: string;
  rate_limit_policy: string;
};

const channelSections = ["overview", "setup", "policies", "sessions", "inbox", "outbox", "logs"] as const;
type ChannelSection = (typeof channelSections)[number];

const defaultConnectionForm: ConnectionFormState = {
  provider_id: "",
  display_name: "NapCatQQ",
  external_account_id: "",
  external_account_name: "",
  endpoint_values: {},
  secret_values: {},
  config: "{\n}"
};

const defaultPolicyForm: PolicyFormState = {
  scope_type: "group_chat",
  external_scope_id: "",
  mode: "mention_only",
  trigger_keywords: "",
  allow_memory_write: true,
  allow_tool_use: true,
  require_approval_for_outbound: true,
  rate_limit_per_minute: "6",
  rate_limit_policy: '{\n  "rate_limits": []\n}'
};

export function ChannelsPage() {
  const { t } = useTranslation();
  const toast = useToast();
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const params = useParams();
  const providersQuery = useChannelProviders();
  const connectionsQuery = useChannelConnections();
  const createMutation = useCreateChannelConnection();
  const updateMutation = useUpdateChannelConnection();
  const deleteConnectionMutation = useDeleteChannelConnection();
  const upsertPolicyMutation = useUpsertChannelPolicy();
  const deletePolicyMutation = useDeleteChannelPolicy();
  const approveMutation = useApproveOutboxMessage();
  const cancelMutation = useCancelOutboxMessage();
  const sendMutation = useSendOutboxMessage();
  const createDraftMutation = useCreateOutboxDraft();
  const [connectionForm, setConnectionForm] =
    useState<ConnectionFormState>(defaultConnectionForm);
  const [policyForm, setPolicyForm] = useState<PolicyFormState>(defaultPolicyForm);
  const [editingConnectionID, setEditingConnectionID] = useState<string | null>(null);
  const [editingPolicyID, setEditingPolicyID] = useState<string | null>(null);
  const [connectionDialogOpen, setConnectionDialogOpen] = useState(false);
  const [policyDialogOpen, setPolicyDialogOpen] = useState(false);
  const [outboxStatus, setOutboxStatus] = useState("pending");
  const [draftContent, setDraftContent] = useState("");
  const [formError, setFormError] = useState<string | null>(null);

  const providers = providersQuery.data ?? [];
  const connections = connectionsQuery.data ?? [];
  const isCreatingConnection = params.connectionId === "new";
  const selectedConnectionID = isCreatingConnection ? undefined : params.connectionId || connections[0]?.id;
  const selectedConnection = connections.find((item) => item.id === selectedConnectionID);
  const activeSection = normalizeChannelSection(params.section);
  const selectedExternalConversationID = params.externalConversationId;
  const policiesQuery = useChannelPolicies(selectedConnectionID);
  const externalConversationsQuery = useExternalConversations(selectedConnectionID);
  const externalMessagesQuery = useExternalConversationMessages(
    selectedConnectionID,
    selectedExternalConversationID
  );
  const inboxQuery = useInboxEvents(selectedConnectionID);
  const outboxQuery = useOutboxMessages(selectedConnectionID, outboxStatus || undefined);
  const policies = policiesQuery.data ?? [];
  const externalConversations = externalConversationsQuery.data ?? [];
  const inboxEvents = inboxQuery.data ?? [];
  const outboxMessages = outboxQuery.data ?? [];
  const mutationError =
    createMutation.error ??
    updateMutation.error ??
    deleteConnectionMutation.error ??
    upsertPolicyMutation.error ??
    deletePolicyMutation.error ??
    approveMutation.error ??
    cancelMutation.error ??
    sendMutation.error ??
    createDraftMutation.error ??
    null;

  useEffect(() => {
    const firstProvider = providers[0];
    if (!connectionForm.provider_id && firstProvider) {
      setConnectionForm((current) =>
        withProviderDefaults({ ...current, provider_id: firstProvider.id }, firstProvider)
      );
    }
  }, [connectionForm.provider_id, providers]);

  useEffect(() => {
    if (!params.connectionId && connections[0]) {
      navigate(`/app/channels/${connections[0].id}/overview`, { replace: true });
    }
  }, [connections, navigate, params.connectionId]);

  useEffect(() => {
    if (params.connectionId && !params.section) {
      navigate(`/app/channels/${params.connectionId}/overview`, { replace: true });
    }
  }, [navigate, params.connectionId, params.section]);

  useEffect(() => {
    if (
      activeSection === "setup" &&
      selectedConnection &&
      editingConnectionID !== selectedConnection.id
    ) {
      setEditingConnectionID(selectedConnection.id);
      const provider = providers.find((item) => item.id === selectedConnection.provider_id);
      setConnectionForm(connectionToForm(selectedConnection, provider));
    }
  }, [activeSection, editingConnectionID, providers, selectedConnection]);

  useEffect(() => {
    if (isCreatingConnection && activeSection === "setup") {
      setConnectionDialogOpen(true);
    }
  }, [activeSection, isCreatingConnection]);

  const selectedProvider = useMemo(
    () => providers.find((provider) => provider.id === selectedConnection?.provider_id),
    [providers, selectedConnection]
  );
  const formProvider = useMemo(
    () => providers.find((provider) => provider.id === connectionForm.provider_id),
    [connectionForm.provider_id, providers]
  );

  function refresh(connectionID = selectedConnectionID) {
    void queryClient.invalidateQueries({ queryKey: channelConnectionsQueryKey });
    if (connectionID) {
      void queryClient.invalidateQueries({ queryKey: channelPoliciesQueryKey });
      void queryClient.invalidateQueries({ queryKey: channelExternalConversationsQueryKey });
      void queryClient.invalidateQueries({ queryKey: channelExternalConversationMessagesQueryKey });
      void queryClient.invalidateQueries({ queryKey: channelInboxQueryKey });
      void queryClient.invalidateQueries({ queryKey: channelOutboxQueryKey });
    }
  }

  function handleSaveConnection(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormError(null);
    let config: Record<string, unknown>;
    try {
      config = parseJSONObject(connectionForm.config);
    } catch {
      setFormError(t("channels.errors.invalidJson"));
      return;
    }
    const payload = {
      provider_id: connectionForm.provider_id,
      display_name: connectionForm.display_name.trim(),
      external_account_id: connectionForm.external_account_id.trim() || null,
      external_account_name: connectionForm.external_account_name.trim() || null,
      endpoints: buildProviderEndpoints(formProvider, connectionForm),
      config: buildProviderConfig(formProvider, connectionForm, config)
    };
    const onSuccess = (connection: PublicChannelConnection) => {
      resetConnectionForm();
      setConnectionDialogOpen(false);
      navigate(`/app/channels/${connection.id}`);
      refresh(connection.id);
      toast.notify(
        editingConnectionID
          ? t("channels.connection.updated")
          : t("channels.connection.created")
      );
    };
    if (editingConnectionID) {
      updateMutation.mutate(
        {
          ...payload,
          connection_id: editingConnectionID
        },
        { onSuccess }
      );
      return;
    }
    createMutation.mutate(payload, { onSuccess });
  }

  function resetConnectionForm() {
    setEditingConnectionID(null);
    setConnectionDialogOpen(false);
    const firstProvider = providers[0];
    setConnectionForm({
      ...withProviderDefaults(defaultConnectionForm, firstProvider),
      provider_id: firstProvider?.id ?? ""
    });
  }

  function loadConnection(connection: PublicChannelConnection) {
    setEditingConnectionID(connection.id);
    const provider = providers.find((item) => item.id === connection.provider_id);
    setConnectionForm(connectionToForm(connection, provider));
    setConnectionDialogOpen(true);
    navigate(`/app/channels/${connection.id}/setup`);
    window.scrollTo({ top: 0, behavior: "smooth" });
  }

  function deleteConnection(connection: PublicChannelConnection) {
    if (!window.confirm(t("channels.connection.deleteConfirm", { name: connection.display_name }))) {
      return;
    }
    deleteConnectionMutation.mutate(connection.id, {
      onSuccess: () => {
        resetConnectionForm();
        navigate("/app/channels", { replace: true });
        refresh();
        toast.notify(t("channels.connection.deleted"));
      }
    });
  }

  function handleUpsertPolicy(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedConnectionID) {
      return;
    }
    setFormError(null);
    let rateLimitPolicy: Record<string, unknown>;
    try {
      rateLimitPolicy = parseJSONObject(policyForm.rate_limit_policy);
    } catch {
      setFormError(t("channels.errors.invalidJson"));
      return;
    }
    upsertPolicyMutation.mutate(
      {
        connection_id: selectedConnectionID,
        scope_type: policyForm.scope_type,
        external_scope_id: policyForm.external_scope_id.trim() || null,
        mode: policyForm.mode,
        trigger_keywords: splitKeywords(policyForm.trigger_keywords),
        allow_memory_write: policyForm.allow_memory_write,
        allow_tool_use: policyForm.allow_tool_use,
        require_approval_for_outbound: policyForm.require_approval_for_outbound,
        rate_limit_per_minute: Number(policyForm.rate_limit_per_minute) || 6,
        rate_limit_policy: rateLimitPolicy
      },
      {
        onSuccess: () => {
          refresh(selectedConnectionID);
          resetPolicyForm();
          toast.notify(t("channels.policy.saved"));
        }
      }
    );
  }

  function loadPolicy(policy: ChannelPolicy) {
    setEditingPolicyID(policy.id);
    setPolicyDialogOpen(true);
    setPolicyForm({
      scope_type: policy.scope_type,
      external_scope_id: policy.external_scope_id ?? "",
      mode: policy.mode,
      trigger_keywords: policy.trigger_keywords.join(", "),
      allow_memory_write: policy.allow_memory_write,
      allow_tool_use: policy.allow_tool_use,
      require_approval_for_outbound: policy.require_approval_for_outbound,
      rate_limit_per_minute: String(policy.rate_limit_per_minute),
      rate_limit_policy: JSON.stringify(policy.metadata ?? {}, null, 2)
    });
  }

  function resetPolicyForm() {
    setEditingPolicyID(null);
    setPolicyDialogOpen(false);
    setPolicyForm(defaultPolicyForm);
  }

  function deletePolicy(policy: ChannelPolicy) {
    if (!selectedConnectionID) {
      return;
    }
    const label = policy.external_scope_id || t("channels.policy.globalScope");
    if (!window.confirm(t("channels.policy.deleteConfirm", { name: label }))) {
      return;
    }
    deletePolicyMutation.mutate(
      {
        connection_id: selectedConnectionID,
        policy_id: policy.id
      },
      {
        onSuccess: () => {
          if (editingPolicyID === policy.id) {
            resetPolicyForm();
          }
          refresh(selectedConnectionID);
          toast.notify(t("channels.policy.deleted"));
        }
      }
    );
  }

  function mutateOutbox(
    outboxID: string,
    action: "approve" | "cancel" | "send"
  ) {
    if (
      action !== "approve" &&
      !window.confirm(t(`channels.outbox.confirm.${action}`))
    ) {
      return;
    }
    const mutation =
      action === "approve"
        ? approveMutation
        : action === "cancel"
          ? cancelMutation
          : sendMutation;
    mutation.mutate(outboxID, {
      onSuccess: () => {
        refresh(selectedConnectionID);
        toast.notify(t(`channels.outbox.success.${action}`));
      }
    });
  }

  function createExternalDraft() {
    if (!selectedConnectionID || !selectedExternalConversationID) {
      return;
    }
    createDraftMutation.mutate(
      {
        connection_id: selectedConnectionID,
        external_conversation_id: selectedExternalConversationID,
        content: draftContent,
        message_type: "text"
      },
      {
        onSuccess: () => {
          setDraftContent("");
          refresh(selectedConnectionID);
          toast.notify(t("channels.sessions.draftCreated"));
        }
      }
    );
  }

  return (
    <section className="max-w-full space-y-5 overflow-x-hidden">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-ink-900">{t("channels.title")}</h1>
          <p className="mt-2 text-sm leading-6 text-ink-500">
            {t("channels.description")}
          </p>
        </div>
        <Button
          icon={<RefreshCw className="h-4 w-4" />}
          onClick={() => refresh()}
          variant="secondary"
        >
          {t("common.refresh")}
        </Button>
      </div>

      <Dialog
        onClose={resetConnectionForm}
        open={connectionDialogOpen}
        title={
          editingConnectionID
            ? t("channels.connection.edit")
            : t("channels.connection.create")
        }
      >
        <ConnectionForm
          editing={Boolean(editingConnectionID)}
          form={connectionForm}
          loading={createMutation.isPending || updateMutation.isPending}
          onChange={setConnectionForm}
          onReset={resetConnectionForm}
          onSubmit={handleSaveConnection}
          providers={providers}
          selectedProvider={formProvider}
        />
      </Dialog>

      <Dialog
        onClose={resetPolicyForm}
        open={policyDialogOpen}
        title={
          editingPolicyID
            ? t("channels.policy.editTitle")
            : t("channels.policy.create")
        }
      >
        <PolicyForm
          editing={Boolean(editingPolicyID)}
          form={policyForm}
          onChange={setPolicyForm}
          onReset={resetPolicyForm}
          onSubmit={handleUpsertPolicy}
          saving={upsertPolicyMutation.isPending}
        />
      </Dialog>

      {mutationError && (
        <Toast
          message={
            mutationError instanceof ApiError
              ? mutationError.message
              : t("channels.errors.operationFailed")
          }
          tone="error"
        />
      )}
      {formError && <Toast message={formError} tone="error" />}
      <div className="grid max-w-full min-w-0 gap-5 xl:grid-cols-[minmax(320px,400px)_minmax(0,1fr)]">
        <div className="min-w-0 space-y-4">
          <ProviderPanel loading={providersQuery.isLoading} providers={providers} />
          <ConnectionList
            connections={connections}
            loading={connectionsQuery.isLoading}
            onCreate={() => {
              resetConnectionForm();
              setConnectionDialogOpen(true);
              navigate("/app/channels/new/setup");
            }}
            onSelect={(connectionID) => navigate(`/app/channels/${connectionID}/overview`)}
            selectedConnectionID={selectedConnectionID}
          />
        </div>

        <div className="min-w-0 space-y-4">
          {!selectedConnection ? (
            <EmptyState
              description={t("channels.connections.empty.description")}
              icon={<PlugZap className="h-8 w-8" />}
              title={t("channels.connections.empty.title")}
            />
          ) : (
            <>
              <ConnectionHeader
                connection={selectedConnection}
                onDelete={deleteConnection}
                onEdit={loadConnection}
                provider={selectedProvider}
              />
              <Tabs
                activeKey={activeSection}
                items={channelSections.map((section) => ({
                  key: section,
                  label: t(`channels.sections.${section}`)
                }))}
                onChange={(section) =>
                  navigate(`/app/channels/${selectedConnection.id}/${section}`)
                }
              />
              {activeSection === "overview" && (
                <ChannelOverview
                  conversations={externalConversations}
                  inboxEvents={inboxEvents}
                  outboxMessages={outboxMessages}
                />
              )}
              {activeSection === "setup" && (
                <SetupPanel
                  connection={selectedConnection}
                  onEdit={loadConnection}
                />
              )}
              {activeSection === "policies" && (
                <div className="min-w-0 space-y-4">
                  <div className="flex justify-end">
                    <Button
                      icon={<ShieldCheck className="h-4 w-4" />}
                      onClick={() => {
                        resetPolicyForm();
                        setPolicyDialogOpen(true);
                      }}
                    >
                      {t("channels.policy.create")}
                    </Button>
                  </div>
                  <PolicyList
                    deletingID={deletePolicyMutation.variables?.policy_id}
                    loading={policiesQuery.isLoading}
                    onDelete={deletePolicy}
                    onEdit={loadPolicy}
                    policies={policies}
                  />
                </div>
              )}
              {activeSection === "inbox" && (
                <InboxPanel
                  events={inboxEvents}
                  loading={inboxQuery.isLoading}
                />
              )}
              {activeSection === "outbox" && (
                <OutboxPanel
                  loading={outboxQuery.isLoading}
                  messages={outboxMessages}
                  onAction={mutateOutbox}
                  onStatusChange={setOutboxStatus}
                  status={outboxStatus}
                />
              )}
              {activeSection === "sessions" && (
                <ChannelSessionsPanel
                  conversations={externalConversations}
                  draftContent={draftContent}
                  draftSaving={createDraftMutation.isPending}
                  loading={externalConversationsQuery.isLoading}
                  messages={externalMessagesQuery.data ?? []}
                  messagesLoading={externalMessagesQuery.isLoading}
                  onCreateDraft={createExternalDraft}
                  onDraftChange={setDraftContent}
                  onSelect={(externalID) =>
                    navigate(`/app/channels/${selectedConnection.id}/sessions/${externalID}`)
                  }
                  selectedExternalConversationID={selectedExternalConversationID}
                />
              )}
              {activeSection === "logs" && (
                <ChannelLogsPanel
                  inboxEvents={inboxEvents}
                  outboxMessages={outboxMessages}
                />
              )}
            </>
          )}
        </div>
      </div>
    </section>
  );
}

function ProviderPanel({
  loading,
  providers
}: {
  loading: boolean;
  providers: ChannelProviderDefinition[];
}) {
  const { t } = useTranslation();
  return (
    <section className="rounded-lg border border-ink-200 bg-white p-4">
      <h2 className="text-base font-semibold text-ink-900">
        {t("channels.providers.title")}
      </h2>
      {loading ? (
        <LoadingState />
      ) : providers.length === 0 ? (
        <p className="mt-3 text-sm text-ink-500">{t("channels.providers.empty")}</p>
      ) : (
        <div className="mt-3 space-y-3">
          {providers.map((provider) => (
            <div className="rounded-md border border-ink-100 p-3" key={provider.id}>
              <div className="flex items-center justify-between gap-3">
                <div>
                  <div className="text-sm font-semibold text-ink-900">
                    {provider.display_name}
                  </div>
                  <div className="mt-1 text-xs text-ink-500">{provider.description}</div>
                </div>
                <Badge>{provider.provider_type}</Badge>
              </div>
              <div className="mt-2 flex flex-wrap gap-2 text-xs text-ink-500">
                <span>{provider.adapter_type}</span>
                <span>{provider.inbound_modes.join(" / ")}</span>
                <span>{provider.outbound_modes.join(" / ")}</span>
              </div>
            </div>
          ))}
        </div>
      )}
    </section>
  );
}

function ConnectionForm({
  editing,
  form,
  loading,
  onChange,
  onReset,
  onSubmit,
  providers,
  selectedProvider
}: {
  editing: boolean;
  form: ConnectionFormState;
  loading: boolean;
  onChange: (form: ConnectionFormState) => void;
  onReset: () => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  providers: ChannelProviderDefinition[];
  selectedProvider?: ChannelProviderDefinition;
}) {
  const { t } = useTranslation();
  const identityFields = identityFieldsForProvider(selectedProvider);
  const endpointFields = endpointFieldsForProvider(selectedProvider);
  const secretFields = secretFieldsForProvider(selectedProvider);
  return (
    <form className="rounded-lg border border-ink-200 bg-white p-4" onSubmit={onSubmit}>
      <h2 className="text-base font-semibold text-ink-900">
        {editing ? t("channels.connection.edit") : t("channels.connection.create")}
      </h2>
      <div className="mt-4 space-y-3">
        <Select
          aria-label={t("channels.connection.provider")}
          onChange={(event) => {
            const provider = providers.find((item) => item.id === event.target.value);
            onChange(withProviderDefaults({ ...form, provider_id: event.target.value }, provider));
          }}
          value={form.provider_id}
        >
          {providers.map((provider) => (
            <option key={provider.id} value={provider.id}>
              {provider.display_name}
            </option>
          ))}
        </Select>
        <Input
          onChange={(event) => onChange({ ...form, display_name: event.target.value })}
          placeholder={t("channels.connection.displayName")}
          required
          value={form.display_name}
        />
        {identityFields.map((field) => {
          const key = field.maps_to ?? field.key;
          const value = key === "external_account_name"
            ? form.external_account_name
            : form.external_account_id;
          return (
            <Input
              key={field.key}
              onChange={(event) =>
                onChange(
                  key === "external_account_name"
                    ? { ...form, external_account_name: event.target.value }
                    : { ...form, external_account_id: event.target.value }
                )
              }
              placeholder={field.placeholder || field.label || field.key}
              value={value}
            />
          );
        })}
        {endpointFields.map((endpoint) => (
          <div className="rounded-md border border-ink-100 bg-ink-50 p-3" key={endpoint.endpoint_type}>
            <label className="text-sm font-medium text-ink-800">
              {endpoint.label || endpoint.endpoint_type}
            </label>
            {endpoint.description && (
              <p className="mt-1 text-xs leading-5 text-ink-500">{endpoint.description}</p>
            )}
            <Input
              className="mt-2"
              onChange={(event) =>
                onChange({
                  ...form,
                  endpoint_values: {
                    ...form.endpoint_values,
                    [endpoint.endpoint_type]: event.target.value
                  }
                })
              }
              placeholder={endpoint.default_url || endpoint.endpoint_type}
              value={form.endpoint_values[endpoint.endpoint_type] ?? ""}
            />
          </div>
        ))}
        {secretFields.map((field) => (
          <SecretInput
            key={field.key}
            onChange={(event) =>
              onChange({
                ...form,
                secret_values: {
                  ...form.secret_values,
                  [field.key]: event.target.value
                }
              })
            }
            placeholder={field.placeholder || field.label || field.key}
            value={form.secret_values[field.key] ?? ""}
          />
        ))}
        <Textarea
          className="h-28 min-h-28 max-h-28 resize-none overflow-auto font-mono text-xs"
          onChange={(event) => onChange({ ...form, config: event.target.value })}
          placeholder={t("channels.connection.advancedConfig")}
          value={form.config}
        />
        <div className="flex flex-wrap gap-2">
          <Button disabled={loading || providers.length === 0} type="submit">
            {loading
              ? t("common.saving")
              : editing
                ? t("channels.connection.save")
                : t("channels.connection.create")}
          </Button>
          {editing && (
            <Button
              icon={<RotateCcw className="h-4 w-4" />}
              onClick={onReset}
              variant="secondary"
            >
              {t("common.cancel")}
            </Button>
          )}
        </div>
      </div>
    </form>
  );
}

function ConnectionList({
  connections,
  loading,
  onCreate,
  onSelect,
  selectedConnectionID
}: {
  connections: PublicChannelConnection[];
  loading: boolean;
  onCreate: () => void;
  onSelect: (connectionID: string) => void;
  selectedConnectionID?: string;
}) {
  const { t } = useTranslation();
  if (loading) {
    return <LoadingState />;
  }
  if (connections.length === 0) {
    return (
      <EmptyState
        description={t("channels.connections.empty.description")}
        icon={<PlugZap className="h-8 w-8" />}
        title={t("channels.connections.empty.title")}
      />
    );
  }
  return (
    <section className="space-y-3">
      <div className="flex items-center justify-between gap-3">
        <h2 className="text-base font-semibold text-ink-900">
          {t("channels.connections.title")}
        </h2>
        <Button className="h-8 px-3" onClick={onCreate} variant="secondary">
          {t("channels.connection.create")}
        </Button>
      </div>
      {connections.map((connection) => (
        <button
          className={[
            "w-full rounded-lg border bg-white p-4 text-left transition",
            connection.id === selectedConnectionID
              ? "border-ocean-300 ring-2 ring-ocean-100"
              : "border-ink-200 hover:border-ink-300"
          ].join(" ")}
          key={connection.id}
          onClick={() => onSelect(connection.id)}
          type="button"
        >
          <div className="flex items-center justify-between gap-3">
            <div className="font-medium text-ink-900">{connection.display_name}</div>
            <Badge>{connection.status}</Badge>
          </div>
          <div className="mt-2 text-xs text-ink-500">
            {connection.external_account_name ||
              connection.external_account_id ||
              t("channels.connection.noExternalAccount")}
          </div>
          <div className="mt-3 flex flex-wrap gap-2 text-xs text-ink-500">
            <span>{connection.has_config ? t("channels.configured") : t("channels.unconfigured")}</span>
            <span>{formatDateTime(connection.created_at)}</span>
          </div>
        </button>
      ))}
    </section>
  );
}

function ConnectionHeader({
  connection,
  onDelete,
  onEdit,
  provider
}: {
  connection: PublicChannelConnection;
  onDelete: (connection: PublicChannelConnection) => void;
  onEdit: (connection: PublicChannelConnection) => void;
  provider?: ChannelProviderDefinition;
}) {
  const { t } = useTranslation();
  return (
    <section className="rounded-lg border border-ink-200 bg-white p-4">
      <div className="flex min-w-0 flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0">
          <h2 className="break-words text-lg font-semibold text-ink-900">{connection.display_name}</h2>
          <p className="mt-1 text-sm text-ink-500">
            {provider?.display_name ?? connection.provider_id}
          </p>
        </div>
        <div className="flex max-w-full flex-wrap justify-start gap-2 sm:justify-end">
          <Badge>{connection.status}</Badge>
          <Badge>{connection.has_config ? t("channels.configured") : t("channels.unconfigured")}</Badge>
          <Button
            className="h-8 px-3"
            icon={<Edit3 className="h-4 w-4" />}
            onClick={() => onEdit(connection)}
            variant="secondary"
          >
            {t("common.edit")}
          </Button>
          <Button
            className="h-8 px-3"
            icon={<Trash2 className="h-4 w-4" />}
            onClick={() => onDelete(connection)}
            variant="danger"
          >
            {t("common.delete")}
          </Button>
        </div>
      </div>
      <div className="mt-4 grid min-w-0 gap-3 text-sm text-ink-600 sm:grid-cols-3">
        <InfoLine label={t("channels.connection.externalAccountId")} value={connection.external_account_id} />
        <InfoLine label={t("channels.connection.externalAccountName")} value={connection.external_account_name} />
        <InfoLine
          label={t("channels.connection.lastEventAt")}
          value={formatOptionalDateTime(connection.last_event_at)}
        />
      </div>
      <div className="mt-4 grid min-w-0 gap-3 lg:grid-cols-3">
        {connection.endpoints.length === 0 ? (
          <p className="text-sm text-ink-500">{t("channels.connection.noEndpoints")}</p>
        ) : (
          connection.endpoints.map((endpoint) => (
            <div className="min-w-0 rounded-md border border-ink-100 p-3" key={endpoint.id}>
              <div className="flex min-w-0 flex-col items-start gap-2">
                <Badge
                  className="max-w-full whitespace-normal break-words text-left leading-5"
                  title={endpoint.endpoint_type}
                >
                  {t(`channels.endpointType.${endpoint.endpoint_type}`, {
                    defaultValue: endpoint.endpoint_type
                  })}
                </Badge>
                <div className="min-w-0 break-words text-sm font-medium leading-6 text-ink-900">
                  {endpoint.display_name}
                </div>
              </div>
              <div className="mt-2 break-all text-xs text-ink-500">{endpoint.url}</div>
              <div className="mt-2 flex min-w-0 flex-wrap gap-2 text-xs text-ink-500">
                <span>{endpoint.direction}</span>
                <span>{endpoint.transport}</span>
                {endpoint.has_secret && <span>{t("channels.connection.hasSecret")}</span>}
              </div>
            </div>
          ))
        )}
      </div>
    </section>
  );
}

function SetupPanel({
  connection,
  onEdit
}: {
  connection: PublicChannelConnection;
  onEdit: (connection: PublicChannelConnection) => void;
}) {
  const { t } = useTranslation();
  return (
    <section className="rounded-lg border border-ink-200 bg-white p-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h2 className="text-base font-semibold text-ink-900">
            {t("channels.sections.setup")}
          </h2>
          <p className="mt-2 text-sm leading-6 text-ink-500">
            {t("channels.connection.setupDescription")}
          </p>
        </div>
        <Button
          icon={<Edit3 className="h-4 w-4" />}
          onClick={() => onEdit(connection)}
          variant="secondary"
        >
          {t("common.edit")}
        </Button>
      </div>
      <div className="mt-4 grid min-w-0 gap-3 sm:grid-cols-2">
        <InfoLine label={t("channels.connection.displayName")} value={connection.display_name} />
        <InfoLine label={t("channels.connection.externalAccountName")} value={connection.external_account_name} />
        <InfoLine label={t("channels.connection.externalAccountId")} value={connection.external_account_id} />
        <InfoLine label={t("channels.connection.lastEventAt")} value={formatOptionalDateTime(connection.last_event_at)} />
      </div>
      <div className="mt-4 space-y-3">
        {connection.endpoints.map((endpoint) => (
          <div
            className="min-w-0 rounded-md border border-ink-100 bg-ink-50 p-3"
            key={endpoint.id}
          >
            <div className="flex min-w-0 flex-wrap items-center gap-2">
              <Badge>{endpoint.endpoint_type}</Badge>
              <span className="min-w-0 break-words text-sm font-medium text-ink-900">
                {endpoint.display_name}
              </span>
            </div>
            <p className="mt-2 break-all text-xs leading-5 text-ink-500">
              {endpoint.url}
            </p>
          </div>
        ))}
      </div>
    </section>
  );
}

function PolicyForm({
  editing,
  form,
  onChange,
  onReset,
  onSubmit,
  saving
}: {
  editing: boolean;
  form: PolicyFormState;
  onChange: (form: PolicyFormState) => void;
  onReset: () => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  saving: boolean;
}) {
  const { t } = useTranslation();
  return (
    <form className="rounded-lg border border-ink-200 bg-white p-4" onSubmit={onSubmit}>
      <h2 className="text-base font-semibold text-ink-900">
        {editing ? t("channels.policy.editTitle") : t("channels.policy.title")}
      </h2>
      <div className="mt-4 space-y-3">
        <Select
          onChange={(event) => onChange({ ...form, scope_type: event.target.value })}
          value={form.scope_type}
        >
          <option value="private_chat">{t("channels.policy.scope.private_chat")}</option>
          <option value="group_chat">{t("channels.policy.scope.group_chat")}</option>
        </Select>
        <Input
          onChange={(event) => onChange({ ...form, external_scope_id: event.target.value })}
          placeholder={t("channels.policy.externalScopeId")}
          value={form.external_scope_id}
        />
        <Select
          onChange={(event) => onChange({ ...form, mode: event.target.value })}
          value={form.mode}
        >
          <option value="auto_reply">{t("channels.policy.mode.auto_reply")}</option>
          <option value="mention_only">{t("channels.policy.mode.mention_only")}</option>
          <option value="keyword">{t("channels.policy.mode.keyword")}</option>
          <option value="off">{t("channels.policy.mode.off")}</option>
        </Select>
        <Input
          onChange={(event) =>
            onChange({ ...form, trigger_keywords: event.target.value })
          }
          placeholder={t("channels.policy.triggerKeywords")}
          value={form.trigger_keywords}
        />
        <Input
          min="0"
          onChange={(event) =>
            onChange({ ...form, rate_limit_per_minute: event.target.value })
          }
          type="number"
          value={form.rate_limit_per_minute}
        />
        <Textarea
          className="min-h-24 font-mono text-xs"
          onChange={(event) =>
            onChange({ ...form, rate_limit_policy: event.target.value })
          }
          value={form.rate_limit_policy}
        />
        <SwitchRow
          checked={form.allow_memory_write}
          label={t("channels.policy.allowMemoryWrite")}
          onToggle={() =>
            onChange({ ...form, allow_memory_write: !form.allow_memory_write })
          }
        />
        <SwitchRow
          checked={form.allow_tool_use}
          label={t("channels.policy.allowToolUse")}
          onToggle={() => onChange({ ...form, allow_tool_use: !form.allow_tool_use })}
        />
        <SwitchRow
          checked={form.require_approval_for_outbound}
          label={t("channels.policy.requireApprovalForOutbound")}
          onToggle={() =>
            onChange({
              ...form,
              require_approval_for_outbound: !form.require_approval_for_outbound
            })
          }
        />
        <div className="flex flex-wrap gap-2">
          <Button disabled={saving} icon={<ShieldCheck className="h-4 w-4" />} type="submit">
            {saving ? t("common.saving") : t("channels.policy.save")}
          </Button>
          {editing && (
            <Button
              icon={<RotateCcw className="h-4 w-4" />}
              onClick={onReset}
              variant="secondary"
            >
              {t("common.cancel")}
            </Button>
          )}
        </div>
      </div>
    </form>
  );
}

function PolicyList({
  deletingID,
  loading,
  onDelete,
  onEdit,
  policies
}: {
  deletingID?: string;
  loading: boolean;
  onDelete: (policy: ChannelPolicy) => void;
  onEdit: (policy: ChannelPolicy) => void;
  policies: ChannelPolicy[];
}) {
  const { t } = useTranslation();
  if (loading) {
    return <LoadingState />;
  }
  return (
    <section className="rounded-lg border border-ink-200 bg-white p-4">
      <h2 className="text-base font-semibold text-ink-900">
        {t("channels.policy.current")}
      </h2>
      {policies.length === 0 ? (
        <p className="mt-3 text-sm text-ink-500">{t("channels.policy.empty")}</p>
      ) : (
        <div className="mt-3 space-y-3">
          {policies.map((policy) => (
            <div
              className="rounded-md border border-ink-100 p-3"
              key={policy.id}
            >
              <div className="flex items-center justify-between gap-3">
                <div className="text-sm font-medium text-ink-900">
                  {t(`channels.policy.scope.${policy.scope_type}`)}
                </div>
                <Badge>{t(`channels.policy.mode.${policy.mode}`)}</Badge>
              </div>
              <div className="mt-2 text-xs text-ink-500">
                {policy.external_scope_id || t("channels.policy.globalScope")}
              </div>
              <div className="mt-2 text-xs text-ink-500">
                {t("channels.policy.rateLimit", {
                  count: policy.rate_limit_per_minute
                })}
              </div>
              <div className="mt-3 flex flex-wrap gap-2">
                <Button
                  className="h-8 px-3"
                  icon={<Edit3 className="h-4 w-4" />}
                  onClick={() => onEdit(policy)}
                  variant="secondary"
                >
                  {t("common.edit")}
                </Button>
                <Button
                  className="h-8 px-3"
                  disabled={deletingID === policy.id}
                  icon={<Trash2 className="h-4 w-4" />}
                  onClick={() => onDelete(policy)}
                  variant="danger"
                >
                  {t("common.delete")}
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}
    </section>
  );
}

function ChannelOverview({
  conversations,
  inboxEvents,
  outboxMessages
}: {
  conversations: ExternalConversation[];
  inboxEvents: ChannelInboxEvent[];
  outboxMessages: ChannelOutboxMessage[];
}) {
  const { t } = useTranslation();
  const latestEvent = inboxEvents[0];
  const latestOutbox = outboxMessages[0];
  return (
    <section className="grid gap-3 md:grid-cols-3">
      <OverviewCard
        label={t("channels.overview.sessions")}
        value={String(conversations.length)}
        detail={conversations[0]?.external_title ?? conversations[0]?.external_conversation_id}
      />
      <OverviewCard
        label={t("channels.overview.latestInbox")}
        value={latestEvent ? formatDateTime(latestEvent.received_at) : "-"}
        detail={summarizeChannelText(latestEvent?.normalized_text)}
      />
      <OverviewCard
        label={t("channels.overview.latestOutbox")}
        value={latestOutbox?.status ?? "-"}
        detail={summarizeChannelText(latestOutbox?.content)}
      />
    </section>
  );
}

function OverviewCard({
  detail,
  label,
  value
}: {
  detail?: string | null;
  label: string;
  value: string;
}) {
  return (
    <div className="min-w-0 rounded-lg border border-ink-200 bg-white p-4">
      <div className="text-xs font-medium uppercase text-ink-500">{label}</div>
      <div className="mt-2 truncate text-lg font-semibold text-ink-900">{value}</div>
      {detail && (
        <div className="mt-2 max-h-20 overflow-hidden break-words text-sm leading-6 text-ink-500">
          {detail}
        </div>
      )}
    </div>
  );
}

function InboxPanel({
  events,
  loading
}: {
  events: ChannelInboxEvent[];
  loading: boolean;
}) {
  const { t } = useTranslation();
  if (loading) {
    return <LoadingState />;
  }
  if (events.length === 0) {
    return (
      <EmptyState
        description={t("channels.inbox.empty.description")}
        icon={<MessageSquareMore className="h-8 w-8" />}
        title={t("channels.inbox.empty.title")}
      />
    );
  }
  return (
    <section className="space-y-3">
      {events.map((event) => (
        <div className="min-w-0 rounded-lg border border-ink-200 bg-white p-4" key={event.id}>
          <div className="flex items-center justify-between gap-3">
            <div className="min-w-0 break-words font-medium text-ink-900">
              {event.external_sender_name || event.external_sender_id || event.event_type}
            </div>
            <Badge>{event.should_trigger_agent ? t("channels.inbox.triggered") : event.status}</Badge>
          </div>
          <p className="mt-3 max-h-64 overflow-auto break-all whitespace-pre-wrap rounded-md bg-ink-50 p-3 text-sm leading-6 text-ink-700">
            {summarizeChannelText(event.normalized_text) || t("channels.inbox.noText")}
          </p>
          <div className="mt-3 flex flex-wrap gap-2 text-xs text-ink-500">
            <span>{formatDateTime(event.received_at)}</span>
            {event.trigger_reason && <span>{event.trigger_reason}</span>}
          </div>
          <details className="mt-3 rounded-md border border-ink-100 bg-white p-3">
            <summary className="cursor-pointer text-xs font-medium text-ink-600">
              {t("channels.inbox.rawPayload")}
            </summary>
            <pre className="mt-2 max-h-72 overflow-auto whitespace-pre-wrap break-all text-xs leading-5 text-ink-600">
              {formatJSON(event.raw_payload)}
            </pre>
          </details>
        </div>
      ))}
    </section>
  );
}

function OutboxPanel({
  loading,
  messages,
  onAction,
  onStatusChange,
  status
}: {
  loading: boolean;
  messages: ChannelOutboxMessage[];
  onAction: (outboxID: string, action: "approve" | "cancel" | "send") => void;
  onStatusChange: (status: string) => void;
  status: string;
}) {
  const { t } = useTranslation();
  return (
    <section className="space-y-3">
      <div className="flex justify-end">
        <Select
          className="w-full sm:w-44"
          onChange={(event) => onStatusChange(event.target.value)}
          value={status}
        >
          <option value="">{t("channels.outbox.status.all")}</option>
          <option value="pending">{t("channels.outbox.status.pending")}</option>
          <option value="approved">{t("channels.outbox.status.approved")}</option>
          <option value="sent">{t("channels.outbox.status.sent")}</option>
          <option value="cancelled">{t("channels.outbox.status.cancelled")}</option>
          <option value="failed">{t("channels.outbox.status.failed")}</option>
        </Select>
      </div>
      {loading ? (
        <LoadingState />
      ) : messages.length === 0 ? (
        <EmptyState
          description={t("channels.outbox.empty.description")}
          icon={<Send className="h-8 w-8" />}
          title={t("channels.outbox.empty.title")}
        />
      ) : (
        messages.map((message) => (
          <div className="min-w-0 rounded-lg border border-ink-200 bg-white p-4" key={message.id}>
            <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div className="min-w-0">
                <div className="flex flex-wrap items-center gap-2">
                  <Badge>{message.status}</Badge>
                  <Badge>{message.message_type}</Badge>
                  {message.requires_approval && <Badge>{t("channels.outbox.requiresApproval")}</Badge>}
                </div>
                <p className="mt-3 max-h-64 overflow-auto break-all whitespace-pre-wrap rounded-md bg-ink-50 p-3 text-sm leading-6 text-ink-700">
                  {summarizeChannelText(message.content)}
                </p>
                {message.error_message && (
                  <p className="mt-2 break-all text-xs text-red-600">{message.error_message}</p>
                )}
                <div className="mt-3 text-xs text-ink-500">
                  {formatDateTime(message.created_at)}
                </div>
              </div>
              <div className="flex flex-wrap gap-2">
                {message.status === "pending" && (
                  <>
                    <Button
                      icon={<Check className="h-4 w-4" />}
                      onClick={() => onAction(message.id, "approve")}
                      variant="secondary"
                    >
                      {t("channels.outbox.approve")}
                    </Button>
                    <Button
                      icon={<X className="h-4 w-4" />}
                      onClick={() => onAction(message.id, "cancel")}
                      variant="danger"
                    >
                      {t("channels.outbox.cancel")}
                    </Button>
                  </>
                )}
                {message.status === "approved" && (
                  <Button
                    icon={<Send className="h-4 w-4" />}
                    onClick={() => onAction(message.id, "send")}
                  >
                    {t("channels.outbox.send")}
                  </Button>
                )}
              </div>
            </div>
          </div>
        ))
      )}
    </section>
  );
}

function ChannelLogsPanel({
  inboxEvents,
  outboxMessages
}: {
  inboxEvents: ChannelInboxEvent[];
  outboxMessages: ChannelOutboxMessage[];
}) {
  const { t } = useTranslation();
  const entries = [
    ...inboxEvents.map((event) => ({
      id: `inbox-${event.id}`,
      at: event.received_at,
      title: event.event_type,
      body: summarizeChannelText(event.normalized_text) || t("channels.inbox.noText"),
      meta: [
        "inbox",
        event.status,
        event.should_trigger_agent ? t("channels.inbox.triggered") : "",
        event.trigger_reason ?? "",
        event.external_sender_name || event.external_sender_id || ""
      ].filter(Boolean)
    })),
    ...outboxMessages.map((message) => ({
      id: `outbox-${message.id}`,
      at: message.created_at,
      title: message.message_type,
      body: summarizeChannelText(message.content),
      meta: [
        "outbox",
        message.status,
        message.requires_approval ? t("channels.outbox.requiresApproval") : "",
        message.error_message ?? ""
      ].filter(Boolean)
    }))
  ].sort((a, b) => new Date(b.at).getTime() - new Date(a.at).getTime());

  if (entries.length === 0) {
    return (
      <EmptyState
        description={t("channels.logs.empty.description")}
        icon={<MessageSquareMore className="h-8 w-8" />}
        title={t("channels.logs.empty.title")}
      />
    );
  }
  return (
    <section className="space-y-3">
      {entries.map((entry) => (
        <div className="min-w-0 rounded-lg border border-ink-200 bg-white p-4" key={entry.id}>
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="font-medium text-ink-900">{entry.title}</div>
            <span className="text-xs text-ink-500">{formatDateTime(entry.at)}</span>
          </div>
          <p className="mt-3 max-h-40 overflow-auto whitespace-pre-wrap break-all rounded-md bg-ink-50 p-3 text-sm leading-6 text-ink-700">
            {entry.body}
          </p>
          <div className="mt-3 flex flex-wrap gap-2">
            {entry.meta.map((item) => (
              <Badge key={item}>{item}</Badge>
            ))}
          </div>
        </div>
      ))}
    </section>
  );
}

function ChannelSessionsPanel({
  conversations,
  draftContent,
  draftSaving,
  loading,
  messages,
  messagesLoading,
  onCreateDraft,
  onDraftChange,
  onSelect,
  selectedExternalConversationID
}: {
  conversations: ExternalConversation[];
  draftContent: string;
  draftSaving: boolean;
  loading: boolean;
  messages: Message[];
  messagesLoading: boolean;
  onCreateDraft: () => void;
  onDraftChange: (value: string) => void;
  onSelect: (externalConversationID: string) => void;
  selectedExternalConversationID?: string;
}) {
  const { t } = useTranslation();
  if (loading) {
    return <LoadingState />;
  }
  if (conversations.length === 0) {
    return (
      <EmptyState
        description={t("channels.conversations.empty.description")}
        icon={<MessageSquareMore className="h-8 w-8" />}
        title={t("channels.conversations.empty.title")}
      />
    );
  }
  const selectedConversation = conversations.find(
    (conversation) => conversation.external_conversation_id === selectedExternalConversationID
  );
  return (
    <section className="grid min-w-0 gap-4 xl:grid-cols-[320px_minmax(0,1fr)]">
      <div className="space-y-3">
        {conversations.map((conversation) => (
          <button
            className={[
              "w-full rounded-lg border bg-white p-4 text-left transition",
              conversation.external_conversation_id === selectedExternalConversationID
                ? "border-ocean-300 ring-2 ring-ocean-100"
                : "border-ink-200 hover:border-ink-300"
            ].join(" ")}
            key={conversation.id}
            onClick={() => onSelect(conversation.external_conversation_id)}
            type="button"
          >
            <div className="flex min-w-0 items-center justify-between gap-3">
              <div className="min-w-0 truncate font-medium text-ink-900">
                {conversation.external_title || conversation.external_conversation_id}
              </div>
              <Badge>{conversation.external_conversation_type}</Badge>
            </div>
            <div className="mt-3 grid gap-2 text-xs text-ink-500">
              <span className="break-all">{conversation.external_conversation_id}</span>
              <span>{formatOptionalDateTime(conversation.last_message_at)}</span>
            </div>
          </button>
        ))}
      </div>
      <ChannelTranscript
        conversation={selectedConversation}
        draftContent={draftContent}
        draftSaving={draftSaving}
        loading={messagesLoading}
        messages={messages}
        onCreateDraft={onCreateDraft}
        onDraftChange={onDraftChange}
      />
    </section>
  );
}

function ChannelTranscript({
  conversation,
  draftContent,
  draftSaving,
  loading,
  messages,
  onCreateDraft,
  onDraftChange
}: {
  conversation?: ExternalConversation;
  draftContent: string;
  draftSaving: boolean;
  loading: boolean;
  messages: Message[];
  onCreateDraft: () => void;
  onDraftChange: (value: string) => void;
}) {
  const { t } = useTranslation();
  if (!conversation) {
    return (
      <EmptyState
        description={t("channels.sessions.selectDescription")}
        icon={<MessageSquareMore className="h-8 w-8" />}
        title={t("channels.sessions.selectTitle")}
      />
    );
  }
  if (loading) {
    return <LoadingState />;
  }
  return (
    <section className="min-w-0 rounded-lg border border-ink-200 bg-white">
      <div className="border-b border-ink-100 p-4">
        <div className="flex flex-wrap items-center gap-2">
          <h2 className="text-base font-semibold text-ink-900">
            {conversation.external_title || conversation.external_conversation_id}
          </h2>
          <Badge>{t("channels.sessions.readonly")}</Badge>
        </div>
        <p className="mt-1 text-sm text-ink-500">
          {t("channels.sessions.readonlyDescription")}
        </p>
      </div>
      <div className="max-h-[560px] space-y-3 overflow-auto p-4">
        {messages.length === 0 ? (
          <p className="text-sm text-ink-500">{t("channels.sessions.noMessages")}</p>
        ) : (
          messages.map((message) => (
            <div
              className="min-w-0 rounded-md border border-ink-100 bg-ink-50 p-3"
              key={message.id}
            >
              <div className="flex flex-wrap items-center gap-2 text-xs text-ink-500">
                <Badge>{message.role}</Badge>
                <span>{channelMessageIdentity(message)}</span>
                <span>{formatDateTime(message.created_at)}</span>
              </div>
              <p className="mt-2 max-h-60 overflow-auto break-all whitespace-pre-wrap text-sm leading-6 text-ink-800">
                {summarizeChannelText(message.content)}
              </p>
            </div>
          ))
        )}
      </div>
      <div className="border-t border-ink-100 p-4">
        <div className="rounded-lg border border-ink-200 bg-ink-50 p-3">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div>
              <h3 className="text-sm font-semibold text-ink-900">
                {t("channels.sessions.outboxDraftTitle")}
              </h3>
              <p className="mt-1 text-xs leading-5 text-ink-500">
                {t("channels.sessions.outboxDraftDescription")}
              </p>
            </div>
            <Badge>{t("channels.sections.outbox")}</Badge>
          </div>
          <Textarea
            className="mt-3 min-h-24"
            onChange={(event) => onDraftChange(event.target.value)}
            placeholder={t("channels.sessions.outboxDraftPlaceholder")}
            value={draftContent}
          />
          <div className="mt-3 flex justify-end">
            <Button
              disabled={!draftContent.trim() || draftSaving}
              icon={<Send className="h-4 w-4" />}
              onClick={onCreateDraft}
              type="button"
            >
              {draftSaving
                ? t("common.saving")
                : t("channels.sessions.createOutboxDraft")}
            </Button>
          </div>
        </div>
      </div>
    </section>
  );
}

function SwitchRow({
  checked,
  label,
  onToggle
}: {
  checked: boolean;
  label: string;
  onToggle: () => void;
}) {
  return (
    <div className="flex min-w-0 items-center justify-between gap-3 rounded-md border border-ink-100 px-3 py-2">
      <span className="min-w-0 break-words text-sm text-ink-700">{label}</span>
      <Switch checked={checked} onClick={onToggle} />
    </div>
  );
}

function InfoLine({ label, value }: { label: string; value?: string | null }) {
  return (
    <div className="min-w-0">
      <div className="break-words text-xs text-ink-500">{label}</div>
      <div className="mt-1 break-all text-sm leading-6 text-ink-800">{value || "-"}</div>
    </div>
  );
}

function parseJSONObject(value: string): Record<string, unknown> {
  if (!value.trim()) {
    return {};
  }
  const parsed = JSON.parse(value) as unknown;
  if (!parsed || Array.isArray(parsed) || typeof parsed !== "object") {
    throw new Error("JSON must be an object");
  }
  return parsed as Record<string, unknown>;
}

function splitKeywords(value: string) {
  return value
    .split(/[,\n]/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function normalizeChannelSection(value?: string): ChannelSection {
  if (value && channelSections.includes(value as ChannelSection)) {
    return value as ChannelSection;
  }
  return "overview";
}

function channelMessageIdentity(message: Message) {
  const metadata = message.metadata ?? {};
  const channel = typeof metadata.channel === "string" ? metadata.channel : "";
  const scopeType = typeof metadata.external_scope_type === "string"
    ? metadata.external_scope_type
    : typeof metadata.external_conversation_type === "string"
      ? metadata.external_conversation_type
      : "";
  const scopeID = typeof metadata.external_scope_id === "string"
    ? metadata.external_scope_id
    : typeof metadata.external_conversation_id === "string"
      ? metadata.external_conversation_id
      : "";
  const senderName = typeof metadata.external_sender_name === "string"
    ? metadata.external_sender_name
    : "";
  const senderID = typeof metadata.external_sender_id === "string"
    ? metadata.external_sender_id
    : "";
  const senderRole = typeof metadata.external_sender_role === "string"
    ? metadata.external_sender_role
    : "";
  const parts = [channel, scopeType, scopeID].filter(Boolean);
  const sender = senderName && senderID
    ? `${senderName}(${senderID})`
    : senderName || senderID;
  if (sender) {
    parts.push(senderRole ? `${sender} / ${senderRole}` : sender);
  }
  if (parts.length > 0) {
    return parts.join(" · ");
  }
  return message.role;
}

function formatJSON(value: unknown) {
  try {
    return JSON.stringify(value ?? {}, null, 2);
  } catch {
    return String(value ?? "");
  }
}

function summarizeChannelText(value?: string | null) {
  if (!value) {
    return "";
  }
  const text = value
    .replace(/&quot;/g, "\"")
    .replace(/&#34;/g, "\"")
    .replace(/&amp;/g, "&")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">");
  if (/^\s*\{/.test(text) || /^\s*\[/.test(text)) {
    try {
      const decoded = JSON.parse(text) as unknown;
      if (decoded && typeof decoded === "object") {
        const record = decoded as Record<string, unknown>;
        const title = valueToText(record.title ?? record.prompt ?? record.desc);
        const app = valueToText(record.app ?? record.app_name ?? record.appid);
        const pieces = [app, title].filter(Boolean);
        if (pieces.length > 0) {
          return `[卡片消息] ${pieces.join(" · ")}`;
        }
        return "[卡片消息]";
      }
    } catch {
      // Fall through to CQ replacements.
    }
  }
  const replacements: Array<[RegExp, string]> = [
    [/\[CQ:image[^\]]*\]/g, "[图片附件]"],
    [/\[CQ:file[^\]]*\]/g, "[文件附件]"],
    [/\[CQ:record[^\]]*\]/g, "[语音附件]"],
    [/\[CQ:video[^\]]*\]/g, "[视频附件]"],
    [/\[CQ:share[^\]]*\]/g, "[分享消息]"],
    [/\[CQ:json[^\]]*\]/g, "[卡片消息]"],
    [/\[CQ:xml[^\]]*\]/g, "[卡片消息]"]
  ];
  return replacements.reduce(
    (current, [pattern, label]) => current.replace(pattern, label),
    text
  );
}

function valueToText(value: unknown) {
  if (typeof value === "string") {
    return value.trim();
  }
  if (typeof value === "number" || typeof value === "boolean") {
    return String(value);
  }
  return "";
}

function formatOptionalDateTime(value?: string | null) {
  return value ? formatDateTime(value) : "-";
}

function connectionToForm(
  connection: PublicChannelConnection,
  provider?: ChannelProviderDefinition
): ConnectionFormState {
  const endpointValues = Object.fromEntries(
    connection.endpoints.map((endpoint) => [endpoint.endpoint_type, endpoint.url])
  );
  return withProviderDefaults({
    provider_id: connection.provider_id,
    display_name: connection.display_name,
    external_account_id: connection.external_account_id ?? "",
    external_account_name: connection.external_account_name ?? "",
    endpoint_values: endpointValues,
    secret_values: {},
    config: "{\n}"
  }, provider);
}

function providerForm(provider?: ChannelProviderDefinition): ChannelProviderFormMetadata {
  const metadata = provider?.metadata ?? {};
  const form = metadata.form;
  if (!form || Array.isArray(form) || typeof form !== "object") {
    return {};
  }
  return form as ChannelProviderFormMetadata;
}

function identityFieldsForProvider(provider?: ChannelProviderDefinition): ChannelProviderFormField[] {
  return providerForm(provider).identity_fields ?? [
    {
      key: "external_account_id",
      label: "External account ID",
      maps_to: "external_account_id"
    },
    {
      key: "external_account_name",
      label: "External account name",
      maps_to: "external_account_name"
    }
  ];
}

function secretFieldsForProvider(provider?: ChannelProviderDefinition): ChannelProviderFormField[] {
  return providerForm(provider).secret_fields ?? [];
}

function endpointFieldsForProvider(provider?: ChannelProviderDefinition): ChannelProviderEndpointField[] {
  return providerForm(provider).endpoint_fields ?? [];
}

function withProviderDefaults(
  form: ConnectionFormState,
  provider?: ChannelProviderDefinition
): ConnectionFormState {
  const endpointValues = { ...form.endpoint_values };
  for (const endpoint of endpointFieldsForProvider(provider)) {
    if (!endpointValues[endpoint.endpoint_type] && endpoint.default_url) {
      endpointValues[endpoint.endpoint_type] = endpoint.default_url;
    }
  }
  return {
    ...form,
    display_name: form.display_name || provider?.display_name || "Channel",
    endpoint_values: endpointValues,
    secret_values: { ...form.secret_values }
  };
}

function buildProviderEndpoints(
  provider: ChannelProviderDefinition | undefined,
  form: ConnectionFormState
) {
  return endpointFieldsForProvider(provider)
    .map((endpoint) => {
      const url = (form.endpoint_values[endpoint.endpoint_type] ?? "").trim();
      const config: Record<string, unknown> = {};
      for (const key of endpoint.secret_keys ?? []) {
        const value = form.secret_values[key]?.trim();
        if (value) {
          config[key] = value;
        }
      }
      return {
        endpoint_type: endpoint.endpoint_type,
        display_name: endpoint.label || endpoint.endpoint_type,
        direction: endpoint.direction || "bidirectional",
        transport: endpoint.transport || "http",
        url,
        config,
        metadata: endpoint.metadata ?? {}
      };
    })
    .filter((endpoint) => endpoint.url !== "");
}

function buildProviderConfig(
  provider: ChannelProviderDefinition | undefined,
  form: ConnectionFormState,
  baseConfig: Record<string, unknown>
) {
  const config = { ...baseConfig };
  for (const binding of providerForm(provider).config_bindings ?? []) {
    const [sourceType, sourceKey] = binding.source.split(":", 2);
    let value = "";
    if (sourceType === "secret") {
      value = form.secret_values[sourceKey]?.trim() ?? "";
    }
    if (sourceType === "identity") {
      value = sourceKey === "external_account_name"
        ? form.external_account_name.trim()
        : form.external_account_id.trim();
    }
    if (value) {
      config[binding.key] = value;
    }
  }
  return config;
}
