"use server";

import { adminApi } from "@/admin/api";

export type AppealChoice = "uphold" | "lift";

export type AppealResult = { ok: true } | { ok: false; error: string; detail: string };

export async function decideAppeal(appealId: number, choice: AppealChoice, reason: string): Promise<AppealResult> {
  const reply = await adminApi<{ state: string }>("POST", `/admin/appeals/${encodeURIComponent(String(appealId))}/${choice}`, { reason });
  if (reply.status !== 200) {
    return { ok: false, error: reply.body.error ?? String(reply.status), detail: reply.body.detail ?? "" };
  }
  return { ok: true };
}
