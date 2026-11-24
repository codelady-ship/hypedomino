import type { MessageKey } from "@/i18n/translate";
import type { Refusal } from "@/config/result";

export type SeasonState = "scheduled" | "live" | "ending" | "closed";

export type Season = {
  id: string;
  name_key: string;
  start: string;
  end: string;
  closed_1v1?: string;
  closed_2v2?: string;
  no_rewards?: boolean;
  rewarded_1v1?: string;
  rewarded_2v2?: string;
  state: SeasonState;
};

export type SeasonMode = "1v1" | "2v2";

export type SeasonProgress = "open" | "closed" | "rewarded";

export type SeasonResult = { ok: true } | Refusal;

const instant = (value: string) => Date.parse(value);

export function seasonIdFromSlug(slug: string): string {
  return `season.${slug}`;
}

export function seasonNameKey(id: string): string {
  return `${id}.name`;
}

export function endAfterMonths(start: string, months: number): string {
  const end = new Date(start);
  end.setUTCMonth(end.getUTCMonth() + months);
  return end.toISOString().replace(".000Z", "Z");
}

export function isLastSeason(seasons: readonly Season[], season: Season): boolean {
  return seasons.at(-1)?.id === season.id;
}

export function canRemove(seasons: readonly Season[], season: Season): boolean {
  return season.state === "scheduled" && isLastSeason(seasons, season);
}

export function startLocked(seasons: readonly Season[], season: Season): boolean {
  return seasons.some((other) => other.id !== season.id && instant(other.end) === instant(season.start));
}

export function shiftedBy(seasons: readonly Season[], season: Season, end: string): string[] {
  if (instant(end) === instant(season.end)) {
    return [];
  }
  return seasons.filter((other) => instant(other.start) >= instant(season.end)).map((other) => other.id);
}

export function createIssues(slug: string, start: string, end: string, nextStart: string | null, now: string): MessageKey[] {
  const found: MessageKey[] = [];
  if (!/^[a-z0-9_]+$/.test(slug)) found.push("ranked.issue.season_id");
  if (!start || !end) {
    return [...found, "ranked.issue.window"];
  }
  if (!(instant(start) > instant(now))) found.push("ranked.issue.start_future");
  if (nextStart && instant(start) !== instant(nextStart)) found.push("ranked.issue.start_next");
  if (!(instant(end) > instant(start))) found.push("ranked.issue.window");
  return found;
}

export function rescheduleIssues(season: Season, start: string, end: string, now: string): MessageKey[] {
  const found: MessageKey[] = [];
  if (instant(start) !== instant(season.start) && !(instant(start) > instant(now))) found.push("ranked.issue.start_future");
  if (!(instant(end) > instant(start))) found.push("ranked.issue.window");
  return found;
}

export function extendIssues(season: Season, end: string): MessageKey[] {
  return instant(end) > instant(season.end) ? [] : ["ranked.issue.extend_later"];
}

export function seasonProgress(season: Season, mode: SeasonMode): SeasonProgress {
  const closed = mode === "1v1" ? season.closed_1v1 : season.closed_2v2;
  const rewarded = mode === "1v1" ? season.rewarded_1v1 : season.rewarded_2v2;
  if (rewarded) {
    return "rewarded";
  }
  return closed ? "closed" : "open";
}
