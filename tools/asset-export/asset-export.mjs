import { copyFileSync, existsSync, mkdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath, pathToFileURL } from "node:url";
import { checkAssetPaths, listFiles } from "../asset-naming/asset-naming.mjs";
import { checkManifest, itemTypes, loadSchema } from "../cosmetics-schema/cosmetics-schema.mjs";

const toolRoot = fileURLToPath(new URL("./", import.meta.url));
const repoRoot = fileURLToPath(new URL("../../", import.meta.url));
const pngSignature = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a]);
const alphaColorTypes = new Set([4, 6]);
const artSources = ["handmade", "ai_assisted"];
const lfsPointerPrefix = "version https://git-lfs";
export const defaultClientRoot = "client/Assets/Cosmetics";

export const loadBudgets = (path = join(toolRoot, "budgets.json")) => JSON.parse(readFileSync(path, "utf8"));

export function readPng(bytes) {
  if (bytes.toString("latin1", 0, lfsPointerPrefix.length) === lfsPointerPrefix) return { error: "is a Git LFS pointer, run git lfs pull" };
  if (bytes.length < 26 || !bytes.subarray(0, 8).equals(pngSignature) || bytes.toString("latin1", 12, 16) !== "IHDR") return { error: "is not a PNG" };
  const png = { width: bytes.readUInt32BE(16), height: bytes.readUInt32BE(20), bitDepth: bytes[24], colorType: bytes[25] };
  return alphaColorTypes.has(png.colorType) ? png : { ...png, error: "has no alpha channel (docs/asset-workflow.md 5.3)" };
}

export function astcBytes(width, height, block) {
  const [blockWidth, blockHeight] = block.split("x").map(Number);
  return Math.ceil(width / blockWidth) * Math.ceil(height / blockHeight) * 16;
}

export function itemOf(directory) {
  const absolute = resolve(directory);
  const [slug, type, assetClass] = absolute.split(/[\\/]/).reverse();
  if (assetClass !== "cosmetic" || !itemTypes.includes(type)) return { error: `${directory} is not a cosmetic item folder assets-src/cosmetic/<type>/<slug>` };
  return { directory: absolute, type, slug, id: `${type}.${slug}`, root: `cosmetic/${type}/${slug}` };
}

function slotOf(path) {
  const [, folder, ...rest] = path.split("/");
  if (folder === "anim") return { slot: "anim", frame: rest.join("/") };
  if (folder === "layer") return { slot: `layer/${rest[0]}`, frame: rest.slice(1).join("/") };
  return null;
}

function frameNameErrors(slot, frames) {
  return frames.flatMap((frame, index) => {
    const expected = `${String(index + 1).padStart(4, "0")}.png`;
    return frame.name === expected ? [] : [`export/${slot}/${frame.name}: frames are named 0001.png, 0002.png and on without gaps, expected ${expected}`];
  });
}

function canvasErrors(still, loops) {
  return loops.flatMap(({ slot, frames }) => frames
    .filter(({ png }) => png.width !== still.width || png.height !== still.height)
    .map(({ name, png }) => `export/${slot}/${name} is ${png.width}x${png.height}, the static is ${still.width}x${still.height}; frames keep the canvas of the static (docs/art-direction.md section 7)`));
}

export function sliceErrors(text, still) {
  let slice;
  try {
    slice = JSON.parse(text);
  } catch {
    return ["export/slice.json is not JSON"];
  }
  const whole = (value) => Number.isInteger(value) && value > 0;
  if (!whole(slice?.depth) || !whole(slice?.step)) return ["export/slice.json needs a whole positive depth and step in pixels of the static (docs/cosmetics.md 8.4)"];
  const errors = [];
  if (slice.depth * 2 >= still.height) errors.push(`export/slice.json depth ${slice.depth} leaves no field in the static height ${still.height}`);
  if (slice.step >= still.width || (still.width - slice.step) % 2 !== 0) errors.push(`export/slice.json step ${slice.step} must be shorter than the static width ${still.width} and leave an even remainder`);
  return errors;
}

function readExportPng(item, path, errors) {
  const png = readPng(readFileSync(join(item.directory, path)));
  if (png.error) errors.push(`${path} ${png.error}`);
  return png;
}

