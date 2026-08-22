import i18next from "i18next";

export function formatDateTime(value: string | number | Date) {
  return new Intl.DateTimeFormat(i18next.language, {
    dateStyle: "medium",
    timeStyle: "short"
  }).format(new Date(value));
}

export function formatNumber(value: number) {
  return new Intl.NumberFormat(i18next.language).format(value);
}
