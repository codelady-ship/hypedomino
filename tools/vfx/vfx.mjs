import { readFileSync, readdirSync } from "node:fs";
import { pathToFileURL } from "node:url";

export const tablePath = "client/Assets/HypeDomino/UI/Effects/effects.json";
export const spritesFolder = "client/Assets/HypeDomino/UI/Effects/Sprites";
export const docPath = "docs/vfx.md";
export const levels = ["low", "medium", "high"];
const materials = ["additive", "alpha"];
const freeTints = ["team", "rarity"];

const isObject = (value) => typeof value === "object" && value !== null && !Array.isArray(value);
const lookup = (root, path) => path.split(".").reduce((node, key) => (isObject(node) ? node[key] : undefined), root);
const roundHalfUp = (numerator) => Math.floor((numerator + 50) / 100);

export const particlesOf = (level) => level.particles.reduce((sum, particle) => sum + particle.count, 0);

function materialOf(table, sprite) {
  return table.sprites.find((entry) => entry.name === sprite)?.material;
}

export function drawCallsOf(table, level) {
  const used = new Set([...level.sprites, ...level.particles].map((element) => materialOf(table, element.sprite)));
  return used.size + level.interfaceDrawCalls;
}

export function stepParticles(tokens, cascade, level, step) {
  const percent = Math.round(tokens.scoreCascade.steps[step - 1].particleShare * 100);
  const total = roundHalfUp(tokens.effects.budgets.scoreCascade[level].particles * percent);
  const mix = Object.entries(cascade.stepMix);
  const counts = Object.fromEntries(mix.slice(0, -1).map(([sprite, share]) => [sprite, roundHalfUp(total * share)]));
  const [lastSprite] = mix[mix.length - 1];
  counts[lastSprite] = total - Object.values(counts).reduce((sum, count) => sum + count, 0);
  return { total, counts };
}

function spriteErrors(table, files) {
  const listed = table.sprites.map((entry) => entry.name);
  const errors = table.sprites.filter((entry) => !materials.includes(entry.material)).map((entry) => `sprite ${entry.name}: material ${entry.material} is not additive or alpha`);
  errors.push(...listed.filter((name) => !files.includes(name)).map((name) => `sprite ${name}: no ${name}.png in ${spritesFolder}`));
  errors.push(...files.filter((name) => !listed.includes(name)).map((name) => `${name}.png is not listed in the sprites of ${tablePath}`));
  return errors;
}

function elementErrors(tokens, table, where, element) {
  const errors = [];
  if (!materialOf(table, element.sprite)) errors.push(`${where}: unknown sprite ${element.sprite}`);
  if (!freeTints.includes(element.tint) && typeof lookup(tokens, element.tint) !== "string") errors.push(`${where}: tint ${element.tint} is not a colour token`);
  if (element.glow !== undefined && element.glow !== "step" && !isObject(tokens.glow[element.glow])) errors.push(`${where}: unknown glow ${element.glow}`);
  if (element.count !== undefined && !(Number.isInteger(element.count) && element.count > 0)) errors.push(`${where}: count must be a positive integer`);
  return errors;
}

function countsBySprite(elements) {
  const counts = new Map();
  for (const element of elements) counts.set(element.sprite, (counts.get(element.sprite) ?? 0) + (element.count ?? 1));
  return counts;
}

function levelErrors(tokens, table, effect, levelName) {
  const where = `${effect.name} ${levelName}`;
  const level = effect.levels[levelName];
  if (!level) return [`${where}: missing`];
  const budget = tokens.effects.budgets[effect.name][levelName];
  const errors = [...level.sprites, ...level.particles].flatMap((element) => elementErrors(tokens, table, where, element));
  if (errors.length > 0) return errors;
  const particles = particlesOf(level);
  const drawCalls = drawCallsOf(table, level);
  const layerLimit = Math.min(budget.overdrawLayers, tokens.effects.levels[levelName].maxOverdrawLayers);
  if (particles > budget.particles) errors.push(`${where}: ${particles} particles over the budget of ${budget.particles}`);
  if (drawCalls > budget.drawCalls) errors.push(`${where}: ${drawCalls} draw calls over the budget of ${budget.drawCalls}`);
  if (level.overdrawLayers > layerLimit) errors.push(`${where}: ${level.overdrawLayers} overdraw layers over the budget of ${layerLimit}`);
  return errors;
}

function growthErrors(effect) {
  const errors = [];
  for (let at = 1; at < levels.length; at++) {
    const lower = effect.levels[levels[at - 1]];
    const higher = effect.levels[levels[at]];
    if (!lower || !higher) continue;
    const higherCounts = countsBySprite([...higher.sprites, ...higher.particles]);
    for (const [sprite, count] of countsBySprite([...lower.sprites, ...lower.particles]))
      if ((higherCounts.get(sprite) ?? 0) < count) errors.push(`${effect.name} ${levels[at]}: has fewer ${sprite} than ${levels[at - 1]}; a higher level only adds`);
  }
  return errors;
}

