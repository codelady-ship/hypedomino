import assert from "node:assert/strict";
import test from "node:test";
import { pendingChanges, shownPrice, statusCounts, visibleRows, type PushPlan, type PushRow, type StoreReport } from "./push.ts";

const rows: PushRow[] = [
  { product: "pass_premium", country: "AZ", target_cents: 299, current: { currency: "AZN", amount: "8.49" }, next: { currency: "AZN", amount: "4.99" }, status: "change" },
  { product: "pass_premium", country: "TR", target_cents: 299, current: { currency: "TRY", amount: "99.99" }, next: { currency: "TRY", amount: "99.99" }, status: "unchanged" },
  { product: "pass_premium", country: "XK", target_cents: 299, status: "not_in_store", detail: "territory has no equalized price point" },
  { product: "crystals_tier1", country: "AZ", target_cents: 59, status: "error", detail: "answered 500" },
  { product: "crystals_tier2", country: "AZ", target_cents: 299, status: "written" },
  { product: "crystals_tier3", country: "AZ", target_cents: 599, status: "failed", detail: "answered 400" },
];

const report: StoreReport = { store: "google_play", configured: true, rows, counts: { change: 1, unchanged: 1, not_in_store: 1, error: 1, written: 1, failed: 1 } };

test("the change filter keeps the rows that need attention and drops unchanged and missing ones", () => {
  assert.deepEqual(
    visibleRows(rows, true).map((row) => `${row.product} ${row.country} ${row.status}`),
    ["pass_premium AZ change", "crystals_tier1 AZ error", "crystals_tier2 AZ written", "crystals_tier3 AZ failed"],
  );
  assert.equal(visibleRows(rows, false).length, rows.length);
  assert.notEqual(visibleRows(rows, false), rows);
});

test("the summary lists only statuses that occur, changes and results first", () => {
  assert.deepEqual(statusCounts(report), [
    ["change", 1],
    ["written", 1],
    ["failed", 1],
    ["error", 1],
    ["not_in_store", 1],
    ["unchanged", 1],
  ]);
  assert.deepEqual(statusCounts({ ...report, counts: { unchanged: 80 } }), [["unchanged", 80]]);
  assert.deepEqual(statusCounts({ store: "app_store", configured: false, rows: [], counts: {} }), []);
});

test("apply is offered only while a store has a pending change", () => {
  const plan: PushPlan = { config_version: 3, stores: [report, { store: "app_store", configured: false, rows: [], counts: {} }] };
  assert.equal(pendingChanges(plan), 1);
  assert.equal(pendingChanges({ ...plan, stores: [{ ...report, counts: { unchanged: 5 } }] }), 0);
});

test("a store price shows its amount and currency as the store reports them", () => {
  assert.equal(shownPrice({ currency: "AZN", amount: "1.003" }), "1.003 AZN");
  assert.equal(shownPrice(undefined), "");
});
