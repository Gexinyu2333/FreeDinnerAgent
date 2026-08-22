import { Languages, LogOut, Menu, Search } from "lucide-react";
import { useTranslation } from "react-i18next";

import { Button } from "../ui/Button";
import { Select } from "../ui/Select";
import { clearTokens } from "../../lib/authToken";
import { changeLocale } from "../../lib/i18n";

export function TopBar() {
  const { i18n, t } = useTranslation();

  function handleLogout() {
    clearTokens();
    window.location.assign("/login");
  }

  return (
    <header className="sticky top-0 z-30 flex h-16 items-center justify-between border-b border-ink-200 bg-white/95 px-4 backdrop-blur sm:px-6 lg:px-8">
      <div className="flex min-w-0 items-center gap-3">
        <button
          aria-label={t("layout.openMenu")}
          className="rounded-md p-2 text-ink-500 hover:bg-ink-100 lg:hidden"
          type="button"
        >
          <Menu className="h-5 w-5" />
        </button>
        <div className="hidden h-10 w-80 items-center gap-2 rounded-md border border-ink-200 bg-ink-50 px-3 text-sm text-ink-500 md:flex">
          <Search className="h-4 w-4" />
          <span>{t("layout.searchPlaceholder")}</span>
        </div>
      </div>

      <div className="flex items-center gap-2">
        <label className="flex items-center gap-2 text-sm text-ink-600">
          <Languages className="h-4 w-4" />
          <Select
            aria-label={t("layout.language")}
            className="w-36"
            onChange={(event) =>
              void changeLocale(event.target.value as "zh-CN" | "en-US")
            }
            value={i18n.language === "en-US" ? "en-US" : "zh-CN"}
          >
            <option value="zh-CN">{t("locale.zhCN")}</option>
            <option value="en-US">{t("locale.enUS")}</option>
          </Select>
        </label>
        <Button icon={<LogOut className="h-4 w-4" />} onClick={handleLogout} variant="ghost">
          {t("auth.logout")}
        </Button>
      </div>
    </header>
  );
}
