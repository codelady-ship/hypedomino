export type Bundle = {
  id: string;
  revision: number;
  coins?: number;
  crystals?: number;
  cosmetics?: string[];
  medals?: string[];
  passXp?: number;
};

export type Offer = {
  id: string;
  kind: string;
  section: string;
  priceCoins?: number;
  priceCrystals?: number;
  productId?: string;
  referencePriceCoins?: number;
  referencePriceCrystals?: number;
  regionalOverrideNote?: string;
  bundle?: Bundle;
  window: { start: string; end?: string };
  limitPerAccount?: number;
  eligibility?: Eligibility;
  nameKey: string;
  variants?: Variant[];
  [field: string]: unknown;
};

export type Variant = { id: string; nameKey?: string; position?: number };

export type Eligibility = { maxAccountAgeHours?: number; firstPurchaseOnly?: boolean };

export type ChestRules = {
  rarityPercent: Record<string, number>;
  rarePityCount: number;
};

export type RotationPeriod = "daily" | "weekly";

export type Rotation = { period: RotationPeriod; start: string; offers: string[] };

export type RotationRules = { daily_slots: number; weekly_slots: number; reset_hour_utc: number };

export type ShopCatalog = {
  schema: number;
  offers: Offer[];
  rotations: Rotation[];
  chests?: ChestRules;
  [field: string]: unknown;
};

export type CosmeticItem = {
  id: string;
  type: string;
  rarity: string;
  availability: string;
  sources: { type: string; ref?: string }[];
};

export type PriceCurrency = "coins" | "crystals" | "product";

export type Price = { currency: PriceCurrency; amount: number; productId: string };

export const chestRarities = ["common", "rare"] as const;

export const crystalPackProducts = ["crystals_tier1", "crystals_tier2", "crystals_tier3", "crystals_tier4", "crystals_tier5"];

const inGame: PriceCurrency[] = ["coins", "crystals"];

export function allowedCurrencies(offer: Offer): PriceCurrency[] {
  switch (true) {
    case offer.kind === "chest":
      return ["coins"];
    case offer.kind === "crystals":
      return ["product"];
    case offer.section === "featured":
      return ["crystals", "product"];
    case offer.section === "pass":
      return ["product", ...inGame];
    default:
      return inGame;
  }
}

export function priceOf(offer: Offer): Price {
  if (offer.productId) {
    return { currency: "product", amount: 0, productId: offer.productId };
  }
  if (offer.priceCrystals) {
    return { currency: "crystals", amount: offer.priceCrystals, productId: "" };
  }
  return { currency: "coins", amount: offer.priceCoins ?? 0, productId: "" };
}

export function withPrice(offer: Offer, price: Price): Offer {
  const { priceCoins, priceCrystals, productId, referencePriceCoins, referencePriceCrystals, regionalOverrideNote, ...rest } = offer;
  switch (price.currency) {
    case "product":
      return { ...rest, productId: price.productId, ...(regionalOverrideNote ? { regionalOverrideNote } : {}) };
    case "crystals":
      return { ...rest, priceCrystals: price.amount, ...(referencePriceCrystals ? { referencePriceCrystals } : {}) };
    case "coins":
      return { ...rest, priceCoins: price.amount, ...(referencePriceCoins ? { referencePriceCoins } : {}) };
  }
}

export function withRegionalNote(offer: Offer, note: string): Offer {
  const { regionalOverrideNote, ...rest } = offer;
  return note.trim() ? { ...rest, regionalOverrideNote: note } : rest;
}

function positiveWhole(value: number): number | undefined {
  return Number.isInteger(value) && value > 0 ? value : undefined;
}

export function withReferencePrice(offer: Offer, amount: number): Offer {
  const { referencePriceCoins, referencePriceCrystals, ...rest } = offer;
  const reference = positiveWhole(amount);
  const price = priceOf(offer);
  if (reference === undefined || price.currency === "product") {
    return rest;
  }
  return price.currency === "coins" ? { ...rest, referencePriceCoins: reference } : { ...rest, referencePriceCrystals: reference };
}

export function withLimitPerAccount(offer: Offer, limit: number): Offer {
  const { limitPerAccount, ...rest } = offer;
  const kept = positiveWhole(limit);
  return kept === undefined ? rest : { ...rest, limitPerAccount: kept };
}

