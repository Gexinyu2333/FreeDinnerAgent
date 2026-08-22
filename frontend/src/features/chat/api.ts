import { apiClient } from "../../lib/apiClient";

import type {
  Conversation,
  CreateConversationInput,
  Message,
  SendMessageInput,
  SendMessageResult
} from "./types";

export function listConversations() {
  return apiClient<Conversation[]>("/conversations");
}

export function createConversation(input: CreateConversationInput) {
  return apiClient<Conversation>("/conversations", {
    method: "POST",
    body: input
  });
}

export function listMessages(conversationID: string) {
  return apiClient<Message[]>(`/conversations/${conversationID}/messages`);
}

export function sendMessage(conversationID: string, input: SendMessageInput) {
  return apiClient<SendMessageResult>(`/conversations/${conversationID}/messages`, {
    method: "POST",
    body: input
  });
}
