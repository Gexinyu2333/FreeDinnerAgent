import type { TextareaHTMLAttributes } from "react";

export function Textarea({
  className = "",
  ...props
}: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return (
    <textarea
      className={[
        "min-h-28 w-full resize-y rounded-md border border-ink-200 bg-white px-3 py-2 text-sm text-ink-900 outline-none transition placeholder:text-ink-500 focus:border-ocean-500 focus:ring-2 focus:ring-ocean-500/15",
        className
      ].join(" ")}
      {...props}
    />
  );
}
