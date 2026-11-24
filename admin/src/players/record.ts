import type { Sanction } from "./sanctions.ts";

export type PlayerSummary = { id: string; nickname: string; state: string; country: string; created_at: string };

export type PlayerAccount = PlayerSummary & {
  emails: string[];
  identities?: { provider: string; subject: string }[];
  banned_until: string | null;
  shadowbanned_at: string | null;
  muted_until: string | null;
  deletion_requested_at: string | null;
  deleted_at: string | null;
};

export type PlayerProfile = { rename_required: boolean; level: number; xp: number };

export type PlayerRecord = {
  account: PlayerAccount;
  profile: PlayerProfile | null;
  nickname_history: { nickname: string; replaced_at: string }[];
  inventory: { item_id: string; source_type: string; source_ref: string; granted_at: string; revoked_at: string | null }[];
  coins: number;
  crystals: number;
  ledger: { id: number; currency: string; amount: number; reason: string; source_type: string; source_ref: string; config_version: number; balance_after: number; at: string }[];
  grants: { id: number; bundle_id: string; revision: number; source_type: string; source_ref: string; config_version: number; at: string; revoked_at: string | null }[];
  purchases: { platform: string; transaction_id: string; offer_id: string; product_id: string; environment: string; state: string; purchased_at: string; refunded_at: string | null }[];
  matches: { match_id: string; started_at: string; ended_at: string | null; mode: string; kind: string; team: number; result: string; score_a: number; score_b: number; cancelled: boolean; suspicious: string[] }[];
  pair_signals: { kind: string; counterpart: string; mode: string; matches: number; wins: number; offenses: number; window_hours: number; first_seen: string; updated_at: string }[];
  missions: MissionAssignment[];
  reports: PlayerReport[];
  sanctions: Sanction[];
};

export type PlayerReport = { id: number; reporter: string; target: string; reason: string; name_text: string; match_id: string | null; state: string; created_at: string; updated_at: string; decided_at: string | null };

export type MissionAssignment = { mission_id: string; type: string; slot: number; period: string; progress: number; target: number; assigned_at: string; completed_at: string | null; replaced_at: string | null };

export type SearchParams = Readonly<Record<string, string | string[] | undefined>>;

export const identityProviders = ["google", "apple"] as const;

export type IdentityProvider = (typeof identityProviders)[number];

export const accountStates = ["active", "deleting", "deleted", "banned"] as const;

export type PlayerListSearch = { q: string; state: (typeof accountStates)[number] | ""; after: string };

export type PlayerSearch = PlayerListSearch | { provider: IdentityProvider; subject: string };

export const playerPageSize = 50;

const accountIdPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

function firstValue(params: SearchParams, key: string): string {
  const value = params[key];
  return (Array.isArray(value) ? value[0] : value)?.trim() ?? "";
}

export function searchQuery(params: SearchParams): string {
  return firstValue(params, "q");
}

export function playerSearch(params: SearchParams): PlayerSearch | null {
  const subject = firstValue(params, "subject");
  if (subject) {
    const provider = identityProviders.find((candidate) => candidate === firstValue(params, "provider")) ?? "google";
    return { provider, subject };
  }
  const q = searchQuery(params);
  const state = accountStates.find((candidate) => candidate === firstValue(params, "state")) ?? "";
  const after = firstValue(params, "after");
  return q || state ? { q, state, after: accountIdPattern.test(after) ? after : "" } : null;
}

function searchOf(search: PlayerSearch): URLSearchParams {
  return new URLSearchParams(Object.entries(search).filter(([, value]) => value));
}

export function playersApiPath(search: PlayerSearch): string {
  const params = searchOf(search);
  if ("q" in search) {
    params.set("limit", String(playerPageSize));
  }
  return `/admin/players?${params}`;
}

export function nextPlayersHref(search: PlayerSearch, ids: readonly string[]): string | null {
  if (!("q" in search) || ids.length < playerPageSize) {
    return null;
  }
  return `/players?${searchOf({ ...search, after: ids[ids.length - 1] })}`;
}

export function playerApiPath(id: string): string {
  return `/admin/players/${encodeURIComponent(id)}`;
}

export function banActive(account: PlayerAccount, now: Date): boolean {
  return account.banned_until !== null && new Date(account.banned_until) > now;
}

export function muteActive(account: PlayerAccount, now: Date): boolean {
  return account.muted_until !== null && new Date(account.muted_until) > now;
}

export function validAdjustment(amount: number): boolean {
  return Number.isSafeInteger(amount) && amount !== 0;
}

export function missionProgress(assignment: Pick<MissionAssignment, "progress" | "target">): string {
  return assignment.target > 0 ? `${assignment.progress}/${assignment.target}` : String(assignment.progress);
}

export function missionPlace(assignment: Pick<MissionAssignment, "type" | "slot">): string {
  return assignment.slot > 0 ? `${assignment.type} ${assignment.slot}` : assignment.type;
}

export function reportSide(playerId: string, report: Pick<PlayerReport, "reporter" | "target">): { about: boolean; counterpart: string } {
  return report.target === playerId ? { about: true, counterpart: report.reporter } : { about: false, counterpart: report.target };
}
