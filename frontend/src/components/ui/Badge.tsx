import type { HTMLAttributes, ReactNode } from "react";

type BadgeTone = "neutral" | "blue" | "green" | "amber";

type BadgeProps = HTMLAttributes<HTMLSpanElement> & {
  children: ReactNode;
  tone?: BadgeTone;
};

const toneClass: Record<BadgeTone, string> = {
  neutral: "bg-ink-100 text-ink-700",
  blue: "bg-ocean-500/10 text-ocean-600",
  green: "bg-mint-500/10 text-mint-600",
  amber: "bg-amber-500/10 text-amber-500"
};

export function Badge({ children, className = "", tone = "neutral", ...props }: BadgeProps) {
  return (
    <span
      className={`inline-flex max-w-full shrink-0 whitespace-normal break-words rounded px-2 py-1 text-left text-xs font-medium leading-5 ${toneClass[tone]} ${className}`}
      {...props}
    >
      {children}
    </span>
  );
}
