import type { MessageKey } from "@/i18n/translate";

export type EconomyAlert = "sink_ratio" | "account_daily_earn" | "median_balance";

export type EconomyReport = {
  from: string;
  to: string;
  day: string;
  coinsCreated: number;
  coinsSpent: number;
  sinkRatioPercent: number | null;
  medianBalance: number;
  sourceMix: Record<string, number>;
  accountsOverDailyEarn: number;
  alerts: EconomyAlert[];
};

export type EconomyComparison = { config_version: number; domain: string; published_at: string; before: EconomyReport; after: EconomyReport };

export const alertLabels: Readonly<Record<EconomyAlert, MessageKey>> = {
  sink_ratio: "analytics.alert.sink_ratio",
  account_daily_earn: "analytics.alert.account_daily_earn",
  median_balance: "analytics.alert.median_balance",
};

export function sourceReasons(reports: readonly EconomyReport[]): string[] {
  const totals = new Map<string, number>();
  for (const report of reports) {
    for (const [reason, coins] of Object.entries(report.sourceMix)) {
      totals.set(reason, (totals.get(reason) ?? 0) + coins);
    }
  }
  return [...totals.keys()].sort((a, b) => totals.get(b)! - totals.get(a)! || a.localeCompare(b));
}

export function comparedVersion(value: string | undefined): number | null {
  return value && /^[1-9][0-9]{0,14}$/.test(value) ? Number(value) : null;
}

export type MonitorWindow = "now" | "before" | "after";

export type WindowReport = { window: MonitorWindow; report: EconomyReport };

const csvHeader = "window,from,to,day,metric,reason,value";

export function monitorsCsv(windows: readonly WindowReport[]): string {
  const reasons = sourceReasons(windows.map((entry) => entry.report));
  const lines = windows.flatMap((entry) => windowLines(entry, reasons));
  return [csvHeader, ...lines].join("\n") + "\n";
}

export function monitorsCsvFileName(now: EconomyReport, version: number | null): string {
  const day = now.to.slice(0, 10);
  return version === null ? `economy-monitors-${day}.csv` : `economy-monitors-${day}-v${version}.csv`;
}

function windowLines({ window, report }: WindowReport, reasons: readonly string[]): string[] {
  const metrics: [string, string, number | null][] = [
    ["coins_created", "", report.coinsCreated],
    ["coins_spent", "", report.coinsSpent],
    ["sink_ratio_percent", "", report.sinkRatioPercent],
    ["median_balance", "", report.medianBalance],
    ["accounts_over_daily_earn", "", report.accountsOverDailyEarn],
    ...reasons.map((reason): [string, string, number] => ["source_mix", reason, report.sourceMix[reason] ?? 0]),
  ];
  return metrics.map(([metric, reason, value]) => [window, report.from, report.to, report.day, metric, reason, value === null ? "" : String(value)].map(csvField).join(","));
}

function csvField(value: string): string {
  return /[",\n\r]/.test(value) ? `"${value.replaceAll('"', '""')}"` : value;
}
