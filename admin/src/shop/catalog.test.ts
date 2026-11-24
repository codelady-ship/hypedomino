import assert from "node:assert/strict";
import { test } from "node:test";
import {
  allowedCurrencies,
  periodStart,
  rotationCandidates,
  rotationFits,
  toggledOffer,
  withRotation,
  withoutRotation,
  type Rotation,
  type RotationRules,
  changedOffers,
  windowEndsBeforeStart,
  windowMinute,
  withWindow,
  chestCandidates,
  crystalPacks,
  listsChest,
  oddsTotal,
  permanentMythics,
  priceOf,
  inEffectWithinDays,
  referencePriceProblem,
  replaceOffer,
  withPackCrystals,
  withPoolItem,
  withPrice,
  withRegionalNote,
  termsProblems,
  withEligibility,
  withLimitPerAccount,
  withReferencePrice,
  draftProblems,
  englishApproval,
  exactBundle,
  lookupProblems,
  newOffer,
  offerProblems,
  searchFrom,
  sectionsFor,
  unlistedCosmetics,
  withOffer,
  emptyShopCatalog,
  withoutOffer,
  isUnpublishedOffer,
  variantProblems,
  previewVariants,
  withVariants,
  editedVariant,
  catalogNameKeys,
  type BundleRow,
  type NewOfferDraft,
  type CosmeticItem,
  type Offer,
  type PublishedCatalog,
  type ShopCatalog,
} from "./catalog.ts";

function offer(fields: Partial<Offer>): Offer {
  return { id: "offer.x", kind: "cosmetic", section: "event", window: { start: "2026-01-01T00:00:00Z" }, nameKey: "k", ...fields };
}

test("each offer offers only the currencies its kind and section allow", () => {
  const cases: [Partial<Offer>, string[]][] = [
    [{ kind: "chest", section: "chests" }, ["coins"]],
    [{ kind: "crystals", section: "crystals" }, ["product"]],
    [{ kind: "bundle", section: "featured" }, ["crystals", "product"]],
    [{ kind: "cosmetic", section: "daily_rotation" }, ["coins", "crystals"]],
    [{ kind: "cosmetic", section: "event" }, ["coins", "crystals"]],
    [{ kind: "pass_levels", section: "pass" }, ["product", "coins", "crystals"]],
  ];
  for (const [fields, want] of cases) {
    assert.deepEqual(allowedCurrencies(offer(fields)), want, JSON.stringify(fields));
  }
});

test("a new price replaces every other price and keeps only its own reference price", () => {
  const coins = offer({ priceCoins: 400, referencePriceCoins: 500, referencePriceCrystals: 70 });
  assert.deepEqual(priceOf(coins), { currency: "coins", amount: 400, productId: "" });
  const crystals = withPrice(coins, { currency: "crystals", amount: 60, productId: "" });
  assert.equal(crystals.priceCrystals, 60);
  assert.equal(crystals.priceCoins, undefined);
  assert.equal(crystals.referencePriceCoins, undefined);
  assert.equal(crystals.referencePriceCrystals, 70);
  const product = withPrice(crystals, { currency: "product", amount: 0, productId: "offer_starter" });
  assert.deepEqual(priceOf(product), { currency: "product", amount: 0, productId: "offer_starter" });
  assert.equal(product.priceCrystals, undefined);
  assert.equal(product.referencePriceCrystals, undefined);
  assert.deepEqual(priceOf(withPrice(product, { currency: "coins", amount: 300, productId: "" })), { currency: "coins", amount: 300, productId: "" });
});

test("a regional override note belongs to a store product price only", () => {
  const product = withRegionalNote(offer({ productId: "offer_starter" }), "TR override approved 2026-10-30");
  assert.equal(product.regionalOverrideNote, "TR override approved 2026-10-30");
  assert.equal(withPrice(product, { currency: "product", amount: 0, productId: "offer_bundle" }).regionalOverrideNote, "TR override approved 2026-10-30");
  assert.equal("regionalOverrideNote" in withPrice(product, { currency: "crystals", amount: 60, productId: "" }), false);
  assert.equal("regionalOverrideNote" in withRegionalNote(product, "  "), false);
});

