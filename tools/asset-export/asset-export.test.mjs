import { after, test } from "node:test";
import assert from "node:assert/strict";
import { mkdirSync, mkdtempSync, readFileSync, writeFileSync, existsSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { crc32, deflateSync } from "node:zlib";
import { addItem, astcBytes, budgetLine, changedItems, checkItems, inspectItem, itemBudget, loadBudgets, parseArguments, readPng, readProvenance, sliceErrors } from "./asset-export.mjs";
import { loadSchema } from "../cosmetics-schema/cosmetics-schema.mjs";

const repoRoot = new URL("../../", import.meta.url);
const schema = loadSchema(fileURLToPath(new URL("docs/cosmetics/schema.json", repoRoot)));
const minimal = fileURLToPath(new URL("docs/cosmetics/examples/minimal.json", repoRoot));
const budgets = loadBudgets();
const temporaryRoots = [];
after(() => temporaryRoots.forEach((root) => rmSync(root, { recursive: true, force: true })));
const approved = "# avatar.x\n\n- artSource: ai_assisted\n- Cultural review: 2026-11-12, reviewer, approved\n";

function chunk(type, data) {
  const body = Buffer.concat([Buffer.from(type, "latin1"), data]);
  const length = Buffer.alloc(4);
  length.writeUInt32BE(data.length);
  const sum = Buffer.alloc(4);
  sum.writeUInt32BE(crc32(body));
  return Buffer.concat([length, body, sum]);
}

function png(width, height, { colorType = 6, shade = 0 } = {}) {
  const channels = { 2: 3, 4: 2, 6: 4 }[colorType];
  const header = Buffer.alloc(13);
  header.writeUInt32BE(width, 0);
  header.writeUInt32BE(height, 4);
  header[8] = 8;
  header[9] = colorType;
  const row = Buffer.alloc(1 + width * channels, shade);
  row[0] = 0;
  const pixels = deflateSync(Buffer.concat(Array.from({ length: height }, () => row)));
  return Buffer.concat([Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]), chunk("IHDR", header), chunk("IDAT", pixels), chunk("IEND", Buffer.alloc(0))]);
}

function makeItem(files, { type = "avatar", slug = "comet_fox", provenance = approved } = {}) {
  const root = mkdtempSync(join(tmpdir(), "asset-export-"));
  temporaryRoots.push(root);
  const directory = join(root, "assets-src", "cosmetic", type, slug);
  mkdirSync(directory, { recursive: true });
  writeFileSync(join(directory, "provenance.md"), provenance);
  for (const [path, bytes] of Object.entries(files)) {
    mkdirSync(dirname(join(directory, path)), { recursive: true });
    writeFileSync(join(directory, path), bytes);
  }
  return { root, directory };
}

function loop(count, size, slot = "export/anim") {
  return Object.fromEntries(Array.from({ length: count }, (_, index) => [`${slot}/${String(index + 1).padStart(4, "0")}.png`, png(size, size, { shade: index })]));
}

function manifestCopy(root) {
  const path = join(root, "manifest.json");
  writeFileSync(path, readFileSync(minimal));
  return path;
}

const addOptions = (manifest, extra = {}) => ({ manifest, sources: [{ type: "shop", ref: "offer.comet_fox" }], rarity: "rare", availability: "live", client: undefined, ...extra });

function assertReported(errors, fragment) {
  assert.ok(errors.some((error) => error.includes(fragment)), `expected "${fragment}" in:\n${errors.join("\n")}`);
}

test("readPng reads size and refuses files without alpha or LFS pointers", () => {
  assert.deepEqual(readPng(png(3, 5)), { width: 3, height: 5, bitDepth: 8, colorType: 6 });
  assert.equal(readPng(png(3, 5, { colorType: 4 })).error, undefined);
  assert.match(readPng(png(3, 5, { colorType: 2 })).error, /no alpha/);
  assert.match(readPng(Buffer.from("version https://git-lfs.github.com/spec/v1\noid sha256:00\nsize 9\n")).error, /LFS pointer/);
  assert.match(readPng(Buffer.from("GIF89a")).error, /not a PNG/);
});

