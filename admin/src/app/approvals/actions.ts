"use server";

import { adminApi } from "@/admin/api";

export type DecisionResult = { ok: true; state: string } | { ok: false; error: string; detail: string };

export async function decide(id: number, decision: "approve" | "reject", reason: string): Promise<DecisionResult> {
  const reply = await adminApi<{ state: string }>("POST", `/admin/approvals/${id}/${decision}`, { reason });
  if (reply.status !== 200) {
    return { ok: false, error: reply.body.error ?? String(reply.status), detail: reply.body.detail ?? "" };
  }
  return { ok: true, state: reply.body.state };
}
