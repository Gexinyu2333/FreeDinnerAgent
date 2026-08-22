import { useQueryClient } from "@tanstack/react-query";
import {
  Check,
  MessageSquareMore,
  PlugZap,
  RefreshCw,
  Send,
  ShieldCheck,
  X
} from "lucide-react";
import { FormEvent, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate, useParams } from "react-router-dom";

import { Badge } from "../../../components/ui/Badge";
import { Button } from "../../../components/ui/Button";
import { EmptyState } from "../../../components/ui/EmptyState";
import { Input } from "../../../components/ui/Input";
import { LoadingState } from "../../../components/ui/LoadingState";
import { Select } from "../../../components/ui/Select";
import { Switch } from "../../../components/ui/Switch";
import { Tabs } from "../../../components/ui/Tabs";
import { Textarea } from "../../../components/ui/Textarea";
import { Toast } from "../../../components/ui/Toast";
import { ApiError } from "../../../lib/errors";
import { formatDateTime } from "../../../lib/format";
import {
  channelConnectionsQueryKey,
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
  useExternalConversations,
  useInboxEvents,
  useOutboxMessages,
  useSendOutboxMessage,
  useUpsertChannelPolicy
} from "../hooks";
import type {
  ChannelInboxEvent,
  ChannelOutboxMessage,
  ChannelPolicy,
  ChannelProviderDefinition,
  ExternalConversation,
  PublicChannelConnection
} from "../types";

