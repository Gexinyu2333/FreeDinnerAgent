import { useQueryClient } from "@tanstack/react-query";
import { AlertCircle } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useNavigate, useParams } from "react-router-dom";

import { Toast } from "../../../components/ui/Toast";
import { ApiError } from "../../../lib/errors";
import { MessageComposer } from "../components/MessageComposer";
import { MessageList } from "../components/MessageList";
import { ConversationList } from "../components/ConversationList";
import {
  conversationsQueryKey,
  messagesQueryKey,
  useConversations,
  useCreateConversation,
  useMessages,
  useSendMessage
} from "../hooks";
import type { Message } from "../types";

export function ChatPage() {
  const { conversationId } = useParams();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { t } = useTranslation();

  const conversationsQuery = useConversations();
  const messagesQuery = useMessages(conversationId);
  const createConversationMutation = useCreateConversation();
  const sendMessageMutation = useSendMessage(conversationId ?? "");

  function handleCreate(title: string) {
    createConversationMutation.mutate(
      { title: title || t("chat.sidebar.defaultTitle") },
      {
        onSuccess: (conversation) => {
          void queryClient.invalidateQueries({ queryKey: conversationsQueryKey });
          navigate(`/app/chat/${conversation.id}`);
        }
      }
    );
  }

  function handleSend(content: string) {
    if (!conversationId) {
      return;
    }
    sendMessageMutation.mutate(content, {
      onSuccess: (result) => {
        queryClient.setQueryData<Message[]>(
          messagesQueryKey(conversationId),
          (previous = []) => [
            ...previous,
            result.user_message,
            result.assistant_message
          ]
        );
        void queryClient.invalidateQueries({
          queryKey: messagesQueryKey(conversationId)
        });
        void queryClient.invalidateQueries({ queryKey: conversationsQueryKey });
      }
    });
  }

  const sendError =
    sendMessageMutation.error instanceof ApiError
      ? sendMessageMutation.error.message
      : sendMessageMutation.error
        ? t("chat.errors.sendFailed")
        : null;

  const createError =
    createConversationMutation.error instanceof ApiError
      ? createConversationMutation.error.message
      : createConversationMutation.error
        ? t("chat.errors.createFailed")
        : null;

  return (
    <section className="grid h-[calc(100vh-6.5rem)] min-h-[620px] overflow-hidden rounded-lg border border-ink-200 bg-ink-50 shadow-soft lg:grid-cols-[320px_1fr]">
      <ConversationList
        activeConversationID={conversationId}
        conversations={conversationsQuery.data ?? []}
        isCreating={createConversationMutation.isPending}
        isLoading={conversationsQuery.isLoading}
        onCreate={handleCreate}
      />

      <div className="flex min-w-0 min-h-0 flex-col">
        <header className="flex h-16 shrink-0 items-center justify-between border-b border-ink-200 bg-white px-5">
          <div className="min-w-0">
            <h1 className="truncate text-lg font-semibold text-ink-900">
              {currentTitle(conversationsQuery.data ?? [], conversationId) ??
                t("chat.header.noConversation")}
            </h1>
            <p className="text-sm text-ink-500">
              {conversationId
                ? t("chat.header.activeDescription")
                : t("chat.header.emptyDescription")}
            </p>
          </div>
        </header>

        {(sendError || createError || conversationsQuery.isError || messagesQuery.isError) && (
          <div className="border-b border-ink-200 bg-white px-5 py-3">
            <Toast
              message={
                sendError ??
                createError ??
                (conversationsQuery.error instanceof ApiError
                  ? conversationsQuery.error.message
                  : messagesQuery.error instanceof ApiError
                    ? messagesQuery.error.message
                    : t("chat.errors.loadFailed"))
              }
              tone="error"
            />
          </div>
        )}

        {sendMessageMutation.isPending && (
          <div className="flex items-center gap-2 border-b border-ink-200 bg-ocean-500/5 px-5 py-2 text-sm text-ocean-600">
            <AlertCircle className="h-4 w-4" />
            <span>{t("chat.header.waitingForAssistant")}</span>
          </div>
        )}

        <div className="min-h-0 flex-1 overflow-y-auto">
          <MessageList
            hasConversation={Boolean(conversationId)}
            isLoading={messagesQuery.isLoading}
            messages={messagesQuery.data ?? []}
          />
        </div>

        <MessageComposer
          disabled={!conversationId}
          isSending={sendMessageMutation.isPending}
          onSend={handleSend}
        />
      </div>
    </section>
  );
}

function currentTitle(conversations: { id: string; title: string }[], id?: string) {
  return conversations.find((conversation) => conversation.id === id)?.title;
}
