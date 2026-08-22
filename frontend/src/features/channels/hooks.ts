import { useMutation, useQuery } from "@tanstack/react-query";

import {
  approveOutboxMessage,
  cancelOutboxMessage,
  createChannelConnection,
  listChannelConnections,
  listChannelPolicies,
  listChannelProviders,
  listExternalConversations,
  listInboxEvents,
  listOutboxMessages,
  sendOutboxMessage,
  upsertChannelPolicy
} from "./api";

export const channelProvidersQueryKey = ["channels", "providers"] as const;
export const channelConnectionsQueryKey = ["channels", "connections"] as const;
export const channelPoliciesQueryKey = ["channels", "policies"] as const;
export const channelExternalConversationsQueryKey = ["channels", "external-conversations"] as const;
export const channelInboxQueryKey = ["channels", "inbox"] as const;
export const channelOutboxQueryKey = ["channels", "outbox"] as const;

export function useChannelProviders() {
  return useQuery({
    queryKey: channelProvidersQueryKey,
    queryFn: listChannelProviders
  });
}

export function useChannelConnections() {
  return useQuery({
    queryKey: channelConnectionsQueryKey,
    queryFn: listChannelConnections
  });
}

export function useCreateChannelConnection() {
  return useMutation({
    mutationFn: createChannelConnection
  });
}

export function useChannelPolicies(connectionID?: string) {
  return useQuery({
    enabled: Boolean(connectionID),
    queryKey: [...channelPoliciesQueryKey, connectionID],
    queryFn: () => listChannelPolicies(connectionID as string)
  });
}

export function useUpsertChannelPolicy() {
  return useMutation({
    mutationFn: upsertChannelPolicy
  });
}

export function useExternalConversations(connectionID?: string) {
  return useQuery({
    enabled: Boolean(connectionID),
    queryKey: [...channelExternalConversationsQueryKey, connectionID],
    queryFn: () => listExternalConversations(connectionID as string)
  });
}

export function useInboxEvents(connectionID?: string) {
  return useQuery({
    enabled: Boolean(connectionID),
    queryKey: [...channelInboxQueryKey, connectionID],
    queryFn: () => listInboxEvents(connectionID as string)
  });
}

export function useOutboxMessages(connectionID?: string, status?: string) {
  return useQuery({
    enabled: Boolean(connectionID),
    queryKey: [...channelOutboxQueryKey, connectionID, status || "all"],
    queryFn: () => listOutboxMessages(connectionID as string, status)
  });
}

export function useApproveOutboxMessage() {
  return useMutation({
    mutationFn: approveOutboxMessage
  });
}

export function useCancelOutboxMessage() {
  return useMutation({
    mutationFn: cancelOutboxMessage
  });
}

export function useSendOutboxMessage() {
  return useMutation({
    mutationFn: sendOutboxMessage
  });
}
