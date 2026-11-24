import type { MessageKey } from "../i18n/translate.ts";

export const banCategories = ["cheating", "collusion", "afk_leaving", "offensive_name", "payment_fraud", "other"] as const;

export type BanCategory = (typeof banCategories)[number];

export type SanctionKind = "temporary_ban" | "permanent_ban" | "ranked_ban" | "penalty_ban" | "forced_rename";

export type Sanction = { id: number; kind: SanctionKind; category: BanCategory; starts_at: string; ends_at: string | null; lifted_at: string | null };

export const banCategoryKeys: Record<BanCategory, MessageKey> = {
  cheating: "ban.category.cheating",
  collusion: "ban.category.collusion",
  afk_leaving: "ban.category.afk_leaving",
  offensive_name: "ban.category.offensive_name",
  payment_fraud: "ban.category.payment_fraud",
  other: "ban.category.other",
};

export const sanctionKindKeys: Record<SanctionKind, MessageKey> = {
  temporary_ban: "sanction.kind.temporary_ban",
  permanent_ban: "sanction.kind.permanent_ban",
  ranked_ban: "sanction.kind.ranked_ban",
  penalty_ban: "sanction.kind.penalty_ban",
  forced_rename: "sanction.kind.forced_rename",
};

export function sanctionRunning(sanction: Pick<Sanction, "ends_at" | "lifted_at">, now: Date): boolean {
  return sanction.lifted_at === null && (sanction.ends_at === null || new Date(sanction.ends_at) > now);
}

export function runningRankedBan(sanctions: readonly Sanction[], now: Date): Sanction | null {
  return sanctions.find((sanction) => sanction.kind === "ranked_ban" && sanctionRunning(sanction, now)) ?? null;
}
