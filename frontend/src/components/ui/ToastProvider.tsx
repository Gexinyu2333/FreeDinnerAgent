import { createContext, PropsWithChildren, useContext, useMemo, useState } from "react";

type ToastTone = "info" | "success" | "error";

type ToastItem = {
  id: string;
  message: string;
  tone: ToastTone;
};

type ToastContextValue = {
  notify: (message: string, tone?: ToastTone) => void;
};

const ToastContext = createContext<ToastContextValue | null>(null);

const toneClass: Record<ToastTone, string> = {
  info: "border-ocean-500 bg-white text-ocean-700",
  success: "border-mint-500 bg-white text-mint-700",
  error: "border-red-500 bg-white text-red-700"
};

export function ToastProvider({ children }: PropsWithChildren) {
  const [items, setItems] = useState<ToastItem[]>([]);

  const value = useMemo<ToastContextValue>(
    () => ({
      notify: (message, tone = "success") => {
        const id =
          typeof crypto !== "undefined" && "randomUUID" in crypto
            ? crypto.randomUUID()
            : `${Date.now()}-${Math.random()}`;
        setItems((current) => [...current, { id, message, tone }].slice(-4));
        window.setTimeout(() => {
          setItems((current) => current.filter((item) => item.id !== id));
        }, 2400);
      }
    }),
    []
  );

  return (
    <ToastContext.Provider value={value}>
      {children}
      <div className="pointer-events-none fixed bottom-5 right-5 z-50 flex w-[min(360px,calc(100vw-40px))] flex-col gap-2">
        {items.map((item) => (
          <div
            className={`rounded-lg border px-4 py-3 text-sm font-medium shadow-soft ${toneClass[item.tone]}`}
            key={item.id}
          >
            {item.message}
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
}

export function useToast() {
  const context = useContext(ToastContext);
  if (!context) {
    throw new Error("useToast must be used within ToastProvider");
  }
  return context;
}
