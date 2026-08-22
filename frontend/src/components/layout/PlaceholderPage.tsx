import { Boxes } from "lucide-react";
import { useTranslation } from "react-i18next";

import { Badge } from "../ui/Badge";
import { EmptyState } from "../ui/EmptyState";

type PlaceholderPageProps = {
  pageKey:
    | "chat"
    | "chatDetail"
    | "agent"
    | "providers"
    | "memory"
    | "knowledge"
    | "market"
    | "tools"
    | "tasks"
    | "channels"
    | "channelDetail"
    | "workspace"
    | "logs";
};

export function PlaceholderPage({ pageKey }: PlaceholderPageProps) {
  const { t } = useTranslation();

  return (
    <section className="space-y-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <Badge tone="blue">{t("common.stepF1")}</Badge>
          <h1 className="mt-3 text-2xl font-semibold tracking-normal text-ink-900">
            {t(`pages.${pageKey}.title`)}
          </h1>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-ink-500">
            {t(`pages.${pageKey}.description`)}
          </p>
        </div>
      </div>

      <EmptyState
        description={t("common.pageWillBeImplemented")}
        icon={<Boxes className="h-8 w-8" />}
        title={t("common.mvpShellReady")}
      />
    </section>
  );
}