export function inspectItem(directory) {
  const item = itemOf(directory);
  if (item.error) return { errors: [item.error] };
  if (!existsSync(join(item.directory, "export"))) return { ...item, errors: [`${item.root}: nothing exported, export/ is missing`] };
  const files = listFiles(item.directory);
  const errors = checkAssetPaths(files.map((file) => `${item.root}/${file}`));
  if (!files.includes("export/static.png")) return { ...item, errors: [...errors, `${item.root}: missing export/static.png`] };
  const still = readExportPng(item, "export/static.png", errors);
  const loopsBySlot = new Map();
  for (const path of files.filter((file) => file.startsWith("export/")).sort()) {
    const place = slotOf(path);
    if (!place) continue;
    if (!loopsBySlot.has(place.slot)) loopsBySlot.set(place.slot, { slot: place.slot, frames: [] });
    loopsBySlot.get(place.slot).frames.push({ name: place.frame, png: readExportPng(item, path, errors) });
  }
  const loops = [...loopsBySlot.values()];
  errors.push(...loops.flatMap(({ slot, frames }) => frameNameErrors(slot, frames)));
  if (!still.error) errors.push(...canvasErrors(still, loops.map(({ slot, frames }) => ({ slot, frames: frames.filter(({ png }) => !png.error) }))));
  if (item.type === "table" && !files.includes("export/slice.json")) errors.push(`${item.root}: missing export/slice.json, the table's ring depth and edge step (docs/cosmetics.md 8.4)`);
  else if (item.type === "table" && !still.error) errors.push(...sliceErrors(readFileSync(join(item.directory, "export/slice.json"), "utf8"), still));
  return { ...item, errors, still, loops };
}

function layerCounts(item) {
  const layers = item.loops.filter(({ slot }) => slot.startsWith("layer/"));
  return { layers: layers.length, animatedLayers: layers.filter(({ frames }) => frames.length > 1).length };
}

export function itemBudget(item, budgets) {
  const typeRule = budgets.types[item.type];
  const rule = typeRule?.[item.loops.length > 0 ? "animated" : "static"];
  const block = typeRule?.astcBlock ?? budgets.astcBlock;
  const sprites = [item.still, ...item.loops.flatMap(({ frames }) => frames.map(({ png }) => png))];
  const bytes = sprites.reduce((sum, png) => sum + astcBytes(png.width, png.height, block), 0);
  const { layers, animatedLayers } = layerCounts(item);
  const measured = { id: item.id, sprites: sprites.length, block, bytes, layers, animatedLayers };
  if (!rule) return { ...measured, ok: true };
  const layersOk = (rule.maxLayers === undefined || layers <= rule.maxLayers) && (rule.maxAnimatedLayers === undefined || animatedLayers <= rule.maxAnimatedLayers);
  return { ...measured, maxBytes: rule.maxBytes, maxLayers: rule.maxLayers, maxAnimatedLayers: rule.maxAnimatedLayers, source: rule.source, ok: bytes < rule.maxBytes && layersOk };
}

function layerText(budget) {
  if (budget.maxLayers === undefined && budget.maxAnimatedLayers === undefined) return "";
  return `, ${budget.layers} layers (${budget.animatedLayers} animated), at most ${budget.maxLayers ?? "any"} (${budget.maxAnimatedLayers ?? "any"} animated)`;
}

export function budgetLine(budget) {
  const measured = `${budget.id}: ${budget.sprites} sprites, ASTC ${budget.block} estimate ${budget.bytes.toLocaleString("en-US")} bytes`;
  if (budget.maxBytes === undefined) return `${measured}, no size limit set for this type and kind (docs/asset-workflow.md O-5)`;
  return `${measured}, limit under ${budget.maxBytes.toLocaleString("en-US")}${layerText(budget)} (${budget.source}): ${budget.ok ? "pass" : "fail"}`;
}

export function readProvenance(text) {
  const field = (label) => text.match(new RegExp(`^- ${label}:[ \\t]*(.*)$`, "m"))?.[1].trim();
  const artSource = field("artSource");
  const verdict = field("Cultural review")?.split(",").pop().trim();
  const errors = [];
  if (!artSources.includes(artSource)) errors.push(`provenance.md artSource is "${artSource ?? ""}", expected handmade or ai_assisted`);
  if (verdict !== "approved") errors.push("provenance.md has no approved Cultural review line; the review of docs/localization.md 6.1 comes before the manifest (W-7)");
  return { artSource, errors };
}

function orderedLayers(item, order) {
  const layers = item.loops.filter(({ slot }) => slot.startsWith("layer/")).map((loop) => ({ ...loop, name: loop.slot.slice("layer/".length) }));
  const names = layers.map(({ name }) => name);
  if (layers.length === 1 && !order) return { layers };
  const wanted = order?.filter((name) => names.includes(name)) ?? [];
  if (wanted.length !== names.length || new Set(order).size !== order.length) return { error: `--layers must name every layer of export/layer/ once, back to front: ${names.join(", ")}` };
  return { layers: wanted.map((name) => layers.find((layer) => layer.name === name)) };
}

