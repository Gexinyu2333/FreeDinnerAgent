import { ArrowRight } from "lucide-react";
import { FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { useMutation } from "@tanstack/react-query";

import { Button } from "../../../components/ui/Button";
import { Input } from "../../../components/ui/Input";
import { setTokens } from "../../../lib/authToken";
import { ApiError } from "../../../lib/errors";
import { queryClient } from "../../../lib/queryClient";
import { register } from "../api";
import { currentUserQueryKey } from "../hooks";

import { AuthLayout } from "./AuthLayout";

export function RegisterPage() {
  const { t } = useTranslation();
  const [username, setUsername] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [password, setPassword] = useState("");

  const registerMutation = useMutation({
    mutationFn: register,
    onSuccess: (result) => {
      setTokens(result.access_token, result.refresh_token);
      queryClient.setQueryData(currentUserQueryKey, result.user);
      window.location.assign("/app/chat");
    }
  });

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    registerMutation.mutate({
      username,
      password,
      display_name: displayName.trim() || undefined
    });
  }

  const errorMessage =
    registerMutation.error instanceof ApiError
      ? registerMutation.error.message
      : registerMutation.error
        ? t("auth.errors.unexpected")
        : null;

  return (
    <AuthLayout description={t("auth.register.description")} title={t("auth.register.title")}>
      <form className="space-y-4" onSubmit={handleSubmit}>
        <label className="block space-y-2">
          <span className="text-sm font-medium text-ink-700">
            {t("auth.fields.username")}
          </span>
          <Input
            autoComplete="username"
            minLength={3}
            onChange={(event) => setUsername(event.target.value)}
            placeholder={t("auth.fields.usernamePlaceholder")}
            required
            value={username}
          />
        </label>

        <label className="block space-y-2">
          <span className="text-sm font-medium text-ink-700">
            {t("auth.fields.displayName")}
          </span>
          <Input
            onChange={(event) => setDisplayName(event.target.value)}
            placeholder={t("auth.fields.displayNamePlaceholder")}
            value={displayName}
          />
        </label>

        <label className="block space-y-2">
          <span className="text-sm font-medium text-ink-700">
            {t("auth.fields.password")}
          </span>
          <Input
            autoComplete="new-password"
            minLength={8}
            onChange={(event) => setPassword(event.target.value)}
            placeholder={t("auth.fields.passwordPlaceholder")}
            required
            type="password"
            value={password}
          />
        </label>

        {errorMessage && (
          <div className="rounded-md border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">
            {errorMessage}
          </div>
        )}

        <Button
          className="w-full"
          disabled={registerMutation.isPending}
          icon={<ArrowRight className="h-4 w-4" />}
          type="submit"
        >
          {registerMutation.isPending
            ? t("auth.register.submitting")
            : t("auth.register.submit")}
        </Button>
      </form>

      <p className="mt-5 text-center text-sm text-ink-500">
        {t("auth.register.hasAccount")}{" "}
        <Link className="font-medium text-ocean-600 hover:text-ocean-500" to="/login">
          {t("auth.register.backToLogin")}
        </Link>
      </p>
    </AuthLayout>
  );
}
