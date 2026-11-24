"use client";

import { useTransition } from "react";
import { chooseLocale } from "./actions";
import { useLocale, useTranslator } from "./client";
import { localeNames, locales } from "./locales";

export function LocaleSwitcher() {
  const current = useLocale();
  const t = useTranslator();
  const [pending, startTransition] = useTransition();
  return (
    <div role="group" aria-label={t("locale.label")} className="flex flex-wrap gap-2">
      {locales.map((locale) => (
        <button
          key={locale}
          type="button"
          lang={locale}
          aria-pressed={locale === current}
          disabled={pending}
          onClick={() => startTransition(() => chooseLocale(locale))}
          className="rounded-md bg-surface px-3 py-1 text-sm text-text-secondary aria-pressed:bg-primary aria-pressed:text-on-primary disabled:opacity-40"
        >
          {localeNames[locale]}
        </button>
      ))}
    </div>
  );
}
