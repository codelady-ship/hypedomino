"use client";

import { useTranslator } from "@/i18n/client";
import { offerProblems, unlistedCosmetics, type CosmeticItem, type Offer } from "@/shop/catalog";

export function OfferChecks({ offer, items }: { offer: Offer; items: CosmeticItem[] }) {
  const t = useTranslator();
  const unlisted = unlistedCosmetics(offer, items);
  return (
    <>
      {offerProblems(offer).map((problem) => (
        <p key={problem} role="alert" className="text-sm text-danger-text">
          {t(`shop.offer.${problem}`)}
        </p>
      ))}
      {unlisted.length > 0 && (
        <p role="alert" className="text-sm text-danger-text">
          {t("shop.offer.source_missing")}: <span className="font-mono">{unlisted.join(", ")}</span>
        </p>
      )}
    </>
  );
}