test("crystal packs list in tier order and change only their crystals", () => {
  const tier2 = offer({ id: "offer.crystals_tier2", kind: "crystals", section: "crystals", productId: "crystals_tier2", bundle: { id: "bundle.crystals_tier2", revision: 1, crystals: 550 } });
  const tier1 = offer({ id: "offer.crystals_tier1", kind: "crystals", section: "crystals", productId: "crystals_tier1", bundle: { id: "bundle.crystals_tier1", revision: 1, crystals: 100 } });
  const catalog: ShopCatalog = { schema: 1, offers: [tier2, offer({}), tier1], rotations: [] };
  assert.deepEqual(crystalPacks(catalog).map((pack) => pack.productId), ["crystals_tier1", "crystals_tier2"]);
  const changed = replaceOffer(catalog, withPackCrystals(tier2, 600));
  assert.equal(changed.offers[0].bundle?.crystals, 600);
  assert.equal(changed.offers[0].bundle?.id, "bundle.crystals_tier2");
  assert.equal(catalog.offers[0].bundle?.crystals, 550);
});

test("the chest pool editor offers only live common and rare items", () => {
  const items: CosmeticItem[] = [
    { id: "avatar.paper_kite", type: "avatar", rarity: "common", availability: "live", sources: [{ type: "chest", ref: "offer.chest_avatar" }] },
    { id: "avatar.moon_owl", type: "avatar", rarity: "rare", availability: "live", sources: [] },
    { id: "frame.season_1_top", type: "frame", rarity: "epic", availability: "live", sources: [] },
    { id: "avatar.old_logo", type: "avatar", rarity: "rare", availability: "retired", sources: [] },
  ];
  assert.deepEqual(chestCandidates(items).map((item) => item.id), ["avatar.paper_kite", "avatar.moon_owl"]);
  const chest = offer({ id: "offer.chest_avatar", kind: "chest", section: "chests", priceCoins: 500, bundle: { id: "bundle.chest_avatar", revision: 1, cosmetics: ["avatar.paper_kite"] } });
  assert.equal(listsChest(items[0], chest), true);
  assert.equal(listsChest(items[1], chest), false);
  const added = withPoolItem(chest, "avatar.moon_owl", true);
  assert.deepEqual(added.bundle?.cosmetics, ["avatar.paper_kite", "avatar.moon_owl"]);
  assert.deepEqual(withPoolItem(withPoolItem(added, "avatar.moon_owl", true), "avatar.paper_kite", false).bundle?.cosmetics, ["avatar.moon_owl"]);
  assert.equal(oddsTotal({ rarityPercent: { common: 85, rare: 15 }, rarePityCount: 10 }), 100);
  assert.equal(oddsTotal({ rarityPercent: { common: 85 }, rarePityCount: 10 }), 85);
});

test("a window instant shows as its UTC minute and an empty end stays empty", () => {
  assert.equal(windowMinute("2026-11-04T09:30:00Z"), "2026-11-04T09:30");
  assert.equal(windowMinute("2026-11-04T13:30:00+04:00"), "2026-11-04T09:30");
  assert.equal(windowMinute(undefined), "");
  assert.equal(windowMinute("not a time"), "");
});

test("editing a window sets UTC instants and an empty end removes the end", () => {
  const start = offer({ window: { start: "2026-01-01T00:00:00Z", end: "2026-02-01T00:00:00Z" } });
  assert.deepEqual(withWindow(start, "2026-11-10T08:00", "2026-11-17T08:00").window, { start: "2026-11-10T08:00:00Z", end: "2026-11-17T08:00:00Z" });
  assert.deepEqual(withWindow(start, "2026-11-10T08:00", "").window, { start: "2026-11-10T08:00:00Z" });
  assert.deepEqual(withWindow(start, "", "").window, { start: "2026-01-01T00:00:00Z" });
});

