"use server";

import { adminApi, adminApiResponse, type ApiRefusal } from "@/admin/api";
import { parseReply } from "@/admin/forwarding";
import { disablePath, listOf, targetsPath, templatePath, templatesPath, type PromoCode, type PromoRequest, type PromoTemplate, type TemplateRequest } from "@/promo/codes";

export type Refused = { ok: false; error: string; detail: string };

export type CreatedResult = { ok: true; state: string; approvalId: number | null; plaintext: string; csv: string; filename: string } | Refused;

export type DisableResult = { ok: true; disabled: number } | Refused;

function refused(status: number, body: ApiRefusal): Refused {
  return { ok: false, error: body.error ?? String(status), detail: body.detail ?? "" };
}

export async function createPromo(request: PromoRequest, reason: string): Promise<CreatedResult> {
  const reply = await adminApi<{ code: PromoCode; plaintext: string; approval_id?: number }>("POST", "/admin/promo-codes", { ...request, reason });
  if (reply.status !== 201) {
    return refused(reply.status, reply.body);
  }
  return { ok: true, state: reply.body.code.state, approvalId: reply.body.approval_id ?? null, plaintext: reply.body.plaintext, csv: "", filename: "" };
}

export async function createPromoBatch(request: PromoRequest, reason: string): Promise<CreatedResult> {
  const response = await adminApiResponse("POST", "/admin/promo-codes/batch", { ...request, reason });
  const text = await response.text();
  if (response.status !== 201) {
    return refused(response.status, parseReply<ApiRefusal>(text));
  }
  const approval = response.headers.get("Hd-Approval-Id");
  return {
    ok: true,
    state: response.headers.get("Hd-Promo-State") ?? "",
    approvalId: approval ? Number(approval) : null,
    plaintext: "",
    csv: text,
    filename: `${request.id}.csv`,
  };
}

export async function disablePromo(id: string, batch: boolean, reason: string): Promise<DisableResult> {
  const reply = await adminApi<{ disabled: number }>("POST", disablePath(id, batch), { reason });
  if (reply.status !== 200) {
    return refused(reply.status, reply.body);
  }
  return { ok: true, disabled: reply.body.disabled };
}

export type TargetsResult = { ok: true; accounts: number; added: number; removed: number } | Refused;

export async function setTargets(id: string, accounts: string, reason: string): Promise<TargetsResult> {
  const reply = await adminApi<{ accounts: number; added: number; removed: number }>("PUT", targetsPath(id), { accounts: listOf(accounts), reason });
  if (reply.status !== 200) {
    return refused(reply.status, reply.body);
  }
  return { ok: true, accounts: reply.body.accounts, added: reply.body.added, removed: reply.body.removed };
}

export type TemplateResult = { ok: true; template: PromoTemplate } | Refused;

export async function saveTemplate(request: TemplateRequest, existing: boolean, reason: string): Promise<TemplateResult> {
  const reply = await adminApi<PromoTemplate>(existing ? "PUT" : "POST", existing ? templatePath(request.id) : templatesPath, { ...request, reason });
  if (reply.status !== (existing ? 200 : 201)) {
    return refused(reply.status, reply.body);
  }
  return { ok: true, template: reply.body };
}
