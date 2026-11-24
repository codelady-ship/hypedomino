import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { checkTokens, contrast, csharpPath, csharpTokens, loadSpec, loadTokens, rarityDefaults, rarityDefaultsPath, stepForPoints } from "./design-tokens.mjs";

const repoRoot = new URL("../../", import.meta.url);
const tokens = loadTokens(fileURLToPath(new URL("docs/design-tokens.json", repoRoot)));
const specText = loadSpec(fileURLToPath(new URL("docs/ui-motion.md", repoRoot)));
const raritySpecText = loadSpec(fileURLToPath(new URL("docs/rarity.md", repoRoot)));

function errorsAfter(change) {
  const copy = structuredClone(tokens);
  change(copy);
  return checkTokens(copy, specText, raritySpecText);
}

function assertReported(errors, fragment) {
  assert.ok(errors.some((error) => error.includes(fragment)), errors.join("\n"));
}

test("the committed tokens and spec pass every check", () => {
  assert.deepEqual(checkTokens(tokens, specText, raritySpecText), []);
});

test("a graphics level that lowers effects or resolution below the level under it is reported", () => {
  assertReported(errorsAfter((copy) => { copy.effects.graphics.levels.ultra.effectLevel = "medium"; }), "levels.ultra: effect level must not fall below high");
  assertReported(errorsAfter((copy) => { copy.effects.graphics.levels.high.maxRenderShortSide = 900; }), "levels.high: maxRenderShortSide must not fall below medium");
});

test("a device tier whose Auto level is above its highest is reported", () => {
  assertReported(errorsAfter((copy) => { copy.effects.graphics.tiers.mid.auto = "high"; }), "tiers.mid: auto is above highest");
  assertReported(errorsAfter((copy) => { copy.effects.graphics.tiers.low.highest = "turbo"; }), "tiers.low: unknown graphics level");
});

test("frame rates that are not offered, fall, or let the saver run faster than menus are reported", () => {
  assertReported(errorsAfter((copy) => { copy.effects.graphics.frameRate.menuMax = 75; }), "frameRate.menuMax: 75 is not an offered rate");
  assertReported(errorsAfter((copy) => { copy.effects.graphics.frameRate.options = [30, 90, 60, 120]; }), "frameRate.options: must rise");
  assertReported(errorsAfter((copy) => { copy.effects.graphics.frameRate.batterySaver = 60; copy.effects.graphics.frameRate.menuMax = 30; }), "batterySaver: must not exceed menuMax");
  assertReported(errorsAfter((copy) => { copy.effects.levels.medium.targetFps = 50; }), "effects.levels.medium.targetFps: 50 is not an offered rate");
});

test("C# frame rate tokens carry the offered rates and each effect level's target", () => {
  const source = csharpTokens(tokens);
  assert.ok(source.includes("public static readonly int[] FrameRateOptions = { 30, 60, 90, 120 };"));
  assert.ok(source.includes("public static readonly int[] TargetFpsOfEffectLevel = { 30, 60, 120 };"));
  assert.ok(source.includes("public const int BatterySaverFrameRate = 30;"));
});

test("C# text size tokens carry each type style's size", () => {
  const source = csharpTokens(tokens);
  assert.ok(source.includes("public const int Caption = 30;"));
  assert.ok(source.includes("public const int Score = 108;"));
});

test("C# typography tokens give each text style its family, weight, size and line height", () => {
  const source = csharpTokens(tokens);
  assert.ok(source.includes("public enum TokenFontFamily"));
  assert.ok(source.includes("        UiCyrillic,"));
  assert.ok(source.includes("public enum TokenTextStyle"));
  const styles = Object.keys(tokens.typography.scale);
  const score = styles.indexOf("score");
  const families = source.match(/FamilyOfStyle = \{ ([^}]*) \}/)[1].split(", ");
  const weights = source.match(/WeightOfStyle = \{ ([^}]*) \}/)[1].split(", ");
  assert.equal(families.length, styles.length);
  assert.equal(families[score], "TokenFontFamily.Numeral");
  assert.equal(weights[score], "900");
  assert.ok(source.includes("public static readonly float[] LineHeightOfStyle = { 1.3f, 1.35f,"));
});

