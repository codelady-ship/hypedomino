import { readFileSync, readdirSync, statSync } from "node:fs";
import { join } from "node:path";
import { pathToFileURL } from "node:url";

export const bytesPerMegabyte = 1_000_000;

export function parseGetSize(csv) {
  const lines = csv.trim().split(/\r?\n/);
  if (lines.length !== 2 || lines[0].trim() !== "MIN,MAX") throw new Error(`expected "MIN,MAX" and one row from bundletool get-size total, got:\n${csv}`);
  const [min, max] = lines[1].split(",").map((cell) => Number(cell.trim()));
  if (!Number.isSafeInteger(min) || !Number.isSafeInteger(max) || min < 0 || max < min) throw new Error(`invalid sizes in "${lines[1]}"`);
  return { min, max };
}

export function folderBytes(directory) {
  let total = 0;
  for (const entry of readdirSync(directory, { withFileTypes: true })) {
    const path = join(directory, entry.name);
    total += entry.isDirectory() ? folderBytes(path) : statSync(path).size;
  }
  return total;
}

const megabytes = (bytes) => (bytes / bytesPerMegabyte).toFixed(1);

export function report({ limitBytes, sizes, remoteBytes }) {
  if (sizes.length === 0) throw new Error("no device spec sizes given");
  const worst = Math.max(...sizes.map((size) => size.max));
  const ok = worst < limitBytes;
  const rows = sizes.map((size) => `| ${size.spec} | ${megabytes(size.min)} | ${megabytes(size.max)} |`);
  const lines = [
    "## Install size",
    "",
    "| Device spec | Download min (MB) | Download max (MB) |",
    "|---|---|---|",
    ...rows,
    "",
    `${ok ? "Pass" : "Fail"}: largest download ${megabytes(worst)} MB, budget under ${megabytes(limitBytes)} MB (docs/performance.md 3).`,
  ];
  if (remoteBytes !== undefined) lines.push("", `Remote content, not counted: ${megabytes(remoteBytes)} MB.`);
  return { ok, markdown: lines.join("\n") + "\n" };
}

export function parseArguments(argv) {
  const options = { sizes: [] };
  for (let at = 0; at < argv.length; at++) {
    const argument = argv[at];
    if (argument === "--limit-bytes") options.limitBytes = Number(argv[++at]);
    else if (argument === "--remote") options.remote = argv[++at];
    else if (argument.includes("=")) {
      const [spec, path] = argument.split("=", 2);
      options.sizes.push({ spec, path });
    } else throw new Error(`unknown argument ${argument}`);
  }
  if (!Number.isSafeInteger(options.limitBytes) || options.limitBytes <= 0) throw new Error("--limit-bytes needs a positive whole number");
  return options;
}

function main(argv) {
  const options = parseArguments(argv);
  const sizes = options.sizes.map(({ spec, path }) => ({ spec, ...parseGetSize(readFileSync(path, "utf8")) }));
  const remoteBytes = options.remote ? folderBytes(options.remote) : undefined;
  const result = report({ limitBytes: options.limitBytes, sizes, remoteBytes });
  process.stdout.write(result.markdown);
  return result.ok ? 0 : 1;
}

if (import.meta.url === pathToFileURL(process.argv[1]).href) {
  try {
    process.exitCode = main(process.argv.slice(2));
  } catch (error) {
    console.error(error.message);
    process.exitCode = 2;
  }
}