test("ASTC bytes follow 16 bytes per block (ADR 0012 size rule)", () => {
  assert.equal(astcBytes(1024, 1024, "6x6"), 467856);
  assert.equal(astcBytes(768, 768, "6x6"), 262144);
  assert.equal(astcBytes(1024, 1024, "8x8"), 262144);
});

test("the default block matches the sprite format the Unity project imports with", () => {
  const setup = readFileSync(fileURLToPath(new URL("client/Assets/HypeDomino/Editor/ProjectSetup.cs", repoRoot)), "utf8");
  assert.match(setup, new RegExp(`SpriteFormat = TextureImporterFormat\\.ASTC_${budgets.astcBlock}\\b`));
});

test("an animated item with a static and a numbered loop passes", () => {
  const { directory } = makeItem({ "export/static.png": png(64, 64), ...loop(4, 64) });
  const item = inspectItem(directory);
  assert.deepEqual(item.errors, []);
  assert.equal(item.id, "avatar.comet_fox");
  assert.deepEqual(item.loops.map(({ slot, frames }) => [slot, frames.length]), [["anim", 4]]);
});

test("export problems are reported", () => {
  assertReported(inspectItem(makeItem({ ...loop(2, 64) }).directory).errors, "missing export/static.png");
  assertReported(inspectItem(makeItem({}).directory).errors, "nothing exported");
  assertReported(inspectItem(makeItem({ "export/static.png": png(64, 64), ...loop(2, 32) }).directory).errors, "frames keep the canvas of the static");
  assertReported(inspectItem(makeItem({ "export/static.png": png(64, 64), "export/anim/0001.png": png(64, 64), "export/anim/0003.png": png(64, 64) }).directory).errors, "expected 0002.png");
  assertReported(inspectItem(makeItem({ "export/static.png": png(64, 64, { colorType: 2 }) }).directory).errors, "no alpha channel");
  assertReported(inspectItem(makeItem({ "export/static.png": png(64, 64), ...loop(1, 64, "export/layer/back") }).directory).errors, "export/layer is not allowed for avatar");
  assertReported(inspectItem(join(tmpdir(), "not-an-item")).errors, "is not a cosmetic item folder");
});

