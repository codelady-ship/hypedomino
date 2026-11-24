"use server";

import { adminApi } from "@/admin/api";
import type { ConfigResult } from "@/config/result";
import type { StorePrices } from "@/storeprices/prices";
import type { PushPlan, PushResult } from "@/storeprices/push";

export async function publishStorePrices(prices: StorePrices, reason: string): Promise<ConfigResult> {
  const reply = await adminApi<{ config_version: number }>("PUT", "/admin/config/store-prices", { catalog: prices, reason });
  if (reply.status !== 200) {
    return { ok: false, error: reply.body.error ?? String(reply.status), detail: reply.body.detail ?? "" };
  }
  return { ok: true, version: reply.body.config_version };
}

export async function rollbackStorePrices(version: number, reason: string): Promise<ConfigResult> {
  const reply = await adminApi<{ config_version: number }>("POST", "/admin/config/store-prices/rollback", { version, reason });
  if (reply.status !== 200) {
    return { ok: false, error: reply.body.error ?? String(reply.status), detail: reply.body.detail ?? "" };
  }
  return { ok: true, version: reply.body.config_version };
}

async function pushRequest(path: string, body?: unknown): Promise<PushResult> {
  const reply = await adminApi<PushPlan>("POST", path, body);
  if (reply.status !== 200) {
    return { ok: false, error: reply.body.error ?? String(reply.status), detail: reply.body.detail ?? "" };
  }
  return { ok: true, plan: reply.body };
}

export async function planStorePush(): Promise<PushResult> {
  return pushRequest("/admin/config/store-prices/sync/plan");
}

export async function applyStorePush(reason: string): Promise<PushResult> {
  return pushRequest("/admin/config/store-prices/sync/apply", { reason });
}
