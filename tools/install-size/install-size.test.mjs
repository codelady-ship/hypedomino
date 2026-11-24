import { test } from "node:test";
import assert from "node:assert/strict";
import { mkdtempSync, mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { fileURLToPath } from "node:url";
import { folderBytes, parseArguments, parseGetSize, report } from "./install-size.mjs";

test("bundletool get-size total output is read as min and max bytes", () => {
  assert.deepEqual(parseGetSize("MIN,MAX\n41234567,42345678\n"), { min: 41234567, max: 42345678 });
  assert.deepEqual(parseGetSize("MIN,MAX\r\n10,10\r\n"), { min: 10, max: 10 });
});

test("output that is not one MIN,MAX row is refused", () => {
  assert.throws(() => parseGetSize("SDK,MIN,MAX\n33,1,2\n"));
  assert.throws(() => parseGetSize("MIN,MAX\n"));
  assert.throws(() => parseGetSize("MIN,MAX\n5,4\n"));
  assert.throws(() => parseGetSize("MIN,MAX\nabc,4\n"));
});

test("the largest download over every spec decides pass or fail", () => {
  const sizes = [
    { spec: "low", min: 90_000_000, max: 100_000_000 },
    { spec: "high", min: 140_000_000, max: 149_999_999 },
  ];
  assert.equal(report({ limitBytes: 150_000_000, sizes }).ok, true);
  sizes.push({ spec: "mid", min: 140_000_000, max: 150_000_000 });
  const failed = report({ limitBytes: 150_000_000, sizes });
  assert.equal(failed.ok, false);
  assert.match(failed.markdown, /Fail: largest download 150\.0 MB/);
  assert.match(failed.markdown, /\| mid \| 140\.0 \| 150\.0 \|/);
});

test("remote content is listed but not counted", () => {
  const result = report({ limitBytes: 150_000_000, sizes: [{ spec: "mid", min: 1, max: 1 }], remoteBytes: 900_000_000 });
  assert.equal(result.ok, true);
  assert.match(result.markdown, /Remote content, not counted: 900\.0 MB/);
});

test("a folder's size is the sum of every file below it", () => {
  const root = mkdtempSync(join(tmpdir(), "install-size-"));
  mkdirSync(join(root, "nested"));
  writeFileSync(join(root, "a.bin"), Buffer.alloc(10));
  writeFileSync(join(root, "nested", "b.bin"), Buffer.alloc(32));
  assert.equal(folderBytes(root), 42);
});

test("arguments need a limit and take spec=path pairs", () => {
  assert.deepEqual(parseArguments(["--limit-bytes", "150000000", "--remote", "ServerData/Android", "low=low.csv"]), {
    limitBytes: 150_000_000,
    remote: "ServerData/Android",
    sizes: [{ spec: "low", path: "low.csv" }],
  });
  assert.throws(() => parseArguments(["low=low.csv"]));
  assert.throws(() => parseArguments(["--limit-bytes", "1", "--verbose"]));
});

test("every tier of the device matrix has an arm64 spec", () => {
  for (const tier of ["low", "mid", "high"]) {
    const spec = JSON.parse(readFileSync(fileURLToPath(new URL(`specs/${tier}.json`, import.meta.url)), "utf8"));
    assert.deepEqual(spec.supportedAbis, ["arm64-v8a"]);
    assert.ok(spec.sdkVersion >= 33 && spec.screenDensity > 0);
  }
});
