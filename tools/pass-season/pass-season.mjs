import { readFileSync } from "node:fs";
import { pathToFileURL } from "node:url";
import { parseBalance } from "../economy-balance/economy-balance.mjs";

const sampleHeading = "### 8.2 Levels and rewards";
const rowPattern = /^\| (\d+) \| (\d+) \| (.+) \| (.+) \|$/;
const coinPattern = /^(\d+) coins/;
const crystalPattern = /^(\d+) crystals/;

export function xpTotalToReach(level, curve) {
  let total = 0;
  for (let from = 1; from < level; from += 1) total += Math.min(curve.base + curve.step * (from - 1), curve.max);
  return total;
}

function sectionBody(docText, heading) {
  const start = docText.indexOf(`\n${heading}\n`);
  if (start < 0) return "";
  const rest = docText.slice(start + heading.length + 2);
  const end = rest.search(/\n#{1,3} /);
  return end < 0 ? rest : rest.slice(0, end);
}

export function parseSampleTable(docText) {
  return sectionBody(docText, sampleHeading)
    .split("\n")
    .map((line) => line.match(rowPattern))
    .filter(Boolean)
    .map(([, level, xpTotal, free, premium]) => ({ level: Number(level), xpTotal: Number(xpTotal), free, premium }));
}

function configValues(docText, keys) {
  const keyCell = keys.map((key) => `\`${key}\``).join(", ");
  const line = docText.split("\n").find((candidate) => candidate.startsWith(`| ${keyCell} | `));
  return line ? line.split(" | ")[1].split(", ").map(Number) : [];
}

const coinsIn = (cell) => Number(cell.match(coinPattern)?.[1] ?? 0);
const crystalsIn = (cell) => Number(cell.match(crystalPattern)?.[1] ?? 0);
const sum = (values) => values.reduce((total, value) => total + value, 0);

function curveFrom(docText) {
  const [levels] = configValues(docText, ["PassLevels"]);
  const [base, step, max] = configValues(docText, ["PassXpBase", "PassXpStep", "PassXpMax"]);
  return [levels, base, step, max].every(Number.isInteger) ? { levels, base, step, max } : null;
}

function rowErrors(rows, curve) {
  const errors = [];
  if (rows.length !== curve.levels) errors.push(`sample table has ${rows.length} levels, PassLevels is ${curve.levels}`);
  rows.forEach((row, index) => {
    if (row.level !== index + 1) errors.push(`row ${index + 1} is level ${row.level}, expected ${index + 1}`);
    const expected = xpTotalToReach(row.level, curve);
    if (row.xpTotal !== expected) errors.push(`level ${row.level}: XP total ${row.xpTotal}, formula gives ${expected}`);
    if (row.premium === "-") errors.push(`level ${row.level}: premium track has no reward`);
  });
  return errors;
}

function coinErrors(rows, balance) {
  const tracks = [
    ["free", "PassFreeTrackCoins", "coins", rows.map((row) => coinsIn(row.free))],
    ["premium", "PassPremiumTrackCrystals", "crystals", rows.map((row) => crystalsIn(row.premium))],
  ];
  return tracks
    .filter(([, key, , amounts]) => sum(amounts) !== balance.get(key).values[0])
    .map(([track, key, unit, amounts]) => `${track} track pays ${sum(amounts)} ${unit}, ${key} is ${balance.get(key).values[0]}`);
}

export function checkPassSeason(docText, csvText) {
  const balance = parseBalance(csvText);
  if (balance.errors.length > 0) return balance.errors;
  const curve = curveFrom(docText);
  if (!curve) return ["config table does not state PassLevels, PassXpBase, PassXpStep and PassXpMax"];
  const rows = parseSampleTable(docText);
  return [...rowErrors(rows, curve), ...coinErrors(rows, balance.rows)];
}

function main() {
  const errors = checkPassSeason(readFileSync("docs/pass-missions-medals.md", "utf8"), readFileSync("docs/economy/balance.csv", "utf8"));
  for (const error of errors) console.error(error);
  if (errors.length > 0) process.exit(1);
  console.log("pass season ok");
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) main();
