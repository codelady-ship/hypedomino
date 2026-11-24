import assert from "node:assert/strict";
import { test } from "node:test";
import { refusalOf } from "./result.ts";

test("a refusal carries the unnamed keys the API lists in keys, not parsed from its detail", () => {
  const detail = "invalid mission catalog: no approved en name for mission.win_50.name, mission.hands_3.name";
  assert.deepEqual(refusalOf({ status: 400, body: { error: "invalid", detail, keys: ["mission.win_50.name", "mission.hands_3.name"] } }), {
    ok: false,
    error: "invalid",
    detail,
    keys: ["mission.win_50.name", "mission.hands_3.name"],
  });
  assert.deepEqual(refusalOf({ status: 400, body: { error: "invalid", detail } }).keys, []);
});

test("a refusal without a body names its status", () => {
  assert.deepEqual(refusalOf({ status: 502, body: {} }), { ok: false, error: "502", detail: "", keys: [] });
});
