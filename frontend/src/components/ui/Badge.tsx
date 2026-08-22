import type { ReactNode } from "react";

type BadgeTone = "neutral" | "blue" | "green" | "amber";

type BadgeProps = {
  children: ReactNode;
  tone?: BadgeTone;
};

const toneClass: Record<BadgeTone, string> = {
  neutral: "bg-ink-100 text-ink-700",
  blue: "bg-ocean-500/10 text-ocean-600",
  green: "bg-mint-500/10 text-mint-600",
  amber: "bg-amber-500/10 text-amber-500"
};

export function Badge({ children, tone = "neutral" }: BadgeProps) {
  return (
    <span className={`inline-flex rounded px-2 py-1 text-xs font-medium ${toneClass[tone]}`}>
      {children}
    </span>
  );
}