test("a window must end after it starts", () => {
  assert.equal(windowEndsBeforeStart(offer({ window: { start: "2026-11-10T08:00:00Z" } })), false);
  assert.equal(windowEndsBeforeStart(offer({ window: { start: "2026-11-10T08:00:00Z", end: "2026-11-10T09:00:00Z" } })), false);
  assert.equal(windowEndsBeforeStart(offer({ window: { start: "2026-11-10T08:00:00Z", end: "2026-11-10T08:00:00Z" } })), true);
});

test("a mythic item is flagged unless its offer rotates or its featured or event window ends", () => {
  const items: CosmeticItem[] = [
    { id: "table.night_market", type: "table", rarity: "mythic", availability: "live", sources: [] },
    { id: "avatar.moon_owl", type: "avatar", rarity: "rare", availability: "live", sources: [] },
  ];
  const bundle = { id: "bundle.night", revision: 1, cosmetics: ["avatar.moon_owl", "table.night_market"] };
  const end = "2027-01-01T00:00:00Z";
  const cases: [Partial<Offer>, string[]][] = [
    [{ section: "weekly_rotation" }, []],
    [{ section: "daily_rotation" }, []],
    [{ section: "featured", window: { start: "2026-01-01T00:00:00Z", end } }, []],
    [{ section: "event", window: { start: "2026-01-01T00:00:00Z", end } }, []],
    [{ section: "featured" }, ["table.night_market"]],
    [{ section: "event" }, ["table.night_market"]],
    [{ kind: "pass", section: "pass", window: { start: "2026-01-01T00:00:00Z", end } }, ["table.night_market"]],
  ];
  for (const [fields, want] of cases) {
    assert.deepEqual(permanentMythics(offer({ kind: "bundle", bundle, ...fields }), items), want, JSON.stringify(fields));
  }
  assert.deepEqual(permanentMythics(offer({ section: "featured", bundle: { id: "bundle.owl", revision: 1, cosmetics: ["avatar.moon_owl"] } }), items), []);
});

test("the versions in effect within the window start with the one in effect at its start", () => {
  const version = (config_version: number, published_at: string): PublishedCatalog => ({
    config_version,
    published_at,
    catalog: { schema: 1, offers: [offer({ priceCoins: config_version })], rotations: [] },
  });
  const newestFirst = [version(4, "2026-11-14T00:00:00Z"), version(3, "2026-11-04T00:00:00Z"), version(2, "2026-10-10T00:00:00Z"), version(1, "2026-10-05T00:00:00Z")];
  const now = new Date("2026-11-14T12:00:00Z");
  const cases: [number, number[]][] = [
    [30, [4, 3, 2]],
    [5, [4, 3]],
    [60, [4, 3, 2, 1]],
  ];
  for (const [days, want] of cases) {
    assert.deepEqual(
      inEffectWithinDays(newestFirst, days, now).map((catalog) => catalog.offers[0].priceCoins),
      want,
      String(days),
    );
  }
});

test("a struck-through price must be the lowest earlier price of its offer", () => {
  const coins = (priceCoins: number, referencePriceCoins?: number) => offer({ priceCoins, referencePriceCoins });
  const crystals = (priceCrystals: number, referencePriceCrystals?: number) => offer({ priceCrystals, referencePriceCrystals });
  const versions = (...offers: Offer[][]): ShopCatalog[] => offers.map((listed) => ({ schema: 1, offers: listed, rotations: [] }));
  const cases: [string, Offer, ShopCatalog[], { reference: number; lowest: number | null } | null][] = [
    ["no reference price", coins(3000), [], null],
    ["the only earlier price", coins(3000, 3500), versions([coins(3500)]), null],
    ["the lowest of several", coins(3000, 3200), versions([coins(3500)], [coins(3200)], [coins(4000)]), null],
    ["above the lowest", coins(3000, 3500), versions([coins(3500)], [coins(3200)]), { reference: 3500, lowest: 3200 }],
    ["below every earlier price", coins(3000, 3400), versions([coins(3500)]), { reference: 3400, lowest: 3500 }],
    ["no earlier version", coins(3000, 3500), [], { reference: 3500, lowest: null }],
    ["offer absent from the window", coins(3000, 3500), versions([]), { reference: 3500, lowest: null }],
    ["earlier price in another currency", coins(3000, 3500), versions([crystals(400)]), { reference: 3500, lowest: null }],
    ["running discount republished unchanged", coins(3000, 3500), versions([coins(3000, 3500)], [coins(3500)]), null],
    ["running discount lowered again", coins(2500, 3500), versions([coins(3000, 3500)], [coins(3500)]), { reference: 3500, lowest: 3000 }],
    ["crystal reference", crystals(400, 500), versions([crystals(500)]), null],
  ];
  for (const [name, candidate, inEffect, want] of cases) {
    assert.deepEqual(referencePriceProblem(candidate, inEffect), want, name);
  }
});

