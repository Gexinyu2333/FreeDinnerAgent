import { apiClient } from "../../lib/apiClient";
import type { Message } from "../chat/types";

import type {
  ChannelInboxEvent,
  ChannelOutboxMessage,
  ChannelPolicy,
  ChannelProviderDefinition,
  CreateChannelOutboxDraftInput,
  CreateChannelConnectionInput,
  ExternalConversation,
  PublicChannelConnection,
  UpdateChannelConnectionInput,
  UpsertChannelPolicyInput
} from "./types";

export function listChannelProviders() {
  return apiClient<ChannelProviderDefinition[]>("/channel-providers");
}

export function createChannelConnection(input: CreateChannelConnectionInput) {
  return apiClient<PublicChannelConnection>("/me/channel-connections", {
    method: "POST",
    body: input
  });
}

export function updateChannelConnection(input: UpdateChannelConnectionInput) {
  const { connection_id, ...body } = input;
  return apiClient<PublicChannelConnection>(`/me/channel-connections/${connection_id}`, {
    method: "PATCH",
    body
  });
}

export function deleteChannelConnection(connectionID: string) {
  return apiClient<{ deleted: boolean }>(`/me/channel-connections/${connectionID}`, {
    method: "DELETE"
  });
}

export function listChannelConnections() {
  return apiClient<PublicChannelConnection[]>("/me/channel-connections");
}

export function listChannelPolicies(connectionID: string) {
  return apiClient<ChannelPolicy[]>(`/me/channel-connections/${connectionID}/policies`);
}

export function upsertChannelPolicy(input: UpsertChannelPolicyInput) {
  const { connection_id, ...body } = input;
  return apiClient<ChannelPolicy>(`/me/channel-connections/${connection_id}/policies`, {
    method: "PATCH",
    body
  });
}

export function deleteChannelPolicy(input: { connection_id: string; policy_id: string }) {
  return apiClient<{ deleted: boolean }>(
    `/me/channel-connections/${input.connection_id}/policies/${input.policy_id}`,
    {
      method: "DELETE"
    }
  );
}

export function listExternalConversations(connectionID: string) {
  return apiClient<ExternalConversation[]>(
    `/me/channel-connections/${connectionID}/external-conversations?limit=50`
  );
}

export function listExternalConversationMessages(
  connectionID: string,
  externalConversationID: string
) {
  return apiClient<Message[]>(
    `/me/channel-connections/${connectionID}/external-conversations/${externalConversationID}/messages`
  );
}

export function createOutboxDraft(input: CreateChannelOutboxDraftInput) {
  const { connection_id, external_conversation_id, ...body } = input;
  return apiClient<ChannelOutboxMessage>(
    `/me/channel-connections/${connection_id}/external-conversations/${external_conversation_id}/outbox-drafts`,
    {
      method: "POST",
      body
    }
  );
}

export function listInboxEvents(connectionID: string) {
  return apiClient<ChannelInboxEvent[]>(
    `/me/channel-connections/${connectionID}/inbox-events?limit=50`
  );
}

export function listOutboxMessages(connectionID: string, status?: string) {
  const params = new URLSearchParams({ limit: "50" });
  if (status) {
    params.set("status", status);
  }
  return apiClient<ChannelOutboxMessage[]>(
    `/me/channel-connections/${connectionID}/outbox-messages?${params.toString()}`
  );
}

export function approveOutboxMessage(outboxID: string) {
  return apiClient<ChannelOutboxMessage>(`/channel-outbox-messages/${outboxID}/approve`, {
    method: "POST"
  });
}

export function cancelOutboxMessage(outboxID: string) {
  return apiClient<ChannelOutboxMessage>(`/channel-outbox-messages/${outboxID}/cancel`, {
    method: "POST"
  });
}

export function sendOutboxMessage(outboxID: string) {
  return apiClient<ChannelOutboxMessage>(`/channel-outbox-messages/${outboxID}/send`, {
    method: "POST"
  });
}
