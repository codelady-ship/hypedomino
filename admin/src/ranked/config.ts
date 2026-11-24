import { fieldChanges, type FieldChange } from "../config/changes.ts";
import { sameContent } from "../config/sameContent.ts";
import type { MessageKey } from "@/i18n/translate";
import type { Bundle } from "@/shop/catalog";

export type Mode = "1v1" | "2v2";

export type ModeConfig = {
  rankRewards?: Record<string, Bundle>;
  seasonResetTable?: Record<string, string>;
  resetTierDrop: number;
  resetFloor: string;
  resetCeiling: string;
  newPlayerPlacementCap: string;
  [field: string]: unknown;
};

export type RatingConfig = {
  schema: number;
  leaderboardTopSize: number;
  leaderboardRegions?: Record<string, string[]>;
  rankRewardCoins: Record<string, number>;
  modes: Record<Mode, ModeConfig>;
  [field: string]: unknown;
};

export const modes: readonly Mode[] = ["1v1", "2v2"];

export const structuralNumberKeys = ["divisionLp", "demotionLp", "apexRecalcHourUtc", "challengerSlots", "challengerMinLp", "grandmasterSlots", "grandmasterMinLp", "placementMatches"] as const;

export const liveNumberKeys = [
  "initialMmr",
  "mmrScale",
  "mmrK",
  "mmrKProvisional",
  "mmrProvisionalMatches",
  "rankMmrBase",
  "rankMmrStep",
  "lpBase",
  "lpGapScale",
  "lpFactorMin",
  "lpFactorMax",
  "lpChangeMin",
  "lpChangeMax",
  "placementMaxDropDivisions",
  "placementMaxClimbDivisions",
  "seasonLengthMonths",
] as const;

export type NumberKey = (typeof structuralNumberKeys)[number] | (typeof liveNumberKeys)[number];

const apexRank = "Master";

export function effectiveConfig(catalog: RatingConfig | null, defaults: RatingConfig): RatingConfig {
  if (!catalog) {
    return defaults;
  }
  return {
    ...defaults,
    ...catalog,
    rankRewardCoins: { ...defaults.rankRewardCoins, ...catalog.rankRewardCoins },
    modes: {
      "1v1": { ...defaults.modes["1v1"], ...catalog.modes?.["1v1"] },
      "2v2": { ...defaults.modes["2v2"], ...catalog.modes?.["2v2"] },
    },
  };
}

export function withMode(config: RatingConfig, mode: Mode, change: Partial<ModeConfig>): RatingConfig {
  return { ...config, modes: { ...config.modes, [mode]: { ...config.modes[mode], ...change } } };
}

export function resetRanks(divisionRanks: readonly string[]): string[] {
  return [...divisionRanks, apexRank];
}

function rankIndex(ranks: readonly string[], name: string): number {
  const wanted = name.trim().toLowerCase();
  return ranks.findIndex((rank) => rank.toLowerCase() === wanted);
}

export function generatedReset(mode: ModeConfig, divisionRanks: readonly string[]): string[] {
  const ranks = resetRanks(divisionRanks);
  const floor = rankIndex(ranks, mode.resetFloor);
  const ceiling = rankIndex(ranks, mode.resetCeiling);
  const drop = Number.isInteger(mode.resetTierDrop) ? mode.resetTierDrop : 0;
  return ranks.map((_, old) => ranks[Math.min(ceiling, Math.max(old - 4 * drop, Math.min(old, floor)))]);
}

export function effectiveReset(mode: ModeConfig, divisionRanks: readonly string[]): string[] {
  const ranks = resetRanks(divisionRanks);
  const table = generatedReset(mode, divisionRanks);
  for (const [old, reset] of Object.entries(mode.seasonResetTable ?? {})) {
    const index = rankIndex(ranks, old);
    if (index >= 0) {
      table[index] = reset;
    }
  }
  return table;
}

export function withResetRow(mode: ModeConfig, divisionRanks: readonly string[], old: string, reset: string): ModeConfig {
  const generated = generatedReset(mode, divisionRanks)[rankIndex(resetRanks(divisionRanks), old)];
  const { [old]: _replaced, ...rows } = mode.seasonResetTable ?? {};
  const table = reset === generated ? rows : { ...rows, [old]: reset };
  const { seasonResetTable: _table, ...rest } = mode;
  return Object.keys(table).length > 0 ? { ...rest, seasonResetTable: table } : (rest as ModeConfig);
}