export function withEligibility(offer: Offer, maxAccountAgeHours: number, firstPurchaseOnly: boolean): Offer {
  const { eligibility, ...rest } = offer;
  const hours = positiveWhole(maxAccountAgeHours);
  const next: Eligibility = { ...(hours === undefined ? {} : { maxAccountAgeHours: hours }), ...(firstPurchaseOnly ? { firstPurchaseOnly } : {}) };
  return Object.keys(next).length === 0 ? rest : { ...rest, eligibility: next };
}

export type TermsProblem = "cosmetic_limit_one" | "chest_without_terms" | "reference_not_above_price";

export function termsProblems(offer: Offer): TermsProblem[] {
  const problems: TermsProblem[] = [];
  if (offer.kind === "cosmetic" && offer.limitPerAccount !== 1) {
    problems.push("cosmetic_limit_one");
  }
  const eligibility = offer.eligibility ?? {};
  if (offer.kind === "chest" && (offer.limitPerAccount !== undefined || eligibility.maxAccountAgeHours !== undefined || eligibility.firstPurchaseOnly)) {
    problems.push("chest_without_terms");
  }
  const price = priceOf(offer);
  const reference = (offer.referencePriceCoins ?? 0) + (offer.referencePriceCrystals ?? 0);
  if (reference > 0 && price.currency !== "product" && reference <= price.amount) {
    problems.push("reference_not_above_price");
  }
  return problems;
}

export function emptyShopCatalog(): ShopCatalog {
  return { schema: 1, offers: [], rotations: [] };
}

export function replaceOffer(catalog: ShopCatalog, offer: Offer): ShopCatalog {
  return { ...catalog, offers: catalog.offers.map((current) => (current.id === offer.id ? offer : current)) };
}

export function crystalPacks(catalog: ShopCatalog): Offer[] {
  return catalog.offers
    .filter((offer) => offer.kind === "crystals")
    .toSorted((a, b) => crystalPackProducts.indexOf(a.productId ?? "") - crystalPackProducts.indexOf(b.productId ?? ""));
}

export function withPackCrystals(pack: Offer, crystals: number): Offer {
  return { ...pack, bundle: { ...(pack.bundle ?? { id: "", revision: 1 }), crystals } };
}

export function chestCandidates(items: CosmeticItem[]): CosmeticItem[] {
  return items.filter((item) => item.availability === "live" && (chestRarities as readonly string[]).includes(item.rarity));
}

export function listsChest(item: CosmeticItem, chest: Offer): boolean {
  return item.sources.some((source) => source.type === "chest" && source.ref === chest.id);
}

export function withPoolItem(chest: Offer, itemId: string, inPool: boolean): Offer {
  const pool = (chest.bundle?.cosmetics ?? []).filter((id) => id !== itemId);
  const cosmetics = inPool ? [...pool, itemId] : pool;
  return { ...chest, bundle: { ...(chest.bundle ?? { id: "", revision: 1 }), cosmetics } };
}

export function chestRulesOf(catalog: ShopCatalog): ChestRules {
  return catalog.chests ?? { rarityPercent: { common: 0, rare: 0 }, rarePityCount: 0 };
}

export function oddsTotal(rules: ChestRules): number {
  return chestRarities.reduce((total, rarity) => total + (rules.rarityPercent[rarity] ?? 0), 0);
}

export function windowMinute(instant: string | undefined): string {
  const date = instant ? new Date(instant) : null;
  return date && !Number.isNaN(date.getTime()) ? date.toISOString().slice(0, 16) : "";
}

export function withWindow(offer: Offer, startMinute: string, endMinute: string): Offer {
  const start = startMinute ? `${startMinute}:00Z` : offer.window.start;
  return { ...offer, window: endMinute ? { start, end: `${endMinute}:00Z` } : { start } };
}

export function windowEndsBeforeStart(offer: Offer): boolean {
  return offer.window.end !== undefined && new Date(offer.window.end) <= new Date(offer.window.start);
}

export function changedOffers(previous: ShopCatalog | null, next: ShopCatalog): string[] {
  const before = new Map((previous?.offers ?? []).map((offer) => [offer.id, JSON.stringify(offer)]));
  return next.offers.filter((offer) => before.get(offer.id) !== JSON.stringify(offer)).map((offer) => offer.id);
}

const rotationSections: Record<RotationPeriod, string> = { daily: "daily_rotation", weekly: "weekly_rotation" };

const windowedSections = ["featured", "event"];

