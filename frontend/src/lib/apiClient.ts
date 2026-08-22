import { clearTokens, getAccessToken } from "./authToken";
import { ApiError } from "./errors";

export type ApiResponse<T> = {
  data: T | null;
  error: null | {
    code: string;
    message: string;
  };
};

type RequestOptions = Omit<RequestInit, "body"> & {
  body?: unknown;
};

export async function apiClient<T>(
  path: string,
  options: RequestOptions = {}
): Promise<T> {
  const headers = new Headers(options.headers);
  headers.set("Content-Type", "application/json");

  const accessToken = getAccessToken();
  if (accessToken) {
    headers.set("Authorization", `Bearer ${accessToken}`);
  }

  const response = await fetch(`/api/v1${path}`, {
    ...options,
    headers,
    body:
      options.body === undefined ? undefined : JSON.stringify(options.body)
  });

  let payload: ApiResponse<T>;
  try {
    payload = (await response.json()) as ApiResponse<T>;
  } catch {
    payload = {
      data: null,
      error: {
        code: "INVALID_RESPONSE",
        message: response.statusText || "Invalid response"
      }
    };
  }

  if (response.status === 401) {
    clearTokens();
    window.location.assign("/login");
  }

  if (!response.ok || payload.error) {
    throw new ApiError(
      response.status,
      payload.error?.code ?? "REQUEST_FAILED",
      payload.error?.message ?? response.statusText
    );
  }

  return payload.data as T;
}
