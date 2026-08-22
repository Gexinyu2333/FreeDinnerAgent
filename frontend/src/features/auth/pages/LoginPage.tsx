import { ArrowRight } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";

import { Button } from "../../../components/ui/Button";
import { Input } from "../../../components/ui/Input";

import { AuthLayout } from "./AuthLayout";

export function LoginPage() {
  const { t } = useTranslation();

  return (
    <AuthLayout description={t("auth.login.description")} title={t("auth.login.title")}>
      <form className="space-y-4">
        <label className="block space-y-2">
          <span className="text-sm font-medium text-ink-700">
            {t("auth.fields.username")}
          </span>
          <Input autoComplete="username" placeholder={t("auth.fields.usernamePlaceholder")} />
        </label>

        <label className="block space-y-2">
          <span className="text-sm font-medium text-ink-700">
            {t("auth.fields.password")}
          </span>
          <Input
            autoComplete="current-password"
            placeholder={t("auth.fields.passwordPlaceholder")}
            type="password"
          />
        </label>

        <Button className="w-full" icon={<ArrowRight className="h-4 w-4" />} type="submit">
          {t("auth.login.submit")}
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
