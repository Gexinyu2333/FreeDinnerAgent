import { SendHorizontal } from "lucide-react";
import { FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";

import { Button } from "../../../components/ui/Button";
import { Textarea } from "../../../components/ui/Textarea";

type MessageComposerProps = {
  disabled: boolean;
  isSending: boolean;
  onSend: (content: string) => void;
};

export function MessageComposer({ disabled, isSending, onSend }: MessageComposerProps) {
  const { t } = useTranslation();
  const [content, setContent] = useState("");

  function submit() {
    const trimmed = content.trim();
    if (!trimmed || disabled || isSending) {
      return;
    }
    onSend(trimmed);
    setContent("");
  }

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    submit();
  }

  return (
    <form className="border-t border-ink-200 bg-white p-4" onSubmit={handleSubmit}>
      <div className="mx-auto flex max-w-4xl gap-3">
        <Textarea
          className="min-h-16"
          disabled={disabled || isSending}
          onChange={(event) => setContent(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Enter" && (event.metaKey || event.ctrlKey)) {
              submit();
            }
          }}
          placeholder={
            disabled
              ? t("chat.composer.disabledPlaceholder")
              : t("chat.composer.placeholder")
          }
          value={content}
        />
        <Button
          className="h-16 shrink-0 px-4"
          disabled={disabled || isSending || content.trim() === ""}
          icon={<SendHorizontal className="h-4 w-4" />}
          type="submit"
        >
          {isSending ? t("chat.composer.sending") : t("chat.composer.send")}
        </Button>
      </div>
    </form>
  );
}
