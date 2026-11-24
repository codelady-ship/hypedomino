"use server";

import { adminApi, type ApiReply, type ApiRefusal } from "@/admin/api";

export type AccountResult = { ok: true; approvalId: number | null } | { ok: false; error: string; detail: string };

export type AccountNote = { reason: string; ref: string };

type Reply = ApiReply<{ id?: number } & ApiRefusal>;

export async function createAccount(email: string, displayName: string, roles: string[], note: AccountNote): Promise<AccountResult> {
  return outcome(await adminApi<{ id?: number }>("POST", "/admin/accounts", { email, display_name: displayName, roles, ...note }), 201);
}

export async function setAccountRoles(id: string, roles: string[], note: AccountNote): Promise<AccountResult> {
  return outcome(await adminApi<{ id?: number }>("PUT", `/admin/accounts/${encodeURIComponent(id)}/roles`, { roles, ...note }), 200);
}

export async function disableAccount(id: string, note: AccountNote): Promise<AccountResult> {
  return outcome(await adminApi<{ id?: number }>("POST", `/admin/accounts/${encodeURIComponent(id)}/disable`, note), 200);
}

function outcome(reply: Reply, success: number): AccountResult {
  if (reply.status === 202) {
    return { ok: true, approvalId: reply.body.id ?? null };
  }
  if (reply.status !== success) {
    return { ok: false, error: reply.body.error ?? String(reply.status), detail: reply.body.detail ?? "" };
  }
  return { ok: true, approvalId: null };
}
