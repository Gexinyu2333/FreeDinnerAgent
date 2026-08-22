import { ArrowRight } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";

import { Button } from "../../../components/ui/Button";
import { Input } from "../../../components/ui/Input";

import { AuthLayout } from "./AuthLayout";

export function RegisterPage() {
  const { t } = useTranslation();

  return (
    <AuthLayout description={t("auth.register.description")} title={t("auth.register.title")}>
      <form className="space-y-4">
        <label className="block space-y-2">
          <span className="text-sm font-medium text-ink-700">
            {t("auth.fields.username")}
          </span>
          <Input autoComplete="username" placeholder={t("auth.fields.usernamePlaceholder")} />
        </label>

        <label className="block space-y-2">
          <span className="text-sm font-medium text-ink-700">
            {t("auth.fields.displayName")}
          </span>
          <Input placeholder={t("auth.fields.displayNamePlaceholder")} />
        </label>

        <label className="block space-y-2">
          <span className="text-sm font-medium text-ink-700">
            {t("auth.fields.password")}
          </span>
          <Input
            autoComplete="new-password"
            placeholder={t("auth.fields.passwordPlaceholder")}
            type="password"
          />
        </label>

        <Button className="w-full" icon={<ArrowRight className="h-4 w-4" />} type="submit">
          {t("auth.register.submit")}
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
