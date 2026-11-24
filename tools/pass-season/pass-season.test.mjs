import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { checkPassSeason, parseSampleTable, xpTotalToReach } from "./pass-season.mjs";

const repoRoot = new URL("../../", import.meta.url);
const docText = readFileSync(new URL("docs/pass-missions-medals.md", repoRoot), "utf8");
const csvText = readFileSync(new URL("docs/economy/balance.csv", repoRoot), "utf8");
const defaultCurve = { base: 300, step: 50, max: 1000 };

function assertReported(errors, fragment) {
  assert.ok(errors.some((error) => error.includes(fragment)), `expected "${fragment}" in:\n${errors.join("\n")}`);
}

test("the committed sample season matches the XP formula and the economy coin budgets", () => {
  assert.deepEqual(checkPassSeason(docText, csvText), []);
});

test("the sample table has one row per level", () => {
  const rows = parseSampleTable(docText);
  assert.equal(rows.length, 100);
  assert.deepEqual(rows[0], { level: 1, xpTotal: 0, free: "30 coins", premium: "6 crystals + `avatar.s1_navigator` (`epic`)" });
});

test("the XP formula grows linearly and then flattens at the maximum", () => {
  assert.equal(xpTotalToReach(1, defaultCurve), 0);
  assert.equal(xpTotalToReach(2, defaultCurve), 300);
  assert.equal(xpTotalToReach(15, defaultCurve), 8750);
  assert.equal(xpTotalToReach(100, defaultCurve), 93750);
});

test("a wrong XP total is reported", () => {
  assertReported(checkPassSeason(docText.replace("| 50 | 43750 |", "| 50 | 43700 |"), csvText), "level 50: XP total 43700, formula gives 43750");
});

test("a free track that does not pay the economy budget is reported", () => {
  assertReported(checkPassSeason(docText.replace("| 3 | 650 | 30 coins |", "| 3 | 650 | 40 coins |"), csvText), "free track pays 1510 coins, PassFreeTrackCoins is 1500");
});

test("a changed economy budget is reported", () => {
  assertReported(checkPassSeason(docText, csvText.replace("PassPremiumTrackCrystals,600,", "PassPremiumTrackCrystals,700,")), "premium track pays 600 crystals, PassPremiumTrackCrystals is 700");
});

test("a missing level is reported", () => {
  assertReported(checkPassSeason(docText.replace("| 42 | 35750 | - | 6 crystals |\n", ""), csvText), "sample table has 99 levels, PassLevels is 100");
});

test("a premium level without a reward is reported", () => {
  assertReported(checkPassSeason(docText.replace("| 42 | 35750 | - | 6 crystals |", "| 42 | 35750 | - | - |"), csvText), "level 42: premium track has no reward");
});

test("a curve missing from the config table is reported", () => {
  assert.deepEqual(checkPassSeason(docText.replace("| `PassLevels` | 100 |", "| `PassLevels` | many |"), csvText), ["config table does not state PassLevels, PassXpBase, PassXpStep and PassXpMax"]);
});
