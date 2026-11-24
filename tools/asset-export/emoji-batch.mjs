import { existsSync, readdirSync } from "node:fs";
import { spawnSync } from "node:child_process";
import { pathToFileURL } from "node:url";
import { checkItems, loadBudgets } from "./asset-export.mjs";

export const emojiRoot = "assets-src/cosmetic/emoji";
export const stages = ["cleanup", "loop"];

export function emojiSlugs(root = emojiRoot) {
  return readdirSync(root, { withFileTypes: true }).filter((entry) => entry.isDirectory()).map((entry) => entry.name).sort();
}

export function emojiSteps(slugs, root = emojiRoot, exists = existsSync) {
  return slugs.flatMap((slug) => stages
    .map((stage) => ({ item: `${root}/${slug}`, script: `${root}/${slug}/work/${slug}_${stage}.py` }))
    .filter((step) => exists(step.script)));
}

export function parseArguments(argv) {
  const options = { python: "python", slugs: [] };
  for (let at = 0; at < argv.length; at++) {
    if (argv[at] === "--python") {
      if (at + 1 >= argv.length) throw new Error("--python needs a value");
      options.python = argv[++at];
    } else if (argv[at].startsWith("--")) throw new Error(`unknown option ${argv[at]}`);
    else options.slugs.push(argv[at]);
  }
  return options;
}

function main(argv) {
  const options = parseArguments(argv);
  const slugs = options.slugs.length > 0 ? options.slugs : emojiSlugs();
  for (const step of emojiSteps(slugs)) {
    const run = spawnSync(options.python, ["-I", step.script, step.item], { stdio: "inherit" });
    if (run.status !== 0) {
      console.error(`${step.script} failed`);
      return 1;
    }
  }
  const result = checkItems(slugs.map((slug) => `${emojiRoot}/${slug}`), loadBudgets());
  for (const line of result.lines) console.log(line);
  return result.ok ? 0 : 1;
}

if (process.argv[1] && import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    process.exitCode = main(process.argv.slice(2));
  } catch (error) {
    console.error(error.message);
    process.exitCode = 2;
  }
}
