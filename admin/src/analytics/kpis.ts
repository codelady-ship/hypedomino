export type MatchDay = { day: string; mode: string; completed: number };

export type Cohort = { start: string; size: number; d1: number | null; d7: number | null };

export type KpiReport = {
  from: string;
  to: string;
  matches: MatchDay[];
  dailyCohorts: Cohort[];
  weeklyCohorts: Cohort[];
};

export const defaultWeeks = 4;
export const maxWeeks = 13;

export function reportWeeks(value: string | undefined): number {
  return value && /^[1-9][0-9]?$/.test(value) && Number(value) <= maxWeeks ? Number(value) : defaultWeeks;
}

export function retentionPercent(retained: number | null, size: number): number | null {
  if (retained === null || size === 0) {
    return null;
  }
  return Math.round((retained * 1000) / size) / 10;
}

export type MatchRow = { day: string; perMode: Record<string, number>; total: number };

export function matchModes(matches: readonly MatchDay[]): string[] {
  return [...new Set(matches.map((match) => match.mode))];
}

export function matchRows(matches: readonly MatchDay[]): MatchRow[] {
  const rows = new Map<string, MatchRow>();
  for (const match of matches) {
    const row = rows.get(match.day) ?? { day: match.day, perMode: {}, total: 0 };
    row.perMode[match.mode] = match.completed;
    row.total += match.completed;
    rows.set(match.day, row);
  }
  return [...rows.values()];
}

export type Retention = { cohort: Cohort; retained: number; percent: number | null };

export function latestRetention(cohorts: readonly Cohort[], day: "d1" | "d7"): Retention | null {
  for (let index = cohorts.length - 1; index >= 0; index--) {
    const cohort = cohorts[index]!;
    const retained = cohort[day];
    if (retained !== null) {
      return { cohort, retained, percent: retentionPercent(retained, cohort.size) };
    }
  }
  return null;
}

const csvHeader = "table,start,mode,metric,value";

export function kpisCsv(report: KpiReport): string {
  const lines = [
    ...report.matches.map((match) => ["matches", match.day, match.mode, "completed", String(match.completed)]),
    ...report.dailyCohorts.flatMap((cohort) => cohortLines("daily_cohort", cohort)),
    ...report.weeklyCohorts.flatMap((cohort) => cohortLines("weekly_cohort", cohort)),
  ];
  return [csvHeader, ...lines.map((fields) => fields.join(","))].join("\n") + "\n";
}

export function kpisCsvFileName(report: KpiReport): string {
  return `kpis-${report.from}-${report.to}.csv`;
}

function cohortLines(table: string, cohort: Cohort): string[][] {
  const metrics: [string, number | null][] = [
    ["size", cohort.size],
    ["d1_retained", cohort.d1],
    ["d1_percent", retentionPercent(cohort.d1, cohort.size)],
    ["d7_retained", cohort.d7],
    ["d7_percent", retentionPercent(cohort.d7, cohort.size)],
  ];
  return metrics.map(([metric, value]) => [table, cohort.start, "", metric, value === null ? "" : String(value)]);
}
