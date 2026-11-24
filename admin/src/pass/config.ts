import type { MessageKey } from "@/i18n/translate";
import type { Bundle } from "@/shop/catalog";
import { entryChanges, fieldChanges, type CatalogChanges } from "../config/changes.ts";

export type Track = "free" | "premium";

export type Curve = { levels: number; xpBase: number; xpStep: number; xpMax: number };

export type Season = {
  id: string;
  start: string;
  end: string;
  curve: Curve;
  free: Record<string, Bundle>;
  premium: Record<string, Bundle>;
};

export type MatchXp = { win: number; loss: number };

export type PassConfig = {
  schema: number;
  freeTrackCoins?: number;
  premiumTrackCrystals?: number;
  premiumPlusLevels?: number;
  matchXp1v1?: MatchXp;
  matchXp2v2?: MatchXp;
  dailyMatchXpCap?: number;
  botSeatRewardPercent?: number;
  levelCoinPrice?: number;
  levelCrystalPrice?: number;
  seasons: Season[];
  [field: string]: unknown;
};

export type SeasonState = "planned" | "running" | "ended";

export type Window = { start: string; end: string };

export type XpRow = { level: number; toNext: number; total: number };

export const tracks: readonly Track[] = ["free", "premium"];

export const settingKeys = [
  "freeTrackCoins",
  "premiumTrackCrystals",
  "premiumPlusLevels",
  "dailyMatchXpCap",
  "botSeatRewardPercent",
  "levelCoinPrice",
  "levelCrystalPrice",
] as const;

export type SettingKey = (typeof settingKeys)[number];

const currencyOf: Record<Track, "coins" | "crystals"> = { free: "coins", premium: "crystals" };

export function emptyPassConfig(): PassConfig {
  return { schema: 1, seasons: [] };
}

export function xpToNext(curve: Curve, level: number): number {
  if (level < 1 || level >= curve.levels) {
    return 0;
  }
  return Math.min(curve.xpBase + curve.xpStep * (level - 1), curve.xpMax);
}

export function xpTable(curve: Curve): XpRow[] {
  const rows: XpRow[] = [];
  let total = 0;
  for (let level = 1; level <= curve.levels; level++) {
    rows.push({ level, toNext: xpToNext(curve, level), total });
    total += xpToNext(curve, level);
  }
  return rows;
}

export function seasonState(season: Season, now: string): SeasonState {
  const at = Date.parse(now);
  if (at < Date.parse(season.start)) {
    return "planned";
  }
  return at < Date.parse(season.end) ? "running" : "ended";
}

export function nextSeasonId(config: PassConfig): string {
  const numbers = config.seasons.map((season) => Number(season.id.replace(/^season_/, ""))).filter(Number.isInteger);
  return `season_${Math.max(0, ...numbers) + 1}`;
}

export function bundleId(seasonId: string, track: Track, level: number): string {
  return `bundle.pass_${seasonId}_${track}_${level}`;
}

export function emptyBundle(seasonId: string, track: Track, level: number): Bundle {
  return { id: bundleId(seasonId, track, level), revision: 1, [currencyOf[track]]: 0 };
}

export function newSeason(config: PassConfig, start: string, end: string, curve: Curve): Season {
  const id = nextSeasonId(config);
  return withPremiumLevels({ id, start, end, curve, free: {}, premium: {} });
}

export function cloneSeason(config: PassConfig, source: Season, start: string, end: string): Season {
  const id = nextSeasonId(config);
  const copy = (track: Track) =>
    Object.fromEntries(
      Object.entries(source[track])
        .map(([level, bundle]): [string, Bundle] => {
          const { cosmetics: _items, ...rest } = bundle;
          return [level, { ...rest, id: bundleId(id, track, Number(level)), revision: 1 }];
        })
        .filter(([, bundle]) => track === "premium" || grantsSomething(bundle)),
    );
  return { id, start, end, curve: { ...source.curve }, free: copy("free"), premium: copy("premium") };
}

function withPremiumLevels(season: Season): Season {
  const premium: Record<string, Bundle> = {};
  for (let level = 1; level <= season.curve.levels; level++) {
    premium[level] = season.premium[level] ?? emptyBundle(season.id, "premium", level);
  }
  const free = Object.fromEntries(Object.entries(season.free).filter(([level]) => Number(level) <= season.curve.levels));
  return { ...season, free, premium };
}

export function withCurve(season: Season, curve: Curve): Season {
  return withPremiumLevels({ ...season, curve });
}

export function levelBundle(season: Season, track: Track, level: number): Bundle | undefined {
  return season[track][level];
}

function grantsSomething(bundle: Bundle): boolean {
  return (bundle.coins ?? 0) > 0 || (bundle.crystals ?? 0) > 0 || (bundle.cosmetics ?? []).length > 0;
}

function bundleContent(bundle: Bundle): string {
  return JSON.stringify({ coins: bundle.coins ?? 0, crystals: bundle.crystals ?? 0, cosmetics: bundle.cosmetics ?? [] });
}

