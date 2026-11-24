"use client";

import { useState, useTransition } from "react";
import { useTranslator } from "@/i18n/client";
import type { MessageKey } from "@/i18n/translate";
import {
  allowedCurrencies,
  catalogNameKeys,
  chestCandidates,
  chestRarities,
  chestRulesOf,
  isUnpublishedOffer,
  crystalPacks,
  listsChest,
  offerProblems,
  oddsTotal,
  permanentMythics,
  priceOf,
  referencePriceProblem,
  replaceOffer,
  withPackCrystals,
  withPoolItem,
  periodStart,
  rotationCandidates,
  rotationFits,
  slotsOf,
  termsProblems,
  toggledOffer,
  unlistedCosmetics,
  variantProblems,
  previewVariants,
  windowEndsBeforeStart,
  windowMinute,
  withPrice,
  withReferencePrice,
  withRegionalNote,
  withRotation,
  withWindow,
  withOffer,
  withoutRotation,
  withoutOffer,
  type CosmeticItem,
  type Offer,
  type PriceCurrency,
  type Rotation,
  type RotationPeriod,
  type RotationRules,
  type ShopCatalog,
} from "@/shop/catalog";
import { previewCatalog, publishCatalog, type PreviewResult, type PublishResult } from "./actions";
import { NewOffer } from "./NewOffer";
import { OfferChecks } from "./OfferChecks";
import { OfferTerms } from "./OfferTerms";
import { OfferVariants } from "./OfferVariants";
import { UnnamedKeys, UnnamedKeysAhead } from "@/app/strings/UnnamedKeys";

const currencyLabel: Record<PriceCurrency, MessageKey> = {
  coins: "currency.coin",
  crystals: "currency.crystal",
  product: "shop.price.product",
};

const fieldClass = "rounded-md bg-surface px-3 py-1 text-text-primary";

type ReferencePrices = { days: number; inEffect: ShopCatalog[] };

