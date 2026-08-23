import { QueryClientProvider } from "@tanstack/react-query";
import type { PropsWithChildren } from "react";

import { ToastProvider } from "../components/ui/ToastProvider";
import { queryClient } from "../lib/queryClient";

export function AppProviders({ children }: PropsWithChildren) {
  return (
    <QueryClientProvider client={queryClient}>
      <ToastProvider>{children}</ToastProvider>
    </QueryClientProvider>
  );
}
