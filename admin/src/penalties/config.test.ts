import assert from "node:assert/strict";
import { test } from "node:test";
import { badFields, configFromFields, fieldsFromConfig } from "./config.ts";

test("an empty field keeps the server default and lists round-trip", () => {
  const fields = fieldsFromConfig({ banHours: 48, banEscalationFactor: 1.5, casualQueueDelayLevels: [4, 8] });
  assert.equal(fields.casualQueueDelayLevels, "4, 8");
  assert.equal(fields.leaveWeight, "");
  assert.deepEqual(configFromFields(fields), { banHours: 48, banEscalationFactor: 1.5, casualQueueDelayLevels: [4, 8] });
  assert.deepEqual(configFromFields(fieldsFromConfig(null)), {});
});

test("negative, fractional or broken values are flagged", () => {
  const fields = { ...fieldsFromConfig(null), banHours: "-1", warningLevel: "1.5", banEscalationFactor: "x", casualQueueDelaySeconds: "60, a", queueDelaySeconds: "300" };
  assert.deepEqual(badFields(fields), ["warningLevel", "banHours", "banEscalationFactor", "casualQueueDelaySeconds"]);
});