test("a GPU pattern that is not a regular expression is reported", () => {
  assertReported(errorsAfter((copy) => { copy.effects.graphics.deviceClass.lowGpuPatterns.push("^Mali-(G52"); }), "^Mali-(G52 is not a regular expression");
});

test("C# graphics tokens list levels low to ultra and tiers low to high", () => {
  const source = csharpTokens(tokens);
  assert.ok(source.includes("public static readonly int[] EffectLevelOfLevel = { 0, 1, 2, 2 };"));
  assert.ok(source.includes("public static readonly int[] MaxRenderShortSideOfLevel = { 720, 1080, 1080, 2160 };"));
  assert.ok(source.includes("public static readonly int[] HighestLevelOfTier = { 0, 1, 3 };"));
  assert.ok(source.includes(String.raw`@"^Adreno \(TM\) (7[3-9]\d|8\d\d)\b"`));
  assert.ok(source.includes("public const float LowWorkTimeP95Ms = 28f;"));
  assert.ok(source.includes("public const float HighWorkTimeP95Ms = 7f;"));
});

test("a level that shows animated cosmetics with no budget, or a budget without them, is reported", () => {
  assertReported(errorsAfter((copy) => { copy.effects.levels.medium.maxAnimatedCosmetics = 0; }), "maxAnimatedCosmetics disagree");
  assertReported(errorsAfter((copy) => { copy.effects.levels.low.maxAnimatedCosmetics = 3; }), "maxAnimatedCosmetics disagree");
});

test("contrast matches the WCAG 2 reference values", () => {
  assert.equal(contrast("#FFFFFF", "#000000"), 21);
  assert.equal(contrast("#777777", "#FFFFFF").toFixed(2), "4.48");
});

test("a text role below AA is reported", () => {
  assertReported(errorsAfter((copy) => { copy.color.role.danger.text = "#7A2030"; }), "contrast role.danger.text");
});

test("a fill without non-text contrast against the base is reported", () => {
  assertReported(errorsAfter((copy) => { copy.color.role.teamA.fill = "#1A1640"; }), "contrast role.teamA.fill on background.base");
});

test("a stale contrast table in the spec is reported", () => {
  assertReported(errorsAfter((copy) => { copy.color.text.muted = "#B0A9E6"; }), "spec contrast table is missing or stale");
});

test("a colour missing from the spec is reported", () => {
  assertReported(errorsAfter((copy) => { copy.color.role.teamB.text = "#FFB088"; }), "spec does not list colour #FFB088");
});

test("rarity ordinals with a gap are reported", () => {
  assertReported(errorsAfter((copy) => { copy.rarity.tiers.mythic.ordinal = 6; }), "ordinals must be 1 to 5");
});

test("repeated rarity pip counts are reported", () => {
  assertReported(errorsAfter((copy) => { copy.rarity.tiers.epic.pips = 2; }), "pip counts must be unique");
});

test("a rarity glow weaker than a lower tier is reported", () => {
  assertReported(errorsAfter((copy) => { copy.rarity.tiers.legendary.glow = "soft"; }), "legendary.glow must not be weaker");
});

test("a rarity tier with a wrong name key is reported", () => {
  assertReported(errorsAfter((copy) => { copy.rarity.tiers.rare.nameKey = "rarity.blue.name"; }), "rare.nameKey");
});

test("a rarity id outside the cosmetics id pattern is reported", () => {
  assertReported(errorsAfter((copy) => { copy.rarity.tiers.Ultra = { ...copy.rarity.tiers.mythic, ordinal: 6, pips: 6, nameKey: "rarity.Ultra.name" }; }), "rarity.tiers.Ultra: id must match");
});

