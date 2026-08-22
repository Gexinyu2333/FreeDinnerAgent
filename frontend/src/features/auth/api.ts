import { apiClient } from "../../lib/apiClient";

import type { AuthResult, LoginInput, RegisterInput } from "./types";

export function login(input: LoginInput) {
  return apiClient<AuthResult>("/auth/login", {
    method: "POST",
    body: input
  });
}

export function register(input: RegisterInput) {
  return apiClient<AuthResult>("/auth/register", {
    method: "POST",
    body: input
  });
}
