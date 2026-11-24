import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { test } from "node:test";
import { fileURLToPath } from "node:url";
import {
  chainOf,
  checkFonts,
  cmapCodePoints,
  coverageErrors,
  fontAssetFacts,
  manifestErrors,
  manifestPath,
  metaGuid,
  nicknameCharacters,
  nicknameCharactersPath,
  nicknameErrors,
  nicknameFace,
  requiredCharacters,
  tokensPath,
} from "./fonts.mjs";

const repoRoot = fileURLToPath(new URL("../../", import.meta.url));
const readJson = (path) => JSON.parse(readFileSync(`${repoRoot}/${path}`, "utf8"));
const manifest = readJson(manifestPath);
const tokens = readJson(tokensPath);
const code = (character) => character.codePointAt(0);

test("the committed font set covers every role chain and matches its font assets", () => {
  const { errors, report } = checkFonts(repoRoot);
  assert.deepEqual(errors, []);
  assert.equal(report.length, manifest.faces.length + 1);
});

test("the nickname characters the server allows are the letters of the name style sets", () => {
  const allowed = nicknameCharacters(readJson(nicknameCharactersPath));
  assert.equal(nicknameFace(tokens), "display-900");
  assert.equal(allowed.length, 52 + 14 + 6 + 66 + 10 + 4);
  for (const character of "AzƏəĞğİıŞşÇçÖöÜüÂâÎîÛûЖжЁё0_.- ") assert.ok(allowed.includes(code(character)), character);
  for (const character of "éßіÄñЎ€!@") assert.ok(!allowed.includes(code(character)), character);
});

test("a nickname character the name chain cannot draw is reported", () => {
  const small = {
    faces: [
      { id: "display-900", fallbacks: ["ui-400"] },
      { id: "ui-400", fallbacks: [] },
    ],
  };
  const glyphs = { "display-900": new Set([code("A")]), "ui-400": new Set([code("ə")]) };
  const allowed = nicknameCharacters({ letters: { latin: "Aə", extra: "é" }, digits: "", separators: "" });
  assert.deepEqual(nicknameErrors({ manifest: small, faceId: "display-900", allowed, glyphsOf: (id) => glyphs[id], source: "font asset" }), [
    `font asset chain display-900 > ui-400 of the name style misses nickname characters é U+00E9 (${nicknameCharactersPath})`,
  ]);
  glyphs["ui-400"].add(code("é"));
  assert.deepEqual(nicknameErrors({ manifest: small, faceId: "display-900", allowed, glyphsOf: (id) => glyphs[id], source: "font asset" }), []);
});

test("the cmap of a committed face lists what the face draws", () => {
  const ui = cmapCodePoints(readFileSync(`${repoRoot}/client/Assets/HypeDomino/UI/Fonts/Source/JosefinSans/ui-400.ttf`));
  assert.ok(ui.has(code("Ə")));
  assert.ok(ui.has(code("₼")));
  assert.ok(!ui.has(code("Ж")));
  const numeral = cmapCodePoints(readFileSync(`${repoRoot}/client/Assets/HypeDomino/UI/Fonts/Source/Cinzel/numeral-900.ttf`));
  assert.ok(numeral.has(code("7")));
  assert.ok(!numeral.has(code("ə")));
});

test("a chain follows fallbacks depth first and visits each face once", () => {
  const faces = [
    { id: "a", fallbacks: ["b", "c"] },
    { id: "b", fallbacks: ["c"] },
    { id: "c", fallbacks: ["a"] },
  ];
  assert.deepEqual(chainOf("a", faces), ["a", "b", "c"]);
  assert.throws(() => chainOf("x", faces), /unknown face x/);
});

test("a style without a face, a renamed family and a weight outside the tokens are reported", () => {
  const copy = structuredClone(manifest);
  copy.faces = copy.faces.filter((face) => face.id !== "numeral-900");
  copy.families.find((family) => family.token === "ui").name = "Other Sans";
  copy.faces.find((face) => face.id === "hint-700").weight = 800;
  const errors = manifestErrors(copy, tokens);
  assert.ok(errors.includes("typography.scale.counter: no face numeral-900 in fonts.json"));
  assert.ok(errors.includes("fonts.json family ui is Other Sans, the tokens say Josefin Sans"));
  assert.ok(errors.includes("face hint-700: weight 800 not in typography.family.hint"));
  assert.deepEqual(manifestErrors(manifest, tokens), []);
});

test("a chain without a glyph and a currency sign set outside the ui family are reported", () => {
  const small = {
    faces: [
      { id: "display-900", family: "display", fallbacks: ["ui-400"] },
      { id: "ui-400", family: "ui", fallbacks: [] },
    ],
    uiOnly: { characters: "₼", families: ["ui"] },
  };
  const glyphs = { "display-900": new Set([code("A"), code("₼")]), "ui-400": new Set([code("A"), code("₼")]) };
  const errors = coverageErrors({ manifest: small, required: [code("A"), code("Ж")], glyphsOf: (id) => glyphs[id], source: "font asset" });
  assert.deepEqual(errors, [
    "font asset chain display-900 > ui-400 misses Ж U+0416",
    "font asset chain display-900 > ui-400 sets ₼ U+20BC in display-900, only ui may",
    "font asset chain ui-400 misses Ж U+0416",
  ]);
  const fileErrors = coverageErrors({ manifest: small, required: [code("A")], glyphsOf: (id) => glyphs[id], source: "font file" });
  assert.deepEqual(fileErrors, [], "a font file may hold a currency sign; only the font asset must leave it out");
});

test("required characters join the character sets and every built-in string", () => {
  const required = requiredCharacters({ characterSets: [{ characters: "ab" }] }, { entries: [{ en: "b", az: "ə", tr: "ı", ru: "Ж" }] });
  assert.deepEqual(required, [code("a"), code("b"), code("ı"), code("ə"), code("Ж")]);
});

test("font asset facts read the character table, fallbacks and population mode", () => {
  const yaml = [
    "MonoBehaviour:",
    "  m_AtlasPopulationMode: 0",
    "  m_CharacterTable:",
    "  - m_ElementType: 1",
    "    m_Unicode: 32",
    "    m_GlyphIndex: 3",
    "  - m_ElementType: 1",
    "    m_Unicode: 1046",
    "    m_GlyphIndex: 9",
    "  m_FallbackFontAssetTable:",
    "  - {fileID: 11400000, guid: 0123456789abcdef0123456789abcdef, type: 2}",
    "  m_FontWeightTable: []",
    "",
  ].join("\n");
  const facts = fontAssetFacts(yaml);
  assert.deepEqual([...facts.characters], [32, 1046]);
  assert.deepEqual(facts.fallbacks, ["0123456789abcdef0123456789abcdef"]);
  assert.equal(facts.populationMode, 0);
  assert.deepEqual(fontAssetFacts("MonoBehaviour:\n  m_FallbackFontAssetTable: []\n  m_AtlasPopulationMode: 1\n").fallbacks, []);
  assert.equal(metaGuid("fileFormatVersion: 2\nguid: 0123456789abcdef0123456789abcdef\n"), "0123456789abcdef0123456789abcdef");
});
