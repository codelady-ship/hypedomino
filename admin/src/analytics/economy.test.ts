import assert from "node:assert/strict";
import { test } from "node:test";
import { comparedVersion, monitorsCsv, monitorsCsvFileName, sourceReasons, type EconomyReport } from "./economy.ts";

function report(sourceMix: Record<string, number>): EconomyReport {
  return { from: "", to: "", day: "", coinsCreated: 0, coinsSpent: 0, sinkRatioPercent: null, medianBalance: 0, sourceMix, accountsOverDailyEarn: 0, alerts: [] };
}

test("source reasons are ordered by coins across every report, then by name", () => {
  const reasons = sourceReasons([report({ mission: 50, match_reward: 400 }), report({ daily_login: 300, mission: 300, first_win: 350 })]);
  assert.deepEqual(reasons, ["match_reward", "first_win", "mission", "daily_login"]);
});

test("only a positive whole version is compared", () => {
  assert.equal(comparedVersion("12"), 12);
  assert.equal(comparedVersion(undefined), null);
  assert.equal(comparedVersion(""), null);
  assert.equal(comparedVersion("0"), null);
  assert.equal(comparedVersion("-3"), null);
  assert.equal(comparedVersion("2.5"), null);
});

test("the CSV lists every monitor and reason per window with its period", () => {
  const now = { ...report({ match_reward: 400 }), from: "2026-11-08T00:00:00Z", to: "2026-11-15T00:00:00Z", day: "2026-11-14", coinsCreated: 900, coinsSpent: 450, sinkRatioPercent: 50, medianBalance: 120, accountsOverDailyEarn: 2 };
  const before = { ...report({ mission: 30 }), from: "2026-10-25T00:00:00Z", to: "2026-11-01T00:00:00Z", day: "2026-10-31" };
  const csv = monitorsCsv([
    { window: "now", report: now },
    { window: "before", report: before },
  ]);
  assert.equal(
    csv,
    [
      "window,from,to,day,metric,reason,value",
      "now,2026-11-08T00:00:00Z,2026-11-15T00:00:00Z,2026-11-14,coins_created,,900",
      "now,2026-11-08T00:00:00Z,2026-11-15T00:00:00Z,2026-11-14,coins_spent,,450",
      "now,2026-11-08T00:00:00Z,2026-11-15T00:00:00Z,2026-11-14,sink_ratio_percent,,50",
      "now,2026-11-08T00:00:00Z,2026-11-15T00:00:00Z,2026-11-14,median_balance,,120",
      "now,2026-11-08T00:00:00Z,2026-11-15T00:00:00Z,2026-11-14,accounts_over_daily_earn,,2",
      "now,2026-11-08T00:00:00Z,2026-11-15T00:00:00Z,2026-11-14,source_mix,match_reward,400",
      "now,2026-11-08T00:00:00Z,2026-11-15T00:00:00Z,2026-11-14,source_mix,mission,0",
      "before,2026-10-25T00:00:00Z,2026-11-01T00:00:00Z,2026-10-31,coins_created,,0",
      "before,2026-10-25T00:00:00Z,2026-11-01T00:00:00Z,2026-10-31,coins_spent,,0",
      "before,2026-10-25T00:00:00Z,2026-11-01T00:00:00Z,2026-10-31,sink_ratio_percent,,",
      "before,2026-10-25T00:00:00Z,2026-11-01T00:00:00Z,2026-10-31,median_balance,,0",
      "before,2026-10-25T00:00:00Z,2026-11-01T00:00:00Z,2026-10-31,accounts_over_daily_earn,,0",
      "before,2026-10-25T00:00:00Z,2026-11-01T00:00:00Z,2026-10-31,source_mix,match_reward,0",
      "before,2026-10-25T00:00:00Z,2026-11-01T00:00:00Z,2026-10-31,source_mix,mission,30",
    ].join("\n") + "\n",
  );
});

test("a CSV field holding a comma or quote is quoted", () => {
  const csv = monitorsCsv([{ window: "now", report: report({ 'odd,"reason"': 5 }) }]);
  assert.match(csv, /,source_mix,"odd,""reason""",5\n$/);
});

test("the CSV file name carries the day and the compared version", () => {
  const now = { ...report({}), to: "2026-11-15T00:00:00Z" };
  assert.equal(monitorsCsvFileName(now, null), "economy-monitors-2026-11-15.csv");
  assert.equal(monitorsCsvFileName(now, 12), "economy-monitors-2026-11-15-v12.csv");
});
