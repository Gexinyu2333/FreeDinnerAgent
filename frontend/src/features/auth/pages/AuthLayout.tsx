import type { ReactNode } from "react";
import { Bot } from "lucide-react";
import { useTranslation } from "react-i18next";

type AuthLayoutProps = {
  title: ReactNode;
  description: ReactNode;
  children: ReactNode;
};

export function AuthLayout({ title, description, children }: AuthLayoutProps) {
  const { t } = useTranslation();

  return (
    <main className="grid min-h-screen bg-ink-50 lg:grid-cols-[1fr_520px]">
      <section className="hidden border-r border-ink-200 bg-white px-12 py-10 lg:flex lg:flex-col lg:justify-between">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-ink-900 text-white">
            <Bot className="h-5 w-5" />
          </div>
          <div>
            <div className="text-sm font-semibold text-ink-900">FreeDinnerAgent</div>
            <div className="text-xs text-ink-500">{t("app.subtitle")}</div>
          </div>
        </div>

        <div className="max-w-xl">
          <p className="text-sm font-medium uppercase tracking-wide text-ocean-600">
            {t("auth.heroEyebrow")}
          </p>
          <h1 className="mt-4 text-4xl font-semibold tracking-normal text-ink-900">
            {t("auth.heroTitle")}
          </h1>
          <p className="mt-5 text-base leading-7 text-ink-500">
            {t("auth.heroDescription")}
          </p>
        </div>

        <p className="text-xs text-ink-500">{t("auth.localFirstHint")}</p>
      </section>

      <section className="flex items-center justify-center px-5 py-10">
        <div className="w-full max-w-md">
          <div className="mb-8 lg:hidden">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-ink-900 text-white">
                <Bot className="h-5 w-5" />
              </div>
              <div>
                <div className="text-sm font-semibold text-ink-900">FreeDinnerAgent</div>
                <div className="text-xs text-ink-500">{t("app.subtitle")}</div>
              </div>
            </div>
          </div>

          <div className="rounded-lg border border-ink-200 bg-white p-6 shadow-soft">
            <h1 className="text-2xl font-semibold tracking-normal text-ink-900">
              {title}
            </h1>
            <p className="mt-2 text-sm leading-6 text-ink-500">{description}</p>
            <div className="mt-6">{children}</div>
          </div>
        </div>
      </section>
    </main>
  );
}
