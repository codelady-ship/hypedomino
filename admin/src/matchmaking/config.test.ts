import assert from "node:assert/strict";
import { test } from "node:test";
import { badFields, configFromFields, fieldsFromConfig } from "./config.ts";

test("an empty field keeps the server default", () => {
  const fields = fieldsFromConfig({ mmrWindowStart: 100, casualBotLevel: "hard" });
  assert.equal(fields.mmrWindowStart, "100");
  assert.equal(fields.mmrWindowMax, "");
  assert.deepEqual(configFromFields(fields), { mmrWindowStart: 100, casualBotLevel: "hard" });
  assert.deepEqual(configFromFields(fieldsFromConfig(null)), {});
});

test("a negative or fractional number and an unknown bot level are flagged", () => {
  const fields = { ...fieldsFromConfig(null), mmrWindowStep: "-5", rankWindowDivisions: "1.5", rankedBotLevel: "expert", casualBotFillSeconds: "30" };
  assert.deepEqual(badFields(fields), ["mmrWindowStep", "rankWindowDivisions", "rankedBotLevel"]);
});
