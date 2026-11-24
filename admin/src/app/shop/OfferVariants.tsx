"use client";

import { useState } from "react";
import { useTranslator } from "@/i18n/client";
import type { MessageKey } from "@/i18n/translate";
import { editedVariant, variantProblems, withVariants, type Offer, type Variant, type VariantProblem } from "@/shop/catalog";

const problemText: Record<VariantProblem, MessageKey> = {
  variant_id: "shop.variant_problem_id",
  variant_twice: "shop.variant_problem_twice",
  variant_position: "shop.variant_problem_position",
  variant_no_change: "shop.variant_problem_no_change",
};

const fieldClass = "rounded-md bg-surface px-3 py-1 text-text-primary";
const labelClass = "flex flex-col gap-1 text-sm text-text-secondary";

export function OfferVariants({ offer, onChange }: { offer: Offer; onChange: (offer: Offer) => void }) {
  const t = useTranslator();
  const [id, setId] = useState("");
  const variants = offer.variants ?? [];
  const replace = (at: number, next: Variant) => onChange(withVariants(offer, variants.map((variant, index) => (index === at ? next : variant))));
  const position = (value: string) => (value === "" ? 0 : Number(value));
  return (
    <fieldset className="flex flex-col gap-2">
      <legend className="text-sm text-text-secondary">{t("shop.variants")}</legend>
      <p className="text-sm text-text-muted">{t("shop.variants_hint")}</p>
      {variants.map((variant, at) => (
        <div key={`${variant.id}:${at}`} className="flex flex-wrap items-end gap-3">
          <span className="font-mono text-sm">{variant.id}</span>
          <label className={labelClass}>
            {t("shop.variant_name_key")}
            <input value={variant.nameKey ?? ""} onChange={(event) => replace(at, editedVariant(variant, event.target.value, variant.position ?? 0))} className={`${fieldClass} font-mono`} />
          </label>
          <label className={labelClass}>
            {t("shop.variant_position")}
            <input type="number" min={1} step={1} value={variant.position ?? ""} onChange={(event) => replace(at, editedVariant(variant, variant.nameKey ?? "", position(event.target.value)))} className={fieldClass} />
          </label>
          <button type="button" onClick={() => onChange(withVariants(offer, variants.filter((_, index) => index !== at)))} className="rounded-md bg-surface px-3 py-1 text-sm text-text-secondary">
            {t("shop.variant_remove")}
          </button>
        </div>
      ))}
      <div className="flex flex-wrap items-end gap-3">
        <label className={labelClass}>
          {t("shop.variant_id")}
          <input value={id} onChange={(event) => setId(event.target.value.trim())} className={`${fieldClass} font-mono`} />
        </label>
        <button
          type="button"
          disabled={id === ""}
          onClick={() => {
            onChange(withVariants(offer, [...variants, { id }]));
            setId("");
          }}
          className="rounded-md bg-surface px-3 py-1 text-sm text-text-secondary disabled:opacity-40"
        >
          {t("shop.variant_add")}
        </button>
      </div>
      {variantProblems(offer).map((problem) => (
        <p key={problem} role="alert" className="text-sm text-danger-text">
          {t(problemText[problem])}
        </p>
      ))}
    </fieldset>
  );
}
