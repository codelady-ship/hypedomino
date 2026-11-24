import { readdirSync } from "node:fs";
import { join } from "node:path";
import { pathToFileURL } from "node:url";
import { itemTypes } from "../cosmetics-schema/cosmetics-schema.mjs";

export const assetClasses = ["cosmetic", "medal", "effect", "ui", "tile", "board", "sound"];
const itemParts = new Set(["provenance.md", "concept", "work", "export"]);
const exportFoldersByType = { avatar: ["anim"], frame: ["anim"], tile: ["anim"], emoji: ["anim"], table: ["layer"] };
const slugPattern = /^[a-z0-9_]+$/;
const boardSlugPattern = /^b_([1-9]|10)$/;
const namePattern = /^[a-z0-9_]+(\.[a-z0-9]+)?$/;
const conceptPattern = /^\d{8}-\d{2}\.(png|jpg|jpeg|webp)$/;

function itemRoot(segments) {
  const [assetClass, typeOrSlug, slug] = segments;
  if (segments.length < 2) return { error: "file outside an item folder" };
  if (!assetClasses.includes(assetClass)) return { error: `unknown asset class ${assetClass}` };
  if (assetClass !== "cosmetic") {
    const valid = assetClass === "board" ? boardSlugPattern.test(typeOrSlug) : slugPattern.test(typeOrSlug);
    return valid ? { root: segments.slice(0, 2), slug: typeOrSlug } : { error: `invalid slug ${typeOrSlug}` };
  }
  if (!itemTypes.includes(typeOrSlug)) return { error: `unknown cosmetic type ${typeOrSlug}` };
  if (!slug || !slugPattern.test(slug)) return { error: `invalid slug ${slug}` };
  return { root: segments.slice(0, 3), type: typeOrSlug, slug };
}

function cosmeticExportErrors(type, rest) {
  if (rest.length === 2 && rest[1] === "static.png") return [];
  if (rest.length === 2 && rest[1] === "slice.json" && type === "table") return [];
  const folder = rest[1];
  if (!exportFoldersByType[type].includes(folder)) return [`export/${folder} is not allowed for ${type}`];
  const depth = folder === "layer" ? 3 : 2;
  if (folder === "layer" && !slugPattern.test(rest[2] ?? "")) return [`invalid layer name ${rest[2]}`];
  return rest.length > depth ? [] : [`export/${folder} needs files inside a folder`];
}

function partErrors(item, rest) {
  const [part, name] = rest;
  if (!itemParts.has(part)) return [`${part} is not allowed in an item folder`];
  if (part === "provenance.md") return rest.length === 1 ? [] : ["provenance.md must be a file"];
  if (rest.length < 2) return [`${part} must be a folder`];
  if (part === "concept") return rest.length === 2 && conceptPattern.test(name) ? [] : [`concept file ${rest.slice(1).join("/")} must be concept/<YYYYMMDD>-<nn>.<png|jpg|jpeg|webp>`];
  if (part === "work") {
    const baseName = name.split(".")[0];
    const ownsName = baseName === item.slug || baseName.startsWith(`${item.slug}_`);
    return rest.length === 2 && ownsName ? [] : [`work file ${rest.slice(1).join("/")} must be work/<slug>[_<part>].<ext>`];
  }
  return item.type ? cosmeticExportErrors(item.type, rest) : [];
}

function segmentErrors(segments) {
  const isConcept = (index) => segments[index - 1] === "concept";
  return segments.filter((segment, index) => !namePattern.test(segment) && !isConcept(index)).map((segment) => `invalid name ${segment}`);
}

export function checkAssetPaths(paths) {
  const errors = [];
  const items = new Map();
  for (const path of paths) {
    if (path === "README.md") continue;
    const segments = path.split("/");
    const report = (message) => errors.push(`${path}: ${message}`);
    segmentErrors(segments).forEach(report);
    const item = itemRoot(segments);
    if (item.error) { report(item.error); continue; }
    const key = item.root.join("/");
    if (!items.has(key)) items.set(key, { ...item, files: new Set() });
    const rest = segments.slice(item.root.length);
    items.get(key).files.add(rest.join("/"));
    if (rest.length === 0) { report("item folder expected, found a file"); continue; }
    partErrors(item, rest).forEach(report);
  }
  for (const [key, item] of items) {
    if (!item.files.has("provenance.md")) errors.push(`${key}: missing provenance.md`);
    if (item.type && [...item.files].some((file) => file.startsWith("export/")) && !item.files.has("export/static.png")) errors.push(`${key}: missing export/static.png`);
  }
  return errors;
}

export function listFiles(directory, prefix = "") {
  return readdirSync(join(directory, prefix), { withFileTypes: true }).flatMap((entry) => {
    const path = prefix ? `${prefix}/${entry.name}` : entry.name;
    return entry.isDirectory() ? listFiles(directory, path) : [path];
  });
}

function main(args) {
  const directory = args[0] ?? "assets-src";
  const paths = listFiles(directory);
  const errors = checkAssetPaths(paths);
  for (const error of errors) console.error(error);
  if (errors.length > 0) process.exit(1);
  console.log(`asset naming ok (${paths.length} files)`);
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) main(process.argv.slice(2));
