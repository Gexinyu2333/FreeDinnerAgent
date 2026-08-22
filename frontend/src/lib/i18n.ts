import i18next from "i18next";
import { initReactI18next } from "react-i18next";

import enUS from "../locales/en-US.json";
import zhCN from "../locales/zh-CN.json";

export const localeStorageKey = "freedinner.locale";

const savedLocale = window.localStorage.getItem(localeStorageKey);
const initialLocale = savedLocale === "en-US" ? "en-US" : "zh-CN";

i18next.use(initReactI18next).init({
  resources: {
    "zh-CN": { translation: zhCN },
    "en-US": { translation: enUS }
  },
  lng: initialLocale,
  fallbackLng: "zh-CN",
  interpolation: {
    escapeValue: false
  }
});

export async function changeLocale(locale: "zh-CN" | "en-US") {
  window.localStorage.setItem(localeStorageKey, locale);
  document.documentElement.lang = locale;
  await i18next.changeLanguage(locale);
}

document.documentElement.lang = initialLocale;

export default i18next;
