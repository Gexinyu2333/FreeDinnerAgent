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
    <div className="inline-flex rounded-md border border-ink-200 bg-white p-1">
      {items.map((item) => (
        <button
          className={[
            "rounded px-3 py-1.5 text-sm font-medium transition",
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
