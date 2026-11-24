"use server";

import { adminApi } from "@/admin/api";
import { staticArtApiPath, staticArtOf, type Asset, type StaticArt } from "@/assets/assets";
import { refusalOf, type ConfigResult } from "@/config/result";
import type { CosmeticItem } from "@/cosmetics/catalog";

export async function publishCosmetics(items: CosmeticItem[], reason: string): Promise<ConfigResult> {
  const reply = await adminApi<{ config_version: number }>("PUT", "/admin/config/cosmetics", { catalog: { items }, reason });
  if (reply.status !== 200) {
    return refusalOf(reply);
  }
  return { ok: true, version: reply.body.config_version };
}

export async function switchOffCosmetic(id: string, reason: string, ref: string): Promise<ConfigResult> {
  const reply = await adminApi<{ config_version: number }>("POST", `/admin/cosmetics/items/${encodeURIComponent(id)}/switch-off`, { reason, ref });
  if (reply.status !== 200) {
    return { ok: false, error: reply.body.error ?? String(reply.status), detail: reply.body.detail ?? "" };
  }
  return { ok: true, version: reply.body.config_version };
}

export async function rollbackCosmetics(version: number, reason: string): Promise<ConfigResult> {
  const reply = await adminApi<{ config_version: number }>("POST", "/admin/config/cosmetics/rollback", { version, reason });
  if (reply.status !== 200) {
    return refusalOf(reply);
  }
  return { ok: true, version: reply.body.config_version };
}

export async function staticArt(itemId: string): Promise<StaticArt> {
  const reply = await adminApi<{ store_configured?: boolean; assets?: Asset[] }>("GET", staticArtApiPath(itemId));
  return staticArtOf(reply.status, reply.body);
}