type ConnectionFormState = {
  provider_id: string;
  display_name: string;
  external_account_id: string;
  external_account_name: string;
  message_api_url: string;
  event_stream_url: string;
  webhook_callback_url: string;
  access_token: string;
  webhook_secret: string;
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

const defaultConnectionForm: ConnectionFormState = {
  provider_id: "",
  display_name: "NapCatQQ",
  external_account_id: "",
  external_account_name: "",
  message_api_url: "http://127.0.0.1:3000",
  event_stream_url: "http://127.0.0.1:3000/sse",
  webhook_callback_url: "http://127.0.0.1:8080/api/v1/channels/<connection_id>/webhook",
  access_token: "",
  webhook_secret: "",
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
  const queryClient = useQueryClient();
  const navigate = useNavigate();
  const params = useParams();
  const providersQuery = useChannelProviders();
  const connectionsQuery = useChannelConnections();
  const createMutation = useCreateChannelConnection();
  const upsertPolicyMutation = useUpsertChannelPolicy();
  const approveMutation = useApproveOutboxMessage();
  const cancelMutation = useCancelOutboxMessage();
  const sendMutation = useSendOutboxMessage();
  const [connectionForm, setConnectionForm] =
    useState<ConnectionFormState>(defaultConnectionForm);
  const [policyForm, setPolicyForm] = useState<PolicyFormState>(defaultPolicyForm);
  const [outboxStatus, setOutboxStatus] = useState("pending");
  const [activeTab, setActiveTab] = useState("inbox");
  const [formError, setFormError] = useState<string | null>(null);

  const providers = providersQuery.data ?? [];
  const connections = connectionsQuery.data ?? [];
  const selectedConnectionID = params.connectionId || connections[0]?.id;
  const selectedConnection = connections.find((item) => item.id === selectedConnectionID);
  const policiesQuery = useChannelPolicies(selectedConnectionID);
  const externalConversationsQuery = useExternalConversations(selectedConnectionID);
  const inboxQuery = useInboxEvents(selectedConnectionID);
  const outboxQuery = useOutboxMessages(selectedConnectionID, outboxStatus || undefined);
  const policies = policiesQuery.data ?? [];
  const externalConversations = externalConversationsQuery.data ?? [];
  const inboxEvents = inboxQuery.data ?? [];
  const outboxMessages = outboxQuery.data ?? [];
  const mutationError =
    createMutation.error ??
    upsertPolicyMutation.error ??
    approveMutation.error ??
    cancelMutation.error ??
    sendMutation.error ??
    null;

  useEffect(() => {
    const firstProvider = providers[0];
    if (!connectionForm.provider_id && firstProvider) {
      setConnectionForm((current) => ({ ...current, provider_id: firstProvider.id }));
    }
  }, [connectionForm.provider_id, providers]);

  useEffect(() => {
    if (!params.connectionId && connections[0]) {
      navigate(`/app/channels/${connections[0].id}`, { replace: true });
    }
  }, [connections, navigate, params.connectionId]);

  const selectedProvider = useMemo(
    () => providers.find((provider) => provider.id === selectedConnection?.provider_id),
    [providers, selectedConnection]
  );

  function refresh(connectionID = selectedConnectionID) {
    void queryClient.invalidateQueries({ queryKey: channelConnectionsQueryKey });
    if (connectionID) {
      void queryClient.invalidateQueries({ queryKey: channelPoliciesQueryKey });
      void queryClient.invalidateQueries({ queryKey: channelExternalConversationsQueryKey });
      void queryClient.invalidateQueries({ queryKey: channelInboxQueryKey });
      void queryClient.invalidateQueries({ queryKey: channelOutboxQueryKey });
    }
  }

  function handleCreateConnection(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setFormError(null);
    let config: Record<string, unknown>;
    try {
      config = parseJSONObject(connectionForm.config);
    } catch {
      setFormError(t("channels.errors.invalidJson"));
      return;
    }
    createMutation.mutate(
      {
        provider_id: connectionForm.provider_id,
        display_name: connectionForm.display_name.trim(),
        external_account_id: connectionForm.external_account_id.trim() || null,
        external_account_name: connectionForm.external_account_name.trim() || null,
        endpoints: buildNapCatEndpoints(connectionForm),
        config: {
          ...config,
          access_token: connectionForm.access_token.trim(),
          webhook_secret: connectionForm.webhook_secret.trim(),
          bot_qq: connectionForm.external_account_id.trim()
        }
      },
      {
        onSuccess: (connection) => {
          setConnectionForm(defaultConnectionForm);
          navigate(`/app/channels/${connection.id}`);
          refresh(connection.id);
        }
      }
    );
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
        onSuccess: () => refresh(selectedConnectionID)
      }
    );
  }

  function loadPolicy(policy: ChannelPolicy) {
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
      onSuccess: () => refresh(selectedConnectionID)
    });
  }

  return (
    <section className="space-y-5">
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
      {createMutation.isSuccess && (
        <Toast message={t("channels.connection.created")} tone="success" />
      )}
      {upsertPolicyMutation.isSuccess && (
        <Toast message={t("channels.policy.saved")} tone="success" />
      )}

      <div className="grid gap-5 xl:grid-cols-[400px_minmax(0,1fr)]">
        <div className="space-y-4">
          <ProviderPanel loading={providersQuery.isLoading} providers={providers} />
          <ConnectionForm
            form={connectionForm}
            loading={createMutation.isPending}
            onChange={setConnectionForm}
            onSubmit={handleCreateConnection}
            providers={providers}
          />
          <ConnectionList
            connections={connections}
            loading={connectionsQuery.isLoading}
            onSelect={(connectionID) => navigate(`/app/channels/${connectionID}`)}
            selectedConnectionID={selectedConnectionID}
          />
        </div>

        <div className="space-y-4">
          {!selectedConnection ? (
            <EmptyState
              description={t("channels.detail.empty.description")}
              icon={<PlugZap className="h-8 w-8" />}
              title={t("channels.detail.empty.title")}
            />
          ) : (
            <>
              <ConnectionHeader
                connection={selectedConnection}
                provider={selectedProvider}
              />
              <div className="grid gap-4 2xl:grid-cols-[360px_minmax(0,1fr)]">
                <div className="space-y-4">
                  <PolicyForm
                    form={policyForm}
                    onChange={setPolicyForm}
                    onSubmit={handleUpsertPolicy}
                    saving={upsertPolicyMutation.isPending}
                  />
                  <PolicyList
                    loading={policiesQuery.isLoading}
                    onEdit={loadPolicy}
                    policies={policies}
                  />
                </div>

                <div className="space-y-4">
                  <Tabs
                    activeKey={activeTab}
                    items={[
                      { key: "inbox", label: t("channels.tabs.inbox") },
                      { key: "outbox", label: t("channels.tabs.outbox") },
                      { key: "conversations", label: t("channels.tabs.conversations") }
                    ]}
                    onChange={setActiveTab}
                  />
                  {activeTab === "inbox" && (
                    <InboxPanel
                      events={inboxEvents}
                      loading={inboxQuery.isLoading}
                    />
                  )}
                  {activeTab === "outbox" && (
                    <OutboxPanel
                      loading={outboxQuery.isLoading}
                      messages={outboxMessages}
                      onAction={mutateOutbox}
                      onStatusChange={setOutboxStatus}
                      status={outboxStatus}
                    />
                  )}
                  {activeTab === "conversations" && (
                    <ExternalConversationsPanel
                      conversations={externalConversations}
                      loading={externalConversationsQuery.isLoading}
                    />
                  )}
                </div>
              </div>
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
  form,
  loading,
  onChange,
  onSubmit,
  providers
}: {
  form: ConnectionFormState;
  loading: boolean;
  onChange: (form: ConnectionFormState) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  providers: ChannelProviderDefinition[];
}) {
  const { t } = useTranslation();
  return (
    <form className="rounded-lg border border-ink-200 bg-white p-4" onSubmit={onSubmit}>
      <h2 className="text-base font-semibold text-ink-900">
        {t("channels.connection.create")}
      </h2>
      <div className="mt-4 space-y-3">
        <Select
          aria-label={t("channels.connection.provider")}
          onChange={(event) => onChange({ ...form, provider_id: event.target.value })}
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
        <Input
          onChange={(event) =>
            onChange({ ...form, external_account_id: event.target.value })
          }
          placeholder={t("channels.connection.externalAccountId")}
          value={form.external_account_id}
        />
        <Input
          onChange={(event) =>
            onChange({ ...form, external_account_name: event.target.value })
          }
          placeholder={t("channels.connection.externalAccountName")}
          value={form.external_account_name}
        />
        <Input
          onChange={(event) =>
            onChange({ ...form, message_api_url: event.target.value })
          }
          placeholder={t("channels.connection.messageApiUrl")}
          value={form.message_api_url}
        />
        <Input
          onChange={(event) =>
            onChange({ ...form, event_stream_url: event.target.value })
          }
          placeholder={t("channels.connection.eventStreamUrl")}
          value={form.event_stream_url}
        />
        <Input
          onChange={(event) =>
            onChange({ ...form, webhook_callback_url: event.target.value })
          }
          placeholder={t("channels.connection.webhookCallbackUrl")}
          value={form.webhook_callback_url}
        />
        <Input
          onChange={(event) => onChange({ ...form, access_token: event.target.value })}
          placeholder={t("channels.connection.accessToken")}
          type="password"
          value={form.access_token}
        />
        <Input
          onChange={(event) =>
            onChange({ ...form, webhook_secret: event.target.value })
          }
          placeholder={t("channels.connection.webhookSecret")}
          type="password"
          value={form.webhook_secret}
        />
        <Textarea
          className="min-h-24 font-mono text-xs"
          onChange={(event) => onChange({ ...form, config: event.target.value })}
          placeholder={t("channels.connection.advancedConfig")}
          value={form.config}
        />
        <Button disabled={loading || providers.length === 0} type="submit">
          {loading ? t("common.saving") : t("channels.connection.create")}
        </Button>
      </div>
    </form>
  );
}

function ConnectionList({
  connections,
  loading,
  onSelect,
  selectedConnectionID
}: {
  connections: PublicChannelConnection[];
  loading: boolean;
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
      <h2 className="text-base font-semibold text-ink-900">
        {t("channels.connections.title")}
      </h2>
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
  provider
}: {
  connection: PublicChannelConnection;
  provider?: ChannelProviderDefinition;
}) {
  const { t } = useTranslation();
  return (
    <section className="rounded-lg border border-ink-200 bg-white p-4">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div>
          <h2 className="text-lg font-semibold text-ink-900">{connection.display_name}</h2>
          <p className="mt-1 text-sm text-ink-500">
            {provider?.display_name ?? connection.provider_id}
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          <Badge>{connection.status}</Badge>
          <Badge>{connection.has_config ? t("channels.configured") : t("channels.unconfigured")}</Badge>
        </div>
      </div>
      <div className="mt-4 grid gap-3 text-sm text-ink-600 sm:grid-cols-3">
        <InfoLine label={t("channels.connection.externalAccountId")} value={connection.external_account_id} />
        <InfoLine label={t("channels.connection.externalAccountName")} value={connection.external_account_name} />
        <InfoLine
          label={t("channels.connection.lastEventAt")}
          value={formatOptionalDateTime(connection.last_event_at)}
        />
      </div>
      <div className="mt-4 grid gap-3 md:grid-cols-3">
        {connection.endpoints.length === 0 ? (
          <p className="text-sm text-ink-500">{t("channels.connection.noEndpoints")}</p>
        ) : (
          connection.endpoints.map((endpoint) => (
            <div className="rounded-md border border-ink-100 p-3" key={endpoint.id}>
              <div className="flex items-center justify-between gap-2">
                <div className="text-sm font-medium text-ink-900">
                  {endpoint.display_name}
                </div>
                <Badge>{endpoint.endpoint_type}</Badge>
              </div>
              <div className="mt-2 break-all text-xs text-ink-500">{endpoint.url}</div>
              <div className="mt-2 flex flex-wrap gap-2 text-xs text-ink-500">
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

function PolicyForm({
  form,
  onChange,
  onSubmit,
  saving
}: {
  form: PolicyFormState;
  onChange: (form: PolicyFormState) => void;
  onSubmit: (event: FormEvent<HTMLFormElement>) => void;
  saving: boolean;
}) {
  const { t } = useTranslation();
  return (
    <form className="rounded-lg border border-ink-200 bg-white p-4" onSubmit={onSubmit}>
      <h2 className="text-base font-semibold text-ink-900">{t("channels.policy.title")}</h2>
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
        <Button disabled={saving} icon={<ShieldCheck className="h-4 w-4" />} type="submit">
          {saving ? t("common.saving") : t("channels.policy.save")}
        </Button>
      </div>
    </form>
  );
}

function PolicyList({
  loading,
  onEdit,
  policies
}: {
  loading: boolean;
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
            <button
              className="w-full rounded-md border border-ink-100 p-3 text-left hover:border-ocean-200"
              key={policy.id}
              onClick={() => onEdit(policy)}
              type="button"
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
            </button>
          ))}
        </div>
      )}
    </section>
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
        <div className="rounded-lg border border-ink-200 bg-white p-4" key={event.id}>
          <div className="flex items-center justify-between gap-3">
            <div className="font-medium text-ink-900">
              {event.external_sender_name || event.external_sender_id || event.event_type}
            </div>
            <Badge>{event.should_trigger_agent ? t("channels.inbox.triggered") : event.status}</Badge>
          </div>
          <p className="mt-3 whitespace-pre-wrap text-sm leading-6 text-ink-700">
            {event.normalized_text || t("channels.inbox.noText")}
          </p>
          <div className="mt-3 flex flex-wrap gap-2 text-xs text-ink-500">
            <span>{formatDateTime(event.received_at)}</span>
            {event.trigger_reason && <span>{event.trigger_reason}</span>}
          </div>
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
          <div className="rounded-lg border border-ink-200 bg-white p-4" key={message.id}>
            <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <div className="flex flex-wrap items-center gap-2">
                  <Badge>{message.status}</Badge>
                  <Badge>{message.message_type}</Badge>
                  {message.requires_approval && <Badge>{t("channels.outbox.requiresApproval")}</Badge>}
                </div>
                <p className="mt-3 whitespace-pre-wrap text-sm leading-6 text-ink-700">
                  {message.content}
                </p>
                {message.error_message && (
                  <p className="mt-2 text-xs text-red-600">{message.error_message}</p>
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

function ExternalConversationsPanel({
  conversations,
  loading
}: {
  conversations: ExternalConversation[];
  loading: boolean;
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
  return (
    <section className="space-y-3">
      {conversations.map((conversation) => (
        <div className="rounded-lg border border-ink-200 bg-white p-4" key={conversation.id}>
          <div className="flex items-center justify-between gap-3">
            <div className="font-medium text-ink-900">
              {conversation.external_title || conversation.external_conversation_id}
            </div>
            <Badge>{conversation.external_conversation_type}</Badge>
          </div>
          <div className="mt-3 grid gap-2 text-xs text-ink-500 sm:grid-cols-2">
            <span>{conversation.external_conversation_id}</span>
            <span>{formatOptionalDateTime(conversation.last_message_at)}</span>
          </div>
        </div>
      ))}
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
    <div className="flex items-center justify-between gap-3 rounded-md border border-ink-100 px-3 py-2">
      <span className="text-sm text-ink-700">{label}</span>
      <Switch checked={checked} onClick={onToggle} />
    </div>
  );
}

function InfoLine({ label, value }: { label: string; value?: string | null }) {
  return (
    <div>
      <div className="text-xs text-ink-500">{label}</div>
      <div className="mt-1 truncate text-sm text-ink-800">{value || "-"}</div>
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

function formatOptionalDateTime(value?: string | null) {
  return value ? formatDateTime(value) : "-";
}

function buildNapCatEndpoints(form: ConnectionFormState) {
  const accessToken = form.access_token.trim();
  const webhookSecret = form.webhook_secret.trim();
  const endpoints = [
    {
      endpoint_type: "message_api",
      display_name: "NapCat HTTP API",
      direction: "outbound",
      transport: "http",
      url: form.message_api_url.trim(),
      config: accessToken ? { access_token: accessToken } : {},
      metadata: { capability: "send_msg" }
    },
    {
      endpoint_type: "event_stream",
      display_name: "NapCat HTTP SSE",
      direction: "inbound",
      transport: "http_sse",
      url: form.event_stream_url.trim(),
      config: accessToken ? { access_token: accessToken } : {},
      metadata: { capability: "listen_events" }
    },
    {
      endpoint_type: "webhook_callback",
      display_name: "FreeDinnerAgent Webhook",
      direction: "inbound",
      transport: "http",
      url: form.webhook_callback_url.trim(),
      config: webhookSecret ? { webhook_secret: webhookSecret } : {},
      metadata: { capability: "receive_events" }
    }
  ];
  return endpoints.filter((endpoint) => endpoint.url !== "");
}