export function withLevel(season: Season, track: Track, level: number, change: { amount?: number; cosmetics?: string[] }, published: Season | undefined): Season {
  const current = season[track][level] ?? emptyBundle(season.id, track, level);
  const next: Bundle = { ...current };
  if (change.amount !== undefined) {
    next[currencyOf[track]] = change.amount;
  }
  if (change.cosmetics !== undefined) {
    if (change.cosmetics.length > 0) {
      next.cosmetics = change.cosmetics;
    } else {
      delete next.cosmetics;
    }
  }
  const before = published?.[track][level];
  next.revision = !before ? 1 : bundleContent(before) === bundleContent(next) ? before.revision : before.revision + 1;
  const levels = { ...season[track] };
  if (track === "free" && !grantsSomething(next)) {
    delete levels[level];
  } else {
    levels[level] = next;
  }
  return { ...season, [track]: levels };
}

export function trackTotal(season: Season, track: Track): number {
  return Object.values(season[track]).reduce((sum, bundle) => sum + (bundle[currencyOf[track]] ?? 0), 0);
}

export function replaceSeason(config: PassConfig, season: Season): PassConfig {
  return { ...config, seasons: config.seasons.map((candidate) => (candidate.id === season.id ? season : candidate)) };
}

export function addSeason(config: PassConfig, season: Season): PassConfig {
  return { ...config, seasons: [...config.seasons, season] };
}

export function removeSeason(config: PassConfig, id: string): PassConfig {
  return { ...config, seasons: config.seasons.filter((season) => season.id !== id) };
}

function overlaps(a: Window, b: Window): boolean {
  return Date.parse(a.start) < Date.parse(b.end) && Date.parse(b.start) < Date.parse(a.end);
}

export function rankedWindow(config: PassConfig, ranked: readonly Window[], now: string): Window | null {
  const free = ranked.find((candidate) => Date.parse(candidate.end) > Date.parse(now) && !config.seasons.some((season) => overlaps(season, candidate)));
  return free ? { start: free.start, end: free.end } : null;
}

export function seasonIssues(config: PassConfig, season: Season): MessageKey[] {
  const found: MessageKey[] = [];
  const { levels, xpBase, xpStep, xpMax } = season.curve;
  if (!(Date.parse(season.end) > Date.parse(season.start))) found.push("pass.issue.window");
  if (config.seasons.some((other) => other.id !== season.id && overlaps(other, season))) found.push("pass.issue.overlap");
  if (![levels, xpBase, xpStep, xpMax].every((value) => Number.isInteger(value) && value >= 1)) found.push("pass.issue.curve");
  if (xpMax < xpBase) found.push("pass.issue.xp_max");
  if (config.premiumPlusLevels !== undefined && levels <= config.premiumPlusLevels) found.push("pass.issue.levels");
  const premium = Object.values(season.premium);
  if (premium.length < levels || premium.some((bundle) => !grantsSomething(bundle))) found.push("pass.issue.premium_empty");
  if (config.freeTrackCoins !== undefined && trackTotal(season, "free") > config.freeTrackCoins) found.push("pass.issue.free_budget");
  if (config.premiumTrackCrystals !== undefined && trackTotal(season, "premium") > config.premiumTrackCrystals) found.push("pass.issue.premium_budget");
  return found;
}

export function settingIssues(config: PassConfig): MessageKey[] {
  const values = [...settingKeys.map((key) => config[key]), config.matchXp1v1?.win, config.matchXp1v1?.loss, config.matchXp2v2?.win, config.matchXp2v2?.loss];
  const found: MessageKey[] = [];
  if (values.some((value) => value !== undefined && !(Number.isInteger(value) && value >= 0))) found.push("pass.issue.settings");
  if ((config.botSeatRewardPercent ?? 0) > 100) found.push("pass.issue.bot_percent");
  if (config.levelCoinPrice === 0 || config.levelCrystalPrice === 0) found.push("pass.issue.level_price");
  return found;
}

export function changedSeasons(before: PassConfig | null, after: PassConfig): string[] {
  const previous = new Map((before?.seasons ?? []).map((season) => [season.id, JSON.stringify(season)]));
  return after.seasons.filter((season) => previous.get(season.id) !== JSON.stringify(season)).map((season) => season.id);
}

export function changedSettings(before: PassConfig | null, after: PassConfig): string[] {
  const keys = [...settingKeys, "matchXp1v1", "matchXp2v2"] as const;
  return keys.filter((key) => JSON.stringify(before?.[key]) !== JSON.stringify(after[key]));
}

export function droppedSeasons(before: PassConfig | null, after: PassConfig): string[] {
  const kept = new Set(after.seasons.map((season) => season.id));
  return (before?.seasons ?? []).filter((season) => !kept.has(season.id)).map((season) => season.id);
}

export function passChanges(before: PassConfig | null, after: PassConfig): CatalogChanges {
  const { seasons, ...settings } = after;
  if (!before) {
    return { settings: [], entries: entryChanges([], seasons) };
  }
  const { seasons: previous, ...previousSettings } = before;
  return { settings: fieldChanges(previousSettings, settings), entries: entryChanges(previous, seasons) };
}