test("a rarity fill without non-text contrast on a surface is reported", () => {
  assertReported(errorsAfter((copy) => { copy.rarity.tiers.rare.fill = "#3A4BB0"; }), "contrast rarity.tiers.rare.fill on surface.raised");
});

test("a rarity label shade below AA is reported", () => {
  assertReported(errorsAfter((copy) => { copy.rarity.tiers.epic.text = "#7A3FB0"; }), "contrast rarity.tiers.epic.text");
});

test("a stale rarity contrast table is reported", () => {
  assertReported(errorsAfter((copy) => { copy.rarity.tiers.common.text = "#D0D2E0"; }), "rarity spec contrast table is missing or stale");
});

test("a rarity colour missing from the rarity spec is reported", () => {
  assertReported(errorsAfter((copy) => { copy.rarity.tiers.mythic.on = "#100828"; }), "rarity spec does not list colour #100828");
});

test("a concurrent set over the level budget is reported", () => {
  assertReported(errorsAfter((copy) => { copy.effects.budgets.domino.medium.gpuMs = 2.0; }), "dominoEnd.medium: gpuMs");
});

test("an effect over the level overdraw limit is reported", () => {
  assertReported(errorsAfter((copy) => { copy.effects.budgets.block.low.overdrawLayers = 3; }), "block.low: overdraw");
});

test("a level whose base plus effects exceeds the frame budget is reported", () => {
  assertReported(errorsAfter((copy) => { copy.effects.levels.high.effectCpuMs = 2.5; }), "levels.high: CPU");
});

test("score steps that shrink are reported", () => {
  assertReported(errorsAfter((copy) => { copy.scoreCascade.steps[3].textScale = 1.1; }), "textScale must not decrease");
});

test("a typography style under the minimum size is reported", () => {
  assertReported(errorsAfter((copy) => { copy.typography.scale.caption.size = 24; }), "under minimum");
});

test("score values map to steps with the chain boost capped at the top step", () => {
  assert.deepEqual([5, 10, 15, 20, 25, 30, 45, 80].map((points) => stepForPoints(tokens, points)), [1, 2, 3, 4, 5, 6, 6, 6]);
  assert.equal(stepForPoints(tokens, 0), 0);
  assert.equal(stepForPoints(tokens, 5, 2), 2);
  assert.equal(stepForPoints(tokens, 10, 3), 4);
  assert.equal(stepForPoints(tokens, 10, 7), 4);
  assert.equal(stepForPoints(tokens, 25, 3), 6);
});

test("the committed C# tokens match the JSON", () => {
  const committed = readFileSync(fileURLToPath(new URL(csharpPath, repoRoot)), "utf8");
  assert.equal(committed, csharpTokens(tokens), "run npm run tokens:gen and commit the result");
});

test("C# tokens carry hex colours, nested names and float easing points", () => {
  const source = csharpTokens(tokens);
  assert.ok(source.includes("public static readonly Color32 BackgroundBase = new Color32(0x0B, 0x06, 0x05, 0xFF);"));
  assert.ok(source.includes("public static readonly Color32 RoleTeamAFill = new Color32(0x3D, 0x8B, 0xFF, 0xFF);"));
  assert.ok(source.includes("public const int Md = 16;"));
  assert.ok(source.includes("public static readonly Vector4 Overshoot = new Vector4(0.34f, 1.56f, 0.64f, 1f);"));
  assert.ok(!source.includes("Legendary"), "rarity colours come from admin config, not constants");
});

test("C# elevation tokens carry each level's Y offset and shadow alpha", () => {
  const source = csharpTokens(tokens);
  assert.ok(source.includes("public static class TokenElevation"));
  assert.ok(source.includes("public const int E0OffsetY = 0;"));
  assert.ok(source.includes("public const int E1OffsetY = 4;"));
  assert.ok(source.includes("public const float E1Alpha = 0.35f;"));
  assert.ok(source.includes("public const int E3OffsetY = 16;"));
});

