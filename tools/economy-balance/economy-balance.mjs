import { readFileSync } from "node:fs";
import { pathToFileURL } from "node:url";

export const rarityIds = ["common", "rare", "epic", "legendary", "mythic"];
export const itemTypes = ["avatar", "frame", "table", "tile", "emoji"];

const header = "key,value,unit,kind,section";
const keyPattern = /^[A-Z][A-Za-z0-9]*$/;
const numberPattern = /^\d+(\.\d+)?$/;
const kinds = new Set(["config", "model"]);

const listLengths = {
  MatchCoinsRanked1v1: 2,
  MatchCoinsRanked2v2: 2,
  MatchCoinsCasual1v1: 2,
  MatchCoinsCasual2v2: 2,
  DailyLoginCoins: 7,
  RankRewardCoins: 10,
  GuidePriceCoins: rarityIds.length,
  GuidePriceTypePercent: itemTypes.length,
  GuidePriceCrystals: rarityIds.length,
  ChestRarityPercent: 2,
  CrystalPackUsd: 5,
  CrystalPackCrystals: 5,
  SinkRatioAlertPercent: 2,
  ModelMatchMinutes: 2,
  ModelProfileHours: 3,
  ModelProfileMissionPercent: 3,
};

const scalarKeys = [
  "DailyMatchCoinCap", "FirstWinBonusCoins", "DailyMissionCoinBudget", "WeeklyMissionCoinBudget",
  "PassFreeTrackCoins", "PassPremiumTrackCrystals", "PassLevelCoinPrice", "PassLevelCrystalPrice", "PassPremiumUsd",
  "PassPremiumPlusUsd", "PassPremiumPlusLevels", "StarterOfferUsd", "StarterOfferCrystals", "ChestCoinPrice",
  "ModelBetweenMatchMinutes", "ModelWinRatePercent", "ModelPassSeasonDays", "CrystalsPerUsdReference",
];

function rowErrors(fields, line, seen) {
  if (fields.length !== 5) return [`line ${line}: expected 5 fields, found ${fields.length}`];
  const [key, value, , kind] = fields;
  const errors = [];
  if (!keyPattern.test(key)) errors.push(`line ${line}: key ${key} is not CamelCase`);
  if (seen.has(key)) errors.push(`line ${line}: duplicate key ${key}`);
  if (!kinds.has(kind)) errors.push(`line ${line}: kind ${kind} is not config or model`);
  if (!value.split(";").every((part) => numberPattern.test(part))) errors.push(`line ${line}: ${key} value ${value} is not a number list`);
  return errors;
}

function shapeErrors(rows) {
  const errors = [];
  for (const [key, length] of Object.entries(listLengths)) {
    if (!rows.has(key)) errors.push(`missing key ${key}`);
    else if (rows.get(key).values.length !== length) errors.push(`${key} has ${rows.get(key).values.length} values, expected ${length}`);
  }
  for (const key of scalarKeys) {
    if (!rows.has(key)) errors.push(`missing key ${key}`);
    else if (rows.get(key).values.length !== 1) errors.push(`${key} must be a single value`);
  }
  return errors;
}

export function parseBalance(text) {
  const lines = text.split("\n").filter((line) => line.length > 0);
  if (lines[0] !== header) return { rows: new Map(), errors: [`header must be ${header}`] };
  const rows = new Map();
  const errors = [];
  lines.slice(1).forEach((line, index) => {
    const fields = line.split(",");
    const lineErrors = rowErrors(fields, index + 2, rows);
    errors.push(...lineErrors);
    if (lineErrors.length === 0) rows.set(fields[0], { values: fields[1].split(";").map(Number), unit: fields[2], kind: fields[3], section: fields[4] });
  });
  if (errors.length === 0) errors.push(...shapeErrors(rows));
  return { rows, errors };
}

