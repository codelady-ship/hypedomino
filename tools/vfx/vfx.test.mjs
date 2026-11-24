import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { checkEffects, docErrors, docPath, drawCallsOf, spriteFiles, spritesFolder, stepParticles, tablePath } from "./vfx.mjs";

const repoRoot = new URL("../../", import.meta.url);
const read = (path) => readFileSync(new URL(path, repoRoot), "utf8");
const tokens = JSON.parse(read("docs/design-tokens.json"));
const table = JSON.parse(read(tablePath));
const files = spriteFiles(fileURLToPath(new URL(spritesFolder, repoRoot)));
const copy = () => structuredClone(table);
const effect = (draft, name) => draft.effects.find((entry) => entry.name === name);

test("the committed effects fit every budget", () => {
  assert.deepEqual(checkEffects(tokens, table, files), []);
});

test("the committed doc holds the current tables", () => {
  assert.deepEqual(docErrors(read(docPath), tokens, table), []);
});

test("particles over an effect budget are reported", () => {
  const draft = copy();
  effect(draft, "tilePlace").levels.low.particles[0].count = 5;
  assert.ok(checkEffects(tokens, draft, files).includes("tilePlace low: 5 particles over the budget of 4"));
});

test("a second material counts as a draw call", () => {
  const draft = copy();
  effect(draft, "tilePlace").levels.low.particles.push({ sprite: "flake_paper", count: 1, tint: "color.gold.hi" });
  assert.equal(drawCallsOf(draft, effect(draft, "tilePlace").levels.low), 2);
});

test("overdraw is capped by the effect and the level", () => {
  const draft = copy();
  effect(draft, "block").levels.low.overdrawLayers = 2;
  assert.ok(checkEffects(tokens, draft, files).includes("block low: 2 overdraw layers over the budget of 1"));
});

test("a higher level that drops particles of a lower one is reported", () => {
  const draft = copy();
  effect(draft, "rankUp").levels.medium.particles = effect(draft, "rankUp").levels.medium.particles.filter((particle) => particle.sprite !== "star");
  assert.ok(checkEffects(tokens, draft, files).some((error) => error.startsWith("rankUp medium: has fewer star")));
});

test("an effect without a budget or a budget without an effect is reported", () => {
  const draft = copy();
  effect(draft, "emoji").name = "confetti";
  const errors = checkEffects(tokens, draft, files);
  assert.ok(errors.includes("effect emoji has a budget but no definition"));
  assert.ok(errors.includes("effect confetti has no budget in docs/design-tokens.json"));
});

test("a tint that is not a colour token is reported", () => {
  const draft = copy();
  effect(draft, "domino").levels.low.particles[0].tint = "color.gold";
  assert.ok(checkEffects(tokens, draft, files).includes("domino low: tint color.gold is not a colour token"));
});

test("an unlisted sprite file and a missing one are reported", () => {
  const errors = checkEffects(tokens, table, [...files.filter((name) => name !== "dust"), "smoke"]);
  assert.ok(errors.includes(`sprite dust: no dust.png in ${spritesFolder}`));
  assert.ok(errors.includes(`smoke.png is not listed in the sprites of ${tablePath}`));
});

test("score steps take the token share of the level and split it by the mix", () => {
  const cascade = effect(table, "scoreCascade");
  assert.deepEqual(stepParticles(tokens, cascade, "low", 1), { total: 5, counts: { star: 1, streak: 1, spark: 3 } });
  assert.deepEqual(stepParticles(tokens, cascade, "high", 6), { total: 180, counts: { star: 36, streak: 36, spark: 108 } });
});

test("a cascade list that is not the mix of the top step is reported", () => {
  const draft = copy();
  effect(draft, "scoreCascade").levels.medium.particles[0].count = 17;
  assert.ok(checkEffects(tokens, draft, files).some((error) => error.startsWith("scoreCascade medium: particles must be the stepMix")));
});