test("a version lists the offers it added or changed against the one before", () => {
  const older: ShopCatalog = { schema: 1, offers: [offer({ id: "offer.a" }), offer({ id: "offer.b" })], rotations: [] };
  const newer: ShopCatalog = { schema: 1, offers: [offer({ id: "offer.a" }), offer({ id: "offer.b", priceCoins: 900 }), offer({ id: "offer.c" })], rotations: [] };
  assert.deepEqual(changedOffers(older, newer), ["offer.b", "offer.c"]);
  assert.deepEqual(changedOffers(null, older), ["offer.a", "offer.b"]);
});

const rules: RotationRules = { daily_slots: 2, weekly_slots: 1, reset_hour_utc: 4 };

test("a rotation starts at the reset hour, and a weekly one only on a Monday", () => {
  assert.equal(periodStart("daily", "2026-11-04", rules), "2026-11-04T04:00:00Z");
  assert.equal(periodStart("weekly", "2026-11-09", rules), "2026-11-09T04:00:00Z");
  assert.equal(periodStart("weekly", "2026-11-04", rules), null);
  assert.equal(periodStart("daily", "", rules), null);
});

test("only offers of the period's rotation section can fill its slots", () => {
  const catalog: ShopCatalog = { schema: 1, offers: [offer({ id: "offer.d", section: "daily_rotation" }), offer({ id: "offer.w", section: "weekly_rotation" }), offer({ id: "offer.e" })], rotations: [] };
  assert.deepEqual(rotationCandidates(catalog, "daily"), ["offer.d"]);
  assert.deepEqual(rotationCandidates(catalog, "weekly"), ["offer.w"]);
});

test("a rotation fills between one offer and the period's slots, in the order picked", () => {
  const empty: Rotation = { period: "daily", start: "2026-11-04T04:00:00Z", offers: [] };
  assert.equal(rotationFits(empty, rules), false);
  const two = toggledOffer(toggledOffer(empty, "offer.b"), "offer.a");
  assert.deepEqual(two.offers, ["offer.b", "offer.a"]);
  assert.equal(rotationFits(two, rules), true);
  assert.equal(rotationFits(toggledOffer(two, "offer.c"), rules), false);
  assert.deepEqual(toggledOffer(two, "offer.b").offers, ["offer.a"]);
});

test("one rotation per period start, kept in date order, and removable", () => {
  const base: ShopCatalog = { schema: 1, offers: [], rotations: [] };
  const later: Rotation = { period: "daily", start: "2026-11-05T04:00:00Z", offers: ["offer.a"] };
  const earlier: Rotation = { period: "daily", start: "2026-11-04T04:00:00.000Z", offers: ["offer.b"] };
  const both = withRotation(withRotation(base, later), earlier);
  assert.deepEqual(both.rotations.map((rotation) => rotation.start), [earlier.start, later.start]);
  const replaced = withRotation(both, { ...earlier, start: "2026-11-04T04:00:00Z", offers: ["offer.c"] });
  assert.equal(replaced.rotations.length, 2);
  assert.deepEqual(withoutRotation(replaced, later).rotations.map((rotation) => rotation.offers), [["offer.c"]]);
});

