import { useMutation, useQuery } from "@tanstack/react-query";

import {
  createConversation,
  listConversations,
  listMessages,
  sendMessage
} from "./api";

export const conversationsQueryKey = ["chat", "conversations"] as const;

export function messagesQueryKey(conversationID: string) {
  return ["chat", "messages", conversationID] as const;
}

export function useConversations() {
  return useQuery({
    queryKey: conversationsQueryKey,
    queryFn: listConversations
  });
}

export function useMessages(conversationID?: string) {
  return useQuery({
    queryKey: conversationID
      ? messagesQueryKey(conversationID)
      : ["chat", "messages", "none"],
    queryFn: () => listMessages(conversationID!),
    enabled: Boolean(conversationID)
  });
}

export function useCreateConversation() {
  return useMutation({
    mutationFn: createConversation
  });
}

export function useSendMessage(conversationID: string) {
  return useMutation({
    mutationFn: (content: string) => sendMessage(conversationID, { content })
  });
}