export function ShopEditor({
  initialCatalog,
  version,
  items,
  rules,
  referencePrices,
}: {
  initialCatalog: ShopCatalog;
  version: number;
  items: CosmeticItem[];
  rules: RotationRules;
  referencePrices: ReferencePrices;
}) {
  const t = useTranslator();
  const [catalog, setCatalog] = useState(initialCatalog);
  const [selected, setSelected] = useState(initialCatalog.offers[0]?.id ?? "");
  const [reason, setReason] = useState("");
  const [result, setResult] = useState<PublishResult | null>(null);
  const [pending, startTransition] = useTransition();
  const offer = catalog.offers.find((candidate) => candidate.id === selected);
  const update = (next: Offer) => setCatalog((current) => replaceOffer(current, next));

  return (
    <div className="flex flex-col gap-8">
      <p className="text-sm text-text-muted">{t("shop.version")}: {version}</p>
      <section className="flex flex-col gap-3">
        <h2 className="text-lg font-semibold">{t("shop.offers")}</h2>
        <select aria-label={t("shop.offers")} value={selected} onChange={(event) => setSelected(event.target.value)} className={fieldClass}>
          {catalog.offers.map((candidate) => (
            <option key={candidate.id} value={candidate.id}>
              {candidate.id}
            </option>
          ))}
        </select>
        {offer && <OfferPrice offer={offer} referencePrices={referencePrices} onChange={update} />}
        {offer && <OfferTerms offer={offer} onChange={update} />}
        {offer && <OfferWindow offer={offer} mythics={permanentMythics(offer, items)} onChange={update} />}
        {offer && <OfferVariants offer={offer} onChange={update} />}
        {offer && <OfferChecks offer={offer} items={items} />}
        {offer && isUnpublishedOffer(initialCatalog, offer.id) && (
          <button
            type="button"
            onClick={() => {
              setCatalog((current) => withoutOffer(current, offer.id));
              setSelected(catalog.offers.find((candidate) => candidate.id !== offer.id)?.id ?? "");
            }}
            className="self-start rounded-md bg-surface px-3 py-1 text-sm text-text-secondary"
          >
            {t("shop.remove_offer")}
          </button>
        )}
        <NewOffer
          catalog={catalog}
          onAdd={(added) => {
            setCatalog((current) => withOffer(current, added));
            setSelected(added.id);
          }}
        />
      </section>
      <CrystalPacks catalog={catalog} onChange={update} />
      <Chests catalog={catalog} items={items} onChange={update} onRules={(chests) => setCatalog((current) => ({ ...current, chests }))} />
      <Rotations catalog={catalog} rules={rules} onChange={setCatalog} />
      <Preview catalog={catalog} />
      <section className="flex flex-col gap-3">
        <label className="flex flex-col gap-1 text-sm text-text-secondary">
          {t("shop.reason")}
          <input value={reason} onChange={(event) => setReason(event.target.value)} className={fieldClass} />
        </label>
        <UnnamedKeysAhead keys={catalogNameKeys(catalog)} />
        <button
          type="button"
          disabled={
            pending ||
            catalog.offers.some(
              (candidate) =>
                windowEndsBeforeStart(candidate) ||
                permanentMythics(candidate, items).length > 0 ||
                termsProblems(candidate).length > 0 ||
                variantProblems(candidate).length > 0 ||
                offerProblems(candidate).length > 0 ||
                unlistedCosmetics(candidate, items).length > 0 ||
                referencePriceProblem(candidate, referencePrices.inEffect) !== null,
            ) ||
            catalog.rotations.some((rotation) => !rotationFits(rotation, rules))
          }
          onClick={() => startTransition(async () => setResult(await publishCatalog(catalog, reason)))}
          className="self-start rounded-md bg-primary px-4 py-2 text-on-primary disabled:opacity-40"
        >
          {t("shop.publish")}
        </button>
        {result && (
          <p role="status" className="text-sm text-text-secondary">
            {result.ok ? `${t("shop.published")}: ${result.version}` : `${t("shop.refused")}: ${result.error} ${result.detail}`}
          </p>
        )}
        {result && !result.ok && <UnnamedKeys keys={result.keys} />}
      </section>
    </div>
  );
}

