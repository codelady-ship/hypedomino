"use client";

import { useTranslator } from "@/i18n/client";
import type { MessageKey } from "@/i18n/translate";
import { termsProblems, withEligibility, withLimitPerAccount, type Offer, type TermsProblem } from "@/shop/catalog";

const problemText: Record<TermsProblem, MessageKey> = {
  cosmetic_limit_one: "shop.terms_cosmetic_limit_one",
  chest_without_terms: "shop.terms_chest_without_terms",
  reference_not_above_price: "shop.terms_reference_not_above_price",
};

const fieldClass = "rounded-md bg-surface px-3 py-1 text-text-primary";

export function OfferTerms({ offer, onChange }: { offer: Offer; onChange: (offer: Offer) => void }) {
  const t = useTranslator();
  const hours = offer.eligibility?.maxAccountAgeHours ?? 0;
  const firstPurchaseOnly = offer.eligibility?.firstPurchaseOnly ?? false;
  return (
    <fieldset className="flex flex-col gap-2">
      <legend className="text-sm text-text-secondary">{t("shop.terms")}</legend>
      <div className="flex flex-wrap items-end gap-3">
        <label className="flex flex-col gap-1 text-sm text-text-secondary">
          {t("shop.limit_per_account")}
          <input
            type="number"
            min={1}
            step={1}
            value={offer.limitPerAccount ?? ""}
            onChange={(event) => onChange(withLimitPerAccount(offer, Number(event.target.value)))}
            className={fieldClass}
          />
        </label>
        <label className="flex flex-col gap-1 text-sm text-text-secondary">
          {t("shop.max_account_age_hours")}
          <input
            type="number"
            min={1}
            step={1}
            value={hours || ""}
            onChange={(event) => onChange(withEligibility(offer, Number(event.target.value), firstPurchaseOnly))}
            className={fieldClass}
          />
        </label>
        <label className="flex items-center gap-2 text-sm text-text-secondary">
          <input type="checkbox" checked={firstPurchaseOnly} onChange={(event) => onChange(withEligibility(offer, hours, event.target.checked))} />
          {t("shop.first_purchase_only")}
        </label>
      </div>
      {termsProblems(offer).map((problem) => (
        <p key={problem} role="alert" className="text-sm text-danger-text">
          {t(problemText[problem])}
        </p>
      ))}
    </fieldset>
  );
}