test("a struck-through price is set in the currency of the price and cleared when empty", () => {
  const coins = withReferencePrice(offer({ priceCoins: 400, referencePriceCrystals: 70 }), 500);
  assert.equal(coins.referencePriceCoins, 500);
  assert.equal("referencePriceCrystals" in coins, false);
  assert.equal(withReferencePrice(offer({ priceCrystals: 60 }), 80).referencePriceCrystals, 80);
  assert.equal("referencePriceCoins" in withReferencePrice(coins, 0), false);
  assert.equal("referencePriceCoins" in withReferencePrice(coins, Number.NaN), false);
  const product = withReferencePrice(offer({ productId: "offer_starter" }), 500);
  assert.equal("referencePriceCoins" in product || "referencePriceCrystals" in product, false);
});

test("a purchase limit and eligibility keep only positive whole values", () => {
  assert.equal(withLimitPerAccount(offer({}), 3).limitPerAccount, 3);
  assert.equal("limitPerAccount" in withLimitPerAccount(offer({ limitPerAccount: 3 }), 0), false);
  assert.equal("limitPerAccount" in withLimitPerAccount(offer({ limitPerAccount: 3 }), 1.5), false);
  assert.deepEqual(withEligibility(offer({}), 72, true).eligibility, { maxAccountAgeHours: 72, firstPurchaseOnly: true });
  assert.deepEqual(withEligibility(offer({}), 0, true).eligibility, { firstPurchaseOnly: true });
  assert.equal("eligibility" in withEligibility(offer({ eligibility: { maxAccountAgeHours: 72 } }), 0, false), false);
});

test("purchase terms the server refuses are flagged", () => {
  const cases: [Partial<Offer>, string[]][] = [
    [{ kind: "cosmetic", priceCoins: 400, limitPerAccount: 1 }, []],
    [{ kind: "cosmetic", priceCoins: 400 }, ["cosmetic_limit_one"]],
    [{ kind: "cosmetic", priceCoins: 400, limitPerAccount: 2 }, ["cosmetic_limit_one"]],
    [{ kind: "chest", section: "chests", priceCoins: 400 }, []],
    [{ kind: "chest", section: "chests", priceCoins: 400, limitPerAccount: 1 }, ["chest_without_terms"]],
    [{ kind: "chest", section: "chests", priceCoins: 400, eligibility: { firstPurchaseOnly: true } }, ["chest_without_terms"]],
    [{ kind: "bundle", priceCrystals: 60, referencePriceCrystals: 60 }, ["reference_not_above_price"]],
    [{ kind: "bundle", priceCrystals: 60, referencePriceCrystals: 80 }, []],
    [{ kind: "bundle", productId: "offer_starter", eligibility: { maxAccountAgeHours: 72, firstPurchaseOnly: true } }, []],
  ];
  for (const [fields, want] of cases) {
    assert.deepEqual(termsProblems(offer(fields)), want, JSON.stringify(fields));
  }
});

test("a new offer takes only the sections that hold its kind", () => {
  assert.deepEqual(sectionsFor("cosmetic"), ["featured", "daily_rotation", "weekly_rotation", "event"]);
  assert.deepEqual(sectionsFor("chest"), ["chests"]);
  assert.deepEqual(sectionsFor("crystals"), ["featured", "crystals"]);
  assert.deepEqual(sectionsFor("pass_levels"), ["featured", "pass"]);
  assert.deepEqual(sectionsFor(""), []);
});

test("a new offer draft is refused for its id, kind and section, bundle and name key", () => {
  const catalog: ShopCatalog = { schema: 1, offers: [offer({ id: "offer.fox" })], rotations: [] };
  const draft: NewOfferDraft = { id: "offer.wolf_2", kind: "cosmetic", section: "event", bundleId: "bundle.wolf", nameKey: "offer.wolf.name" };
  const cases: [Partial<NewOfferDraft>, string[]][] = [
    [{}, []],
    [{ id: "wolf" }, ["id_format"]],
    [{ id: "offer.Wolf" }, ["id_format"]],
    [{ id: "offer." }, ["id_format"]],
    [{ id: "offer.fox" }, ["id_taken"]],
    [{ section: "chests" }, ["section_kind"]],
    [{ kind: "", section: "" }, ["section_kind"]],
    [{ bundleId: " " }, ["bundle_required"]],
    [{ kind: "pass", section: "pass", bundleId: "" }, []],
    [{ nameKey: "" }, ["name_required"]],
  ];
  for (const [fields, want] of cases) {
    assert.deepEqual(draftProblems({ ...draft, ...fields }, catalog), want, JSON.stringify(fields));
  }
});

