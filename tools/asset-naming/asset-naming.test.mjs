import { test } from "node:test";
import assert from "node:assert/strict";
import { fileURLToPath } from "node:url";
import { checkAssetPaths, listFiles } from "./asset-naming.mjs";

const avatar = "cosmetic/avatar/comet_fox";
const validTree = [
  "README.md",
  `${avatar}/provenance.md`,
  `${avatar}/concept/20261026-01.png`,
  `${avatar}/work/comet_fox.psb`,
  `${avatar}/work/comet_fox_head.ai`,
  `${avatar}/export/static.png`,
  `${avatar}/export/anim/sheet.png`,
  "cosmetic/table/night_garden/provenance.md",
  "cosmetic/table/night_garden/export/static.png",
  "cosmetic/table/night_garden/export/slice.json",
  "cosmetic/table/night_garden/export/layer/back/0001.png",
  "medal/first_win/provenance.md",
  "medal/first_win/export/static.png",
  "board/b_5/provenance.md",
  "board/b_5/work/b_5.psd",
];

function assertReported(paths, fragment) {
  const errors = checkAssetPaths(paths);
  assert.ok(errors.some((error) => error.includes(fragment)), `expected "${fragment}" in:\n${errors.join("\n")}`);
}

test("a tree that follows the convention passes", () => {
  assert.deepEqual(checkAssetPaths(validTree), []);
});

test("the repository assets-src passes", () => {
  const directory = fileURLToPath(new URL("../../assets-src/", import.meta.url));
  assert.deepEqual(checkAssetPaths(listFiles(directory)), []);
});

test("capitals, spaces and unknown classes are rejected", () => {
  assertReported([`${avatar}/work/Comet Fox.psb`, `${avatar}/provenance.md`], "invalid name Comet Fox.psb");
  assertReported(["props/lamp/provenance.md"], "unknown asset class props");
  assertReported(["cosmetic/hat/top_hat/provenance.md"], "unknown cosmetic type hat");
  assertReported(["board/b_11/provenance.md"], "invalid slug b_11");
});

test("an item folder without provenance is rejected", () => {
  assertReported([`${avatar}/concept/20261026-01.png`], "missing provenance.md");
});

test("an export without the static fallback is rejected", () => {
  assertReported([`${avatar}/provenance.md`, `${avatar}/export/anim/0001.png`], "missing export/static.png");
});

test("export folders must match the cosmetic type and its address slots", () => {
  assertReported([`${avatar}/provenance.md`, `${avatar}/export/static.png`, `${avatar}/export/layer/back/0001.png`], "export/layer is not allowed for avatar");
  assertReported([`${avatar}/provenance.md`, `${avatar}/export/static.png`, `${avatar}/export/model/comet_fox.fbx`], "export/model is not allowed for avatar");
  assertReported(["cosmetic/table/t/provenance.md", "cosmetic/table/t/export/static.png", "cosmetic/table/t/export/anim/0001.png"], "export/anim is not allowed for table");
  assertReported(["cosmetic/table/t/provenance.md", "cosmetic/table/t/export/static.png", "cosmetic/table/t/export/layer/0001.png"], "invalid layer name 0001.png");
  assertReported([`${avatar}/provenance.md`, `${avatar}/export/static.png`, `${avatar}/export/slice.json`], "export/slice.json is not allowed for avatar");
  assertReported(["cosmetic/emoji/e/provenance.md", "cosmetic/emoji/e/export/static.png", "cosmetic/emoji/e/export/anim"], "needs files inside a folder");
});

test("concept and work names follow the convention", () => {
  assertReported([`${avatar}/provenance.md`, `${avatar}/concept/final_v2.png`], "must be concept/<YYYYMMDD>-<nn>");
  assertReported([`${avatar}/provenance.md`, `${avatar}/work/other_fox.psd`], "must be work/<slug>[_<part>].<ext>");
  assertReported([`${avatar}/provenance.md`, `${avatar}/notes.txt`], "notes.txt is not allowed in an item folder");
});
