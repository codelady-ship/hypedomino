import { existsSync, readFileSync } from "node:fs";
import { pathToFileURL } from "node:url";

export const fontsFolder = "client/Assets/HypeDomino/UI/Fonts";
export const manifestPath = `${fontsFolder}/fonts.json`;
export const builtinStringsPath = "client/Assets/HypeDomino/UI/Localization/Resources/builtin-strings.json";
export const tokensPath = "docs/design-tokens.json";
export const nicknameCharactersPath = "server/internal/profile/nickname-characters.json";
const nicknameStyle = "name";
const staticPopulationMode = 0;
const builtinLanguages = ["en", "az", "tr", "ru"];
const licenseMarker = "SIL Open Font License, Version 1.1";

export const sourcePath = (family, face) => `${fontsFolder}/Source/${family.folder}/${face.id}.ttf`;
export const licensePath = (family) => `${fontsFolder}/Source/${family.folder}/OFL.txt`;
export const fontAssetPath = (face) => `${fontsFolder}/${face.id}.asset`;

function tableOffset(font, tag) {
  const tables = font.readUInt16BE(4);
  for (let at = 0; at < tables; at++) {
    const record = 12 + at * 16;
    if (font.toString("latin1", record, record + 4) === tag) return font.readUInt32BE(record + 8);
  }
  throw new Error(`no ${tag} table`);
}

function format4(font, start, codePoints) {
  const segments = font.readUInt16BE(start + 6) / 2;
  const ends = start + 14;
  const starts = ends + segments * 2 + 2;
  const deltas = starts + segments * 2;
  const rangeOffsets = deltas + segments * 2;
  for (let segment = 0; segment < segments; segment++) {
    const end = font.readUInt16BE(ends + segment * 2);
    const first = font.readUInt16BE(starts + segment * 2);
    const delta = font.readUInt16BE(deltas + segment * 2);
    const rangeAt = rangeOffsets + segment * 2;
    const range = font.readUInt16BE(rangeAt);
    for (let code = first; code <= end && code !== 0xffff; code++) {
      let glyph;
      if (range === 0) glyph = (code + delta) & 0xffff;
      else {
        const raw = font.readUInt16BE(rangeAt + range + (code - first) * 2);
        glyph = raw === 0 ? 0 : (raw + delta) & 0xffff;
      }
      if (glyph !== 0) codePoints.add(code);
    }
  }
}

function format12(font, start, codePoints) {
  const groups = font.readUInt32BE(start + 12);
  for (let group = 0; group < groups; group++) {
    const record = start + 16 + group * 12;
    const first = font.readUInt32BE(record);
    const last = font.readUInt32BE(record + 4);
    const glyph = font.readUInt32BE(record + 8);
    for (let code = first; code <= last; code++) if (glyph + code - first !== 0) codePoints.add(code);
  }
}

export function cmapCodePoints(font) {
  const cmap = tableOffset(font, "cmap");
  const subtables = font.readUInt16BE(cmap + 2);
  const codePoints = new Set();
  for (let at = 0; at < subtables; at++) {
    const record = cmap + 4 + at * 8;
    const platform = font.readUInt16BE(record);
    const encoding = font.readUInt16BE(record + 2);
    const start = cmap + font.readUInt32BE(record + 4);
    const unicode = platform === 0 || (platform === 3 && (encoding === 1 || encoding === 10));
    if (!unicode) continue;
    const format = font.readUInt16BE(start);
    if (format === 4) format4(font, start, codePoints);
    else if (format === 12) format12(font, start, codePoints);
  }
  return codePoints;
}

export function fontAssetFacts(yaml) {
  const characters = new Set();
  const table = yaml.match(/\n  m_CharacterTable:\n((?: {2}[- ] .*\n)*)/);
  if (table) for (const match of table[1].matchAll(/m_Unicode: (\d+)/g)) characters.add(Number(match[1]));
  const fallbacks = [];
  const fallbackTable = yaml.match(/\n  m_FallbackFontAssetTable:(?: \[\])?\n((?: {2}- .*\n)*)/);
  if (fallbackTable) for (const match of fallbackTable[1].matchAll(/guid: ([0-9a-f]{32})/g)) fallbacks.push(match[1]);
  const mode = yaml.match(/\n  m_AtlasPopulationMode: (\d+)/);
  return { characters, fallbacks, populationMode: mode ? Number(mode[1]) : undefined };
}

