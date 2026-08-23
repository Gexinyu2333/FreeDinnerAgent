import { Eye, EyeOff } from "lucide-react";
import type { InputHTMLAttributes } from "react";
import { useState } from "react";
import { useTranslation } from "react-i18next";

import { Input } from "./Input";

type SecretInputProps = Omit<InputHTMLAttributes<HTMLInputElement>, "type">;

export function SecretInput({ className = "", ...props }: SecretInputProps) {
  const { t } = useTranslation();
  const [visible, setVisible] = useState(false);
  const Icon = visible ? EyeOff : Eye;

  return (
    <div className="relative">
      <Input
        className={`pr-11 ${className}`}
        type={visible ? "text" : "password"}
        {...props}
      />
      <button
        aria-label={visible ? t("common.hideSecret") : t("common.showSecret")}
        className="absolute right-2 top-1/2 inline-flex h-8 w-8 -translate-y-1/2 items-center justify-center rounded-md text-ink-500 transition hover:bg-ink-100 hover:text-ink-900"
        onClick={() => setVisible((current) => !current)}
        type="button"
      >
        <Icon className="h-4 w-4" />
      </button>
    </div>
  );
}
