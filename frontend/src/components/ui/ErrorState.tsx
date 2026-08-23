import { AlertTriangle } from "lucide-react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";

import { Button } from "./Button";

type ErrorStateProps = {
  title?: ReactNode;
  description: ReactNode;
  action?: ReactNode;
  onRetry?: () => void;
};

export function ErrorState({ action, description, onRetry, title }: ErrorStateProps) {
  const { t } = useTranslation();

  return (
    <div className="flex min-h-64 flex-col items-center justify-center rounded-lg border border-red-200 bg-red-50 px-6 py-10 text-center">
      <AlertTriangle className="h-8 w-8 text-red-600" />
      <h2 className="mt-4 text-base font-semibold text-red-900">
        {title ?? t("common.errorTitle")}
      </h2>
      <p className="mt-2 max-w-md text-sm leading-6 text-red-700">{description}</p>
      {(action || onRetry) && (
        <div className="mt-5">
          {action ?? (
            <Button onClick={onRetry} variant="secondary">
              {t("common.retry")}
            </Button>
          )}
        </div>
      )}
    </div>
  );
}
