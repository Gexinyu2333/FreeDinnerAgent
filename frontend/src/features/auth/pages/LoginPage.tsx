import { ArrowRight } from "lucide-react";
import { FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { useMutation } from "@tanstack/react-query";

import { Button } from "../../../components/ui/Button";
import { Field } from "../../../components/ui/Field";
import { Input } from "../../../components/ui/Input";
import { Toast } from "../../../components/ui/Toast";
import { setTokens } from "../../../lib/authToken";
import { ApiError } from "../../../lib/errors";
import { queryClient } from "../../../lib/queryClient";
import { currentUserQueryKey } from "../hooks";
import { login } from "../api";

import { AuthLayout } from "./AuthLayout";

export function LoginPage() {
  const { t } = useTranslation();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");

  const loginMutation = useMutation({
    mutationFn: login,
    onSuccess: (result) => {
      setTokens(result.access_token, result.refresh_token);
      queryClient.setQueryData(currentUserQueryKey, result.user);
      window.location.assign("/app/chat");
    }
  });

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    loginMutation.mutate({ username, password });
  }

  const errorMessage =
    loginMutation.error instanceof ApiError
      ? loginMutation.error.message
      : loginMutation.error
        ? t("auth.errors.unexpected")
        : null;

  return (
    <AuthLayout description={t("auth.login.description")} title={t("auth.login.title")}>
      <form className="space-y-4" onSubmit={handleSubmit}>
        <Field label={t("auth.fields.username")}>
          <Input
            autoComplete="username"
            onChange={(event) => setUsername(event.target.value)}
            placeholder={t("auth.fields.usernamePlaceholder")}
            required
            value={username}
          />
        </Field>

        <Field label={t("auth.fields.password")}>
          <Input
            autoComplete="current-password"
            minLength={8}
            onChange={(event) => setPassword(event.target.value)}
            placeholder={t("auth.fields.passwordPlaceholder")}
            required
            type="password"
            value={password}
          />
        </Field>

        {errorMessage && <Toast message={errorMessage} tone="error" />}

        <Button
          className="w-full"
          disabled={loginMutation.isPending}
          icon={<ArrowRight className="h-4 w-4" />}
          type="submit"
        >
          {loginMutation.isPending ? t("auth.login.submitting") : t("auth.login.submit")}
        </Button>
      </form>

      <p className="mt-5 text-center text-sm text-ink-500">
        {t("auth.login.noAccount")}{" "}
        <Link className="font-medium text-ocean-600 hover:text-ocean-500" to="/register">
          {t("auth.login.createAccount")}
        </Link>
      </p>
    </AuthLayout>
  );
}
