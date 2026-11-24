"use server";

import { adminApi } from "@/admin/api";
import type { GrantRequest } from "@/players/grant";
import { playerApiPath } from "@/players/record";
import type { BanCategory } from "@/players/sanctions";

export type PlayerActionResult = { ok: true; proposed: boolean; repeated: boolean } | { ok: false; error: string; detail: string };

async function act(method: string, path: string, body: Record<string, unknown>): Promise<PlayerActionResult> {
  const reply = await adminApi<{ repeated?: boolean }>(method, path, body);
  if (reply.status !== 200 && reply.status !== 202) {
    return { ok: false, error: reply.body.error ?? String(reply.status), detail: reply.body.detail ?? "" };
  }
  return { ok: true, proposed: reply.status === 202, repeated: reply.body.repeated ?? false };
}

export async function adjustCoins(id: string, amount: number, requestId: string, reason: string): Promise<PlayerActionResult> {
  return act("POST", `${playerApiPath(id)}/coins`, { amount, request_id: requestId, reason });
}

export async function grantReward(id: string, grant: GrantRequest, requestId: string, reason: string): Promise<PlayerActionResult> {
  return act("POST", `${playerApiPath(id)}/grant`, { ...grant, request_id: requestId, reason });
}

export async function banTemporarily(id: string, reason: string, category: BanCategory): Promise<PlayerActionResult> {
  return act("POST", `${playerApiPath(id)}/temporary-ban`, { reason, category });
}

export async function liftTemporaryBan(id: string, reason: string): Promise<PlayerActionResult> {
  return act("DELETE", `${playerApiPath(id)}/temporary-ban`, { reason });
}

export async function banFromRanked(id: string, reason: string, category: BanCategory): Promise<PlayerActionResult> {
  return act("POST", `${playerApiPath(id)}/ranked-ban`, { reason, category });
}

export async function liftRankedBan(id: string, reason: string): Promise<PlayerActionResult> {
  return act("DELETE", `${playerApiPath(id)}/ranked-ban`, { reason });
}

export async function shadowban(id: string, reason: string): Promise<PlayerActionResult> {
  return act("POST", `${playerApiPath(id)}/shadowban`, { reason });
}

export async function liftShadowban(id: string, reason: string): Promise<PlayerActionResult> {
  return act("DELETE", `${playerApiPath(id)}/shadowban`, { reason });
}

export async function forceRename(id: string, reason: string): Promise<PlayerActionResult> {
  return act("POST", `${playerApiPath(id)}/force-rename`, { reason });
}

export async function mute(id: string, reason: string): Promise<PlayerActionResult> {
  return act("POST", `${playerApiPath(id)}/mute`, { reason });
}

export async function unmute(id: string, reason: string): Promise<PlayerActionResult> {
  return act("DELETE", `${playerApiPath(id)}/mute`, { reason });
}

export async function removeFromLeaderboard(id: string, season: string, mode: string, reason: string): Promise<PlayerActionResult> {
  return act("POST", `${playerApiPath(id)}/leaderboard-removal`, { season, mode, reason });
}

export async function liftPenalty(id: string, reason: string): Promise<PlayerActionResult> {
  return act("POST", `${playerApiPath(id)}/penalties/lift`, { reason });
}

export async function removeOffense(id: string, queue: string, matchId: string, reason: string): Promise<PlayerActionResult> {
  return act("POST", `${playerApiPath(id)}/penalties/remove`, { queue, match_id: matchId, reason });
}
