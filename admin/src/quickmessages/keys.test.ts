import assert from "node:assert/strict";
import { test } from "node:test";
import { badKeys, keysFromText, sameKeys } from "./keys.ts";

test("one key per line, blanks dropped", () => {
  assert.deepEqual(keysFromText(" quick.hello \n\nquick.bye\n"), ["quick.hello", "quick.bye"]);
});

test("a key outside quick.<slug> or listed twice is flagged", () => {
  assert.deepEqual(badKeys(["quick.hello", "match.result.win", "quick.Hello", "quick.hello"]), ["match.result.win", "quick.Hello", "quick.hello"]);
  assert.deepEqual(badKeys(["quick.hello", "quick.well_played"]), []);
});

test("order counts as a change", () => {
  assert.ok(sameKeys(["quick.a"], ["quick.a"]));
  assert.ok(!sameKeys(["quick.a", "quick.b"], ["quick.b", "quick.a"]));
});
