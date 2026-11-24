"use server";

import { adminApi } from "@/admin/api";
import { refusalOf, type ConfigResult } from "@/config/result";
import type { RatingConfig } from "@/ranked/config";
import type { SeasonResult } from "@/ranked/seasons";

export async function publishRating(catalog: RatingConfig, reason: string): Promise<ConfigResult> {
  const reply = await adminApi<{ config_version: number }>("PUT", "/admin/config/rating", { catalog, reason });
  if (reply.status !== 200) {
    return { ok: false, error: reply.body.error ?? String(reply.status), detail: reply.body.detail ?? "" };
  }
  return { ok: true, version: reply.body.config_version };
}

export async function rollbackRating(version: number, reason: string): Promise<ConfigResult> {
  const reply = await adminApi<{ config_version: number }>("POST", "/admin/config/rating/rollback", { version, reason });
  if (reply.status !== 200) {
    return { ok: false, error: reply.body.error ?? String(reply.status), detail: reply.body.detail ?? "" };
  }
  return { ok: true, version: reply.body.config_version };
}

async function seasonChange(method: string, path: string, body: unknown, success: number): Promise<SeasonResult> {
  const reply = await adminApi<object>(method, path, body);
  if (reply.status !== success) {
    return refusalOf(reply);
  }
  return { ok: true };
}

export async function createSeason(id: string, start: string, end: string, noRewards: boolean, reason: string): Promise<SeasonResult> {
  return seasonChange("POST", "/admin/ranked/seasons", { id, start, end, no_rewards: noRewards, reason }, 201);
}

export async function rescheduleSeason(id: string, start: string, end: string, noRewards: boolean, reason: string): Promise<SeasonResult> {
  return seasonChange("PATCH", `/admin/ranked/seasons/${encodeURIComponent(id)}`, { start, end, no_rewards: noRewards, reason }, 200);
}

export async function extendSeason(id: string, end: string, reason: string): Promise<SeasonResult> {
  return seasonChange("POST", `/admin/ranked/seasons/${encodeURIComponent(id)}/extend`, { end, reason }, 200);
}

export async function deleteSeason(id: string, reason: string): Promise<SeasonResult> {
  return seasonChange("DELETE", `/admin/ranked/seasons/${encodeURIComponent(id)}`, { reason }, 204);
}