function canonical(value) {
  if (Array.isArray(value)) return `[${value.map(canonical).join(",")}]`;
  if (value && typeof value === "object") return `{${Object.keys(value).sort().map((key) => `${JSON.stringify(key)}:${canonical(value[key])}`).join(",")}}`;
  return JSON.stringify(value);
}

const sameContent = (left, right) => canonical({ ...left, revision: 0 }) === canonical({ ...right, revision: 0 });

function missingChoices(options, previous) {
  const missing = [];
  if (!options.rarity && !previous) missing.push("--rarity");
  if (!options.availability && !previous) missing.push("--availability");
  if (options.sources.length === 0 && !previous) missing.push("--source");
  if (!options.group && !previous?.group && options.type === "emoji") missing.push("--group");
  return missing;
}

export function buildEntry(item, options, previous, provenance) {
  const missing = missingChoices({ ...options, type: item.type }, previous);
  if (missing.length > 0) return { errors: [`a new ${item.type} needs ${missing.join(", ")}`] };
  const isTable = item.type === "table";
  const ordered = isTable ? orderedLayers(item, options.layers ?? previous?.layers?.map(({ name }) => name)) : { layers: [] };
  if (ordered.error) return { errors: [ordered.error] };
  const base = `cosmetic/${item.type}/${item.slug}`;
  const animated = isTable ? ordered.layers.length > 0 : item.loops.some(({ slot }) => slot === "anim");
  const entry = { id: item.id, type: item.type, revision: previous?.revision ?? 1, nameKey: `cosmetic.${item.id}.name`, rarityId: options.rarity ?? previous.rarityId, animated, staticAddress: `${base}/static` };
  if (animated && !isTable) entry.animationAddress = `${base}/anim`;
  if (animated && isTable) entry.layers = ordered.layers.map(({ name, frames }) => ({ name, address: `${base}/layer/${name}`, animated: frames.length > 1 }));
  const group = options.group ?? previous?.group;
  if (group) entry.group = group;
  entry.availability = options.availability ?? previous.availability;
  entry.sources = options.sources.length > 0 ? options.sources : previous.sources;
  entry.artSource = provenance.artSource;
  entry.culturalReview = "approved";
  if (previous && !sameContent(previous, entry)) entry.revision = previous.revision + 1;
  return { entry, errors: [] };
}

export function upsertEntry(manifest, entry) {
  const at = manifest.items.findIndex((item) => item.id === entry.id);
  const items = [...manifest.items];
  if (at < 0) items.push(entry);
  else items[at] = entry;
  const change = at < 0 ? "added" : manifest.items[at].revision === entry.revision ? "unchanged" : "updated";
  return { manifest: { ...manifest, items }, change };
}

const otherRoot = (builtin) => (builtin ? "remote" : "builtin");

export function syncClient(item, clientRoot, builtin) {
  const target = join(clientRoot, builtin ? "builtin" : "remote", item.type, item.slug);
  const stale = join(clientRoot, otherRoot(builtin), item.type, item.slug);
  rmSync(stale, { recursive: true, force: true });
  rmSync(`${stale}.meta`, { force: true });
  const source = join(item.directory, "export");
  const wanted = new Set(listFiles(source));
  const present = existsSync(target) ? listFiles(target) : [];
  for (const file of present.filter((path) => path.endsWith(".png") && !wanted.has(path))) {
    rmSync(join(target, file));
    rmSync(join(target, `${file}.meta`), { force: true });
  }
  for (const file of wanted) {
    mkdirSync(dirname(join(target, file)), { recursive: true });
    copyFileSync(join(source, file), join(target, file));
  }
  return target;
}

