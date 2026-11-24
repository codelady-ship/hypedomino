"use server";

import { adminApi } from "@/admin/api";
import { refusalOf, type ConfigResult } from "@/config/result";
import type { BundleEdit } from "@/rewards/bundles";

export async function editRewardBundle(id: string, revision: number, edit: BundleEdit, reason: string): Promise<ConfigResult> {
  const reply = await adminApi<{ bundle: { revision: number } }>("PUT", `/admin/reward-bundles/${encodeURIComponent(id)}`, { revision, ...edit, reason });
  if (reply.status !== 200) {
    return refusalOf(reply);
  }
  return { ok: true, version: reply.body.bundle.revision };
}

export async function createRewardBundle(id: string, bundle: BundleEdit, reason: string): Promise<ConfigResult> {
  const reply = await adminApi<{ config_version: number }>("POST", "/admin/reward-bundles", { id, ...bundle, reason });
  if (reply.status !== 201) {
    return refusalOf(reply);
  }
  return { ok: true, version: reply.body.config_version };
}