test("a new offer copies an existing reward bundle used the same everywhere", () => {
  const row = (fields: Partial<BundleRow>): BundleRow => ({ id: "bundle.wolf", revision: 3, coins: 0, cosmetics: ["avatar.wolf"], medals: null, passXp: 0, differs: false, ...fields });
  assert.deepEqual(exactBundle([row({ id: "bundle.wolf_pack" }), row({})], "bundle.wolf"), { id: "bundle.wolf", revision: 3, coins: 0, cosmetics: ["avatar.wolf"], medals: [], passXp: 0 });
  assert.equal(exactBundle([row({ id: "bundle.wolf_pack" })], "bundle.wolf"), "unknown");
  assert.equal(exactBundle([row({ differs: true }), row({ cosmetics: ["frame.wolf"], differs: true })], "bundle.wolf"), "differs");
  assert.equal(searchFrom("bundle.wolf"), "bundle.wol");
});

test("a new offer name key needs an approved en value", () => {
  assert.equal(englishApproval(["offer.wolf.name"], "offer.wolf.name"), "not_approved");
  assert.equal(englishApproval([], "offer.fox.name"), "approved");
  assert.equal(englishApproval(["offer.wolf.name_long"], "offer.wolf.name"), "approved");
  assert.deepEqual(lookupProblems({ id: "bundle.wolf", revision: 1 }, "approved"), []);
  assert.deepEqual(lookupProblems(null, "approved"), []);
  assert.deepEqual(lookupProblems("unknown", "not_approved"), ["bundle_unknown", "name_not_approved"]);
  assert.deepEqual(lookupProblems("unchecked", "unchecked"), ["bundle_unchecked", "name_unchecked"]);
  assert.deepEqual(lookupProblems("differs", "approved"), ["bundle_differs"]);
});

test("an added offer has no price and no window until the admin sets them", () => {
  const draft: NewOfferDraft = { id: "offer.wolf", kind: "cosmetic", section: "event", bundleId: "bundle.wolf", nameKey: " offer.wolf.name " };
  const bundle = { id: "bundle.wolf", revision: 3, coins: 0, cosmetics: ["avatar.wolf"], medals: [], passXp: 0 };
  const added = newOffer(draft, bundle);
  assert.deepEqual(added, { id: "offer.wolf", kind: "cosmetic", section: "event", bundle, window: { start: "" }, nameKey: "offer.wolf.name" });
  assert.deepEqual(offerProblems(added), ["price_required", "window_start_required"]);
  assert.equal("bundle" in newOffer({ ...draft, kind: "pass", section: "pass", bundleId: "" }, null), false);
  const catalog = withOffer({ schema: 1, offers: [offer({})], rotations: [] }, added);
  assert.deepEqual(catalog.offers.map((listed) => listed.id), ["offer.x", "offer.wolf"]);
});

test("the first catalog starts with no offers and takes a new offer that is removable until published", () => {
  const first = emptyShopCatalog();
  assert.deepEqual(first, { schema: 1, offers: [], rotations: [] });
  const draft: NewOfferDraft = { id: "offer.wolf", kind: "cosmetic", section: "event", bundleId: "bundle.wolf", nameKey: "offer.wolf.name" };
  assert.deepEqual(draftProblems(draft, first), []);
  const added = withOffer(first, newOffer(draft, { id: "bundle.wolf", revision: 1, cosmetics: ["avatar.wolf"] }));
  assert.equal(isUnpublishedOffer(first, "offer.wolf"), true);
  assert.deepEqual(withoutOffer(added, "offer.wolf"), first);
});

