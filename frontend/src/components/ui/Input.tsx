import type { InputHTMLAttributes } from "react";

export function Input({ className = "", ...props }: InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      className={[
        "h-10 w-full rounded-md border border-ink-200 bg-white px-3 text-sm text-ink-900 outline-none transition placeholder:text-ink-500 focus:border-ocean-500 focus:ring-2 focus:ring-ocean-500/15",
        className
      ].join(" ")}
      {...props}
    />
  );
}
