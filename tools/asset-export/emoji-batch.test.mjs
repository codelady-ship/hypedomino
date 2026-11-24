import { after, test } from "node:test";
import assert from "node:assert/strict";
import { mkdirSync, mkdtempSync, rmSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { emojiSlugs, emojiSteps, parseArguments } from "./emoji-batch.mjs";

const root = mkdtempSync(join(tmpdir(), "emoji-batch-"));
after(() => rmSync(root, { recursive: true, force: true }));

function item(slug, scripts) {
  mkdirSync(join(root, slug, "work"), { recursive: true });
  for (const stage of scripts) writeFileSync(join(root, slug, "work", `${slug}_${stage}.py`), "");
}

item("wink", ["loop", "cleanup"]);
item("grin", ["cleanup"]);
item("bare", []);
writeFileSync(join(root, "notes.txt"), "");

test("lists every emoji folder in name order", () => {
  assert.deepEqual(emojiSlugs(root), ["bare", "grin", "wink"]);
});

test("runs the clean-up before the loop and skips missing scripts", () => {
  const steps = emojiSteps(["bare", "grin", "wink"], root);
  assert.deepEqual(steps.map((step) => step.script.slice(root.length + 1)), [
    "grin/work/grin_cleanup.py",
    "wink/work/wink_cleanup.py",
    "wink/work/wink_loop.py",
  ]);
  assert.deepEqual(steps.map((step) => step.item), [`${root}/grin`, `${root}/wink`, `${root}/wink`]);
});

test("reads the python executable and the chosen slugs", () => {
  assert.deepEqual(parseArguments([]), { python: "python", slugs: [] });
  assert.deepEqual(parseArguments(["--python", "python3", "grin", "wink"]), { python: "python3", slugs: ["grin", "wink"] });
  assert.throws(() => parseArguments(["--python"]), /needs a value/);
  assert.throws(() => parseArguments(["--fast"]), /unknown option/);
});
