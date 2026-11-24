"use server";

import { adminApiResponse, type ApiRefusal } from "@/admin/api";
import { parseReply } from "@/admin/forwarding";
import { auditExportPath, type AuditFilter } from "@/audit/query";

export type ExportResult = { ok: true; json: string } | { ok: false; error: string; detail: string };

export async function exportAudit(filter: AuditFilter, reason: string): Promise<ExportResult> {
  const response = await adminApiResponse("POST", auditExportPath(filter), { reason });
  const text = await response.text();
  if (response.status !== 200) {
    const refusal = parseReply<ApiRefusal>(text);
    return { ok: false, error: refusal.error ?? String(response.status), detail: refusal.detail ?? "" };
  }
  return { ok: true, json: text };
}
