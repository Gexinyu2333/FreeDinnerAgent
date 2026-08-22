import { apiClient } from "../../lib/apiClient";

import type { AuthResult, LoginInput, PublicUser, RegisterInput } from "./types";

export function login(input: LoginInput) {
  return apiClient<AuthResult>("/auth/login", {
    method: "POST",
    body: input,
    redirectOnUnauthorized: false
  });
}

export function register(input: RegisterInput) {
  return apiClient<AuthResult>("/auth/register", {
    method: "POST",
    body: input,
    redirectOnUnauthorized: false
  });
}

export function getCurrentUser() {
  return apiClient<PublicUser>("/me");
}
