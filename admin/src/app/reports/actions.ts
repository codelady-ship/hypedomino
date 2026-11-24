"use server";

import { adminApi } from "@/admin/api";

export type ReportDecision = "uphold" | "reject";

export type ReportResult = { ok: true } | { ok: false; error: string; detail: string };

export async function decideReports(accountId: string, decision: ReportDecision, reason: string): Promise<ReportResult> {
  const reply = await adminApi<{ closed: number }>("POST", `/admin/players/${encodeURIComponent(accountId)}/reports/${decision}`, { reason });
  if (reply.status !== 200) {
    return { ok: false, error: reply.body.error ?? String(reply.status), detail: reply.body.detail ?? "" };
  }
  return { ok: true };
}
