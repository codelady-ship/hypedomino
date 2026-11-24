import type { SearchParams } from "../audit/query.ts";
import type { Sanction } from "../players/sanctions.ts";

export type ReviewSignal ={ kind: string; weight: number; evidence: unknown; at: string };

export type QueueDeadline = { due_at: string; overdue: boolean };

export type Review = QueueDeadline & {
  id: number;
  account_id: string;
  nickname: string;
  cause: string;
  opened_at: string;
  shadowbanned_at: string | null;
  signals: ReviewSignal[];
  decisions: EarlierDecision[];
};

export function signalWeight(signals: readonly ReviewSignal[]): number {
  return signals.reduce((total, signal) => total + signal.weight, 0);
}

export function evidenceSummary(evidence: unknown): string {
  if (evidence === null || typeof evidence !== "object" || Array.isArray(evidence)) {
    return JSON.stringify(evidence) ?? "";
  }
  return Object.entries(evidence)
    .map(([key, value]) => `${key}: ${typeof value === "string" ? value : JSON.stringify(value)}`)
    .join("; ");
}

export type ReportEntry = { id: number; reporter: string; reason: string; name_text: string; match_id: string | null; updated_at: string };

export type EarlierDecision = { queue: "reports" | "reviews" | "appeals"; outcome: string; reasons: string[]; reports: number; decided_at: string };

export type ReportTarget = QueueDeadline & { account_id: string; nickname: string; rename_required: boolean; reports: ReportEntry[]; decisions: EarlierDecision[] };

export type Appeal = QueueDeadline & {
  id: number;
  account_id: string;
  nickname: string;
  sanction_id: number;
  text: string;
  state: "open" | "upheld" | "lifted";
  created_at: string;
  sanction: Sanction;
  decisions: EarlierDecision[];
};

export function decidedReports(decisions: readonly EarlierDecision[]): { upheld: number; rejected: number } {
  const totals = { upheld: 0, rejected: 0 };
  for (const decision of decisions) {
    if (decision.queue === "reports" && (decision.outcome === "upheld" || decision.outcome === "rejected")) {
      totals[decision.outcome] += decision.reports;
    }
  }
  return totals;
}

export const queuePageSize = 50;

export const reviewCauses = ["score", "hard_signal"] as const;

export const reportReasons = ["offensive_nickname", "offensive_room_name", "impersonation", "other"] as const;

export type ReviewQuery = { cause: (typeof reviewCauses)[number] | ""; account: string; after: string };

export type QueueQuery = ReviewQuery | ReportQuery | AppealQuery | SuspiciousQuery;

export type ReportQuery = { reason: (typeof reportReasons)[number] | ""; account: string; after: string };

export const appealKinds = ["temporary_ban", "permanent_ban", "ranked_ban", "penalty_ban", "forced_rename"] as const;

export type AppealQuery = { kind: (typeof appealKinds)[number] | ""; account: string; after: string };

export const suspiciousSignals = ["win_trade", "feeding", "partner_fit"] as const;

export type SuspiciousQuery = { signal: (typeof suspiciousSignals)[number] | ""; account: string; after: string };

const reviewIdPattern = /^[1-9][0-9]{0,17}$/;

const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

function single(value: string | string[] | undefined): string {
  return (Array.isArray(value) ? value[0] : value)?.trim() ?? "";
}

function known<T extends string>(values: readonly T[], value: string): T | "" {
  return values.find((candidate) => candidate === value) ?? "";
}

export function readReviewQuery(params: SearchParams): ReviewQuery {
  const after = single(params.after);
  return { cause: known(reviewCauses, single(params.cause)), account: single(params.account), after: reviewIdPattern.test(after) ? after : "" };
}

export function readReportQuery(params: SearchParams): ReportQuery {
  const after = single(params.after);
  return { reason: known(reportReasons, single(params.reason)), account: single(params.account), after: uuidPattern.test(after) ? after : "" };
}

export function readAppealQuery(params: SearchParams): AppealQuery {
  const after = single(params.after);
  return { kind: known(appealKinds, single(params.kind)), account: single(params.account), after: reviewIdPattern.test(after) ? after : "" };
}

export function readSuspiciousQuery(params: SearchParams): SuspiciousQuery {
  const after = single(params.after);
  return { signal: known(suspiciousSignals, single(params.signal)), account: single(params.account), after: uuidPattern.test(after) ? after : "" };
}

function searchOf(query: QueueQuery): URLSearchParams {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(query)) {
    if (value) {
      search.set(key, value);
    }
  }
  return search;
}

export function queueApiPath(path: "/admin/reviews" | "/admin/reports" | "/admin/appeals" | "/admin/matches/suspicious", query: QueueQuery): string {
  const search = searchOf(query);
  search.set("limit", String(queuePageSize));
  return `${path}?${search}`;
}

export function nextQueueHref(screen: "/reviews" | "/reports" | "/appeals" | "/matches", query: QueueQuery, cursors: readonly string[]): string | null {
  if (cursors.length < queuePageSize) {
    return null;
  }
  return `${screen}?${searchOf({ ...query, after: cursors[cursors.length - 1] })}`;
}
