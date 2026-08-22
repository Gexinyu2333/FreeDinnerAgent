import type { ButtonHTMLAttributes } from "react";

type SwitchProps = Omit<ButtonHTMLAttributes<HTMLButtonElement>, "value"> & {
  checked: boolean;
};

export function Switch({ checked, className = "", ...props }: SwitchProps) {
  return (
    <button
      aria-checked={checked}
      className={[
        "relative h-6 w-11 rounded-full transition",
        checked ? "bg-ocean-600" : "bg-ink-200",
        className
      ].join(" ")}
      role="switch"
      type="button"
      {...props}
    >
      <span
        className={[
          "absolute top-1 h-4 w-4 rounded-full bg-white transition",
          checked ? "left-6" : "left-1"
        ].join(" ")}
      />
    </button>
  );
}