const roundHalfAway = (value) => Math.sign(value) * Math.round(Math.abs(value));
const sum = (values) => values.reduce((total, value) => total + value, 0);
const fixed = (value, digits) => value.toFixed(digits);
const tableRow = (cells) => `| ${cells.join(" | ")} |`;
const separatorRow = (head) => `|${head.map(() => "---").join("|")}|`;
const table = (head, body) => [tableRow(head), separatorRow(head), ...body.map(tableRow)].join("\n");

function modeRates(get) {
  const [minutes1v1, minutes2v2] = get("ModelMatchMinutes");
  const between = get("ModelBetweenMatchMinutes")[0];
  const winRate = get("ModelWinRatePercent")[0] / 100;
  const cap = get("DailyMatchCoinCap")[0];
  const modes = [
    ["Ranked 1v1", "MatchCoinsRanked1v1", minutes1v1],
    ["Ranked 2v2", "MatchCoinsRanked2v2", minutes2v2],
    ["Casual 1v1", "MatchCoinsCasual1v1", minutes1v1],
    ["Casual 2v2", "MatchCoinsCasual2v2", minutes2v2],
  ];
  return modes.map(([name, key, minutes]) => {
    const [win, loss] = get(key);
    const matchesPerHour = 60 / (minutes + between);
    const coinsPerHour = matchesPerHour * (win * winRate + loss * (1 - winRate));
    return { name, win, loss, matchesPerHour, coinsPerHour, hoursToCap: cap / coinsPerHour };
  });
}

function profileDays(get, rankedRate) {
  const cap = get("DailyMatchCoinCap")[0];
  const login = sum(get("DailyLoginCoins")) / 7;
  const missions = get("DailyMissionCoinBudget")[0] + get("WeeklyMissionCoinBudget")[0] / 7;
  const passFree = get("PassFreeTrackCoins")[0] / get("ModelPassSeasonDays")[0];
  const shares = get("ModelProfileMissionPercent").map((percent) => percent / 100);
  return get("ModelProfileHours").map((hours, index) => {
    const parts = [Math.min(cap, hours * rankedRate), get("FirstWinBonusCoins")[0], login, missions * shares[index], passFree * shares[index]];
    return { hours, parts, total: sum(parts) };
  });
}

function packRates(get) {
  const reference = get("CrystalsPerUsdReference")[0];
  const crystals = get("CrystalPackCrystals");
  return get("CrystalPackUsd").map((usd, index) => ({
    tier: index + 1,
    usd,
    crystals: crystals[index],
    bonus: crystals[index] / (usd * reference) - 1,
    perUsd: crystals[index] / usd,
  }));
}

function renderModes(modes) {
  const body = modes.map((mode) => [mode.name, mode.win, mode.loss, fixed(mode.matchesPerHour, 2), roundHalfAway(mode.coinsPerHour), fixed(mode.hoursToCap, 1)]);
  return table(["Mode", "Win", "Loss", "Matches per hour", "Coins per hour", "Hours to daily cap"], body);
}

function renderProfiles(profiles) {
  const labels = ["Matches (ranked 1v1)", "First win of the day", "Daily login (7-day average)", "Missions (daily + weekly / 7)", "Pass free track"];
  const body = labels.map((label, index) => [label, ...profiles.map((profile) => roundHalfAway(profile.parts[index]))]);
  body.push(["**Total per day**", ...profiles.map((profile) => `**${roundHalfAway(profile.total)}**`)]);
  return table(["Source", ...profiles.map((profile) => `${profile.hours} h/day`)], body);
}

function renderPacks(packs) {
  const body = packs.map((pack) => [`crystals_tier${pack.tier}`, fixed(pack.usd, 2), pack.crystals, `${roundHalfAway(pack.bonus * 100)}%`, roundHalfAway(pack.perUsd)]);
  return table(["Product", "USD reference", "Crystals", "Bonus over reference rate", "Crystals per USD"], body);
}

