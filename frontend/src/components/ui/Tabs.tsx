import type { ReactNode } from "react";

type TabItem = {
  key: string;
  label: ReactNode;
};

type TabsProps = {
  items: TabItem[];
  activeKey: string;
  onChange: (key: string) => void;
};

export function Tabs({ items, activeKey, onChange }: TabsProps) {
  return (
    <div className="flex max-w-full flex-wrap gap-1 rounded-md border border-ink-200 bg-white p-1">
      {items.map((item) => (
        <button
          className={[
            "max-w-full rounded px-3 py-1.5 text-sm font-medium leading-5 transition",
            item.key === activeKey
              ? "bg-ink-900 text-white"
              : "text-ink-600 hover:bg-ink-100"
          ].join(" ")}
          key={item.key}
          onClick={() => onChange(item.key)}
          type="button"
        >
          {item.label}
        </button>
      ))}
    </div>
  );
}
