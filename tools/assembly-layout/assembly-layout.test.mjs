import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { checkLayout, parseLayout, readAssemblies } from "./assembly-layout.mjs";

const clientRoot = fileURLToPath(new URL("../../client/", import.meta.url));
const readme = readFileSync(`${clientRoot}README.md`, "utf8");

const sampleReadme = [
  "# client",
  "",
  "## Layout",
  "",
  "| Path | Assembly | References | Notes |",
  "|---|---|---|---|",
  "| `Assets/A/Core` | `HypeDomino.Core` | none | |",
  "| `Assets/A/Net` | `HypeDomino.Net` | Core | `Google.Protobuf.dll` |",
  "| `Assets/A/Game` | `HypeDomino.Game` | Core, Net | URP |",
  "| `Assets/A/Scenes` | | | Boot scene |",
  "",
  "## Settings",
  "",
  "| Area | Setting | Why |",
  "|---|---|---|",
  "| Scripting | IL2CPP | size |",
].join("\n");

function sampleAssemblies() {
  return [
    { path: "Assets/A/Core", name: "HypeDomino.Core", guid: "c".repeat(32), references: [], precompiledReferences: [] },
    { path: "Assets/A/Net", name: "HypeDomino.Net", guid: "e".repeat(32), references: ["HypeDomino.Core"], precompiledReferences: [] },
    { path: "Assets/A/Game", name: "HypeDomino.Game", guid: "a".repeat(32), references: [`GUID:${"c".repeat(32)}`, "HypeDomino.Net", "Unity.Addressables"], precompiledReferences: [] },
  ];
}

function errorsAfter(change) {
  const rows = parseLayout(sampleReadme);
  const assemblies = sampleAssemblies();
  change(rows, assemblies);
  return checkLayout(rows, assemblies);
}

function assertReported(errors, expected) {
  assert.ok(errors.includes(expected), errors.join("\n"));
}

test("the asmdefs under client/Assets match the Layout table of client/README.md", () => {
  assert.deepEqual(checkLayout(parseLayout(readme), readAssemblies(clientRoot)), []);
});

test("the Layout table is read in order with its project references, and rows without an assembly are skipped", () => {
  assert.deepEqual(parseLayout(sampleReadme), [
    { path: "Assets/A/Core", assembly: "HypeDomino.Core", references: [] },
    { path: "Assets/A/Net", assembly: "HypeDomino.Net", references: ["HypeDomino.Core"] },
    { path: "Assets/A/Game", assembly: "HypeDomino.Game", references: ["HypeDomino.Core", "HypeDomino.Net"] },
  ]);
  assert.throws(() => parseLayout("# client\n\n## Settings\n"), /no Layout section/);
});

test("a matching sample passes, with GUID references resolved and package references ignored", () => {
  assert.deepEqual(errorsAfter(() => {}), []);
});

test("a reference to an assembly below or level with it in the table is reported", () => {
  const errors = errorsAfter((rows, assemblies) => {
    assemblies[1].references.push("HypeDomino.Game");
    rows[1].references.push("HypeDomino.Game");
  });
  assert.deepEqual(errors, ["HypeDomino.Net: references HypeDomino.Game, which is not above it in the table"]);
});

test("a reference the table does not list, and a listed one the asmdef lacks, are reported", () => {
  assertReported(errorsAfter((rows) => { rows[2].references = ["HypeDomino.Net"]; }), "HypeDomino.Game: references HypeDomino.Core, which the table does not list");
  assertReported(errorsAfter((rows, assemblies) => { assemblies[2].references = ["HypeDomino.Net"]; }), "HypeDomino.Game: the table lists HypeDomino.Core, which the asmdef does not reference");
});

test("Core referencing anything, even a package or a dll, is reported", () => {
  assertReported(errorsAfter((rows, assemblies) => { assemblies[0].references.push("Unity.Addressables"); }), "HypeDomino.Core: must reference nothing");
  assertReported(errorsAfter((rows, assemblies) => { assemblies[0].precompiledReferences.push("Google.Protobuf.dll"); }), "HypeDomino.Core: must reference nothing");
});

test("an asmdef missing from the table, a row with no asmdef, a moved asmdef and a duplicate row are reported", () => {
  assertReported(errorsAfter((rows, assemblies) => { assemblies.push({ path: "Assets/A/Bench", name: "HypeDomino.Bench", guid: "b".repeat(32), references: [], precompiledReferences: [] }); }), "HypeDomino.Bench: not in the table");
  assertReported(errorsAfter((rows) => { rows.push({ path: "Assets/A/UI", assembly: "HypeDomino.UI", references: [] }); }), "HypeDomino.UI: in the table but no asmdef defines it");
  assertReported(errorsAfter((rows, assemblies) => { assemblies[1].path = "Assets/A/Network"; }), "HypeDomino.Net: defined in Assets/A/Network, the table says Assets/A/Net");
  assertReported(errorsAfter((rows) => { rows.push({ ...rows[0] }); }), "HypeDomino.Core: listed twice in the table");
});
