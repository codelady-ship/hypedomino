import assert from "node:assert/strict";
import { test } from "node:test";
import { changeText, entryChanges, fieldChanges } from "./changes.ts";

test("a nested field is named by its path and a missing one shows as a dash", () => {
  assert.deepEqual(fieldChanges({ target: 3, filters: {} }, { target: 5, filters: { mode: "2v2" } }), [
    { field: "target", before: "3", after: "5" },
    { field: "filters.mode", before: "—", after: "2v2" },
  ]);
  assert.deepEqual(fieldChanges({ filters: { queue: "ranked" } }, { filters: {} }), [{ field: "filters.queue", before: "ranked", after: "—" }]);
});

test("a list is compared and shown whole", () => {
  assert.deepEqual(fieldChanges({ medals: ["medal.a"] }, { medals: ["medal.a", "medal.b"] }), [{ field: "medals", before: '["medal.a"]', after: '["medal.a","medal.b"]' }]);
  assert.deepEqual(fieldChanges({ medals: ["medal.a"] }, { medals: ["medal.a"] }), []);
});

test("entries are matched by id: new ones are added, missing ones removed, unchanged ones left out", () => {
  const before = [
    { id: "a", weight: 1 },
    { id: "b", weight: 1 },
    { id: "d", weight: 1 },
  ];
  const after = [
    { id: "a", weight: 2 },
    { id: "c", weight: 1 },
    { id: "d", weight: 1 },
  ];
  assert.deepEqual(entryChanges(before, after), [
    { id: "a", change: "changed", fields: [{ field: "weight", before: "1", after: "2" }] },
    { id: "c", change: "added", fields: [] },
    { id: "b", change: "removed", fields: [] },
  ]);
});

test("a change reads as field, before and after", () => {
  assert.equal(changeText({ field: "bundle.coins", before: "0", after: "50" }), "bundle.coins 0 → 50");
});
