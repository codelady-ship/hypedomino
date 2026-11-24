"use client";

import { useTranslator } from "@/i18n/client";
import type { MessageKey } from "@/i18n/translate";
import type { BundleForm } from "@/rewards/bundles";

export const fieldClass = "rounded-md bg-surface px-3 py-1 text-text-primary";
export const labelClass = "flex flex-col gap-1 text-sm text-text-secondary";

type Props = { form: BundleForm; onChange: (form: BundleForm) => void; issues: MessageKey[] };

export function BundleFields({ form, onChange, issues }: Props) {
  const t = useTranslator();
  const number = (value: string) => (value === "" ? Number.NaN : Number(value));
  const shown = (value: number) => (Number.isNaN(value) ? "" : value);
  return (
    <>
      <div className="flex flex-wrap gap-3">
        <label className={labelClass}>
          {t("rewards.coins")}
          <input type="number" min={0} value={shown(form.coins)} onChange={(event) => onChange({ ...form, coins: number(event.target.value) })} className={fieldClass} />
        </label>
        <label className={labelClass}>
          {t("rewards.crystals")}
          <input type="number" min={0} value={shown(form.crystals)} onChange={(event) => onChange({ ...form, crystals: number(event.target.value) })} className={fieldClass} />
        </label>
        <label className={labelClass}>
          {t("rewards.pass_xp")}
          <input type="number" min={0} value={shown(form.passXp)} onChange={(event) => onChange({ ...form, passXp: number(event.target.value) })} className={fieldClass} />
        </label>
      </div>
      <label className={labelClass}>
        {t("rewards.cosmetics")}
        <input value={form.cosmetics} onChange={(event) => onChange({ ...form, cosmetics: event.target.value })} className={`${fieldClass} font-mono`} />
      </label>
      <label className={labelClass}>
        {t("rewards.medals")}
        <input value={form.medals} onChange={(event) => onChange({ ...form, medals: event.target.value })} className={`${fieldClass} font-mono`} />
      </label>
      {issues.length > 0 && (
        <ul role="alert" className="flex flex-col gap-1 text-sm text-danger-text">
          {issues.map((issue) => (
            <li key={issue}>{t(issue)}</li>
          ))}
        </ul>
      )}
    </>
  );
}
