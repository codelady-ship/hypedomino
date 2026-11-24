"use server";

import { adminApi } from "@/admin/api";
import { refusalOf, type ConfigResult } from "@/config/result";
import { englishApproval, exactBundle, searchFrom, type BundleLookup, type BundleRow, type NameApproval, type ShopCatalog } from "@/shop/catalog";

const lookupLimit = "200";

export async function findRewardBundle(id: string): Promise<BundleLookup> {
  const search = new URLSearchParams({ q: id, after: searchFrom(id), limit: lookupLimit });
  const reply = await adminApi<{ bundles: BundleRow[] }>("GET", `/admin/reward-bundles?${search}`);
  return reply.status === 200 ? exactBundle(reply.body.bundles, id) : "unchecked";
}

export async function nameKeyApproval(key: string): Promise<NameApproval> {
  const reply = await adminApi<{ keys: string[] }>("POST", "/admin/strings/unapproved", { keys: [key] });
  return reply.status === 200 ? englishApproval(reply.body.keys, key) : "unchecked";
}

export type PublishResult = ConfigResult;

export async function publishCatalog(catalog: ShopCatalog, reason: string): Promise<PublishResult> {
  const reply = await adminApi<{ config_version: number }>("PUT", "/admin/config/shop", { catalog, reason });
  if (reply.status !== 200) {
    return refusalOf(reply);
  }
  return { ok: true, version: reply.body.config_version };
}

export type PreviewOffer = { id: string; name_key: string; variant?: string };
export type PreviewSection = { section: string; ends_at?: string; offers: PreviewOffer[] };
export type PreviewResult = { ok: true; at: string; sections: PreviewSection[] } | { ok: false; error: string; detail: string };

export async function previewCatalog(catalog: ShopCatalog, at: string | undefined, variants: Record<string, string>): Promise<PreviewResult> {
  const reply = await adminApi<{ at: string; sections: PreviewSection[] }>("POST", "/admin/config/shop/preview", { catalog, at, variants });
  if (reply.status !== 200) {
    return { ok: false, error: reply.body.error ?? String(reply.status), detail: reply.body.detail ?? "" };
  }
  return { ok: true, at: reply.body.at, sections: reply.body.sections };
}

export async function rollbackCatalog(version: number, reason: string): Promise<ConfigResult> {
  const reply = await adminApi<{ config_version: number }>("POST", "/admin/config/shop/rollback", { version, reason });
  if (reply.status !== 200) {
    return refusalOf(reply);
  }
  return { ok: true, version: reply.body.config_version };
}
