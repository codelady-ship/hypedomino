import assert from "node:assert/strict";
import { test } from "node:test";
import { changedKeys, invalidKeys, publishedValues, versionChanges, withinBounds, type ConfigKey } from "./keys.ts";

const turn: ConfigKey = { key: "TurnSeconds", type: "integer", unit: "seconds", min: 10, max: 60, default: 20, value: 25, set: true };
const afk: ConfigKey = { key: "AfkConsecutiveTimeouts", type: "integer", unit: "timeouts", min: 1, max: 10, default: 3, value: 3, set: false };
const frames: ConfigKey = { key: "ContentGateMinFrames", type: "integer", unit: "", min: 0, default: 20, value: 20, set: false };
const keys = [turn, afk];

test("a value is valid only as an integer inside the key's bounds", () => {
  assert.ok(withinBounds(turn, 10));
  assert.ok(withinBounds(turn, 60));
  assert.ok(!withinBounds(turn, 9));
  assert.ok(!withinBounds(turn, 61));
  assert.ok(!withinBounds(turn, 20.5));
  assert.ok(!withinBounds(turn, Number.NaN));
});

test("a key without a maximum takes any whole number from its minimum", () => {
  for (const value of [0, 20, 1_000_000]) {
    assert.ok(withinBounds(frames, value), String(value));
  }
  for (const value of [-1, 0.5, Number.NaN]) {
    assert.ok(!withinBounds(frames, value), String(value));
  }
  assert.deepEqual(invalidKeys([frames], { ContentGateMinFrames: -1 }), ["ContentGateMinFrames"]);
});

test("untouched keys keep their current value and are neither changed nor invalid", () => {
  assert.deepEqual(changedKeys(keys, {}), []);
  assert.deepEqual(invalidKeys(keys, {}), []);
  assert.deepEqual(changedKeys(keys, { TurnSeconds: 25 }), []);
});

test("edits are reported as changed and out-of-bounds edits as invalid", () => {
  assert.deepEqual(changedKeys(keys, { TurnSeconds: 30, AfkConsecutiveTimeouts: 11 }), ["TurnSeconds", "AfkConsecutiveTimeouts"]);
  assert.deepEqual(invalidKeys(keys, { TurnSeconds: 30, AfkConsecutiveTimeouts: 11 }), ["AfkConsecutiveTimeouts"]);
});

test("a publish carries every value that differs from the default and leaves defaults out", () => {
  assert.deepEqual(publishedValues(keys, {}), { TurnSeconds: 25 });
  assert.deepEqual(publishedValues(keys, { TurnSeconds: 20 }), {});
  assert.deepEqual(publishedValues(keys, { AfkConsecutiveTimeouts: 5 }), { TurnSeconds: 25, AfkConsecutiveTimeouts: 5 });
});

test("a version lists each key whose published value differs from the version before", () => {
  assert.deepEqual(versionChanges(keys, { TurnSeconds: 25 }, { TurnSeconds: 30, AfkConsecutiveTimeouts: 5 }), [
    { key: "TurnSeconds", before: 25, after: 30 },
    { key: "AfkConsecutiveTimeouts", before: 3, after: 5 },
  ]);
  assert.deepEqual(versionChanges(keys, { TurnSeconds: 25 }, { TurnSeconds: 25 }), []);
});

test("a key left out of a version counts as its default", () => {
  assert.deepEqual(versionChanges(keys, { TurnSeconds: 30 }, {}), [{ key: "TurnSeconds", before: 30, after: 20 }]);
  assert.deepEqual(versionChanges(keys, {}, { TurnSeconds: 20 }), []);
  assert.deepEqual(versionChanges(keys, {}, { AfkConsecutiveTimeouts: 4 }), [{ key: "AfkConsecutiveTimeouts", before: 3, after: 4 }]);
});