export function isOverride(mode: ModeConfig, old: string): boolean {
  return mode.seasonResetTable?.[old] !== undefined;
}

export function resetIssues(mode: ModeConfig, divisionRanks: readonly string[]): MessageKey[] {
  const floor = divisionRanks.indexOf(mode.resetFloor);
  const ceiling = divisionRanks.indexOf(mode.resetCeiling);
  if (floor < 0 || ceiling < 0 || floor > ceiling) {
    return ["ranked.issue.floor"];
  }
  const ranks = resetRanks(divisionRanks);
  const table = effectiveReset(mode, divisionRanks).map((reset) => rankIndex(ranks, reset));
  const found = new Set<MessageKey>();
  table.forEach((reset, old) => {
    if (reset < 0 || reset >= divisionRanks.length) found.add("ranked.issue.reset_apex");
    if (reset > old) found.add("ranked.issue.reset_above");
    if (old > 0 && reset < table[old - 1]) found.add("ranked.issue.reset_order");
  });
  return [...found];
}

function whole(mode: ModeConfig, key: NumberKey): number {
  const value = mode[key];
  return typeof value === "number" && Number.isInteger(value) ? value : Number.NaN;
}

function amount(mode: ModeConfig, key: NumberKey): number {
  const value = mode[key];
  return typeof value === "number" ? value : Number.NaN;
}

export function modeIssues(mode: ModeConfig, divisionRanks: readonly string[]): MessageKey[] {
  const found: MessageKey[] = [];
  const atLeast = (keys: readonly NumberKey[], least: number) => keys.every((key) => whole(mode, key) >= least);
  const divisionLp = whole(mode, "divisionLp");
  const demotionLp = whole(mode, "demotionLp");
  const hour = whole(mode, "apexRecalcHourUtc");
  if (!(divisionLp >= 1) || !(demotionLp >= 0 && demotionLp < divisionLp)) found.push("ranked.issue.division_lp");
  if (!(hour >= 0 && hour <= 23)) found.push("ranked.issue.hour");
  if (!atLeast(["challengerSlots", "challengerMinLp", "grandmasterSlots", "grandmasterMinLp"], 0)) found.push("ranked.issue.apex");
  if (!atLeast(["mmrScale", "mmrK", "mmrKProvisional", "rankMmrStep", "lpBase", "lpGapScale", "placementMatches", "seasonLengthMonths"], 1)) found.push("ranked.issue.at_least_one");
  if (!atLeast(["initialMmr", "rankMmrBase", "mmrProvisionalMatches", "placementMaxDropDivisions", "placementMaxClimbDivisions"], 0)) found.push("ranked.issue.not_negative");
  if (!(amount(mode, "lpFactorMin") > 0 && amount(mode, "lpFactorMin") <= amount(mode, "lpFactorMax"))) found.push("ranked.issue.lp_factor");
  if (!(whole(mode, "lpChangeMin") >= 0 && whole(mode, "lpChangeMin") <= whole(mode, "lpChangeMax"))) found.push("ranked.issue.lp_change");
  if (!(whole(mode, "lpChangeMax") < divisionLp)) found.push("ranked.issue.lp_division");
  if (!divisionRanks.includes(mode.newPlayerPlacementCap)) found.push("ranked.issue.placement_cap");
  if (!(Number.isInteger(mode.resetTierDrop) && mode.resetTierDrop >= 0)) found.push("ranked.issue.tier_drop");
  return [...found, ...resetIssues(mode, divisionRanks)];
}

export function bundleIdFor(tier: string): string {
  return `bundle.rank_${tier.toLowerCase()}`;
}

function differs(a: unknown, b: unknown): boolean {
  return !sameContent(a, b);
}

function bundleContent(bundle: Bundle): Bundle {
  return { ...bundle, revision: 0, cosmetics: bundle.cosmetics ?? [], medals: bundle.medals ?? [] };
}

function grantsSomething(bundle: Bundle): boolean {
  return (bundle.coins ?? 0) > 0 || (bundle.crystals ?? 0) > 0 || (bundle.passXp ?? 0) > 0 || (bundle.cosmetics ?? []).length > 0 || (bundle.medals ?? []).length > 0;
}

