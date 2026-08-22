import { apiClient } from "../../lib/apiClient";

import type {
  ChannelInboxEvent,
  ChannelOutboxMessage,
  ChannelPolicy,
  ChannelProviderDefinition,
  CreateChannelConnectionInput,
  ExternalConversation,
  PublicChannelConnection,
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

export function listExternalConversations(connectionID: string) {
  return apiClient<ExternalConversation[]>(
    `/me/channel-connections/${connectionID}/external-conversations?limit=50`
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
