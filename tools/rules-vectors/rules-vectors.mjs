import { readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";

const seatsByMode = { "1v1": ["X", "Y"], "2v2": ["N", "E", "S", "W"] };
const teamBySeat = { N: "A", S: "A", E: "B", W: "B", X: "A", Y: "B" };
const firstHandOpeningTile = "2-3";
const firstHandDoubleOrder = ["1-1", "2-2", "3-3", "4-4", "5-5", "6-6", "0-0"];
const scoringOpeningTile = "5-5";
const scoringOpeningPoints = 10;
const scoringMultiple = 5;
const handPointsLockThreshold = 300;
const targetScore = 365;
const handSize = 7;
const redealDoublesThreshold = 6;
const setSize = 28;
const minimumVectorsPerRule = 3;
const ends = ["left", "right", "top", "bottom"];
const tilePattern = /^[0-6]-[0-6]$/;
const annotationKeywords = ["$schema", "$id", "$defs", "title", "description"];
const supportedKeywords = new Set([
  ...annotationKeywords, "$ref", "type", "const", "enum", "pattern", "minLength", "minimum", "exclusiveMinimum",
  "minItems", "uniqueItems", "items", "required", "properties", "additionalProperties", "minProperties",
  "maxProperties", "allOf", "anyOf", "if", "then", "else",
]);

const isObject = (value) => typeof value === "object" && value !== null && !Array.isArray(value);
const canonical = (value) =>
  JSON.stringify(value, (key, inner) => (isObject(inner) ? Object.fromEntries(Object.entries(inner).sort(([a], [b]) => (a < b ? -1 : 1))) : inner));
const sum = (values) => values.reduce((total, value) => total + value, 0);
const halves = (tile) => tile.split("-").map(Number);
const pipsOf = (tile) => sum(halves(tile));
const isDouble = (tile) => halves(tile)[0] === halves(tile)[1];
const tileKey = (tile) => halves(tile).sort((a, b) => a - b).join("-");
const flip = (tile) => halves(tile).reverse().join("-");
const roundUp = (pips) => Math.ceil(pips / scoringMultiple) * scoringMultiple;
const scoreForTotal = (total) => (total > 0 && total % scoringMultiple === 0 ? total : 0);
const otherTeam = (team) => (team === "A" ? "B" : "A");
const nextSeat = (mode, seat) => {
  const seats = seatsByMode[mode];
  return seats[(seats.indexOf(seat) + 1) % seats.length];
};

const typeChecks = {
  string: (value) => typeof value === "string",
  number: (value) => typeof value === "number",
  integer: (value) => Number.isInteger(value),
  boolean: (value) => typeof value === "boolean",
  null: (value) => value === null,
  array: Array.isArray,
  object: isObject,
};

function resolveRef(root, ref) {
  return ref.slice(2).split("/").reduce((node, part) => node[part], root);
}

export function schemaErrors(schema, value, root = schema, path = "$") {
  const unsupported = Object.keys(schema).filter((keyword) => !supportedKeywords.has(keyword));
  if (unsupported.length > 0) throw new Error(`unsupported schema keyword(s): ${unsupported.join(", ")}`);
  if (schema.$ref) return schemaErrors(resolveRef(root, schema.$ref), value, root, path);
  const types = [schema.type ?? []].flat();
  if (types.length > 0 && !types.some((type) => typeChecks[type](value))) return [`${path}: expected ${types.join(" or ")}`];
  const errors = [];
  const fail = (message) => errors.push(`${path}: ${message}`);
  const nested = (subschema, inner, innerPath = path) => errors.push(...schemaErrors(subschema, inner, root, innerPath));
  if ("const" in schema && canonical(value) !== canonical(schema.const)) fail(`expected ${JSON.stringify(schema.const)}`);
  if (schema.enum && !schema.enum.some((option) => canonical(option) === canonical(value))) fail(`expected one of ${schema.enum.join(", ")}`);
  if (typeof value === "string") {
    if (schema.pattern && !new RegExp(schema.pattern, "u").test(value)) fail(`"${value}" does not match ${schema.pattern}`);
    if (value.length < (schema.minLength ?? 0)) fail(`shorter than ${schema.minLength}`);
  }
  if (typeof value === "number") {
    if (value < (schema.minimum ?? -Infinity)) fail(`below ${schema.minimum}`);
    if (value <= (schema.exclusiveMinimum ?? -Infinity)) fail(`not above ${schema.exclusiveMinimum}`);
  }
  if (Array.isArray(value)) {
    if (value.length < (schema.minItems ?? 0)) fail(`fewer than ${schema.minItems} items`);
    if (schema.uniqueItems && new Set(value.map(canonical)).size !== value.length) fail("items are not unique");
    if (schema.items) value.forEach((item, index) => nested(schema.items, item, `${path}[${index}]`));
  }
  if (isObject(value)) {
    const keys = Object.keys(value);
    for (const key of schema.required ?? []) if (!keys.includes(key)) fail(`missing ${key}`);
    if (keys.length < (schema.minProperties ?? 0)) fail(`fewer than ${schema.minProperties} properties`);
    if (keys.length > (schema.maxProperties ?? Infinity)) fail(`more than ${schema.maxProperties} properties`);
    for (const key of keys) {
      const declared = schema.properties && Object.hasOwn(schema.properties, key);
      const child = declared ? schema.properties[key] : schema.additionalProperties;
      if (child === false) fail(`unexpected property ${key}`);
      else if (isObject(child)) nested(child, value[key], `${path}.${key}`);
    }
  }
  for (const part of schema.allOf ?? []) nested(part, value);
  if (schema.anyOf && !schema.anyOf.some((option) => schemaErrors(option, value, root, path).length === 0)) fail("matches no allowed form");
  if (schema.if) {
    const branch = schemaErrors(schema.if, value, root, path).length === 0 ? schema.then : schema.else;
    if (branch) nested(branch, value);
  }
  return errors;
}

function exposedPip(board, end) {
  if (end === "left") return halves(board.line[0])[0];
  if (end === "right") return halves(board.line.at(-1))[1];
  if (!board.spinner) return null;
  const arm = board[end];
  return arm.length > 0 ? halves(arm.at(-1))[1] : halves(board.spinner)[0];
}

function openEnds(board) {
  const counted = (tile, end) => (isDouble(tile) ? pipsOf(tile) : exposedPip(board, end));
  const result = { left: counted(board.line[0], "left"), right: counted(board.line.at(-1), "right") };
  for (const arm of ["top", "bottom"]) if (board[arm].length > 0) result[arm] = counted(board[arm].at(-1), arm);
  return result;
}

function place(board, tile, end) {
  const pip = exposedPip(board, end);
  if (pip === null || !halves(tile).includes(pip)) return null;
  const outward = halves(tile)[0] === pip ? tile : flip(tile);
  const next = { line: [...board.line], spinner: board.spinner, top: [...board.top], bottom: [...board.bottom] };
  if (end === "left") next.line.unshift(flip(outward));
  else if (end === "right") next.line.push(outward);
  else next[end].push(outward);
  const lineEnd = end === "left" || end === "right";
  const neighbour = end === "left" ? next.line[1] : next.line.at(-2);
  if (!next.spinner && lineEnd && next.line.length >= 3 && isDouble(neighbour)) next.spinner = neighbour;
  return next;
}

const canPlace = (board, tile) => ends.some((end) => place(board, tile, end) !== null);

function placement(board, tile, end) {
  const next = place(board, tile, end);
  if (!next) return { legal: false };
  const open = openEnds(next);
  const total = sum(Object.values(open));
  return { legal: true, board: next, openEnds: open, total, score: scoreForTotal(total) };
}

function isValidDeal(mode, { hands, boneyard }) {
  const seats = seatsByMode[mode];
  const all = [...Object.values(hands).flat(), ...boneyard].map(tileKey);
  return (
    seats.every((seat) => hands[seat]?.length === handSize) &&
    boneyard.length === setSize - seats.length * handSize &&
    all.length === setSize &&
    new Set(all).size === setSize
  );
}

function firstOpener(mode, { hands }) {
  const holder = (tile) => seatsByMode[mode].find((seat) => hands[seat].some((held) => tileKey(held) === tile));
  const candidates = mode === "1v1" ? [firstHandOpeningTile, ...firstHandDoubleOrder] : [firstHandOpeningTile];
  const tile = candidates.find(holder);
  if (!tile) return { redeal: true, opener: null, tile: null, next: null };
  return { redeal: false, opener: holder(tile), tile, next: nextSeat(mode, holder(tile)) };
}

function doublesRedeal(mode, { hands }) {
  const doublesOf = (seat) => hands[seat].filter(isDouble).length;
  return { redeal: seatsByMode[mode].some((seat) => doublesOf(seat) >= redealDoublesThreshold) };
}

function turn(board, placements) {
  const steps = [];
  let current = board;
  let previous = null;
  for (const { tile, end } of placements) {
    const result = placement(current, tile, end);
    const continues = previous === null || (isDouble(previous.tile) && previous.score > 0 && isDouble(tile) && result.score > 0);
    if (!result.legal || !continues) {
      steps.push({ placed: false });
      break;
    }
    steps.push({ placed: true, total: result.total, score: result.score });
    current = result.board;
    previous = { tile, score: result.score };
  }
  return { steps, turnScore: sum(steps.map((step) => step.score ?? 0)) };
}

function draw({ board, hand, boneyard }) {
  if (hand.some((tile) => canPlace(board, tile))) return { drawn: [], outcome: "place" };
  const drawn = [];
  for (const tile of boneyard.slice(0, -1)) {
    drawn.push(tile);
    if (canPlace(board, tile)) return { drawn, outcome: "place" };
  }
  return { drawn, outcome: "pass" };
}

function handStatus({ board, hands, boneyard }) {
  const held = Object.values(hands);
  if (held.some((tiles) => tiles.length === 0)) return { status: "domino" };
  if (held.flat().some((tile) => canPlace(board, tile)) || boneyard.length > 1) return { status: "continue" };
  return { status: "block" };
}

function handEnd(given) {
  const pips = { A: 0, B: 0 };
  for (const [seat, tiles] of Object.entries(given.hands)) pips[teamBySeat[seat]] += sum(tiles.map(pipsOf));
  const placingTeam = teamBySeat[given.lastPlacedBy];
  const lowerTeam = pips.A === pips.B ? null : pips.A < pips.B ? "A" : "B";
  const winner = given.end === "domino" ? placingTeam : lowerTeam;
  const awarded = { A: 0, B: 0 };
  let carryOver = 0;
  if (winner === null) carryOver = given.carryOver + roundUp(pips.A + pips.B);
  else if (given.scoresAtHandStart[winner] < handPointsLockThreshold) awarded[winner] = roundUp(pips[otherTeam(winner)]) + given.carryOver;
  const scores = { A: given.scores.A + awarded.A, B: given.scores.B + awarded.B };
  return { pips, winner, awarded, carryOver, scores, nextOpeningTeam: placingTeam };
}

function matchEnd({ end, scores }) {
  if (end === "none") return { outcome: "continue", winner: null };
  const leader = scores.A === scores.B ? null : scores.A > scores.B ? "A" : "B";
  const reached = Math.max(scores.A, scores.B) >= targetScore;
  if (end === "domino" && reached && leader) return { outcome: "won", winner: leader };
  return { outcome: "next-hand", winner: null };
}

function afk(mode, { afkTimeouts, events }) {
  const states = Object.fromEntries(seatsByMode[mode].map((seat) => [seat, { status: "active", timeouts: 0 }]));
  for (const { seat, type } of events) {
    const state = states[seat];
    if (type === "timeout") state.timeouts++;
    if (type === "timeout" && state.timeouts >= afkTimeouts) state.status = "afk";
    if (type === "move") state.timeouts = 0;
    if (type === "reconnect") Object.assign(state, { status: "active", timeouts: 0 });
    if (type === "leave") state.status = "left";
  }
  const controllers = Object.entries(states).map(([seat, { status }]) => [seat, status === "active" ? "player" : "bot"]);
  return { controllers: Object.fromEntries(controllers) };
}

const derive = {
  deal: ({ mode, given }) => ({ valid: isValidDeal(mode, given) }),
  seat: ({ mode, given }) => ({ team: teamBySeat[given.seat], next: nextSeat(mode, given.seat) }),
  "first-opener": ({ mode, given }) => firstOpener(mode, given),
  "opening-score": ({ given }) => {
    const scores = !given.firstHand && given.tile === scoringOpeningTile && given.teamScoreAtHandStart < handPointsLockThreshold;
    return { score: scores ? scoringOpeningPoints : 0 };
  },
  place: ({ given }) => placement(given.board, given.tile, given.end),
  turn: ({ given }) => turn(given.board, given.placements),
  draw: ({ given }) => draw(given),
  "hand-status": ({ given }) => handStatus(given),
  round: ({ given }) => ({ points: roundUp(given.pips) }),
  "hand-end": ({ given }) => handEnd(given),
  "match-end": ({ given }) => matchEnd(given),
  "turn-timer": ({ given }) => {
    const inTime = given.actedAtSeconds !== null && given.actedAtSeconds < given.turnSeconds;
    return { movedBy: inTime ? "player" : "bot" };
  },
  afk: ({ mode, given }) => afk(mode, given),
  "doubles-redeal": ({ mode, given }) => doublesRedeal(mode, given),
};

function boardErrors({ line, spinner, top, bottom }) {
  const errors = [];
  line.slice(1).forEach((tile, index) => {
    if (halves(line[index])[1] !== halves(tile)[0]) errors.push(`board line breaks between ${line[index]} and ${tile}`);
  });
  const inner = line.slice(1, -1);
  if (!spinner) {
    if (top.length + bottom.length > 0) errors.push("board has arms but no spinner");
    if (inner.some(isDouble)) errors.push("board has a double with both sides taken but no spinner");
    return errors;
  }
  if (!isDouble(spinner) || !inner.includes(spinner)) errors.push(`spinner ${spinner} is not an inner double of the line`);
  for (const arm of [top, bottom]) {
    arm.forEach((tile, index) => {
      const touching = index === 0 ? halves(spinner)[0] : halves(arm[index - 1])[1];
      if (halves(tile)[0] !== touching) errors.push(`board arm breaks at ${tile}`);
    });
  }
  return errors;
}

function looseTiles(value, key = "") {
  if (key === "board") return [];
  if (typeof value === "string") return tilePattern.test(value) ? [value] : [];
  if (Array.isArray(value)) return value.flatMap((item) => looseTiles(item));
  if (isObject(value)) return Object.entries(value).flatMap(([childKey, child]) => looseTiles(child, childKey));
  return [];
}

function stateErrors({ kind, mode, given, expect }) {
  const errors = [];
  const seats = seatsByMode[mode];
  const seatsUsed = [given.seat, given.lastPlacedBy, ...(given.events ?? []).map((event) => event.seat)].filter(Boolean);
  for (const seat of seatsUsed) if (!seats.includes(seat)) errors.push(`seat ${seat} is not part of ${mode}`);
  for (const seatMap of [given.hands, expect.controllers].filter(Boolean)) {
    if (canonical(Object.keys(seatMap).sort()) !== canonical([...seats].sort())) errors.push(`seats must be exactly ${seats.join(", ")}`);
  }
  if (errors.length > 0 || kind === "deal") return errors;
  const loose = looseTiles(given);
  for (const tile of loose) if (halves(tile)[0] > halves(tile)[1]) errors.push(`tile ${tile} must be written low-high`);
  const boardTiles = given.board ? [given.board.line, given.board.top, given.board.bottom].flat() : [];
  const keys = [...boardTiles, ...loose].map(tileKey);
  if (new Set(keys).size !== keys.length) errors.push("a tile appears more than once");
  if (given.board) errors.push(...boardErrors(given.board));
  if (given.boneyard && mode === "2v2" && given.boneyard.length > 0) errors.push("2v2 has no boneyard");
  const fullDeal = kind === "first-opener" || kind === "doubles-redeal";
  if (given.boneyard && mode === "1v1" && !fullDeal && given.boneyard.length === 0) errors.push("1v1 boneyard always keeps its last tile");
  if (fullDeal && !isValidDeal(mode, given)) errors.push(`${kind} needs a valid full deal`);
  if (kind === "turn" && expect.steps.length !== given.placements.length) errors.push("placements continue after the turn ended");
  if (kind === "hand-end") {
    const emptyHands = Object.keys(given.hands).filter((seat) => given.hands[seat].length === 0);
    const expectedEmpty = given.end === "domino" ? [given.lastPlacedBy] : [];
    if (canonical(emptyHands) !== canonical(expectedEmpty)) errors.push("only the player who dominoed has an empty hand");
    if (given.scores.A < given.scoresAtHandStart.A || given.scores.B < given.scoresAtHandStart.B) errors.push("scores are below scoresAtHandStart");
  }
  return errors;
}

export function ruleIds(rulesText) {
  return [...rulesText.matchAll(/^- \*\*(R-\d+)\*\*/gm)].map((match) => match[1]);
}

export function excludedRules(readme) {
  const section = readme.split(/^## /m).find((part) => part.startsWith("Excluded rules")) ?? "";
  return [...section.matchAll(/^- \*\*(R-\d+)\*\*/gm)].map((match) => match[1]);
}

export function loadSuite(directory, rulesPath) {
  const read = (path) => readFileSync(path, "utf8");
  const names = readdirSync(directory).filter((name) => name.endsWith(".json") && name !== "schema.json").sort();
  return {
    schema: JSON.parse(read(join(directory, "schema.json"))),
    files: names.map((name) => ({ name, content: JSON.parse(read(join(directory, name))) })),
    readme: read(join(directory, "README.md")),
    rulesText: read(rulesPath),
  };
}

export function checkSuite({ schema, files, readme, rulesText }) {
  const errors = [];
  const known = new Set(ruleIds(rulesText));
  const excluded = new Set(excludedRules(readme));
  const coverage = new Map([...known].map((id) => [id, 0]));
  const seenIds = new Set();
  for (const id of excluded) if (!known.has(id)) errors.push(`README excludes unknown rule ${id}`);
  for (const { name, content } of files) {
    const fileErrors = schemaErrors(schema, content);
    if (fileErrors.length > 0) {
      errors.push(...fileErrors.map((error) => `${name} ${error}`));
      continue;
    }
    for (const vector of content.vectors) {
      const where = `${name} ${vector.id}`;
      if (seenIds.has(vector.id)) errors.push(`${where}: duplicate id`);
      seenIds.add(vector.id);
      for (const rule of vector.rules) {
        if (!known.has(rule)) errors.push(`${where}: unknown rule ${rule}`);
        else if (excluded.has(rule)) errors.push(`${where}: references excluded rule ${rule}`);
        else coverage.set(rule, coverage.get(rule) + 1);
      }
      const problems = stateErrors(vector);
      errors.push(...problems.map((problem) => `${where}: ${problem}`));
      if (problems.length > 0) continue;
      const derived = canonical(derive[vector.kind](vector));
      if (derived !== canonical(vector.expect)) errors.push(`${where}: expect ${canonical(vector.expect)} but rules give ${derived}`);
    }
  }
  for (const [id, count] of coverage) {
    if (!excluded.has(id) && count < minimumVectorsPerRule) errors.push(`${id}: covered by ${count} vector(s), needs ${minimumVectorsPerRule}`);
  }
  return errors;
}
