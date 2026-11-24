import assert from "node:assert/strict";
import { test } from "node:test";
import { auditApiPath, auditExportPath, beforeIdOf, olderPageHref, pageSize, readFilter, utcInstant, type AuditEntry } from "./query.ts";

function entries(count: number, lastId: number): AuditEntry[] {
  return Array.from({ length: count }, (_, index) => ({
    id: lastId + count - 1 - index,
    at: "2026-11-03T17:50:29Z",
    actor_id: "a1",
    actor_roles: ["super_admin"],
    environment: "staging",
    action: "publish",
    object_type: "shop_offers",
    object_id: "shop",
    session_id: "s1",
  }));
}

test("the filter keeps known non-empty fields and the first of repeated values", () => {
  assert.deepEqual(readFilter({ actor: " a1 ", action: ["publish", "edit"], object_id: "", unknown: "x" }), { actor: "a1", action: "publish" });
});

test("minute inputs from the form are read as UTC instants", () => {
  assert.equal(utcInstant("2026-11-03T17:50"), "2026-11-03T17:50:00Z");
  assert.equal(utcInstant("2026-11-03T17:50:00+04:00"), "2026-11-03T17:50:00+04:00");
});

test("the API path carries the filter, the page cursor and the page size", () => {
  const path = auditApiPath({ actor: "a1", from: "2026-11-01T00:00", reason: "fix price" }, "120");
  assert.equal(path, "/admin/audit?actor=a1&reason=fix+price&from=2026-11-01T00%3A00%3A00Z&before_id=120&limit=50");
  assert.equal(auditApiPath({}, ""), "/admin/audit?limit=50");
});

test("the export path carries the filter without a page cursor or size", () => {
  assert.equal(auditExportPath({ action: "publish", to: "2026-11-02T00:00" }), "/admin/audit/export?action=publish&to=2026-11-02T00%3A00%3A00Z");
  assert.equal(auditExportPath({}), "/admin/audit/export?");
});

test("an older page exists only after a full page and starts below its last id", () => {
  assert.equal(olderPageHref({ action: "publish" }, entries(pageSize - 1, 10)), null);
  assert.equal(olderPageHref({ action: "publish" }, entries(pageSize, 10)), "/audit?action=publish&before_id=10");
});

test("the page cursor accepts only a positive id", () => {
  assert.equal(beforeIdOf({ before_id: "42" }), "42");
  assert.equal(beforeIdOf({ before_id: "0" }), "");
  assert.equal(beforeIdOf({ before_id: "-1" }), "");
  assert.equal(beforeIdOf({ before_id: "4x" }), "");
});