test("only an offer missing from the published catalog is removed, from its offers and rotations", () => {
  const published: ShopCatalog = { schema: 1, offers: [offer({ id: "offer.daily_a", section: "daily" })], rotations: [] };
  const added = offer({ id: "offer.daily_b", section: "daily" });
  const rotation: Rotation = { period: "daily", start: "2026-02-02T00:00:00Z", offers: ["offer.daily_a", "offer.daily_b"] };
  const draft = { ...withOffer(published, added), rotations: [rotation] };
  assert.equal(isUnpublishedOffer(published, "offer.daily_b"), true);
  assert.equal(isUnpublishedOffer(published, "offer.daily_a"), false);
  const removed = withoutOffer(draft, "offer.daily_b");
  assert.deepEqual(removed.offers.map((listed) => listed.id), ["offer.daily_a"]);
  assert.deepEqual(removed.rotations, [{ ...rotation, offers: ["offer.daily_a"] }]);
});

test("offer prices and bundle contents the server refuses are flagged", () => {
  const bundle = (fields: Partial<NonNullable<Offer["bundle"]>>) => ({ id: "bundle.x", revision: 1, ...fields });
  const cases: [Partial<Offer>, string[]][] = [
    [{ priceCoins: 400, bundle: bundle({ cosmetics: ["avatar.fox"] }) }, []],
    [{ bundle: bundle({ cosmetics: ["avatar.fox"] }) }, ["price_required"]],
    [{ priceCoins: 400, priceCrystals: 60, bundle: bundle({ cosmetics: ["avatar.fox"] }) }, ["price_required"]],
    [{ section: "featured", productId: "Offer-Fox", bundle: bundle({ cosmetics: ["avatar.fox"] }) }, ["price_required"]],
    [{ section: "featured", priceCoins: 400, bundle: bundle({ cosmetics: ["avatar.fox"] }) }, ["price_required"]],
    [{ section: "event", productId: "offer_fox", bundle: bundle({ cosmetics: ["avatar.fox"] }) }, ["price_required"]],
    [{ priceCoins: 400, window: { start: "" }, bundle: bundle({ cosmetics: ["avatar.fox"] }) }, ["window_start_required"]],
    [{ priceCoins: 400, bundle: bundle({ cosmetics: ["avatar.fox"], medals: ["medal.fox"] }) }, ["bundle_medal"]],
    [{ priceCoins: 400, bundle: bundle({ cosmetics: ["avatar.fox"], crystals: 10 }) }, ["bundle_currency", "cosmetic_one_item"]],
    [{ priceCoins: 400, bundle: bundle({ cosmetics: ["avatar.fox", "frame.fox"] }) }, ["cosmetic_one_item"]],
    [{ kind: "bundle", priceCrystals: 60, bundle: bundle({ cosmetics: ["avatar.fox", "frame.fox"], passXp: 100 }) }, []],
    [{ kind: "bundle", priceCrystals: 60, bundle: bundle({ passXp: 100 }) }, ["bundle_needs_cosmetic"]],
    [{ kind: "bundle", section: "featured", productId: "offer_starter", bundle: bundle({ crystals: 300, cosmetics: ["avatar.fox"] }) }, []],
    [{ kind: "bundle", section: "featured", productId: "offer_starter", bundle: bundle({ coins: 300, cosmetics: ["avatar.fox"] }) }, ["bundle_currency"]],
    [{ kind: "crystals", section: "crystals", productId: "crystals_tier1", bundle: bundle({ crystals: 100 }) }, []],
    [{ kind: "crystals", section: "crystals", productId: "crystals_tier1", bundle: bundle({ crystals: 100, cosmetics: ["avatar.fox"] }) }, ["crystal_pack_content"]],
    [{ kind: "chest", section: "chests", priceCoins: 500, bundle: bundle({ cosmetics: ["avatar.fox"] }) }, []],
    [{ kind: "chest", section: "chests", priceCoins: 500, bundle: bundle({ cosmetics: [] }) }, ["chest_content"]],
    [{ kind: "pass", section: "pass", productId: "pass_premium" }, []],
    [{ kind: "pass_levels", section: "pass", priceCrystals: 60, bundle: bundle({ passXp: 100 }) }, []],
  ];
  for (const [fields, want] of cases) {
    assert.deepEqual(offerProblems(offer(fields)), want, JSON.stringify(fields));
  }
});

