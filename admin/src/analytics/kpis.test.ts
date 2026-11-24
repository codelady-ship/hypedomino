import assert from "node:assert/strict";
import { test } from "node:test";
import { kpisCsv, kpisCsvFileName, latestRetention, matchModes, matchRows, reportWeeks, retentionPercent, type KpiReport } from "./kpis.ts";

const report: KpiReport = {
  from: "2026-11-09",
  to: "2026-11-11",
  matches: [
    { day: "2026-11-09", mode: "1v1", completed: 3 },
    { day: "2026-11-09", mode: "2v2", completed: 1 },
    { day: "2026-11-10", mode: "1v1", completed: 0 },
    { day: "2026-11-10", mode: "2v2", completed: 2 },
  ],
  dailyCohorts: [
    { start: "2026-11-09", size: 3, d1: 1, d7: null },
    { start: "2026-11-10", size: 0, d1: null, d7: null },
  ],
  weeklyCohorts: [{ start: "2026-11-09", size: 3, d1: null, d7: null }],
};

test("only a whole number of weeks from 1 to 13 is asked for, otherwise 4", () => {
  assert.equal(reportWeeks("1"), 1);
  assert.equal(reportWeeks("13"), 13);
  assert.equal(reportWeeks(undefined), 4);
  assert.equal(reportWeeks("0"), 4);
  assert.equal(reportWeeks("14"), 4);
  assert.equal(reportWeeks("2.5"), 4);
  assert.equal(reportWeeks("x"), 4);
});

test("retention is the share of the cohort with one decimal, none while pending or empty", () => {
  assert.equal(retentionPercent(1, 3), 33.3);
  assert.equal(retentionPercent(2, 3), 66.7);
  assert.equal(retentionPercent(0, 4), 0);
  assert.equal(retentionPercent(null, 3), null);
  assert.equal(retentionPercent(0, 0), null);
});

test("matches become one row per day with every mode and the total", () => {
  assert.deepEqual(matchModes(report.matches), ["1v1", "2v2"]);
  assert.deepEqual(matchRows(report.matches), [
    { day: "2026-11-09", perMode: { "1v1": 3, "2v2": 1 }, total: 4 },
    { day: "2026-11-10", perMode: { "1v1": 0, "2v2": 2 }, total: 2 },
  ]);
});

test("the latest retention is the newest cohort whose return day has ended", () => {
  assert.deepEqual(latestRetention(report.dailyCohorts, "d1"), { cohort: report.dailyCohorts[0], retained: 1, percent: 33.3 });
  assert.equal(latestRetention(report.dailyCohorts, "d7"), null);
});

test("the CSV lists the completed matches and every cohort with its counts and shares", () => {
  assert.equal(kpisCsvFileName(report), "kpis-2026-11-09-2026-11-11.csv");
  assert.equal(
    kpisCsv(report),
    [
      "table,start,mode,metric,value",
      "matches,2026-11-09,1v1,completed,3",
      "matches,2026-11-09,2v2,completed,1",
      "matches,2026-11-10,1v1,completed,0",
      "matches,2026-11-10,2v2,completed,2",
      "daily_cohort,2026-11-09,,size,3",
      "daily_cohort,2026-11-09,,d1_retained,1",
      "daily_cohort,2026-11-09,,d1_percent,33.3",
      "daily_cohort,2026-11-09,,d7_retained,",
      "daily_cohort,2026-11-09,,d7_percent,",
      "daily_cohort,2026-11-10,,size,0",
      "daily_cohort,2026-11-10,,d1_retained,",
      "daily_cohort,2026-11-10,,d1_percent,",
      "daily_cohort,2026-11-10,,d7_retained,",
      "daily_cohort,2026-11-10,,d7_percent,",
      "weekly_cohort,2026-11-09,,size,3",
      "weekly_cohort,2026-11-09,,d1_retained,",
      "weekly_cohort,2026-11-09,,d1_percent,",
      "weekly_cohort,2026-11-09,,d7_retained,",
      "weekly_cohort,2026-11-09,,d7_percent,",
    ].join("\n") + "\n",
  );
});
