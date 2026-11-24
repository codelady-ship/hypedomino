"use server";

import { adminApi } from "@/admin/api";
import type { AssetCheckResult, AssetUploadStart } from "@/assets/assets";

export async function startAssetUpload(itemId: string): Promise<AssetUploadStart> {
  const reply = await adminApi<{ asset: { id: string }; upload_url: string }>("POST", "/admin/assets/uploads", { item_id: itemId, slot: "static" });
  if (reply.status !== 200) {
    return { ok: false, error: reply.body.error ?? String(reply.status), detail: reply.body.detail ?? "" };
  }
  return { ok: true, id: reply.body.asset.id, uploadUrl: reply.body.upload_url };
}

export async function checkAsset(id: string): Promise<AssetCheckResult> {
  const reply = await adminApi<{ asset: { id: string } }>("POST", `/admin/assets/${encodeURIComponent(id)}/check`);
  if (reply.status !== 200) {
    return { ok: false, error: reply.body.error ?? String(reply.status), detail: reply.body.detail ?? "" };
  }
  return { ok: true };
}
