import { Bot, User } from "lucide-react";
import { useTranslation } from "react-i18next";

import { EmptyState } from "../../../components/ui/EmptyState";
import { LoadingState } from "../../../components/ui/LoadingState";
import { formatDateTime } from "../../../lib/format";
import type { Message } from "../types";

type MessageListProps = {
  messages: Message[];
  isLoading: boolean;
  hasConversation: boolean;
};

export function MessageList({ messages, isLoading, hasConversation }: MessageListProps) {
  const { t } = useTranslation();

  if (!hasConversation) {
    return (
      <div className="flex min-h-full items-center justify-center p-6">
        <EmptyState
          description={t("chat.empty.noConversationDescription")}
          icon={<Bot className="h-8 w-8" />}
          title={t("chat.empty.noConversationTitle")}
        />
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="flex min-h-full items-center justify-center">
        <LoadingState />
      </div>
    );
  }

  if (messages.length === 0) {
    return (
      <div className="flex min-h-full items-center justify-center p-6">
        <EmptyState
          description={t("chat.empty.noMessagesDescription")}
          icon={<Bot className="h-8 w-8" />}
          title={t("chat.empty.noMessagesTitle")}
        />
      </div>
    );
  }

  return (
    <div className="mx-auto flex w-full max-w-4xl flex-col gap-4 px-4 py-6">
      {messages.map((message) => (
        <MessageBubble key={message.id} message={message} />
      ))}
    </div>
  );
}

function MessageBubble({ message }: { message: Message }) {
  const isUser = message.role === "user";
  const Icon = isUser ? User : Bot;

  return (
    <article
      className={[
        "flex gap-3",
        isUser ? "justify-end" : "justify-start"
      ].join(" ")}
    >
      {!isUser && (
        <div className="mt-1 flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-ink-900 text-white">
          <Icon className="h-4 w-4" />
        </div>
      )}
      <div
        className={[
          "max-w-[min(760px,85%)] rounded-lg px-4 py-3 shadow-sm",
          isUser
            ? "bg-ocean-600 text-white"
            : "border border-ink-200 bg-white text-ink-900"
        ].join(" ")}
      >
        <div className="whitespace-pre-wrap text-sm leading-6">{message.content}</div>
        <div
          className={[
            "mt-2 text-xs",
            isUser ? "text-white/70" : "text-ink-400"
          ].join(" ")}
        >
          {formatDateTime(message.created_at)}
        </div>
      </div>
      {isUser && (
        <div className="mt-1 flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-ocean-600 text-white">
          <Icon className="h-4 w-4" />
        </div>
      )}
    </article>
  );
}
