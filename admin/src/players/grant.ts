import { bundleSummary, listOf } from "../promo/codes.ts";
import type { Bundle } from "../shop/catalog.ts";

export const grantSourceTypes = ["event", "promo", "shop", "pass", "mission", "rank_reward", "first_purchase", "chest"] as const;

export type GrantSourceType = (typeof grantSourceTypes)[number];

export type GrantDraft = {
  sourceType: GrantSourceType;
  sourceRef: string;
  bundleId: string;
  coins: number;
  passXp: number;
  cosmetics: string;
  medals: string;
};

export const emptyGrant: GrantDraft = { sourceType: "event", sourceRef: "", bundleId: "bundle.", coins: 0, passXp: 0, cosmetics: "", medals: "" };

export type GrantRequest = { bundle: Bundle; source: { type: GrantSourceType; ref: string } };

export function grantRequest(draft: GrantDraft): GrantRequest {
  const bundle: Bundle = { id: draft.bundleId.trim(), revision: 1 };
  const cosmetics = listOf(draft.cosmetics);
  const medals = listOf(draft.medals);
  if (draft.coins > 0) {
    bundle.coins = draft.coins;
  }
  if (draft.passXp > 0) {
    bundle.passXp = draft.passXp;
  }
  if (cosmetics.length > 0) {
    bundle.cosmetics = cosmetics;
  }
  if (medals.length > 0) {
    bundle.medals = medals;
  }
  return { bundle, source: { type: draft.sourceType, ref: draft.sourceRef.trim() } };
}

export function grantReady(draft: GrantDraft): boolean {
  const { bundle, source } = grantRequest(draft);
  const counts = [draft.coins, draft.passXp];
  if (source.ref === "" || counts.some((count) => !Number.isSafeInteger(count) || count < 0)) {
    return false;
  }
  return (bundle.coins ?? 0) > 0 || (bundle.passXp ?? 0) > 0 || (bundle.cosmetics?.length ?? 0) > 0 || (bundle.medals?.length ?? 0) > 0;
}

type GrantChange = { bundle?: Bundle; source?: { type: string; ref: string } };

export function grantChangeSummary(change: unknown): string | null {
  const grant = change as GrantChange | null;
  if (!grant || typeof grant !== "object" || !grant.bundle || typeof grant.bundle !== "object" || !grant.source) {
    return null;
  }
  const parts = [bundleSummary(grant.bundle), grant.bundle.medals?.join(", ")].filter(Boolean);
  return `${grant.bundle.id}: ${parts.join(" + ")}; ${grant.source.type}:${grant.source.ref}`;
}
