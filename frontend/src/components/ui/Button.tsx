import type { ButtonHTMLAttributes, ReactNode } from "react";

type ButtonVariant = "primary" | "secondary" | "ghost" | "danger";

type ButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: ButtonVariant;
  icon?: ReactNode;
};

const variantClass: Record<ButtonVariant, string> = {
  primary: "bg-ocean-600 text-white hover:bg-ocean-500",
  secondary: "border border-ink-200 bg-white text-ink-900 hover:bg-ink-50",
  ghost: "text-ink-700 hover:bg-ink-100",
  danger: "bg-red-600 text-white hover:bg-red-500"
};

export function Button({
  className = "",
  variant = "primary",
  icon,
  children,
  type = "button",
  ...props
}: ButtonProps) {
  return (
    <button
      className={[
        "inline-flex h-10 items-center justify-center gap-2 rounded-md px-4 text-sm font-medium transition disabled:cursor-not-allowed disabled:opacity-60",
        variantClass[variant],
        className
      ].join(" ")}
      type={type}
      {...props}
    >
      {icon}
      {children}
    </button>
  );
}
