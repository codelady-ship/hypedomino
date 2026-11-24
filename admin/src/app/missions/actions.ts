"use server";

import { adminApi } from "@/admin/api";
import { refusalOf, type ConfigResult } from "@/config/result";
import type { MissionCatalog } from "@/missions/catalog";

export async function publishMissions(catalog: MissionCatalog, reason: string): Promise<ConfigResult> {
  const reply = await adminApi<{ config_version: number }>("PUT", "/admin/config/missions", { catalog, reason });
  if (reply.status !== 200) {
    return refusalOf(reply);
  }
  return { ok: true, version: reply.body.config_version };
}

export async function rollbackMissions(version: number, reason: string): Promise<ConfigResult> {
  const reply = await adminApi<{ config_version: number }>("POST", "/admin/config/missions/rollback", { version, reason });
  if (reply.status !== 200) {
    return refusalOf(reply);
  }
  return { ok: true, version: reply.body.config_version };
}