function cascadeErrors(tokens, cascade) {
  if (!cascade) return ["scoreCascade is missing"];
  const shares = Object.values(cascade.stepMix ?? {});
  if (shares.length === 0 || shares.reduce((sum, share) => sum + share, 0) !== 100) return ["scoreCascade: stepMix must add up to 100"];
  const lastStep = tokens.scoreCascade.steps.length;
  return levels.flatMap((levelName) => {
    const { counts } = stepParticles(tokens, cascade, levelName, lastStep);
    const listed = countsBySprite(cascade.levels[levelName]?.particles ?? []);
    const matches = listed.size === Object.keys(counts).length && Object.entries(counts).every(([sprite, count]) => listed.get(sprite) === count);
    return matches ? [] : [`scoreCascade ${levelName}: particles must be the stepMix of step ${lastStep}, ${JSON.stringify(counts)}`];
  });
}

function markErrors(tokens, table) {
  return table.marks.flatMap((mark) => {
    const errors = elementErrors(tokens, table, `mark ${mark.name}`, mark);
    if (mark.overdrawLayers !== 1) errors.push(`mark ${mark.name}: a mark is one overdraw layer`);
    return errors;
  });
}

export function checkEffects(tokens, table, files) {
  const budgeted = Object.keys(tokens.effects.budgets);
  const defined = table.effects.map((effect) => effect.name);
  const errors = spriteErrors(table, files);
  errors.push(...budgeted.filter((name) => !defined.includes(name)).map((name) => `effect ${name} has a budget but no definition`));
  errors.push(...defined.filter((name) => !budgeted.includes(name)).map((name) => `effect ${name} has no budget in docs/design-tokens.json`));
  for (const effect of table.effects.filter((entry) => budgeted.includes(entry.name))) {
    errors.push(...levels.flatMap((levelName) => levelErrors(tokens, table, effect, levelName)));
    errors.push(...growthErrors(effect));
  }
  errors.push(...cascadeErrors(tokens, table.effects.find((effect) => effect.name === "scoreCascade")));
  errors.push(...markErrors(tokens, table));
  return errors;
}

const spriteList = (level) => [...level.sprites.map((element) => (element.glow ? `${element.sprite} (${element.glow})` : element.sprite)), ...level.particles.map((element) => `${element.sprite} ${element.count}`)].join(", ") || "none";

export function budgetTable(tokens, table) {
  const rows = ["| Effect | Level | Particles / budget | Sprites and particles | Overdraw layers / budget | Draw calls / budget | Pooled |", "|---|---|---|---|---|---|---|"];
  for (const effect of table.effects) {
    for (const levelName of levels) {
      const level = effect.levels[levelName];
      const budget = tokens.effects.budgets[effect.name][levelName];
      const layerLimit = Math.min(budget.overdrawLayers, tokens.effects.levels[levelName].maxOverdrawLayers);
      rows.push(`| \`${effect.name}\` | ${levelName} | ${particlesOf(level)} / ${budget.particles} | ${spriteList(level)} | ${level.overdrawLayers} / ${layerLimit} | ${drawCallsOf(table, level)} / ${budget.drawCalls} | ${budget.pooled} |`);
    }
  }
  return rows.join("\n");
}

export function stepTable(tokens, table) {
  const cascade = table.effects.find((effect) => effect.name === "scoreCascade");
  const sprites = Object.keys(cascade.stepMix);
  const rows = [`| Step | Points | Glow | ${levels.map((levelName) => `${levelName}: ${sprites.join(" / ")}`).join(" | ")} |`, `|---|---|---|${levels.map(() => "---").join("|")}|`];
  for (const step of tokens.scoreCascade.steps) {
    const cells = levels.map((levelName) => {
      const { total, counts } = stepParticles(tokens, cascade, levelName, step.step);
      return `${total}: ${sprites.map((sprite) => counts[sprite]).join(" / ")}`;
    });
    rows.push(`| ${step.step} | ${step.minPoints} | \`glow.${step.glow}\` | ${cells.join(" | ")} |`);
  }
  return rows.join("\n");
}

export function docErrors(doc, tokens, table) {
  const tables = { "budget table": budgetTable(tokens, table), "score step table": stepTable(tokens, table) };
  return Object.entries(tables).filter(([, text]) => !doc.includes(text)).map(([name]) => `${docPath}: the ${name} is stale; run node tools/vfx/vfx.mjs --print and paste it`);
}

export function spriteFiles(folder) {
  return readdirSync(folder).filter((name) => name.endsWith(".png")).map((name) => name.slice(0, -4)).sort();
}

function main(args) {
  const tokens = JSON.parse(readFileSync("docs/design-tokens.json", "utf8"));
  const table = JSON.parse(readFileSync(tablePath, "utf8"));
  if (args.includes("--print")) {
    console.log(`${budgetTable(tokens, table)}\n\n${stepTable(tokens, table)}`);
    return;
  }
  const errors = [...checkEffects(tokens, table, spriteFiles(spritesFolder)), ...docErrors(readFileSync(docPath, "utf8"), tokens, table)];
  for (const error of errors) console.error(error);
  if (errors.length > 0) process.exit(1);
  console.log(`vfx ok (${table.effects.length} effects, ${table.sprites.length} sprites)`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) main(process.argv.slice(2));