function renderGuidePrices(get) {
  const base = get("GuidePriceCoins");
  const percents = get("GuidePriceTypePercent");
  const body = rarityIds.map((rarity, index) => [`\`${rarity}\``, ...percents.map((percent) => roundHalfAway((base[index] * percent) / 100))]);
  return table(["Rarity", ...itemTypes.map((type) => `\`${type}\``)], body);
}

function renderPriceVsEarn(get, profiles, packs) {
  const base = get("GuidePriceCoins");
  const crystals = get("GuidePriceCrystals");
  const bestRate = Math.max(...packs.map((pack) => pack.perUsd));
  const worstRate = Math.min(...packs.map((pack) => pack.perUsd));
  const body = rarityIds.map((rarity, index) => [
    `\`${rarity}\``,
    base[index],
    ...profiles.map((profile) => fixed(base[index] / profile.total, 1)),
    crystals[index],
    `${fixed(crystals[index] / bestRate, 2)} to ${fixed(crystals[index] / worstRate, 2)}`,
  ]);
  return table(["Rarity", "Guide price in coins (avatar, frame)", ...profiles.map((profile) => `Days at ${profile.hours} h/day`), "Guide price in crystals", "USD for the crystals (best to smallest pack)"], body);
}

function renderPassAndOffers(get) {
  const reference = get("CrystalsPerUsdReference")[0];
  const levelPrice = get("PassLevelCrystalPrice")[0];
  const premiumCrystals = get("PassPremiumTrackCrystals")[0];
  const products = [
    ["pass_premium", get("PassPremiumUsd")[0], premiumCrystals],
    ["pass_premium_plus", get("PassPremiumPlusUsd")[0], premiumCrystals + get("PassPremiumPlusLevels")[0] * levelPrice],
    ["offer_starter", get("StarterOfferUsd")[0], get("StarterOfferCrystals")[0]],
  ];
  const body = products.map(([id, usd, crystals]) => [id, fixed(usd, 2), crystals, fixed(crystals / reference, 2), fixed(crystals / reference / usd, 2)]);
  return table(["Product", "USD reference", "Crystal value (crystals and levels, items excluded)", "USD at reference rate", "Value multiple"], body);
}

export function renderBalance(rows) {
  const get = (key) => rows.get(key).values;
  const modes = modeRates(get);
  const profiles = profileDays(get, modes[0].coinsPerHour);
  const packs = packRates(get);
  return [
    "#### Earn rate per mode",
    renderModes(modes),
    "#### Earn per day by player profile",
    renderProfiles(profiles),
    "#### Crystal packs",
    renderPacks(packs),
    "#### Guide prices by rarity and type (coins)",
    renderGuidePrices(get),
    "#### Price against earn rate",
    renderPriceVsEarn(get, profiles, packs),
    "#### Pass and offer value",
    renderPassAndOffers(get),
  ].join("\n\n");
}

export function checkEconomy(csvText, docText) {
  const { rows, errors } = parseBalance(csvText);
  if (errors.length > 0) return errors;
  const missingKeys = [...rows.keys()].filter((key) => !docText.includes(`\`${key}\``)).map((key) => `docs/economy.md does not name ${key}`);
  const stale = docText.includes(renderBalance(rows)) ? [] : ["docs/economy.md balance sheet is stale; paste the output of node tools/economy-balance/economy-balance.mjs --print"];
  return [...missingKeys, ...stale];
}

function main(args) {
  const csvText = readFileSync("docs/economy/balance.csv", "utf8");
  if (args.includes("--print")) {
    const { rows, errors } = parseBalance(csvText);
    if (errors.length > 0) {
      for (const error of errors) console.error(error);
      process.exit(1);
    }
    console.log(renderBalance(rows));
    return;
  }
  const errors = checkEconomy(csvText, readFileSync("docs/economy.md", "utf8"));
  for (const error of errors) console.error(error);
  if (errors.length > 0) process.exit(1);
  console.log("economy balance ok");
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) main(process.argv.slice(2));
