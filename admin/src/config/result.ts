import type { ApiRefusal, ApiReply } from "../admin/api.ts";

export type Refusal = { ok: false; error: string; detail: string; keys?: string[] };

export type ConfigResult = { ok: true; version: number } | Refusal;

export function refusalOf(reply: ApiReply<ApiRefusal>): Refusal {
  return { ok: false, error: reply.body.error ?? String(reply.status), detail: reply.body.detail ?? "", keys: reply.body.keys ?? [] };
}