export const metaGuid = (meta) => meta.match(/^guid: ([0-9a-f]{32})$/m)?.[1];

export function requiredCharacters(manifest, builtinStrings) {
  const characters = new Set();
  for (const set of manifest.characterSets) for (const character of set.characters) characters.add(character.codePointAt(0));
  for (const entry of builtinStrings.entries)
    for (const language of builtinLanguages) for (const character of entry[language] ?? "") characters.add(character.codePointAt(0));
  return [...characters].sort((a, b) => a - b);
}

export function nicknameCharacters(set) {
  const characters = new Set();
  for (const text of [...Object.values(set.letters), set.digits, set.separators]) for (const character of text) characters.add(character.codePointAt(0));
  return [...characters].sort((a, b) => a - b);
}

export function nicknameFace(tokens) {
  const style = tokens.typography.scale[nicknameStyle];
  return `${style.family}-${style.weight}`;
}

export function chainOf(faceId, faces) {
  const byId = new Map(faces.map((face) => [face.id, face]));
  const chain = [];
  const visit = (id) => {
    if (chain.includes(id)) return;
    const face = byId.get(id);
    if (!face) throw new Error(`unknown face ${id}`);
    chain.push(id);
    for (const fallback of face.fallbacks) visit(fallback);
  };
  visit(faceId);
  return chain;
}

export function manifestErrors(manifest, tokens) {
  const errors = [];
  const { family: tokenFamilies, scale } = tokens.typography;
  const families = new Map(manifest.families.map((family) => [family.token, family]));
  const faces = new Map(manifest.faces.map((face) => [face.id, face]));
  for (const [token, family] of Object.entries(tokenFamilies)) {
    const entry = families.get(token);
    if (!entry) errors.push(`fonts.json has no family for typography.family.${token}`);
    else if (entry.name !== family.name) errors.push(`fonts.json family ${token} is ${entry.name}, the tokens say ${family.name}`);
    else if (entry.italic !== (family.style === "italic")) errors.push(`fonts.json family ${token} italic is ${entry.italic}, the tokens say ${family.style}`);
  }
  for (const face of manifest.faces) {
    const family = tokenFamilies[face.family];
    if (!family) errors.push(`face ${face.id}: unknown family ${face.family}`);
    else if (!family.weights.includes(face.weight)) errors.push(`face ${face.id}: weight ${face.weight} not in typography.family.${face.family}`);
    if (face.id !== `${face.family}-${face.weight}`) errors.push(`face ${face.id}: id must be ${face.family}-${face.weight}`);
    for (const fallback of face.fallbacks) if (!faces.has(fallback)) errors.push(`face ${face.id}: unknown fallback ${fallback}`);
  }
  for (const [name, style] of Object.entries(scale))
    if (!faces.has(`${style.family}-${style.weight}`)) errors.push(`typography.scale.${name}: no face ${style.family}-${style.weight} in fonts.json`);
  if (!faces.has(manifest.defaultFace)) errors.push(`defaultFace ${manifest.defaultFace} is not a face`);
  return errors;
}

const label = (codePoint) => `${String.fromCodePoint(codePoint)} U+${codePoint.toString(16).toUpperCase().padStart(4, "0")}`;

export function coverageErrors({ manifest, required, glyphsOf, source }) {
  const errors = [];
  const familyOf = new Map(manifest.faces.map((face) => [face.id, face.family]));
  const uiOnly = [...manifest.uiOnly.characters].map((character) => character.codePointAt(0));
  for (const face of manifest.faces) {
    const chain = chainOf(face.id, manifest.faces);
    const missing = required.filter((codePoint) => !chain.some((id) => glyphsOf(id).has(codePoint)));
    if (missing.length > 0) errors.push(`${source} chain ${chain.join(" > ")} misses ${missing.map(label).join(", ")}`);
    if (source !== "font asset") continue;
    for (const codePoint of uiOnly) {
      const holder = chain.find((id) => glyphsOf(id).has(codePoint));
      if (holder && !manifest.uiOnly.families.includes(familyOf.get(holder))) errors.push(`${source} chain ${chain.join(" > ")} sets ${label(codePoint)} in ${holder}, only ${manifest.uiOnly.families.join(" or ")} may`);
    }
  }
  return errors;
}

