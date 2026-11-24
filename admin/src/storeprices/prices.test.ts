import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import test from "node:test";
import {
  centsOf,
  countriesInTier,
  csvFileName,
  formatUsd,
  matchesCountry,
  pricedCountries,
  sameStorePrices,
  storeCsv,
  storePriceIssues,
  targetOf,
  withCountryTier,
  withOverride,
  withoutTier,
  withReference,
  withTierPercent,
  type StorePrices,
} from "./prices.ts";

const example: StorePrices = JSON.parse(readFileSync(new URL("../../../docs/economy/examples/store_prices.json", import.meta.url), "utf8"));
const products = Object.keys(example.referenceUsd);
const countries = ["AZ", "TR", "DE", "US", "PL", "IN", "GB"];

test("a listed country pays its tier percent of the reference, rounded to the cent", () => {
  assert.deepEqual(targetOf(example, "crystals_tier1", "AZ"), { product: "crystals_tier1", country: "AZ", tier: "mid", cents: 59 });
  assert.equal(targetOf(example, "crystals_tier5", "PL")?.cents, 3999);
  assert.equal(targetOf(example, "offer_starter", "IN")?.cents, 90);
});

test("Azerbaijan and Turkey stay at sixty percent", () => {
  const sixty = { crystals_tier1: 59, crystals_tier2: 299, crystals_tier3: 599, crystals_tier4: 1199, crystals_tier5: 2999, pass_premium: 299, pass_premium_plus: 599, offer_starter: 119 };
  for (const country of ["AZ", "TR"]) {
    for (const [product, cents] of Object.entries(sixty)) {
      assert.equal(targetOf(example, product, country)?.cents, cents, `${product} ${country}`);
    }
  }
});

test("an override wins over the tier and an unlisted country has no target", () => {
  const edited = withOverride(withOverride(example, "pass_premium", "AZ", 2.49), "pass_premium", "DE", 3.99);
  assert.deepEqual(targetOf(edited, "pass_premium", "AZ"), { product: "pass_premium", country: "AZ", cents: 249, override: true });
  assert.equal(targetOf(edited, "pass_premium", "DE")?.cents, 399);
  assert.equal(targetOf(edited, "pass_premium", "US"), null);
  assert.equal(targetOf(example, "pass_premium", "DE"), null);
  assert.deepEqual(pricedCountries(edited).filter((code) => ["AZ", "DE", "US"].includes(code)), ["AZ", "DE"]);
});

test("removing the last override of a product drops the product", () => {
  const added = withOverride(example, "pass_premium", "AZ", 2.49);
  assert.deepEqual(withOverride(added, "pass_premium", "AZ", null).overrides, {});
  assert.deepEqual(example.overrides, {});
});

test("assigning and clearing a tier leaves the other countries", () => {
  const listed = withCountryTier(example, "US", "full");
  assert.equal(targetOf(listed, "crystals_tier5", "US")?.cents, 4999);
  const cleared = withCountryTier(listed, "AZ", null);
  assert.equal(targetOf(cleared, "pass_premium", "AZ"), null);
  assert.equal(cleared.countries.TR, "mid");
  assert.equal(example.countries.AZ, "mid");
});

test("a tier percent change moves every country in it", () => {
  const edited = withTierPercent(example, "mid", 50);
  assert.equal(targetOf(edited, "pass_premium", "AZ")?.cents, 250);
  assert.equal(targetOf(edited, "pass_premium", "PL")?.cents, 399);
  assert.equal(countriesInTier(example, "mid"), 37);
  assert.equal(countriesInTier(example, "high"), 17);
  assert.equal(countriesInTier(example, "low"), 26);
});

test("the CSV lists product, country and USD target with a header", () => {
  const csv = storeCsv([targetOf(example, "pass_premium", "AZ")!, targetOf(withOverride(example, "crystals_tier1", "TR", 0.49), "crystals_tier1", "TR")!]);
  assert.equal(csv, "product_id,country_code,target_usd\npass_premium,AZ,2.99\ncrystals_tier1,TR,0.49\n");
  assert.equal(storeCsv([]), "product_id,country_code,target_usd\n");
  assert.equal(csvFileName("google_play", 4), "store-prices-google-play-v4.csv");
  assert.equal(csvFileName("app_store", 4), "store-prices-app-store-v4.csv");
});

test("money is a positive amount with at most two decimals", () => {
  assert.equal(centsOf(4.99), 499);
  assert.equal(centsOf(19.99), 1999);
  assert.equal(centsOf(0), null);
  assert.equal(centsOf(4.995), null);
  assert.equal(centsOf(Number.NaN), null);
  assert.equal(formatUsd(59), "0.59");
  assert.equal(formatUsd(4999), "49.99");
});

test("the issues follow the server checks", () => {
  const all = countries.concat(Object.keys(example.countries));
  assert.deepEqual(storePriceIssues(example, products, all), []);
  assert.deepEqual(storePriceIssues(withTierPercent(example, "low", 0), products, all), ["store_prices.issue.percent"]);
  assert.deepEqual(storePriceIssues(withTierPercent(example, "Cheap", 40), products, all), ["store_prices.issue.tier_name"]);
  assert.deepEqual(storePriceIssues(withoutTier(example, "mid"), products, all), ["store_prices.issue.unknown_tier"]);
  assert.deepEqual(storePriceIssues(withCountryTier(example, "RU", "mid"), products, all), ["store_prices.issue.country"]);
  assert.deepEqual(storePriceIssues(withReference(example, "pass_premium", 0), products, all), ["store_prices.issue.reference"]);
  assert.deepEqual(storePriceIssues(withOverride(example, "pass_premium", "AZ", 5.99), products, all), ["store_prices.issue.override_price"]);
  assert.deepEqual(storePriceIssues(withOverride(example, "pass_premium", "RU", 2.99), products, all), ["store_prices.issue.override_target"]);
});

test("the country search matches the code or the name", () => {
  assert.equal(matchesCountry("AZ", "Azerbaijan", "aze"), true);
  assert.equal(matchesCountry("AZ", "Azerbaijan", "az"), true);
  assert.equal(matchesCountry("TR", "Türkiye", "aze"), false);
  assert.equal(matchesCountry("TR", "Türkiye", "  "), true);
});

test("two configs match whatever the key order", () => {
  const reordered = { ...example, countries: Object.fromEntries(Object.entries(example.countries).reverse()) };
  assert.equal(sameStorePrices(example, reordered), true);
  assert.equal(sameStorePrices(example, withTierPercent(example, "mid", 55)), false);
});
