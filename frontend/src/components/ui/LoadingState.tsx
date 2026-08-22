import { Loader2 } from "lucide-react";
import { useTranslation } from "react-i18next";

export function LoadingState() {
  const { t } = useTranslation();

  return (
    <div className="flex items-center gap-2 text-sm text-ink-500">
      <Loader2 className="h-4 w-4 animate-spin" />
      <span>{t("common.loading")}</span>
    </div>
  );
}
