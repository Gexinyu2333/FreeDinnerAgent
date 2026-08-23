import type { ReactNode } from "react";

type FieldProps = {
  label: ReactNode;
  children: ReactNode;
  className?: string;
  hint?: ReactNode;
  error?: ReactNode;
};

export function Field({ children, className = "", error, hint, label }: FieldProps) {
  return (
    <label className={`block space-y-1.5 ${className}`}>
      <span className="block text-sm font-medium text-ink-700">{label}</span>
      {children}
      {hint && !error && <span className="block text-xs leading-5 text-ink-500">{hint}</span>}
      {error && <span className="block text-xs leading-5 text-red-600">{error}</span>}
    </label>
  );
}
