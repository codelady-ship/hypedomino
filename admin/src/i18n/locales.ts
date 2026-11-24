export const locales = ["az", "en"] as const;

export type Locale = (typeof locales)[number];

export const defaultLocale: Locale = "az";

export const localeCookie = "hd_admin_locale";

export const localeNames: Record<Locale, string> = {
  az: "Azərbaycanca",
  en: "English",
};

export function isLocale(value: string | undefined): value is Locale {
  return locales.some((locale) => locale === value);
}

export function preferredLocale(account: string | undefined, cookie: string | undefined): Locale {
  if (isLocale(account)) {
    return account;
  }
  return isLocale(cookie) ? cookie : defaultLocale;
}