export function withRankReward(mode: ModeConfig, tier: string, change: Partial<Pick<Bundle, "id" | "cosmetics" | "medals">>, published: ModeConfig | undefined): ModeConfig {
  const before = published?.rankRewards?.[tier];
  const current: Bundle = mode.rankRewards?.[tier] ?? { id: bundleIdFor(tier), revision: 1 };
  const next: Bundle = { ...current, ...change };
  next.revision = !before ? 1 : differs(bundleContent(before), bundleContent(next)) ? before.revision + 1 : before.revision;
  const { [tier]: _replaced, ...rewards } = mode.rankRewards ?? {};
  const table = grantsSomething(next) ? { ...rewards, [tier]: next } : rewards;
  const { rankRewards: _rewards, ...rest } = mode;
  return Object.keys(table).length > 0 ? { ...rest, rankRewards: table } : (rest as ModeConfig);
}

export function rewardIssues(mode: ModeConfig): MessageKey[] {
  const ids = Object.values(mode.rankRewards ?? {}).map((bundle) => bundle.id);
  return ids.every((id) => /^bundle\.[a-z0-9_]+$/.test(id)) ? [] : ["ranked.issue.bundle_id"];
}

export function countryList(text: string): string[] {
  return [...new Set(text.split(/[\s,]+/).filter(Boolean).map((code) => code.toUpperCase()))];
}

export function withRegion(config: RatingConfig, id: string, countries: string[]): RatingConfig {
  return { ...config, leaderboardRegions: { ...config.leaderboardRegions, [id]: countries } };
}

export function withoutRegion(config: RatingConfig, id: string): RatingConfig {
  const { [id]: _removed, ...regions } = config.leaderboardRegions ?? {};
  const { leaderboardRegions: _regions, ...rest } = config;
  return Object.keys(regions).length > 0 ? { ...rest, leaderboardRegions: regions } : (rest as RatingConfig);
}

export function snapshotIssues(config: RatingConfig): MessageKey[] {
  const found: MessageKey[] = [];
  if (!(Number.isInteger(config.leaderboardTopSize) && config.leaderboardTopSize >= 1)) found.push("ranked.issue.top_size");
  if (Object.values(config.rankRewardCoins).some((coins) => !(Number.isInteger(coins) && coins >= 0))) found.push("ranked.issue.coins");
  const regions = Object.entries(config.leaderboardRegions ?? {});
  if (regions.some(([id]) => !/^[a-z0-9_]+$/.test(id))) found.push("ranked.issue.region_id");
  const countries = regions.flatMap(([, codes]) => [...new Set(codes)]);
  if (countries.some((code) => !/^[A-Z]{2}$/.test(code))) found.push("ranked.issue.country");
  if (new Set(countries).size !== countries.length) found.push("ranked.issue.country_twice");
  return found;
}

export function configIssues(config: RatingConfig, divisionRanks: readonly string[]): MessageKey[] {
  const found = new Set<MessageKey>(snapshotIssues(config));
  for (const mode of modes) {
    for (const key of [...modeIssues(config.modes[mode], divisionRanks), ...rewardIssues(config.modes[mode])]) {
      found.add(key);
    }
  }
  return [...found];
}

export function ratingChanges(before: RatingConfig, after: RatingConfig): FieldChange[] {
  const { modes: _previousModes, ...previous } = before;
  const { modes: _nextModes, ...next } = after;
  const perMode = modes.flatMap((mode) => fieldChanges(before.modes[mode], after.modes[mode]).map((change) => ({ ...change, field: `${mode}.${change.field}` })));
  return [...fieldChanges(previous, next), ...perMode];
}

export function changedStructural(before: RatingConfig, after: RatingConfig, structuralKeys: readonly string[], divisionRanks: readonly string[]): string[] {
  return modes.flatMap((mode) => {
    const previous = before.modes[mode];
    const next = after.modes[mode];
    const direct = structuralKeys.filter((key) => key !== "seasonResetTable" && differs(previous[key], next[key]));
    const table = differs(effectiveReset(previous, divisionRanks), effectiveReset(next, divisionRanks)) ? ["seasonResetTable"] : [];
    return [...direct, ...table].map((key) => `${mode}.${key}`);
  });
}
