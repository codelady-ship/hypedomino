"use server";

import { adminApi } from "@/admin/api";
import { refusalOf, type ConfigResult } from "@/config/result";
import type { MedalCatalog } from "@/medals/catalog";

export async function publishMedals(catalog: MedalCatalog, reason: string): Promise<ConfigResult> {
  const reply = await adminApi<{ config_version: number }>("PUT", "/admin/config/medals", { catalog, reason });
  if (reply.status !== 200) {
    return refusalOf(reply);
  }
  return { ok: true, version: reply.body.config_version };
}

export async function rollbackMedals(version: number, reason: string): Promise<ConfigResult> {
  const reply = await adminApi<{ config_version: number }>("POST", "/admin/config/medals/rollback", { version, reason });
  if (reply.status !== 200) {
    return refusalOf(reply);
  }
  return { ok: true, version: reply.body.config_version };
}
