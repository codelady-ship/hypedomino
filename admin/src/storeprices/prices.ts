import { sameContent } from "../config/sameContent.ts";
import type { MessageKey } from "../i18n/translate.ts";

export type StorePrices = {
  schema: number;
  referenceUsd: Record<string, number>;
  tiers: Record<string, number>;
  countries: Record<string, string>;
  overrides: Record<string, Record<string, number>>;
};

export type Target = { product: string; country: string; tier?: string; cents: number; override?: boolean };

export type Store = "google_play" | "app_store";

export const stores: readonly Store[] = ["google_play", "app_store"];

const tierName = /^[a-z][a-z0-9_]{0,31}$/;

export function centsOf(usd: number): number | null {
  const scaled = usd * 100;
  const rounded = Math.round(scaled);
  return Number.isFinite(usd) && rounded >= 1 && Math.abs(scaled - rounded) <= 1e-6 ? rounded : null;
}

export function formatUsd(cents: number): string {
  return (cents / 100).toFixed(2);
}

export function targetOf(prices: StorePrices, product: string, country: string): Target | null {
  const override = prices.overrides[product]?.[country];
  if (override !== undefined) {
    return { product, country, cents: centsOf(override) ?? 0, override: true };
  }
  const tier = prices.countries[country];
  const reference = centsOf(prices.referenceUsd[product] ?? Number.NaN);
  if (tier === undefined || reference === null) {
    return null;
  }
  return { product, country, tier, cents: Math.floor((reference * (prices.tiers[tier] ?? 0) + 50) / 100) };
}

export function pricedCountries(prices: StorePrices): string[] {
  const overridden = Object.values(prices.overrides).flatMap((byCountry) => Object.keys(byCountry));
  return [...new Set([...Object.keys(prices.countries), ...overridden])].sort();
}

export function storeCsv(rows: readonly Target[]): string {
  const lines = rows.map((row) => `${row.product},${row.country},${formatUsd(row.cents)}`);
  return ["product_id,country_code,target_usd", ...lines].join("\n") + "\n";
}

export function csvFileName(store: Store, version: number): string {
  return `store-prices-${store.replace("_", "-")}-v${version}.csv`;
}

export function storePriceIssues(prices: StorePrices, products: readonly string[], countries: readonly string[]): MessageKey[] {
  const issues = new Set<MessageKey>();
  if (products.some((product) => centsOf(prices.referenceUsd[product] ?? Number.NaN) === null)) {
    issues.add("store_prices.issue.reference");
  }
  for (const [tier, percent] of Object.entries(prices.tiers)) {
    if (!tierName.test(tier)) {
      issues.add("store_prices.issue.tier_name");
    }
    if (!Number.isInteger(percent) || percent < 1 || percent > 100) {
      issues.add("store_prices.issue.percent");
    }
  }
  for (const [country, tier] of Object.entries(prices.countries)) {
    if (!countries.includes(country)) {
      issues.add("store_prices.issue.country");
    }
    if (!(tier in prices.tiers)) {
      issues.add("store_prices.issue.unknown_tier");
    }
  }
  for (const [product, byCountry] of Object.entries(prices.overrides)) {
    for (const [country, usd] of Object.entries(byCountry)) {
      overrideIssues(prices, product, country, usd, products, countries).forEach((issue) => issues.add(issue));
    }
  }
  return [...issues];
}

function overrideIssues(prices: StorePrices, product: string, country: string, usd: number, products: readonly string[], countries: readonly string[]): MessageKey[] {
  const reference = centsOf(prices.referenceUsd[product] ?? Number.NaN);
  const amount = centsOf(usd);
  const found: MessageKey[] = [];
  if (!products.includes(product) || !countries.includes(country)) {
    found.push("store_prices.issue.override_target");
  }
  if (amount === null || (reference !== null && amount > reference)) {
    found.push("store_prices.issue.override_price");
  }
  return found;
}

export function withReference(prices: StorePrices, product: string, usd: number): StorePrices {
  return { ...prices, referenceUsd: { ...prices.referenceUsd, [product]: usd } };
}

export function withTierPercent(prices: StorePrices, tier: string, percent: number): StorePrices {
  return { ...prices, tiers: { ...prices.tiers, [tier]: percent } };
}

export function withoutTier(prices: StorePrices, tier: string): StorePrices {
  const tiers = { ...prices.tiers };
  delete tiers[tier];
  return { ...prices, tiers };
}

export function withCountryTier(prices: StorePrices, country: string, tier: string | null): StorePrices {
  const countries = { ...prices.countries };
  if (tier === null) {
    delete countries[country];
  } else {
    countries[country] = tier;
  }
  return { ...prices, countries };
}

export function withOverride(prices: StorePrices, product: string, country: string, usd: number | null): StorePrices {
  const byCountry = { ...prices.overrides[product] };
  if (usd === null) {
    delete byCountry[country];
  } else {
    byCountry[country] = usd;
  }
  const overrides = { ...prices.overrides, [product]: byCountry };
  if (Object.keys(byCountry).length === 0) {
    delete overrides[product];
  }
  return { ...prices, overrides };
}

export function countriesInTier(prices: StorePrices, tier: string): number {
  return Object.values(prices.countries).filter((assigned) => assigned === tier).length;
}

export function matchesCountry(code: string, name: string, query: string): boolean {
  const needle = query.trim().toLowerCase();
  return needle === "" || code.toLowerCase().includes(needle) || name.toLowerCase().includes(needle);
}

export function sameStorePrices(a: StorePrices, b: StorePrices): boolean {
  return sameContent(a, b);
}
