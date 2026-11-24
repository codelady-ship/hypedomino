import type { SearchParams } from "../audit/query.ts";
import type { MessageKey } from "../i18n/translate.ts";

export type BundleUse = { domain: string; object: string; detail?: string };

export type RewardBundle = {
  id: string;
  revision: number;
  coins: number;
  crystals?: number;
  cosmetics: string[] | null;
  medals: string[] | null;
  passXp: number;
  guideValue: number;
  usedBy: BundleUse[];
  differs: boolean;
  editable: boolean;
};

export type BundleEdit = { coins: number; crystals: number; cosmetics: string[]; medals: string[]; passXp: number };

export const bundlePageSize = 50;

export const bundleDomains = ["shop", "pass", "missions", "rating", "daily_login", "promo", "promo_template", "reward_bundles"] as const;

export type BundleQuery = { q: string; domain: (typeof bundleDomains)[number] | ""; differing: "true" | ""; after: string };

function single(value: string | string[] | undefined): string {
  return (Array.isArray(value) ? value[0] : value)?.trim() ?? "";
}

export function readBundleQuery(params: SearchParams): BundleQuery {
  return {
    q: single(params.q),
    domain: bundleDomains.find((domain) => domain === single(params.domain)) ?? "",
    differing: single(params.differing) === "true" ? "true" : "",
    after: single(params.after),
  };
}

function searchOf(query: BundleQuery): URLSearchParams {
  return new URLSearchParams(Object.entries(query).filter(([, value]) => value));
}

export function rewardBundlesApiPath(query: BundleQuery): string {
  const search = searchOf(query);
  search.set("limit", String(bundlePageSize));
  return `/admin/reward-bundles?${search}`;
}

export function nextRewardsHref(query: BundleQuery, ids: readonly string[]): string | null {
  if (ids.length < bundlePageSize) {
    return null;
  }
  return `/rewards?${searchOf({ ...query, after: ids[ids.length - 1] })}`;
}

export function useLabel(use: BundleUse): string {
  return use.detail ? `${use.object} ${use.detail}` : use.object;
}

export type NameKeysByDomain = Readonly<Record<string, readonly string[]>>;

export function republishedNameKeys(bundle: RewardBundle, nameKeys: NameKeysByDomain): string[] {
  return [...new Set(bundle.usedBy.map((use) => use.domain))].flatMap((domain) => nameKeys[domain] ?? []);
}

export function editOf(bundle: RewardBundle): BundleEdit {
  return { coins: bundle.coins, crystals: bundle.crystals ?? 0, cosmetics: bundle.cosmetics ?? [], medals: bundle.medals ?? [], passXp: bundle.passXp };
}

export function parsedIds(text: string): string[] {
  return [...new Set(text.split(",").map((id) => id.trim()).filter((id) => id !== ""))];
}

function sameList(a: string[], b: string[]): boolean {
  return [...a].sort().join() === [...b].sort().join();
}

export function bundleEditChanged(bundle: RewardBundle, edit: BundleEdit): boolean {
  const before = editOf(bundle);
  return before.coins !== edit.coins || before.crystals !== edit.crystals || before.passXp !== edit.passXp || !sameList(before.cosmetics, edit.cosmetics) || !sameList(before.medals, edit.medals);
}

export function bundleEditIssues(edit: BundleEdit): MessageKey[] {
  const found: MessageKey[] = [];
  if (![edit.coins, edit.crystals, edit.passXp].every((value) => Number.isInteger(value) && value >= 0)) found.push("rewards.issue.number");
  if (edit.coins === 0 && edit.crystals === 0 && edit.passXp === 0 && edit.cosmetics.length === 0 && edit.medals.length === 0) found.push("rewards.issue.empty");
  return found;
}

const bundleIdPattern = /^bundle\.[a-z0-9_]+$/;

export const emptyBundleEdit: BundleEdit = { coins: 0, crystals: 0, cosmetics: [], medals: [], passXp: 0 };

export function newBundleIssues(id: string, edit: BundleEdit): MessageKey[] {
  const idIssues: MessageKey[] = bundleIdPattern.test(id) ? [] : ["rewards.issue.id"];
  return [...idIssues, ...bundleEditIssues(edit)];
}

export type BundleForm = { coins: number; crystals: number; passXp: number; cosmetics: string; medals: string };

export function formOf(edit: BundleEdit): BundleForm {
  return { coins: edit.coins, crystals: edit.crystals, passXp: edit.passXp, cosmetics: edit.cosmetics.join(", "), medals: edit.medals.join(", ") };
}

export function editFrom(form: BundleForm): BundleEdit {
  return { coins: form.coins, crystals: form.crystals, passXp: form.passXp, cosmetics: parsedIds(form.cosmetics), medals: parsedIds(form.medals) };
}
