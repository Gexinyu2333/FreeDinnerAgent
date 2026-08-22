import type { ReactNode } from "react";

type EmptyStateProps = {
  icon?: ReactNode;
  title: ReactNode;
  description: ReactNode;
  action?: ReactNode;
};

export function EmptyState({ icon, title, description, action }: EmptyStateProps) {
  return (
    <div className="flex min-h-64 flex-col items-center justify-center rounded-lg border border-dashed border-ink-200 bg-white px-6 py-10 text-center">
      {icon && <div className="mb-4 text-ocean-600">{icon}</div>}
      <h2 className="text-base font-semibold text-ink-900">{title}</h2>
      <p className="mt-2 max-w-md text-sm leading-6 text-ink-500">{description}</p>
      {action && <div className="mt-5">{action}</div>}
    </div>
  );
}