function Preview({ catalog }: { catalog: ShopCatalog }) {
  const t = useTranslator();
  const [minute, setMinute] = useState("");
  const [chosen, setChosen] = useState<Record<string, string>>({});
  const [result, setResult] = useState<PreviewResult | null>(null);
  const tested = catalog.offers.filter((offer) => (offer.variants ?? []).length > 0);
  const [pending, startTransition] = useTransition();
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-lg font-semibold">{t("shop.preview")}</h2>
      <p className="text-sm text-text-muted">{t("shop.preview_hint")}</p>
      <div className="flex flex-wrap items-end gap-3">
        <label className="flex flex-col gap-1 text-sm text-text-secondary">
          {t("shop.preview_at")}
          <input type="datetime-local" value={minute} onChange={(event) => setMinute(event.target.value)} className={fieldClass} />
        </label>
        {tested.map((offer) => (
          <label key={offer.id} className="flex flex-col gap-1 text-sm text-text-secondary">
            <span className="font-mono">{offer.id}</span>
            <select value={chosen[offer.id] ?? ""} onChange={(event) => setChosen({ ...chosen, [offer.id]: event.target.value })} className={fieldClass}>
              <option value="">{t("shop.preview_as_written")}</option>
              {(offer.variants ?? []).map((variant) => (
                <option key={variant.id} value={variant.id}>
                  {variant.id}
                </option>
              ))}
            </select>
          </label>
        ))}
        <button
          type="button"
          disabled={pending}
          onClick={() => startTransition(async () => setResult(await previewCatalog(catalog, minute ? `${minute}:00Z` : undefined, previewVariants(catalog, chosen))))}
          className="rounded-md bg-surface px-4 py-2 text-text-secondary disabled:opacity-40"
        >
          {t("shop.preview_show")}
        </button>
      </div>
      {result && !result.ok && (
        <p role="alert" className="text-sm text-danger-text">
          {t("shop.refused")}: {result.error} {result.detail}
        </p>
      )}
      {result?.ok && (
        <table className="text-sm">
          <caption className="text-left text-text-muted">{t.when(result.at)}</caption>
          <tbody>
            {result.sections.map((section) => (
              <tr key={section.section} className="border-t border-surface">
                <td className="py-1 pr-4 font-mono">{section.section}</td>
                <td className="py-1 pr-4 font-mono">{section.offers.map((offer) => (offer.variant ? `${offer.id} [${offer.variant}: ${offer.name_key}]` : offer.id)).join(", ") || "—"}</td>
                <td className="py-1 pr-4 text-text-muted">{section.ends_at ? t.when(section.ends_at) : ""}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  );
}

const periods: readonly RotationPeriod[] = ["daily", "weekly"];

function Rotations({ catalog, rules, onChange }: { catalog: ShopCatalog; rules: RotationRules; onChange: (catalog: ShopCatalog) => void }) {
  const t = useTranslator();
  const [period, setPeriod] = useState<RotationPeriod>("daily");
  const [day, setDay] = useState("");
  const start = periodStart(period, day, rules);
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-lg font-semibold">{t("shop.rotations")}</h2>
      <p className="text-sm text-text-muted">
        {t("shop.rotation_rules")}: {t("shop.rotation_daily")} {rules.daily_slots}, {t("shop.rotation_weekly")} {rules.weekly_slots}, {t("shop.rotation_reset")} {rules.reset_hour_utc}:00 UTC
      </p>
      {catalog.rotations.length === 0 && <p className="text-sm text-text-secondary">{t("shop.rotations_empty")}</p>}
      {catalog.rotations.map((rotation) => (
        <RotationRow
          key={`${rotation.period}-${rotation.start}`}
          rotation={rotation}
          candidates={rotationCandidates(catalog, rotation.period)}
          slots={slotsOf(rules, rotation.period)}
          fits={rotationFits(rotation, rules)}
          onChange={(next) => onChange(withRotation(catalog, next))}
          onRemove={() => onChange(withoutRotation(catalog, rotation))}
        />
      ))}
      <div className="flex flex-wrap items-end gap-3">
        <label className="flex flex-col gap-1 text-sm text-text-secondary">
          {t("shop.rotation_period")}
          <select value={period} onChange={(event) => setPeriod(event.target.value as RotationPeriod)} className={fieldClass}>
            {periods.map((candidate) => (
              <option key={candidate} value={candidate}>
                {t(candidate === "daily" ? "shop.rotation_daily" : "shop.rotation_weekly")}
              </option>
            ))}
          </select>
        </label>
        <label className="flex flex-col gap-1 text-sm text-text-secondary">
          {t("shop.rotation_day")}
          <input type="date" value={day} onChange={(event) => setDay(event.target.value)} className={fieldClass} />
        </label>
        <button
          type="button"
          disabled={start === null}
          onClick={() => start && onChange(withRotation(catalog, { period, start, offers: [] }))}
          className="rounded-md bg-surface px-4 py-2 text-text-secondary disabled:opacity-40"
        >
          {t("shop.rotation_add")}
        </button>
      </div>
      {day && start === null && (
        <p role="alert" className="text-sm text-danger-text">
          {t("shop.rotation_monday")}
        </p>
      )}
    </section>
  );
}

function RotationRow({
  rotation,
  candidates,
  slots,
  fits,
  onChange,
  onRemove,
}: {
  rotation: Rotation;
  candidates: string[];
  slots: number;
  fits: boolean;
  onChange: (rotation: Rotation) => void;
  onRemove: () => void;
}) {
  const t = useTranslator();
  return (
    <fieldset className="flex flex-col gap-2 rounded-md border border-surface p-3">
      <legend className="text-sm text-text-secondary">
        {t(rotation.period === "daily" ? "shop.rotation_daily" : "shop.rotation_weekly")} · {t.when(rotation.start)}
      </legend>
      <div className="flex flex-wrap gap-3">
        {candidates.map((id) => (
          <label key={id} className="flex items-center gap-1 font-mono text-sm">
            <input type="checkbox" checked={rotation.offers.includes(id)} onChange={() => onChange(toggledOffer(rotation, id))} />
            {id}
          </label>
        ))}
      </div>
      <p className="text-sm text-text-muted">
        {t("shop.rotation_order")}: {rotation.offers.join(" → ") || "—"} ({rotation.offers.length}/{slots})
      </p>
      {!fits && (
        <p role="alert" className="text-sm text-danger-text">
          {t("shop.rotation_slots")}
        </p>
      )}
      <button type="button" onClick={onRemove} className="self-start rounded-md bg-surface px-3 py-1 text-sm text-text-secondary">
        {t("shop.rotation_remove")}
      </button>
    </fieldset>
  );
}

function OfferWindow({ offer, mythics, onChange }: { offer: Offer; mythics: string[]; onChange: (offer: Offer) => void }) {
  const t = useTranslator();
  const start = windowMinute(offer.window.start);
  const end = windowMinute(offer.window.end);
  return (
    <fieldset className="flex flex-col gap-2">
      <legend className="text-sm text-text-secondary">{t("shop.window")}</legend>
      <div className="flex flex-wrap gap-3">
        <label className="flex flex-col gap-1 text-sm text-text-secondary">
          {t("shop.window_start")}
          <input type="datetime-local" required value={start} onChange={(event) => onChange(withWindow(offer, event.target.value, end))} className={fieldClass} />
        </label>
        <label className="flex flex-col gap-1 text-sm text-text-secondary">
          {t("shop.window_end")}
          <input type="datetime-local" value={end} onChange={(event) => onChange(withWindow(offer, start, event.target.value))} className={fieldClass} />
        </label>
      </div>
      {windowEndsBeforeStart(offer) && (
        <p role="alert" className="text-sm text-danger-text">
          {t("shop.window_end_before_start")}
        </p>
      )}
      {mythics.length > 0 && (
        <p role="alert" className="text-sm text-danger-text">
          {t("shop.mythic_time_limited")}: <span className="font-mono">{mythics.join(", ")}</span>
        </p>
      )}
    </fieldset>
  );
}

function OfferPrice({ offer, referencePrices, onChange }: { offer: Offer; referencePrices: ReferencePrices; onChange: (offer: Offer) => void }) {
  const t = useTranslator();
  const price = priceOf(offer);
  const problem = referencePriceProblem(offer, referencePrices.inEffect);
  return (
    <fieldset className="flex flex-col gap-2">
      <legend className="text-sm text-text-secondary">{t("shop.price")}</legend>
      <div className="flex flex-wrap gap-3">
        {allowedCurrencies(offer).map((currency) => (
          <label key={currency} className="flex items-center gap-1 text-sm">
            <input type="radio" name={`currency-${offer.id}`} checked={price.currency === currency} onChange={() => onChange(withPrice(offer, { ...price, currency }))} />
            {t(currencyLabel[currency])}
          </label>
        ))}
      </div>
      {price.currency === "product" ? (
        <>
          <input aria-label={t("shop.price.product")} value={price.productId} onChange={(event) => onChange(withPrice(offer, { ...price, productId: event.target.value }))} className={fieldClass} />
          <label className="flex flex-col gap-1 text-sm text-text-secondary">
            {t("shop.regional_note")}
            <textarea rows={2} value={offer.regionalOverrideNote ?? ""} onChange={(event) => onChange(withRegionalNote(offer, event.target.value))} className={fieldClass} />
            <span className="text-text-muted">{t("shop.regional_note_hint")}</span>
          </label>
        </>
      ) : (
        <div className="flex flex-wrap items-end gap-3">
          <input
            type="number"
            min={1}
            aria-label={t(currencyLabel[price.currency])}
            value={price.amount}
            onChange={(event) => onChange(withPrice(offer, { ...price, amount: Number(event.target.value) }))}
            className={fieldClass}
          />
          <label className="flex flex-col gap-1 text-sm text-text-secondary">
            {t("shop.reference_price_input")}
            <input
              type="number"
              min={1}
              step={1}
              value={(offer.referencePriceCoins ?? 0) + (offer.referencePriceCrystals ?? 0) || ""}
              onChange={(event) => onChange(withReferencePrice(offer, Number(event.target.value)))}
              className={fieldClass}
            />
          </label>
        </div>
      )}
      {problem && (
        <p role="alert" className="text-sm text-danger-text">
          {t("shop.reference_price")} {t.number(problem.reference)} {t("shop.reference_price_not_lowest")} ({t("shop.reference_price_days")}: {t.number(referencePrices.days)}).{" "}
          {problem.lowest === null ? t("shop.reference_price_no_earlier") : `${t("shop.reference_price_lowest")}: ${t.number(problem.lowest)}`}
        </p>
      )}
    </fieldset>
  );
}

function CrystalPacks({ catalog, onChange }: { catalog: ShopCatalog; onChange: (offer: Offer) => void }) {
  const t = useTranslator();
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-lg font-semibold">{t("shop.crystal_packs")}</h2>
      <table className="text-sm">
        <thead className="text-left text-text-muted">
          <tr>
            <th className="py-1 pr-4">{t("shop.price.product")}</th>
            <th className="py-1">{t("currency.crystal")}</th>
          </tr>
        </thead>
        <tbody>
          {crystalPacks(catalog).map((pack) => (
            <tr key={pack.id}>
              <td className="py-1 pr-4">{pack.productId}</td>
              <td className="py-1">
                <input
                  type="number"
                  min={1}
                  aria-label={`${pack.productId} ${t("currency.crystal")}`}
                  value={pack.bundle?.crystals ?? 0}
                  onChange={(event) => onChange(withPackCrystals(pack, Number(event.target.value)))}
                  className={fieldClass}
                />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}

function Chests({
  catalog,
  items,
  onChange,
  onRules,
}: {
  catalog: ShopCatalog;
  items: CosmeticItem[];
  onChange: (offer: Offer) => void;
  onRules: (rules: ShopCatalog["chests"]) => void;
}) {
  const t = useTranslator();
  const rules = chestRulesOf(catalog);
  const candidates = chestCandidates(items);
  const chests = catalog.offers.filter((offer) => offer.kind === "chest");
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-lg font-semibold">{t("shop.chests")}</h2>
      <div className="flex flex-wrap gap-4 text-sm">
        {chestRarities.map((rarity) => (
          <label key={rarity} className="flex flex-col gap-1">
            {t(`rarity.${rarity}`)} %
            <input
              type="number"
              min={0}
              max={100}
              value={rules.rarityPercent[rarity] ?? 0}
              onChange={(event) => onRules({ ...rules, rarityPercent: { ...rules.rarityPercent, [rarity]: Number(event.target.value) } })}
              className={fieldClass}
            />
          </label>
        ))}
        <label className="flex flex-col gap-1">
          {t("shop.chest_pity")}
          <input type="number" min={1} value={rules.rarePityCount} onChange={(event) => onRules({ ...rules, rarePityCount: Number(event.target.value) })} className={fieldClass} />
        </label>
      </div>
      {oddsTotal(rules) !== 100 && <p className="text-sm text-primary">{t("shop.chest_odds_total")}</p>}
      {chests.map((chest) => (
        <fieldset key={chest.id} className="flex flex-col gap-1">
          <legend className="text-sm text-text-secondary">{chest.id}</legend>
          {candidates.map((item) => (
            <label key={item.id} className="flex items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={chest.bundle?.cosmetics?.includes(item.id) ?? false}
                onChange={(event) => onChange(withPoolItem(chest, item.id, event.target.checked))}
              />
              {item.id} · {t(`rarity.${item.rarity as "common" | "rare"}`)}
              {!listsChest(item, chest) && <span className="text-text-muted">{t("shop.chest_source_missing")}</span>}
            </label>
          ))}
        </fieldset>
      ))}
    </section>
  );
}
