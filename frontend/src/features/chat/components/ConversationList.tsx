import { MessageSquarePlus, Plus } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { NavLink } from "react-router-dom";

import { Button } from "../../../components/ui/Button";
import { Input } from "../../../components/ui/Input";
import { LoadingState } from "../../../components/ui/LoadingState";
import { formatDateTime } from "../../../lib/format";
import type { Conversation } from "../types";

type ConversationListProps = {
  conversations: Conversation[];
  isLoading: boolean;
  activeConversationID?: string;
  onCreate: (title: string) => void;
  isCreating: boolean;
};

export function ConversationList({
  conversations,
  isLoading,
  activeConversationID,
  onCreate,
  isCreating
}: ConversationListProps) {
  const { t } = useTranslation();
  const [title, setTitle] = useState("");

  function handleCreate() {
    onCreate(title.trim());
    setTitle("");
  }

  return (
    <aside className="flex min-h-0 flex-col border-r border-ink-200 bg-white">
      <div className="border-b border-ink-100 p-4">
        <h2 className="text-sm font-semibold text-ink-900">{t("chat.sidebar.title")}</h2>
        <div className="mt-3 flex gap-2">
          <Input
            aria-label={t("chat.sidebar.newConversationTitle")}
            onChange={(event) => setTitle(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === "Enter") {
                handleCreate();
              }
            }}
            placeholder={t("chat.sidebar.newConversationPlaceholder")}
            value={title}
          />
          <Button
            aria-label={t("chat.sidebar.create")}
            className="shrink-0 px-3"
            disabled={isCreating}
            icon={<Plus className="h-4 w-4" />}
            onClick={handleCreate}
          />
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto p-2">
        {isLoading ? (
          <div className="p-3">
            <LoadingState />
          </div>
        ) : conversations.length === 0 ? (
          <div className="flex flex-col items-center justify-center px-5 py-12 text-center">
            <MessageSquarePlus className="h-8 w-8 text-ink-300" />
            <p className="mt-3 text-sm font-medium text-ink-700">
              {t("chat.sidebar.emptyTitle")}
            </p>
            <p className="mt-1 text-xs leading-5 text-ink-500">
              {t("chat.sidebar.emptyDescription")}
            </p>
          </div>
        ) : (
          <div className="space-y-1">
            {conversations.map((conversation) => (
              <NavLink
                className={[
                  "block rounded-md px-3 py-2 transition",
                  activeConversationID === conversation.id
                    ? "bg-ink-900 text-white"
                    : "text-ink-700 hover:bg-ink-100"
                ].join(" ")}
                key={conversation.id}
                to={`/app/chat/${conversation.id}`}
              >
                <div className="truncate text-sm font-medium">{conversation.title}</div>
                <div
                  className={[
                    "mt-1 truncate text-xs",
                    activeConversationID === conversation.id
                      ? "text-white/70"
                      : "text-ink-500"
                  ].join(" ")}
                >
                  {formatDateTime(conversation.updated_at)}
                </div>
              </NavLink>
            ))}
          </div>
        )}
      </div>
    </aside>
  );
}
