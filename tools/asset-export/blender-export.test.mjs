import { test } from "node:test";
import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { existsSync, mkdirSync, mkdtempSync, readdirSync, readFileSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
import { inspectItem, readPng } from "./asset-export.mjs";

const exportScript = fileURLToPath(new URL("blender/export_loop.py", import.meta.url));
const testScene = fileURLToPath(new URL("blender/test_scene.py", import.meta.url));
const windowsInstalls = "C:/Program Files/Blender Foundation";

function findBlender() {
  if (process.env.BLENDER) return process.env.BLENDER;
  const installed = existsSync(windowsInstalls) ? readdirSync(windowsInstalls).sort().reverse().map((name) => join(windowsInstalls, name, "blender.exe")).find(existsSync) : undefined;
  if (installed) return installed;
  return spawnSync("blender", ["--version"]).status === 0 ? "blender" : undefined;
}

const blender = findBlender();
const skip = blender ? false : "Blender is not installed; set BLENDER to its executable";

function runExport(extraSetup, scriptArguments) {
  const setup = extraSetup ? ["--python-expr", extraSetup] : [];
  return spawnSync(blender, ["--background", "--factory-startup", "--python-exit-code", "1", "--python", testScene, ...setup, "--python", exportScript, "--", ...scriptArguments], { encoding: "utf8" });
}

function makeItem() {
  const root = mkdtempSync(join(tmpdir(), "blender-export-"));
  const directory = join(root, "assets-src", "cosmetic", "avatar", "blender_cube");
  mkdirSync(join(directory, "export", "anim"), { recursive: true });
  writeFileSync(join(directory, "provenance.md"), "- artSource: handmade\n");
  writeFileSync(join(directory, "export", "anim", "0009.png"), "stale");
  return { root, directory };
}

test("Blender renders the loop and the chosen static into export/ with alpha at the full canvas", { skip, timeout: 120000 }, () => {
  const { root, directory } = makeItem();
  try {
    const run = runExport(undefined, ["--frames", join(directory, "export", "anim"), "--static", join(directory, "export", "static.png"), "--static-frame", "2"]);
    assert.equal(run.status, 0, run.stderr || run.stdout);
    const item = inspectItem(directory);
    assert.deepEqual(item.errors, []);
    assert.deepEqual(item.loops[0].frames.map(({ name }) => name), ["0001.png", "0002.png", "0003.png"]);
    for (const path of ["static.png", "anim/0001.png", "anim/0003.png"]) {
      assert.deepEqual(readPng(readFileSync(join(directory, "export", path))), { width: 32, height: 32, bitDepth: 8, colorType: 6 }, path);
    }
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});

test("Blender export refuses a perspective camera and a static frame outside the range", { skip, timeout: 120000 }, () => {
  const { root, directory } = makeItem();
  try {
    const perspective = runExport("import bpy; bpy.context.scene.camera.data.type = 'PERSP'", ["--frames", join(directory, "export", "anim")]);
    assert.notEqual(perspective.status, 0);
    assert.match(perspective.stderr, /orthographic/);
    const outside = runExport(undefined, ["--static", join(directory, "export", "static.png"), "--static-frame", "9"]);
    assert.notEqual(outside.status, 0);
    assert.match(outside.stderr, /outside the scene range/);
    assert.equal(existsSync(join(directory, "export", "static.png")), false);
  } finally {
    rmSync(root, { recursive: true, force: true });
  }
});
