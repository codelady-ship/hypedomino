"use client";

import { useState } from "react";
import { BundleEditor } from "./BundleEditor";
import { useTranslator } from "@/i18n/client";
import type { MessageKey } from "@/i18n/translate";
import { bundleDomains, useLabel, type BundleQuery, type NameKeysByDomain, type RewardBundle } from "@/rewards/bundles";

const fieldClass = "rounded-md bg-surface px-3 py-1 text-text-primary";
const labelClass = "flex flex-col gap-1 text-sm text-text-secondary";

const domainLabel: Record<string, MessageKey> = {
  shop: "rewards.domain.shop",
  pass: "rewards.domain.pass",
  missions: "rewards.domain.missions",
  rating: "rewards.domain.rating",
  daily_login: "rewards.domain.daily_login",
  promo: "rewards.domain.promo",
  promo_template: "rewards.domain.promo_template",
  reward_bundles: "rewards.domain.reward_bundles",
};

export function BundleList({ bundles, query, canEdit, nameKeys }: { bundles: RewardBundle[]; query: BundleQuery; canEdit: boolean; nameKeys: NameKeysByDomain }) {
  const t = useTranslator();
  const [selected, setSelected] = useState("");
  const label = (domain: string) => (domainLabel[domain] ? t(domainLabel[domain]) : domain);
  const editing = bundles.find((bundle) => bundle.id === selected && bundle.editable);
  return (
    <section className="flex flex-col gap-3">
      <form method="get" action="/rewards" className="flex flex-wrap items-end gap-3">
        <label className={labelClass}>
          {t("rewards.search")}
          <input name="q" defaultValue={query.q} className={fieldClass} />
        </label>
        <label className={labelClass}>
          <span>{t("rewards.used_in")}</span>
          <select name="domain" aria-label={t("rewards.used_in")} defaultValue={query.domain} className={fieldClass}>
            <option value="">{t("rewards.all")}</option>
            {bundleDomains.map((domain) => (
              <option key={domain} value={domain}>
                {label(domain)}
              </option>
            ))}
          </select>
        </label>
        <label className="flex items-center gap-2 text-sm text-text-secondary">
          <input type="checkbox" name="differing" value="true" defaultChecked={query.differing === "true"} />
          {t("rewards.only_differing")}
        </label>
        <button type="submit" className="rounded-md bg-primary px-4 py-2 font-semibold text-on-primary">
          {t("queues.search")}
        </button>
      </form>      <table className="w-full text-sm">
        <thead className="text-left text-text-muted">
          <tr>
            <th className="py-1 pr-3">{t("rewards.id")}</th>
            <th className="py-1 pr-3">{t("rewards.revision")}</th>
            <th className="py-1 pr-3">{t("rewards.contents")}</th>
            <th className="py-1 pr-3">{t("rewards.guide_value")}</th>
            <th className="py-1 pr-3">{t("rewards.used_in")}</th>
            {canEdit && <th className="py-1 pr-3" />}
          </tr>
        </thead>
        <tbody>
          {bundles.map((bundle) => (
            <tr key={`${bundle.id}:${useLabel(bundle.usedBy[0])}`} className="border-t border-surface align-top">
              <td className="py-1 pr-3 font-mono break-all">
                {bundle.id}
                {bundle.differs && <p className="font-sans text-danger-text">{t("rewards.differs")}</p>}
              </td>
              <td className="py-1 pr-3">{bundle.revision}</td>
              <td className="py-1 pr-3">
                <Contents bundle={bundle} />
              </td>
              <td className="py-1 pr-3">{bundle.guideValue}</td>
              <td className="py-1 pr-3">
                <ul className="flex flex-col gap-1">
                  {bundle.usedBy.map((use) => (
                    <li key={`${use.domain}:${useLabel(use)}`}>
                      {label(use.domain)}: <span className="font-mono break-all">{useLabel(use)}</span>
                    </li>
                  ))}
                </ul>
              </td>
              {canEdit && (
                <td className="py-1 pr-3">
                  {bundle.editable ? (
                    <button type="button" aria-label={`${t("rewards.edit")} ${bundle.id}`} onClick={() => setSelected(bundle.id)} className="rounded-md bg-surface px-3 py-1 text-text-secondary">
                      {t("rewards.edit")}
                    </button>
                  ) : (
                    <span className="text-text-muted">{t("rewards.edit_elsewhere")}</span>
                  )}
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
      {bundles.length === 0 && <p className="text-sm text-text-secondary">{t("rewards.no_match")}</p>}
      {canEdit && editing && <BundleEditor key={`${editing.id}:${editing.revision}`} bundle={editing} domainLabel={label} nameKeys={nameKeys} />}
    </section>
  );
}

function Contents({ bundle }: { bundle: RewardBundle }) {
  const t = useTranslator();
  return (
    <ul className="flex flex-col gap-1">
      {bundle.coins > 0 && (
        <li>
          {t("rewards.coins")}: {bundle.coins}
        </li>
      )}
      {(bundle.crystals ?? 0) > 0 && (
        <li>
          {t("rewards.crystals")}: {bundle.crystals}
        </li>
      )}
      {bundle.passXp > 0 && (
        <li>
          {t("rewards.pass_xp")}: {bundle.passXp}
        </li>
      )}
      {(bundle.cosmetics ?? []).length > 0 && (
        <li>
          {t("rewards.cosmetics")}: <span className="font-mono break-all">{(bundle.cosmetics ?? []).join(", ")}</span>
        </li>
      )}
      {(bundle.medals ?? []).length > 0 && (
        <li>
          {t("rewards.medals")}: <span className="font-mono break-all">{(bundle.medals ?? []).join(", ")}</span>
        </li>
      )}
    </ul>
  );
}
