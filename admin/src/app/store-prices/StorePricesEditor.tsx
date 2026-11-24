"use client";

import { useRouter } from "next/navigation";
import { useMemo, useState, useTransition } from "react";
import type { ConfigResult } from "@/config/result";
import { useLocale, useTranslator } from "@/i18n/client";
import {
  centsOf,
  countriesInTier,
  formatUsd,
  matchesCountry,
  pricedCountries,
  sameStorePrices,
  storePriceIssues,
  targetOf,
  withCountryTier,
  withOverride,
  withoutTier,
  withReference,
  withTierPercent,
  type StorePrices,
} from "@/storeprices/prices";
import { publishStorePrices } from "./actions";

const fieldClass = "w-24 rounded-md bg-surface px-3 py-1 text-text-primary disabled:opacity-50";
const labelClass = "flex flex-col gap-1 text-sm text-text-secondary";
const buttonClass = "self-start rounded-md bg-surface px-3 py-1 text-text-secondary disabled:opacity-40";

type Props = { published: StorePrices | null; starting: StorePrices; products: string[]; countries: string[]; version: number; canPublish: boolean };

type Edit = { prices: StorePrices; change: (next: StorePrices) => void; disabled: boolean };

function amount(value: string): number {
  return value === "" ? Number.NaN : Number(value);
}

function useCountryNames(countries: string[]): Record<string, string> {
  const locale = useLocale();
  return useMemo(() => {
    const names = new Intl.DisplayNames([locale], { type: "region" });
    return Object.fromEntries(countries.map((code) => [code, names.of(code) ?? code]));
  }, [locale, countries]);
}

export function StorePricesEditor({ published, starting, products, countries, version, canPublish }: Props) {
  const t = useTranslator();
  const [prices, setPrices] = useState(starting);
  const [query, setQuery] = useState("");
  const names = useCountryNames(countries);
  const issues = storePriceIssues(prices, products, countries);
  const edit: Edit = { prices, change: setPrices, disabled: !canPublish };
  return (
    <section className="flex flex-col gap-8">
      <p className="text-sm text-text-muted">
        {t("store_prices.version")}: {version}
      </p>
      <References edit={edit} products={products} />
      <Tiers edit={edit} />
      <label className={labelClass}>
        {t("store_prices.search")}
        <input value={query} onChange={(event) => setQuery(event.target.value)} className="w-64 rounded-md bg-surface px-3 py-1 text-text-primary" />
      </label>
      <Countries edit={edit} countries={countries.filter((code) => matchesCountry(code, names[code], query))} names={names} />
      <Overrides edit={edit} products={products} countries={countries} names={names} />
      <Preview prices={prices} products={products} countries={pricedCountries(prices).filter((code) => matchesCountry(code, names[code] ?? code, query))} names={names} />
      {issues.length > 0 && (
        <ul role="alert" className="flex flex-col gap-1 text-sm text-danger-text">
          {issues.map((issue) => (
            <li key={issue}>{t(issue)}</li>
          ))}
        </ul>
      )}
      {canPublish && <Publish prices={prices} ready={issues.length === 0 && (published === null || !sameStorePrices(published, prices))} />}
    </section>
  );
}

function References({ edit, products }: { edit: Edit; products: string[] }) {
  const t = useTranslator();
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-lg font-semibold">{t("store_prices.references")}</h2>
      <p className="text-sm text-text-muted">{t("store_prices.references_hint")}</p>
      <div className="flex flex-wrap gap-3">
        {products.map((product) => (
          <label key={product} className={labelClass}>
            {product}
            <input
              type="number"
              min={0.01}
              step={0.01}
              disabled={edit.disabled}
              value={Number.isNaN(edit.prices.referenceUsd[product] ?? Number.NaN) ? "" : edit.prices.referenceUsd[product]}
              onChange={(event) => edit.change(withReference(edit.prices, product, amount(event.target.value)))}
              className={fieldClass}
            />
          </label>
        ))}
      </div>
    </section>
  );
}

