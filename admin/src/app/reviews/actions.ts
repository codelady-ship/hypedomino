"use server";

import { adminApi } from "@/admin/api";
import type { BanCategory } from "@/players/sanctions";

export type ReviewDecision = "clear" | "confirm" | "permanent-ban";

export type ReviewResult = { ok: true } | { ok: false; error: string; detail: string };

export async function decideReview(id: number, decision: ReviewDecision, reason: string, category: BanCategory | ""): Promise<ReviewResult> {
  const body = decision === "permanent-ban" ? { reason, category } : { reason };
  const reply = await adminApi<{ account_id: string }>("POST", `/admin/reviews/${id}/${decision}`, body);
  if (reply.status !== 200 && reply.status !== 202) {
    return { ok: false, error: reply.body.error ?? String(reply.status), detail: reply.body.detail ?? "" };
  }
  return { ok: true };
}
