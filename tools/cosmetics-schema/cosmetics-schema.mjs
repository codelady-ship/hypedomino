import { readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";
import { pathToFileURL } from "node:url";
import { schemaErrors } from "../rules-vectors/rules-vectors.mjs";

export const itemTypes = ["avatar", "frame", "table", "tile", "emoji"];

const assetBase = (id) => `cosmetic/${id.replace(".", "/")}`;

function addressErrors(item) {
  const base = assetBase(item.id);
  const expected = [["staticAddress", item.staticAddress, `${base}/static`]];
  if (item.animationAddress) expected.push(["animationAddress", item.animationAddress, `${base}/anim`]);
  for (const layer of item.layers ?? []) expected.push([`layers.${layer.name}`, layer.address, `${base}/layer/${layer.name}`]);
  return expected.filter(([, actual, wanted]) => actual !== wanted).map(([field, actual, wanted]) => `${field} is ${actual}, expected ${wanted}`);
}

function layerErrors(item) {
  if (!item.layers) return [];
  const errors = [];
  const names = item.layers.map((layer) => layer.name);
  if (new Set(names).size !== names.length) errors.push("layer names are not unique");
  if (!item.layers.some((layer) => layer.animated)) errors.push("animated table has no animated layer");
  return errors;
}

const isDefault = (item) => item.sources.some((source) => source.type === "default");

function itemErrors(item) {
  const errors = [];
  if (item.id.split(".")[0] !== item.type) errors.push(`id prefix does not match type ${item.type}`);
  if (item.nameKey !== `cosmetic.${item.id}.name`) errors.push(`nameKey is ${item.nameKey}, expected cosmetic.${item.id}.name`);
  if (isDefault(item) && item.availability !== "live") errors.push(`default item is ${item.availability}, must be live`);
  if (isDefault(item) && item.animated) errors.push("default item is animated, must be static");
  return [...errors, ...addressErrors(item), ...layerErrors(item)];
}

export function checkManifest(schema, manifest) {
  const errors = schemaErrors(schema, manifest);
  if (errors.length > 0) return errors;
  const seen = new Set();
  for (const item of manifest.items) {
    if (seen.has(item.id)) errors.push(`${item.id}: duplicate id`);
    seen.add(item.id);
    errors.push(...itemErrors(item).map((error) => `${item.id}: ${error}`));
  }
  for (const type of itemTypes) {
    const hasLiveDefault = manifest.items.some((item) => item.type === type && item.availability === "live" && isDefault(item));
    if (!hasLiveDefault) errors.push(`${type}: no live item with a default source`);
  }
  return errors;
}

const readJson = (path) => JSON.parse(readFileSync(path, "utf8"));

export function loadExamples(directory) {
  return readdirSync(directory)
    .filter((name) => name.endsWith(".json"))
    .sort()
    .map((name) => ({ name, manifest: readJson(join(directory, name)) }));
}

export const loadSchema = (path) => readJson(path);

function main(args) {
  const schema = loadSchema("docs/cosmetics/schema.json");
  const targets = args.length > 0 ? args.map((path) => ({ name: path, manifest: readJson(path) })) : loadExamples("docs/cosmetics/examples");
  const errors = targets.flatMap(({ name, manifest }) => checkManifest(schema, manifest).map((error) => `${name} ${error}`));
  for (const error of errors) console.error(error);
  if (errors.length > 0) process.exit(1);
  console.log(`cosmetics manifests ok (${targets.length})`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) main(process.argv.slice(2));