function Tiers({ edit }: { edit: Edit }) {
  const t = useTranslator();
  const [name, setName] = useState("");
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-lg font-semibold">{t("store_prices.tiers")}</h2>
      <table className="text-sm">
        <thead className="text-left text-text-muted">
          <tr>
            <th className="py-2 pr-4">{t("store_prices.tier")}</th>
            <th className="py-2 pr-4">{t("store_prices.percent")}</th>
            <th className="py-2 pr-4">{t("store_prices.countries_in_tier")}</th>
            <th className="py-2 pr-4" />
          </tr>
        </thead>
        <tbody>
          {Object.entries(edit.prices.tiers).map(([tier, percent]) => (
            <tr key={tier} className="border-t border-surface">
              <td className="py-2 pr-4">{tier}</td>
              <td className="py-2 pr-4">
                <input
                  type="number"
                  min={1}
                  max={100}
                  disabled={edit.disabled}
                  aria-label={`${tier} ${t("store_prices.percent")}`}
                  value={Number.isNaN(percent) ? "" : percent}
                  onChange={(event) => edit.change(withTierPercent(edit.prices, tier, amount(event.target.value)))}
                  className={fieldClass}
                />
              </td>
              <td className="py-2 pr-4">{countriesInTier(edit.prices, tier)}</td>
              <td className="py-2 pr-4">
                <button type="button" disabled={edit.disabled || countriesInTier(edit.prices, tier) > 0} onClick={() => edit.change(withoutTier(edit.prices, tier))} className={buttonClass}>
                  {t("store_prices.remove")}
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      {!edit.disabled && (
        <div className="flex flex-wrap items-end gap-3">
          <label className={labelClass}>
            {t("store_prices.new_tier")}
            <input value={name} onChange={(event) => setName(event.target.value.trim())} className="w-40 rounded-md bg-surface px-3 py-1 text-text-primary" />
          </label>
          <button type="button" disabled={name === "" || name in edit.prices.tiers} onClick={() => edit.change(withTierPercent(edit.prices, name, 100))} className={buttonClass}>
            {t("store_prices.add_tier")}
          </button>
        </div>
      )}
    </section>
  );
}

function Countries({ edit, countries, names }: { edit: Edit; countries: string[]; names: Record<string, string> }) {
  const t = useTranslator();
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-lg font-semibold">{t("store_prices.countries")}</h2>
      <div className="max-h-96 overflow-y-auto">
        <table className="w-full text-sm">
          <thead className="sticky top-0 bg-background-base text-left text-text-muted">
            <tr>
              <th className="py-2 pr-4">{t("store_prices.country")}</th>
              <th className="py-2 pr-4">{t("store_prices.tier")}</th>
            </tr>
          </thead>
          <tbody>
            {countries.map((code) => (
              <tr key={code} className="border-t border-surface">
                <td className="py-1 pr-4">{`${code} · ${names[code]}`}</td>
                <td className="py-1 pr-4">
                  <select
                    aria-label={`${code} ${t("store_prices.tier")}`}
                    disabled={edit.disabled}
                    value={edit.prices.countries[code] ?? ""}
                    onChange={(event) => edit.change(withCountryTier(edit.prices, code, event.target.value === "" ? null : event.target.value))}
                    className="rounded-md bg-surface px-2 py-1 text-text-primary disabled:opacity-50"
                  >
                    <option value="">{t("store_prices.not_listed")}</option>
                    {Object.keys(edit.prices.tiers).map((tier) => (
                      <option key={tier} value={tier}>{`${tier} (${edit.prices.tiers[tier]}%)`}</option>
                    ))}
                  </select>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function Overrides({ edit, products, countries, names }: { edit: Edit; products: string[]; countries: string[]; names: Record<string, string> }) {
  const t = useTranslator();
  const [product, setProduct] = useState(products[0] ?? "");
  const [country, setCountry] = useState(countries[0] ?? "");
  const [usd, setUsd] = useState("");
  const rows = Object.entries(edit.prices.overrides).flatMap(([id, byCountry]) => Object.entries(byCountry).map(([code, value]) => ({ id, code, value })));
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-lg font-semibold">{t("store_prices.overrides")}</h2>
      {rows.length > 0 && (
        <table className="text-sm">
          <thead className="text-left text-text-muted">
            <tr>
              <th className="py-2 pr-4">{t("store_prices.product")}</th>
              <th className="py-2 pr-4">{t("store_prices.country")}</th>
              <th className="py-2 pr-4">{t("store_prices.target_usd")}</th>
              <th className="py-2 pr-4" />
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <tr key={`${row.id}-${row.code}`} className="border-t border-surface">
                <td className="py-2 pr-4">{row.id}</td>
                <td className="py-2 pr-4">{`${row.code} · ${names[row.code] ?? row.code}`}</td>
                <td className="py-2 pr-4">{row.value}</td>
                <td className="py-2 pr-4">
                  <button type="button" disabled={edit.disabled} onClick={() => edit.change(withOverride(edit.prices, row.id, row.code, null))} className={buttonClass}>
                    {t("store_prices.remove")}
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {!edit.disabled && (
        <div className="flex flex-wrap items-end gap-3">
          <label className={labelClass}>
            {t("store_prices.product")}
            <select value={product} onChange={(event) => setProduct(event.target.value)} className="rounded-md bg-surface px-2 py-1 text-text-primary">
              {products.map((id) => (
                <option key={id} value={id}>
                  {id}
                </option>
              ))}
            </select>
          </label>
          <label className={labelClass}>
            {t("store_prices.country")}
            <select value={country} onChange={(event) => setCountry(event.target.value)} className="rounded-md bg-surface px-2 py-1 text-text-primary">
              {countries.map((code) => (
                <option key={code} value={code}>{`${code} · ${names[code]}`}</option>
              ))}
            </select>
          </label>
          <label className={labelClass}>
            {t("store_prices.target_usd")}
            <input type="number" min={0.01} step={0.01} value={usd} onChange={(event) => setUsd(event.target.value)} className={fieldClass} />
          </label>
          <button type="button" disabled={centsOf(amount(usd)) === null} onClick={() => edit.change(withOverride(edit.prices, product, country, amount(usd)))} className={buttonClass}>
            {t("store_prices.add_override")}
          </button>
        </div>
      )}
    </section>
  );
}

function Preview({ prices, products, countries, names }: { prices: StorePrices; products: string[]; countries: string[]; names: Record<string, string> }) {
  const t = useTranslator();
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-lg font-semibold">{t("store_prices.preview")}</h2>
      <p className="text-sm text-text-muted">{t("store_prices.preview_hint")}</p>
      <div className="max-h-[32rem] overflow-auto">
        <table className="text-sm">
          <thead className="sticky top-0 bg-background-base text-left text-text-muted">
            <tr>
              <th className="py-2 pr-4">{t("store_prices.country")}</th>
              {products.map((product) => (
                <th key={product} className="py-2 pr-4 whitespace-nowrap">
                  {product}
                  <br />
                  {`${t("store_prices.reference_usd")} ${prices.referenceUsd[product] ?? "—"}`}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {countries.map((code) => (
              <tr key={code} className="border-t border-surface">
                <td className="py-1 pr-4 whitespace-nowrap">{`${code} · ${names[code] ?? code}${prices.countries[code] ? ` (${prices.countries[code]})` : ""}`}</td>
                {products.map((product) => {
                  const target = targetOf(prices, product, code);
                  return (
                    <td key={product} className={target?.override ? "py-1 pr-4 font-semibold" : "py-1 pr-4"}>
                      {target ? formatUsd(target.cents) : "—"}
                    </td>
                  );
                })}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function Publish({ prices, ready }: { prices: StorePrices; ready: boolean }) {
  const t = useTranslator();
  const router = useRouter();
  const [reason, setReason] = useState("");
  const [result, setResult] = useState<ConfigResult | null>(null);
  const [pending, startTransition] = useTransition();
  return (
    <div className="flex flex-col gap-3">
      <label className={labelClass}>
        {t("store_prices.reason")}
        <input value={reason} onChange={(event) => setReason(event.target.value)} className="rounded-md bg-surface px-3 py-1 text-text-primary" />
      </label>
      <button
        type="button"
        disabled={!ready || pending || reason.trim() === ""}
        onClick={() =>
          startTransition(async () => {
            const outcome = await publishStorePrices(prices, reason);
            setResult(outcome);
            if (outcome.ok) {
              router.refresh();
            }
          })
        }
        className="self-start rounded-md bg-primary px-4 py-2 text-on-primary disabled:opacity-40"
      >
        {t("store_prices.publish")}
      </button>
      {result && !result.ok && (
        <p role="status" className="text-sm text-text-secondary">
          {t("store_prices.refused")}: {result.error} {result.detail}
        </p>
      )}
    </div>
  );
}
