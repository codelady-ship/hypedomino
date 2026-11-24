import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync, readdirSync } from "node:fs";
import { join, relative } from "node:path";
import { fileURLToPath } from "node:url";

const clientAssets = fileURLToPath(new URL("../../client/Assets/HypeDomino/", import.meta.url));

function csharpFiles(directory) {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = join(directory, entry.name);
    if (entry.isDirectory()) return csharpFiles(path);
    return entry.name.endsWith(".cs") ? [path] : [];
  });
}

test("client tests measure allocations with Allocates.Nothing, never the engine's shared-recorder constraint", () => {
  const offenders = csharpFiles(clientAssets)
    .filter((file) => readFileSync(file, "utf8").includes("AllocatingGCMemory"))
    .map((file) => relative(clientAssets, file));
  assert.deepEqual(offenders, []);
});
