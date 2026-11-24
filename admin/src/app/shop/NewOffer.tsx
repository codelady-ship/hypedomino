"use client";

import { useState, useTransition } from "react";
import { useTranslator } from "@/i18n/client";
import { draftProblems, lookupProblems, newOffer, offerKinds, sectionsFor, type LookupProblem, type NewOfferDraft, type Offer, type ShopCatalog } from "@/shop/catalog";
import { findRewardBundle, nameKeyApproval } from "./actions";

const fieldClass = "rounded-md bg-surface px-3 py-1 text-text-primary";

const emptyDraft: NewOfferDraft = { id: "", kind: "", section: "", bundleId: "", nameKey: "" };

export function NewOffer({ catalog, onAdd }: { catalog: ShopCatalog; onAdd: (offer: Offer) => void }) {
  const t = useTranslator();
  const [draft, setDraft] = useState(emptyDraft);
  const [refused, setRefused] = useState<LookupProblem[]>([]);
  const [pending, startTransition] = useTransition();
  const problems = draftProblems(draft, catalog);
  const edit = (fields: Partial<NewOfferDraft>) => {
    setDraft((current) => ({ ...current, ...fields }));
    setRefused([]);
  };
  const add = () =>
    startTransition(async () => {
      const bundleId = draft.bundleId.trim();
      const [bundle, name] = await Promise.all([bundleId ? findRewardBundle(bundleId) : null, nameKeyApproval(draft.nameKey.trim())]);
      const found = lookupProblems(bundle, name);
      setRefused(found);
      if (found.length === 0) {
        onAdd(newOffer(draft, typeof bundle === "string" ? null : bundle));
        setDraft(emptyDraft);
      }
    });
  return (
    <fieldset className="flex flex-col gap-2">
      <legend className="text-sm text-text-secondary">{t("shop.new_offer")}</legend>
      <p className="text-sm text-text-muted">{t("shop.new_offer_hint")}</p>
      <div className="flex flex-wrap items-end gap-3">
        <label className="flex flex-col gap-1 text-sm text-text-secondary">
          {t("shop.new_offer_id")}
          <input value={draft.id} onChange={(event) => edit({ id: event.target.value.trim() })} className={`${fieldClass} font-mono`} />
        </label>
        <label className="flex flex-col gap-1 text-sm text-text-secondary">
          {t("shop.new_offer_kind")}
          <select value={draft.kind} onChange={(event) => edit({ kind: event.target.value, section: "" })} className={fieldClass}>
            <option value="">—</option>
            {offerKinds.map((kind) => (
              <option key={kind} value={kind}>
                {kind}
              </option>
            ))}
          </select>
        </label>
        <label className="flex flex-col gap-1 text-sm text-text-secondary">
          {t("shop.new_offer_section")}
          <select value={draft.section} onChange={(event) => edit({ section: event.target.value })} className={fieldClass}>
            <option value="">—</option>
            {sectionsFor(draft.kind).map((section) => (
              <option key={section} value={section}>
                {section}
              </option>
            ))}
          </select>
        </label>
        <label className="flex flex-col gap-1 text-sm text-text-secondary">
          {t("shop.new_offer_bundle")}
          <input value={draft.bundleId} onChange={(event) => edit({ bundleId: event.target.value })} className={`${fieldClass} font-mono`} />
        </label>
        <label className="flex flex-col gap-1 text-sm text-text-secondary">
          {t("shop.new_offer_name_key")}
          <input value={draft.nameKey} onChange={(event) => edit({ nameKey: event.target.value })} className={`${fieldClass} font-mono`} />
        </label>
        <button type="button" disabled={pending || problems.length > 0} onClick={add} className="rounded-md bg-surface px-3 py-1 text-sm text-text-secondary disabled:opacity-40">
          {t("shop.new_offer_add")}
        </button>
      </div>
      {[...problems, ...refused].map((problem) => (
        <p key={problem} role="alert" className="text-sm text-danger-text">
          {t(`shop.new_offer.${problem}`)}
        </p>
      ))}
    </fieldset>
  );
}
