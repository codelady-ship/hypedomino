import assert from "node:assert/strict";
import { test } from "node:test";
import { translator } from "./translate.ts";

test("times show in UTC in the chosen language", () => {
  assert.equal(translator("en").when("2026-11-03T21:50:29+04:00"), "Nov 3, 2026, 17:50:29 UTC");
  assert.equal(translator("az").when("2026-11-03T21:50:29+04:00"), "3 noy 2026, 17:50:29 UTC");
  assert.equal(translator("az").when("not a time"), "not a time");
});

test("numbers group by the chosen language and ledger amounts show their sign", () => {
  assert.equal(translator("en").number(1234567), "1,234,567");
  assert.equal(translator("az").number(1234567), "1.234.567");
  assert.equal(translator("en").signed(50), "+50");
  assert.equal(translator("en").signed(-20), "-20");
  assert.equal(translator("en").signed(0), "0");
});