export function nicknameErrors({ manifest, faceId, allowed, glyphsOf, source }) {
  const chain = chainOf(faceId, manifest.faces);
  const missing = allowed.filter((codePoint) => !chain.some((id) => glyphsOf(id).has(codePoint)));
  if (missing.length === 0) return [];
  return [`${source} chain ${chain.join(" > ")} of the ${nicknameStyle} style misses nickname characters ${missing.map(label).join(", ")} (${nicknameCharactersPath})`];
}

export function checkFonts(root = ".") {
  const read = (path) => readFileSync(`${root}/${path}`);
  const exists = (path) => existsSync(`${root}/${path}`);
  const manifest = JSON.parse(read(manifestPath).toString("utf8"));
  const tokens = JSON.parse(read(tokensPath).toString("utf8"));
  const builtinStrings = JSON.parse(read(builtinStringsPath).toString("utf8"));
  const allowed = nicknameCharacters(JSON.parse(read(nicknameCharactersPath).toString("utf8")));
  const errors = manifestErrors(manifest, tokens);
  if (errors.length > 0) return { errors, report: [] };
  const families = new Map(manifest.families.map((family) => [family.token, family]));
  for (const family of manifest.families) {
    const path = licensePath(family);
    if (!exists(path) || !read(path).toString("utf8").includes(licenseMarker)) errors.push(`${path}: the OFL 1.1 licence text is missing`);
  }
  const sources = new Map();
  const assets = new Map();
  const guidToFace = new Map();
  for (const face of manifest.faces) {
    const path = sourcePath(families.get(face.family), face);
    if (!exists(path)) errors.push(`${path} is missing`);
    else sources.set(face.id, cmapCodePoints(read(path)));
    const asset = fontAssetPath(face);
    if (!exists(asset) || !exists(`${asset}.meta`)) {
      errors.push(`${asset} is missing; run HypeDomino.Editor.FontSetup.GenerateAndExit`);
      continue;
    }
    assets.set(face.id, fontAssetFacts(read(asset).toString("utf8")));
    guidToFace.set(metaGuid(read(`${asset}.meta`).toString("utf8")), face.id);
  }
  if (errors.length > 0) return { errors, report: [] };
  const required = requiredCharacters(manifest, builtinStrings);
  errors.push(...coverageErrors({ manifest, required, glyphsOf: (id) => sources.get(id), source: "font file" }));
  errors.push(...coverageErrors({ manifest, required, glyphsOf: (id) => assets.get(id).characters, source: "font asset" }));
  const faceId = nicknameFace(tokens);
  errors.push(...nicknameErrors({ manifest, faceId, allowed, glyphsOf: (id) => sources.get(id), source: "font file" }));
  errors.push(...nicknameErrors({ manifest, faceId, allowed, glyphsOf: (id) => assets.get(id).characters, source: "font asset" }));
  for (const face of manifest.faces) {
    const facts = assets.get(face.id);
    if (facts.populationMode !== staticPopulationMode) errors.push(`${fontAssetPath(face)}: atlas population mode ${facts.populationMode}, must be static`);
    const linked = facts.fallbacks.map((guid) => guidToFace.get(guid) ?? guid);
    if (linked.join(",") !== face.fallbacks.join(",")) errors.push(`${fontAssetPath(face)}: fallbacks ${linked.join(", ") || "none"}, fonts.json says ${face.fallbacks.join(", ") || "none"}`);
  }
  const report = manifest.faces.map((face) => `${chainOf(face.id, manifest.faces).join(" > ")}: ${required.length}/${required.length}`);
  report.push(`nickname characters in the ${nicknameStyle} style (${chainOf(faceId, manifest.faces).join(" > ")}): ${allowed.length}/${allowed.length}`);
  return { errors, report };
}

function main() {
  const { errors, report } = checkFonts();
  for (const error of errors) console.error(error);
  if (errors.length > 0) {
    process.exitCode = 1;
    return;
  }
  for (const line of report) console.log(line);
  console.log("font coverage ok");
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) main();