test("a cosmetic sold by an offer lists that offer as its shop source", () => {
  const item = (id: string, ref: string): CosmeticItem => ({ id, type: "avatar", rarity: "rare", availability: "live", sources: [{ type: "shop", ref }] });
  const items = [item("avatar.fox", "offer.x"), item("avatar.wolf", "offer.other")];
  const selling = (cosmetics: string[]) => offer({ bundle: { id: "bundle.x", revision: 1, cosmetics } });
  assert.deepEqual(unlistedCosmetics(selling(["avatar.fox"]), items), []);
  assert.deepEqual(unlistedCosmetics(selling(["avatar.fox", "avatar.wolf", "avatar.bear"]), items), ["avatar.wolf", "avatar.bear"]);
  assert.deepEqual(unlistedCosmetics({ ...selling(["avatar.wolf"]), kind: "chest" }, items), []);
});

test("a variant changes its name key or its position and nothing else", () => {
  assert.deepEqual(variantProblems(offer({ variants: [{ id: "bold", nameKey: "k2" }, { id: "first", position: 1 }] })), []);
  assert.deepEqual(variantProblems(offer({ variants: [{ id: "Bold", nameKey: "k2" }] })), ["variant_id"]);
  assert.deepEqual(variantProblems(offer({ variants: [{ id: "bold", nameKey: "k2" }, { id: "bold", position: 2 }] })), ["variant_twice"]);
  assert.deepEqual(variantProblems(offer({ variants: [{ id: "bold", position: 1.5 }] })), ["variant_position"]);
  assert.deepEqual(variantProblems(offer({ variants: [{ id: "bold" }] })), ["variant_no_change"]);
  assert.deepEqual(variantProblems(offer({ variants: [{ id: "bold", nameKey: "k" }] })), ["variant_no_change"]);
});

test("a variant edit keeps only what it changes and an offer without variants drops the field", () => {
  assert.deepEqual(editedVariant({ id: "bold" }, " k2 ", 0), { id: "bold", nameKey: "k2" });
  assert.deepEqual(editedVariant({ id: "bold", nameKey: "k2" }, "", 3), { id: "bold", position: 3 });
  assert.deepEqual(editedVariant({ id: "bold", position: 3 }, "", Number.NaN), { id: "bold" });
  assert.equal("variants" in withVariants(offer({ variants: [{ id: "bold", nameKey: "k2" }] }), []), false);
  assert.deepEqual(withVariants(offer({}), [{ id: "bold", nameKey: "k2" }]).variants, [{ id: "bold", nameKey: "k2" }]);
});

test("the name keys of a catalog include those of its variants", () => {
  const catalog: ShopCatalog = { ...emptyShopCatalog(), offers: [offer({ variants: [{ id: "bold", nameKey: "k2" }, { id: "first", position: 1 }] }), offer({ id: "offer.y", nameKey: "y" })] };
  assert.deepEqual(catalogNameKeys(catalog), ["k", "k2", "y"]);
});

test("a preview asks only for variants the draft still has", () => {
  const catalog: ShopCatalog = { ...emptyShopCatalog(), offers: [offer({ variants: [{ id: "bold", nameKey: "k2" }] }), offer({ id: "offer.y", nameKey: "y" })] };
  assert.deepEqual(previewVariants(catalog, { "offer.x": "bold" }), { "offer.x": "bold" });
  assert.deepEqual(previewVariants(catalog, { "offer.x": "gone", "offer.y": "bold", "offer.removed": "bold" }), {});
  assert.deepEqual(previewVariants(catalog, {}), {});
});
