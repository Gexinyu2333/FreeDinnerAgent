import { useQuery } from "@tanstack/react-query";

import { getAccessToken } from "../../lib/authToken";

import { getCurrentUser } from "./api";

export const currentUserQueryKey = ["auth", "current-user"] as const;

export function useCurrentUser() {
  return useQuery({
    queryKey: currentUserQueryKey,
    queryFn: getCurrentUser,
    enabled: Boolean(getAccessToken()),
    retry: false
  });
}