test("an animated avatar over the limit of #171 fails, under it passes", () => {
  const over = inspectItem(makeItem({ "export/static.png": png(512, 512), ...loop(16, 512) }).directory);
  const budget = itemBudget(over, budgets);
  assert.equal(budget.ok, false);
  assert.match(budgetLine(budget), /limit under 300,000 \(#171.*\): fail/);
  const under = itemBudget(inspectItem(makeItem({ "export/static.png": png(192, 192), ...loop(12, 192) }).directory), budgets);
  assert.equal(under.bytes, 13 * astcBytes(192, 192, "6x6"));
  assert.equal(under.ok, true);
});

test("a type or kind without a documented limit is measured and not judged", () => {
  const still = itemBudget(inspectItem(makeItem({ "export/static.png": png(2048, 2048) }).directory), budgets);
  assert.equal(still.ok, true);
  assert.match(budgetLine(still), /no size limit set/);
  const tile = itemBudget(inspectItem(makeItem({ "export/static.png": png(1024, 1024), ...loop(8, 1024) }, { type: "tile" }).directory), budgets);
  assert.equal(tile.ok, true);
  assert.match(budgetLine(tile), /no size limit set/);
});

test("frames are limited static and animated by the owner decision of 2026-11-16", () => {
  const frame = (files) => itemBudget(inspectItem(makeItem(files, { type: "frame" }).directory), budgets);
  assert.equal(frame({ "export/static.png": png(512, 512) }).ok, true);
  const bigStill = frame({ "export/static.png": png(1024, 1024) });
  assert.equal(bigStill.ok, false);
  assert.match(budgetLine(bigStill), /limit under 125,000 \(owner decision 2026-11-16.*\): fail/);
  const animated = frame({ "export/static.png": png(256, 256), ...loop(16, 256) });
  assert.equal(animated.bytes, 502928);
  assert.equal(animated.ok, true);
  assert.match(budgetLine(animated), /limit under 600,000/);
  assert.equal(frame({ "export/static.png": png(512, 512), ...loop(16, 512) }).ok, false);
});

test("tables are limited by size and an animated table by its layers", () => {
  const table = (files) => itemBudget(inspectItem(makeItem(files, { type: "table" }).directory), budgets);
  assert.equal(table({ "export/static.png": png(1440, 1440) }).ok, true);
  assert.equal(table({ "export/static.png": png(1500, 1500) }).ok, false);
  const base = loop(1, 64, "export/layer/base");
  const animated = table({ "export/static.png": png(64, 64), ...base, ...loop(10, 64, "export/layer/motion") });
  assert.deepEqual([animated.layers, animated.animatedLayers, animated.ok], [2, 1, true]);
  assert.match(budgetLine(animated), /limit under 5,000,000, 2 layers \(1 animated\), at most 2 \(1 animated\)/);
  const threeLayers = table({ "export/static.png": png(64, 64), ...base, ...loop(1, 64, "export/layer/rim"), ...loop(10, 64, "export/layer/motion") });
  assert.equal(threeLayers.ok, false);
  const twoAnimated = table({ "export/static.png": png(64, 64), ...loop(4, 64, "export/layer/cloth"), ...loop(10, 64, "export/layer/motion") });
  assert.equal(twoAnimated.ok, false);
  assert.equal(table({ "export/static.png": png(1200, 1200), ...base, ...loop(10, 1200, "export/layer/motion") }).ok, false);
});

test("emoji are limited static or animated (#175)", () => {
  const emoji = itemBudget(inspectItem(makeItem({ "export/static.png": png(512, 512) }, { type: "emoji", slug: "wave" }).directory), budgets);
  assert.equal(emoji.ok, false);
});

test("checkItems reports every item and fails on any problem", () => {
  const good = makeItem({ "export/static.png": png(64, 64), ...loop(2, 64) }).directory;
  const bad = makeItem({ "export/static.png": png(512, 512), ...loop(16, 512) }).directory;
  assert.equal(checkItems([good], budgets).ok, true);
  const both = checkItems([good, bad], budgets);
  assert.equal(both.ok, false);
  assert.equal(both.lines.length, 2);
});

test("provenance gives artSource and requires an approved cultural review", () => {
  assert.deepEqual(readProvenance(approved), { artSource: "ai_assisted", errors: [] });
  assertReported(readProvenance("- artSource: ai_assisted | handmade\n- Cultural review: <YYYY-MM-DD>, <reviewer>, approved | rejected\n").errors, "artSource");
  assertReported(readProvenance("- artSource: handmade\n- Cultural review: 2026-11-12, reviewer, rejected\n").errors, "Cultural review");
  assertReported(readProvenance("- artSource: handmade\n").errors, "Cultural review");
});

test("add puts a new animated avatar into the manifest and the result is valid", () => {
  const { root, directory } = makeItem({ "export/static.png": png(64, 64), ...loop(3, 64) });
  const manifest = manifestCopy(root);
  const result = addItem(directory, addOptions(manifest), { schema, budgets });
  assert.deepEqual(result.errors, []);
  assert.equal(result.change, "added");
  const written = JSON.parse(readFileSync(manifest, "utf8"));
  assert.deepEqual(written.items.at(-1), {
    id: "avatar.comet_fox",
    type: "avatar",
    revision: 1,
    nameKey: "cosmetic.avatar.comet_fox.name",
    rarityId: "rare",
    animated: true,
    staticAddress: "cosmetic/avatar/comet_fox/static",
    animationAddress: "cosmetic/avatar/comet_fox/anim",
    availability: "live",
    sources: [{ type: "shop", ref: "offer.comet_fox" }],
    artSource: "ai_assisted",
    culturalReview: "approved",
  });
});

test("add again keeps the revision and the choices when nothing changed", () => {
  const { root, directory } = makeItem({ "export/static.png": png(64, 64) });
  const manifest = manifestCopy(root);
  addItem(directory, addOptions(manifest), { schema, budgets });
  const again = addItem(directory, { manifest, sources: [], client: undefined }, { schema, budgets });
  assert.deepEqual([again.change, again.entry.revision, again.entry.rarityId], ["unchanged", 1, "rare"]);
});

test("add raises the revision when the item becomes animated", () => {
  const { root, directory } = makeItem({ "export/static.png": png(64, 64) });
  const manifest = manifestCopy(root);
  addItem(directory, addOptions(manifest), { schema, budgets });
  mkdirSync(join(directory, "export/anim"));
  writeFileSync(join(directory, "export/anim/0001.png"), png(64, 64));
  writeFileSync(join(directory, "export/anim/0002.png"), png(64, 64));
  const changed = addItem(directory, { manifest, sources: [], client: undefined }, { schema, budgets });
  assert.deepEqual([changed.change, changed.entry.revision, changed.entry.animationAddress], ["updated", 2, "cosmetic/avatar/comet_fox/anim"]);
});

test("add writes table layers back to front and needs the order for a new table", () => {
  const files = { "export/static.png": png(64, 64), "export/slice.json": '{"depth":8,"step":16}', ...loop(3, 64, "export/layer/sky"), ...loop(1, 64, "export/layer/cloth") };
  const { root, directory } = makeItem(files, { type: "table", slug: "garden" });
  const manifest = manifestCopy(root);
  assertReported(addItem(directory, addOptions(manifest), { schema, budgets }).errors, "--layers must name every layer");
  const result = addItem(directory, addOptions(manifest, { layers: ["sky", "cloth"] }), { schema, budgets });
  assert.deepEqual(result.errors, []);
  assert.deepEqual(result.entry.layers, [
    { name: "sky", address: "cosmetic/table/garden/layer/sky", animated: true },
    { name: "cloth", address: "cosmetic/table/garden/layer/cloth", animated: false },
  ]);
  const again = addItem(directory, { manifest, sources: [], client: undefined }, { schema, budgets });
  assert.deepEqual(again.entry.layers.map(({ name }) => name), ["sky", "cloth"]);
});

test("a table needs a slice that fits its static", () => {
  const { directory } = makeItem({ "export/static.png": png(64, 36) }, { type: "table", slug: "plain" });
  assertReported(inspectItem(directory).errors, "missing export/slice.json");
  const still = { width: 64, height: 36 };
  assert.deepEqual(sliceErrors('{"depth":8,"step":16}', still), []);
  assertReported(sliceErrors("{", still), "is not JSON");
  assertReported(sliceErrors('{"depth":8.5,"step":16}', still), "whole positive depth and step");
  assertReported(sliceErrors('{"depth":18,"step":16}', still), "leaves no field");
  assertReported(sliceErrors('{"depth":8,"step":15}', still), "even remainder");
  assertReported(sliceErrors('{"depth":8,"step":64}', still), "shorter than the static width");
});

test("add refuses without the choices a new item needs, an unreviewed item and an invalid manifest result", () => {
  const { root, directory } = makeItem({ "export/static.png": png(64, 64) });
  const manifest = manifestCopy(root);
  assertReported(addItem(directory, { manifest, sources: [], client: undefined }, { schema, budgets }).errors, "needs --rarity, --availability, --source");
  const emoji = makeItem({ "export/static.png": png(64, 64) }, { type: "emoji", slug: "wave" });
  assertReported(addItem(emoji.directory, addOptions(manifestCopy(emoji.root)), { schema, budgets }).errors, "needs --group");
  const unreviewed = makeItem({ "export/static.png": png(64, 64) }, { provenance: "- artSource: handmade\n" });
  assertReported(addItem(unreviewed.directory, addOptions(manifestCopy(unreviewed.root)), { schema, budgets }).errors, "Cultural review");
  const before = readFileSync(manifest, "utf8");
  assertReported(addItem(directory, addOptions(manifest, { sources: [{ type: "default" }], availability: "retired" }), { schema, budgets }).errors, "must be live");
  assert.equal(readFileSync(manifest, "utf8"), before, "an invalid result is never written");
  assertReported(addItem(directory, addOptions(join(root, "missing.json")), { schema, budgets }).errors, "--manifest must name an existing manifest");
});

test("add copies the export into the client, removes stale frames and moves default items to the built-in root", () => {
  const { root, directory } = makeItem({ "export/static.png": png(64, 64), ...loop(3, 64) });
  const client = join(root, "client");
  const manifest = manifestCopy(root);
  const first = addItem(directory, addOptions(manifest, { client }), { schema, budgets });
  assert.equal(first.client, join(client, "remote", "avatar", "comet_fox"));
  for (const file of ["static.png", "anim/0001.png", "anim/0003.png"]) assert.ok(existsSync(join(first.client, file)), file);
  writeFileSync(join(first.client, "anim/0003.png.meta"), "guid");
  rmSync(join(directory, "export/anim/0003.png"));
  addItem(directory, addOptions(manifest, { client }), { schema, budgets });
  assert.equal(existsSync(join(first.client, "anim/0003.png")), false);
  assert.equal(existsSync(join(first.client, "anim/0003.png.meta")), false);
  const still = makeItem({ "export/static.png": png(64, 64) }, { slug: "plain" });
  const remote = addItem(still.directory, addOptions(manifest, { client }), { schema, budgets });
  assert.deepEqual(remote.errors, []);
  const builtin = addItem(still.directory, addOptions(manifest, { client, sources: [{ type: "default" }] }), { schema, budgets });
  assert.deepEqual(builtin.errors, []);
  assert.equal(builtin.client, join(client, "builtin", "avatar", "plain"));
  assert.equal(existsSync(join(client, "remote", "avatar", "plain")), false);
});

test("changed paths map to the cosmetic items that still have an export", () => {
  const existing = new Set(["assets-src/cosmetic/avatar/comet_fox/export", "assets-src/cosmetic/table/garden/export"]);
  const paths = [
    "assets-src/cosmetic/avatar/comet_fox/export/anim/0001.png",
    "assets-src/cosmetic/avatar/comet_fox/provenance.md",
    "assets-src/cosmetic/table/garden/export/static.png",
    "assets-src/cosmetic/frame/removed/export/static.png",
    "assets-src/board/b_1/export/b_1.jpg",
    "tools/asset-export/asset-export.mjs",
  ];
  assert.deepEqual(changedItems(paths, (path) => existing.has(path.replaceAll("\\", "/"))), ["assets-src/cosmetic/avatar/comet_fox", "assets-src/cosmetic/table/garden"]);
});

test("arguments", () => {
  const options = parseArguments(["add", "assets-src/cosmetic/avatar/x", "--manifest", "m.json", "--source", "default", "--source", "shop:offer.x", "--layers", "a,b", "--no-client"]);
  assert.deepEqual(options.sources, [{ type: "default" }, { type: "shop", ref: "offer.x" }]);
  assert.deepEqual([options.command, options.paths, options.manifest, options.layers, options.client], ["add", ["assets-src/cosmetic/avatar/x"], "m.json", ["a", "b"], undefined]);
  assert.equal(parseArguments(["check", "x"]).client, "client/Assets/Cosmetics");
  assert.throws(() => parseArguments(["add", "--rarity"]), /needs a value/);
  assert.throws(() => parseArguments(["add", "--colour", "red"]), /unknown option/);
});