export function timeLimited(offer: Offer): boolean {
  return Object.values(rotationSections).includes(offer.section) || (windowedSections.includes(offer.section) && offer.window.end !== undefined);
}

export function permanentMythics(offer: Offer, items: CosmeticItem[]): string[] {
  if (offer.kind === "chest" || timeLimited(offer)) {
    return [];
  }
  const mythics = new Set(items.filter((item) => item.rarity === "mythic").map((item) => item.id));
  return (offer.bundle?.cosmetics ?? []).filter((id) => mythics.has(id));
}

export type PublishedCatalog = { config_version: number; published_at: string; catalog: ShopCatalog };

export type ReferencePriceProblem = { reference: number; lowest: number | null };

const dayMs = 24 * 60 * 60 * 1000;

export function inEffectWithinDays(newestFirst: PublishedCatalog[], days: number, now: Date): ShopCatalog[] {
  const windowStart = now.getTime() - days * dayMs;
  const startsBefore = newestFirst.findIndex((version) => new Date(version.published_at).getTime() <= windowStart);
  const inEffect = startsBefore === -1 ? newestFirst : newestFirst.slice(0, startsBefore + 1);
  return inEffect.map((version) => version.catalog);
}

function samePricing(offer: Offer, earlier: Offer | undefined): boolean {
  const fields = ["priceCoins", "priceCrystals", "referencePriceCoins", "referencePriceCrystals"] as const;
  return earlier !== undefined && fields.every((field) => (offer[field] ?? 0) === (earlier[field] ?? 0));
}

export function referencePriceProblem(offer: Offer, inEffect: ShopCatalog[]): ReferencePriceProblem | null {
  const reference = (offer.referencePriceCoins ?? 0) + (offer.referencePriceCrystals ?? 0);
  const find = (catalog: ShopCatalog) => catalog.offers.find((candidate) => candidate.id === offer.id);
  if (reference === 0 || (inEffect.length > 0 && samePricing(offer, find(inEffect[0])))) {
    return null;
  }
  const field = offer.referencePriceCoins ? "priceCoins" : "priceCrystals";
  const earlier = inEffect.map((catalog) => find(catalog)?.[field] ?? 0).filter((price) => price > 0);
  const lowest = earlier.length > 0 ? Math.min(...earlier) : null;
  return lowest === reference ? null : { reference, lowest };
}

export function rotationCandidates(catalog: ShopCatalog, period: RotationPeriod): string[] {
  return catalog.offers.filter((offer) => offer.section === rotationSections[period]).map((offer) => offer.id);
}

export function slotsOf(rules: RotationRules, period: RotationPeriod): number {
  return period === "weekly" ? rules.weekly_slots : rules.daily_slots;
}

export function periodStart(period: RotationPeriod, day: string, rules: RotationRules): string | null {
  const start = new Date(`${day}T${String(rules.reset_hour_utc).padStart(2, "0")}:00:00Z`);
  if (Number.isNaN(start.getTime()) || (period === "weekly" && start.getUTCDay() !== 1)) {
    return null;
  }
  return start.toISOString().replace(".000Z", "Z");
}

function sameSlot(a: Rotation, period: RotationPeriod, start: string): boolean {
  return a.period === period && new Date(a.start).getTime() === new Date(start).getTime();
}

export function withRotation(catalog: ShopCatalog, rotation: Rotation): ShopCatalog {
  const others = catalog.rotations.filter((current) => !sameSlot(current, rotation.period, rotation.start));
  const rotations = [...others, rotation].toSorted((a, b) => a.period.localeCompare(b.period) || new Date(a.start).getTime() - new Date(b.start).getTime());
  return { ...catalog, rotations };
}

export function withoutRotation(catalog: ShopCatalog, rotation: Rotation): ShopCatalog {
  return { ...catalog, rotations: catalog.rotations.filter((current) => !sameSlot(current, rotation.period, rotation.start)) };
}

export function toggledOffer(rotation: Rotation, offerId: string): Rotation {
  const offers = rotation.offers.includes(offerId) ? rotation.offers.filter((id) => id !== offerId) : [...rotation.offers, offerId];
  return { ...rotation, offers };
}

export function rotationFits(rotation: Rotation, rules: RotationRules): boolean {
  return rotation.offers.length >= 1 && rotation.offers.length <= slotsOf(rules, rotation.period);
}

export const offerKinds = ["cosmetic", "bundle", "crystals", "pass", "pass_levels", "chest"];