export function addItem(directory, options, { schema, budgets }) {
  const item = inspectItem(directory);
  if (item.errors.length > 0) return { errors: item.errors };
  const budget = itemBudget(item, budgets);
  if (!budget.ok) return { errors: [budgetLine(budget)] };
  const provenance = readProvenance(readFileSync(join(item.directory, "provenance.md"), "utf8"));
  if (provenance.errors.length > 0) return { errors: provenance.errors };
  if (!options.manifest || !existsSync(options.manifest)) return { errors: ["--manifest must name an existing manifest file, the published one or docs/cosmetics/examples/minimal.json"] };
  const manifest = JSON.parse(readFileSync(options.manifest, "utf8"));
  const previous = manifest.items.find((entry) => entry.id === item.id);
  const built = buildEntry(item, options, previous, provenance);
  if (built.errors.length > 0) return { errors: built.errors };
  const result = upsertEntry(manifest, built.entry);
  const manifestErrors = checkManifest(schema, result.manifest);
  if (manifestErrors.length > 0) return { errors: manifestErrors };
  writeFileSync(options.manifest, `${JSON.stringify(result.manifest, null, 2)}\n`);
  const builtin = built.entry.sources.some(({ type }) => type === "default");
  const client = options.client ? syncClient(item, options.client, builtin) : undefined;
  return { errors: [], budget, entry: built.entry, change: result.change, client };
}

export function checkItems(directories, budgets) {
  const lines = [];
  let ok = true;
  for (const directory of directories) {
    const item = inspectItem(directory);
    if (item.errors.length > 0) {
      ok = false;
      lines.push(...item.errors.map((error) => `${item.id ?? directory}: ${error}`));
      continue;
    }
    const budget = itemBudget(item, budgets);
    ok &&= budget.ok;
    lines.push(budgetLine(budget));
  }
  return { ok, lines };
}

export function changedItems(paths, exists = existsSync) {
  const items = new Set();
  for (const path of paths) {
    const segments = path.trim().split("/");
    if (segments[0] !== "assets-src" || segments[1] !== "cosmetic" || segments.length < 5) continue;
    items.add(segments.slice(0, 4).join("/"));
  }
  return [...items].filter((item) => exists(join(item, "export"))).sort();
}

function parseSource(text) {
  const at = text.indexOf(":");
  return at < 0 ? { type: text } : { type: text.slice(0, at), ref: text.slice(at + 1) };
}

export function parseArguments(argv) {
  const [command, ...rest] = argv;
  const options = { command, paths: [], sources: [], client: defaultClientRoot };
  for (let at = 0; at < rest.length; at++) {
    const argument = rest[at];
    const value = () => {
      if (at + 1 >= rest.length) throw new Error(`${argument} needs a value`);
      return rest[++at];
    };
    if (argument === "--manifest") options.manifest = value();
    else if (argument === "--rarity") options.rarity = value();
    else if (argument === "--availability") options.availability = value();
    else if (argument === "--group") options.group = value();
    else if (argument === "--layers") options.layers = value().split(",");
    else if (argument === "--source") options.sources.push(parseSource(value()));
    else if (argument === "--client") options.client = value();
    else if (argument === "--no-client") options.client = undefined;
    else if (argument.startsWith("--")) throw new Error(`unknown option ${argument}`);
    else options.paths.push(argument);
  }
  return options;
}

const usage = "usage: asset-export.mjs add <assets-src/cosmetic/<type>/<slug>> --manifest <file> [--rarity <id>] [--availability <state>] [--source <type>[:<ref>]]... [--group <ref>] [--layers <back,...,front>] [--client <dir> | --no-client]\n       asset-export.mjs check <item folder>...\n       asset-export.mjs items < changed paths";

function runAdd(options) {
  if (options.paths.length !== 1) throw new Error(usage);
  const schema = loadSchema(join(repoRoot, "docs/cosmetics/schema.json"));
  const result = addItem(options.paths[0], options, { schema, budgets: loadBudgets() });
  for (const error of result.errors) console.error(error);
  if (result.errors.length > 0) return 1;
  console.log(budgetLine(result.budget));
  console.log(`${options.manifest}: ${result.change} ${result.entry.id} at revision ${result.entry.revision}`);
  if (result.client) console.log(`client art: ${result.client}; next run HypeDomino/Import Cosmetics in Unity (CosmeticContent.ImportAllAndExit in batch mode)`);
  return 0;
}

function runCheck(options) {
  if (options.paths.length === 0) throw new Error(usage);
  const result = checkItems(options.paths, loadBudgets());
  for (const line of result.lines) console.log(line);
  return result.ok ? 0 : 1;
}

function main(argv) {
  const options = parseArguments(argv);
  if (options.command === "add") return runAdd(options);
  if (options.command === "check") return runCheck(options);
  if (options.command === "items") {
    for (const item of changedItems(readFileSync(0, "utf8").split("\n").filter(Boolean))) console.log(item);
    return 0;
  }
  throw new Error(usage);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    process.exitCode = main(process.argv.slice(2));
  } catch (error) {
    console.error(error.message);
    process.exitCode = 2;
  }
}