test("an elevation offset that is not a whole unit is reported", () => {
  const copy = structuredClone(tokens);
  copy.elevation.e1.offsetY = 4.5;
  assert.throws(() => csharpTokens(copy), /elevation\.e1\.offsetY: 4\.5 is not an integer/);
});

test("C# tokens carry each effect's duration and pool size", () => {
  const source = csharpTokens(tokens);
  assert.ok(source.includes("public const int EmojiDurationMs = 1800;"));
  assert.ok(source.includes("public const int EmojiPooled = 4;"));
  assert.ok(source.includes("public const int ScorePopupPooled = 6;"));
});

test("C# tokens carry each effect's particles per level by kind and each level's particle cap", () => {
  const source = csharpTokens(tokens);
  const kinds = Object.keys(tokens.effects.budgets);
  assert.ok(source.includes("public enum TokenEffectKind"));
  assert.ok(source.includes(`public static readonly string[] NameOfKind = { ${kinds.map((name) => `@"${name}"`).join(", ")} };`));
  for (const level of ["low", "medium", "high"]) {
    const particles = kinds.map((name) => tokens.effects.budgets[name][level].particles);
    const name = level[0].toUpperCase() + level.slice(1);
    assert.ok(source.includes(`public static readonly int[] ${name}ParticlesOfKind = { ${particles.join(", ")} };`), level);
    assert.ok(source.includes(`public const int ${name}MaxParticles = ${tokens.effects.levels[level].maxParticles};`), level);
  }
  assert.ok(source.includes("public static readonly int[] PooledOfKind = { 4, 6, 2, 1, 1, 1, 1, 1, 4 };"));
});

test("C# tokens carry what the effect player resolves at load: colour paths, glow levels, overdraw caps and step glows", () => {
  const source = csharpTokens(tokens);
  assert.ok(source.includes(`@"color.gold.hi"`));
  assert.ok(source.includes("public static readonly Color32[] OfPath = { BackgroundBase,"));
  const glows = Object.keys(tokens.glow);
  assert.ok(source.includes(`public static readonly string[] NameOfLevel = { ${glows.map((name) => `@"${name}"`).join(", ")} };`));
  assert.ok(source.includes(`public static readonly float[] AlphaOfLevel = { ${glows.map((name) => `${tokens.glow[name].alpha}f`).join(", ")} };`));
  for (const level of ["low", "medium", "high"]) {
    const name = level[0].toUpperCase() + level.slice(1);
    assert.ok(source.includes(`public const int ${name}MaxOverdrawLayers = ${tokens.effects.levels[level].maxOverdrawLayers};`), level);
  }
  const stepGlows = tokens.scoreCascade.steps.map((step) => glows.indexOf(step.glow));
  assert.ok(source.includes(`public static readonly int[] StepGlowLevel = { ${stepGlows.join(", ")} };`));
});

test("the built-in rarity defaults of the client match the tokens", () => {
  const committed = readFileSync(fileURLToPath(new URL(rarityDefaultsPath, repoRoot)), "utf8");
  assert.equal(committed, rarityDefaults(tokens), "run npm run tokens:gen and commit the result");
});

test("rarity defaults list tiers by ordinal with their glow values", () => {
  const { tiers } = JSON.parse(rarityDefaults(tokens));
  assert.deepEqual(tiers.map((tier) => tier.ordinal), tiers.map((_, at) => at + 1));
  const mythic = tiers.find((tier) => tier.id === "mythic");
  assert.equal(mythic.glowAlpha, tokens.glow[tokens.rarity.tiers.mythic.glow].alpha);
  assert.equal(mythic.glowScale, tokens.glow[tokens.rarity.tiers.mythic.glow].scale);
  assert.equal(mythic.flow, true);
});

test("a fractional integer token stops C# generation", () => {
  const copy = structuredClone(tokens);
  copy.spacing.md = 16.5;
  assert.throws(() => csharpTokens(copy), /md: 16\.5 is not an integer/);
});
