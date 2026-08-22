type ToastProps = {
  message: string;
  tone?: "info" | "success" | "error";
};

const toneClass = {
  info: "border-ocean-500 bg-ocean-500/10 text-ocean-600",
  success: "border-mint-500 bg-mint-500/10 text-mint-600",
  error: "border-red-500 bg-red-500/10 text-red-700"
};

export function Toast({ message, tone = "info" }: ToastProps) {
  return (
    <div className={`rounded-md border px-4 py-3 text-sm ${toneClass[tone]}`}>
      {message}
    </div>
  );
}
