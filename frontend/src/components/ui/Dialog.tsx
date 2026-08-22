import type { ReactNode } from "react";
import { X } from "lucide-react";
import { useTranslation } from "react-i18next";

type DialogProps = {
  open: boolean;
  title: ReactNode;
  children: ReactNode;
  onClose: () => void;
};

export function Dialog({ open, title, children, onClose }: DialogProps) {
  const { t } = useTranslation();

  if (!open) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-ink-900/30 p-4">
      <section className="w-full max-w-lg rounded-lg bg-white shadow-soft">
        <header className="flex items-center justify-between border-b border-ink-100 px-5 py-4">
          <h2 className="text-base font-semibold text-ink-900">{title}</h2>
          <button
            aria-label={t("common.close")}
            className="rounded-md p-2 text-ink-500 hover:bg-ink-100"
            onClick={onClose}
            type="button"
          >
            <X className="h-4 w-4" />
          </button>
        </header>
        <div className="px-5 py-4">{children}</div>
      </section>
    </div>
  );
}
