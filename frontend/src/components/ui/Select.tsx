import type { SelectHTMLAttributes } from "react";

export function Select({
  className = "",
  children,
  ...props
}: SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <select
      className={[
        "h-10 rounded-md border border-ink-200 bg-white px-3 text-sm text-ink-900 outline-none transition focus:border-ocean-500 focus:ring-2 focus:ring-ocean-500/15",
        className
      ].join(" ")}
      {...props}
    >
      {children}
    </select>
  );
}