const cosmeticKinds = ["cosmetic", "bundle"];

const passKinds = ["pass", "pass_levels"];

const sectionKinds: Record<string, string[]> = {
  featured: [...cosmeticKinds, "crystals", ...passKinds],
  daily_rotation: cosmeticKinds,
  weekly_rotation: cosmeticKinds,
  event: cosmeticKinds,
  chests: ["chest"],
  pass: passKinds,
  crystals: ["crystals"],
};

export function sectionsFor(kind: string): string[] {
  return Object.keys(sectionKinds).filter((section) => sectionKinds[section].includes(kind));
}

export type NewOfferDraft = { id: string; kind: string; section: string; bundleId: string; nameKey: string };

export type DraftProblem = "id_format" | "id_taken" | "section_kind" | "bundle_required" | "name_required";

const offerIdPattern = /^offer\.[a-z0-9_]+$/;

export function draftProblems(draft: NewOfferDraft, catalog: ShopCatalog): DraftProblem[] {
  const checks: [boolean, DraftProblem][] = [
    [!offerIdPattern.test(draft.id), "id_format"],
    [catalog.offers.some((offer) => offer.id === draft.id), "id_taken"],
    [!sectionsFor(draft.kind).includes(draft.section), "section_kind"],
    [draft.bundleId.trim() === "" && !passKinds.includes(draft.kind), "bundle_required"],
    [draft.nameKey.trim() === "", "name_required"],
  ];
  return checks.filter(([failed]) => failed).map(([, problem]) => problem);
}

export type BundleRow = { id: string; revision: number; coins: number; crystals?: number; cosmetics: string[] | null; medals: string[] | null; passXp: number; differs: boolean };

export type BundleLookup = Bundle | "unknown" | "differs" | "unchecked";

export function exactBundle(rows: BundleRow[], id: string): BundleLookup {
  const matches = rows.filter((row) => row.id === id);
  if (matches.length === 0) {
    return "unknown";
  }
  if (matches.length > 1 || matches[0].differs) {
    return "differs";
  }
  const { revision, coins, crystals, cosmetics, medals, passXp } = matches[0];
  return { id, revision, coins, ...(crystals ? { crystals } : {}), cosmetics: cosmetics ?? [], medals: medals ?? [], passXp };
}

export type NameApproval = "approved" | "not_approved" | "unchecked";

export function englishApproval(unapproved: readonly string[], key: string): NameApproval {
  return unapproved.includes(key) ? "not_approved" : "approved";
}

export function searchFrom(id: string): string {
  return id.slice(0, -1);
}

export type LookupProblem = "bundle_unknown" | "bundle_differs" | "bundle_unchecked" | "name_not_approved" | "name_unchecked";

const bundleProblem: Record<Exclude<BundleLookup, Bundle>, LookupProblem> = { unknown: "bundle_unknown", differs: "bundle_differs", unchecked: "bundle_unchecked" };

const nameProblem: Record<Exclude<NameApproval, "approved">, LookupProblem> = { not_approved: "name_not_approved", unchecked: "name_unchecked" };

export function lookupProblems(bundle: BundleLookup | null, name: NameApproval): LookupProblem[] {
  const problems: LookupProblem[] = typeof bundle === "string" ? [bundleProblem[bundle]] : [];
  return name === "approved" ? problems : [...problems, nameProblem[name]];
}

export function newOffer(draft: NewOfferDraft, bundle: Bundle | null): Offer {
  return { id: draft.id, kind: draft.kind, section: draft.section, ...(bundle ? { bundle } : {}), window: { start: "" }, nameKey: draft.nameKey.trim() };
}

export function withOffer(catalog: ShopCatalog, offer: Offer): ShopCatalog {
  return { ...catalog, offers: [...catalog.offers, offer] };
}

export function isUnpublishedOffer(published: ShopCatalog, offerId: string): boolean {
  return !published.offers.some((offer) => offer.id === offerId);
}

export function withoutOffer(catalog: ShopCatalog, offerId: string): ShopCatalog {
  return {
    ...catalog,
    offers: catalog.offers.filter((offer) => offer.id !== offerId),
    rotations: catalog.rotations.map((rotation) => ({ ...rotation, offers: rotation.offers.filter((id) => id !== offerId) })),
  };
}

export type OfferProblem =
  | "price_required"
  | "window_start_required"
  | "bundle_medal"
  | "bundle_currency"
  | "cosmetic_one_item"
  | "bundle_needs_cosmetic"
  | "crystal_pack_content"
  | "chest_content";

const productIdPattern = /^[a-z0-9_]+$/;

function priced(offer: Offer): boolean {
  const prices = [(offer.priceCoins ?? 0) > 0, (offer.priceCrystals ?? 0) > 0, Boolean(offer.productId)].filter(Boolean).length;
  return prices === 1 && (!offer.productId || productIdPattern.test(offer.productId)) && allowedCurrencies(offer).includes(priceOf(offer).currency);
}

function bundleProblems(offer: Offer): OfferProblem[] {
  if (passKinds.includes(offer.kind) && !offer.bundle?.id) {
    return [];
  }
  const cosmetics = offer.bundle?.cosmetics ?? [];
  const coins = offer.bundle?.coins ?? 0;
  const crystals = offer.bundle?.crystals ?? 0;
  const passXp = offer.bundle?.passXp ?? 0;
  const product = Boolean(offer.productId);
  const checks: [boolean, OfferProblem][] = [
    [(offer.bundle?.medals ?? []).length > 0, "bundle_medal"],
    [product ? coins > 0 : coins > 0 || crystals > 0, "bundle_currency"],
    [offer.kind === "cosmetic" && (cosmetics.length !== 1 || coins > 0 || crystals > 0 || passXp > 0), "cosmetic_one_item"],
    [offer.kind === "bundle" && cosmetics.length === 0, "bundle_needs_cosmetic"],
    [offer.kind === "crystals" && (crystals === 0 || !product || cosmetics.length > 0 || passXp > 0), "crystal_pack_content"],
    [offer.kind === "chest" && (passXp > 0 || cosmetics.length === 0), "chest_content"],
  ];
  return checks.filter(([failed]) => failed).map(([, problem]) => problem);
}

export function offerProblems(offer: Offer): OfferProblem[] {
  const own: OfferProblem[] = [];
  if (!priced(offer)) {
    own.push("price_required");
  }
  if (windowMinute(offer.window.start) === "") {
    own.push("window_start_required");
  }
  return [...own, ...bundleProblems(offer)];
}

export function unlistedCosmetics(offer: Offer, items: CosmeticItem[]): string[] {
  if (offer.kind === "chest") {
    return [];
  }
  const listed = (id: string) => items.some((item) => item.id === id && item.sources.some((source) => source.type === "shop" && source.ref === offer.id));
  return (offer.bundle?.cosmetics ?? []).filter((id) => !listed(id));
}

const variantIdPattern = /^[a-z0-9_]+$/;

export type VariantProblem = "variant_id" | "variant_twice" | "variant_position" | "variant_no_change";

export function variantProblems(offer: Offer): VariantProblem[] {
  const found = new Set<VariantProblem>();
  const variants = offer.variants ?? [];
  variants.forEach((variant, at) => {
    if (!variantIdPattern.test(variant.id)) found.add("variant_id");
    if (variants.findIndex((other) => other.id === variant.id) !== at) found.add("variant_twice");
    if (variant.position !== undefined && !(Number.isInteger(variant.position) && variant.position > 0)) found.add("variant_position");
    const keepsName = (variant.nameKey ?? "") === "" || variant.nameKey === offer.nameKey;
    if (keepsName && variant.position === undefined) found.add("variant_no_change");
  });
  return [...found];
}

export function withVariants(offer: Offer, next: Variant[]): Offer {
  const { variants, ...rest } = offer;
  return next.length === 0 ? rest : { ...rest, variants: next };
}

export function editedVariant(variant: Variant, nameKey: string, position: number): Variant {
  const key = nameKey.trim();
  return { id: variant.id, ...(key === "" ? {} : { nameKey: key }), ...(position === 0 || Number.isNaN(position) ? {} : { position }) };
}

export function catalogNameKeys(catalog: ShopCatalog): string[] {
  return catalog.offers.flatMap((offer) => [offer.nameKey, ...(offer.variants ?? []).flatMap((variant) => (variant.nameKey ? [variant.nameKey] : []))]);
}

export function previewVariants(catalog: ShopCatalog, chosen: Record<string, string>): Record<string, string> {
  const kept = catalog.offers.flatMap((offer) => (offer.variants ?? []).some((variant) => variant.id === chosen[offer.id]) ? [[offer.id, chosen[offer.id]] as const] : []);
  return Object.fromEntries(kept);
}
