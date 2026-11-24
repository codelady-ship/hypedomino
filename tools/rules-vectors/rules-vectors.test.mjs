import { test } from "node:test";
import assert from "node:assert/strict";
import { fileURLToPath } from "node:url";
import { checkSuite, excludedRules, loadSuite, ruleIds, schemaErrors } from "./rules-vectors.mjs";

const repoRoot = new URL("../../", import.meta.url);
const suite = loadSuite(fileURLToPath(new URL("docs/rules-vectors/", repoRoot)), fileURLToPath(new URL("docs/rules.md", repoRoot)));
const allVectors = (files) => files.flatMap((file) => file.content.vectors);

function mutated(change) {
  const copy = structuredClone(suite);
  change(copy);
  return checkSuite(copy);
}

function findVector(files, id) {
  const vector = allVectors(files).find((candidate) => candidate.id === id);
  assert.ok(vector, `vector ${id} exists`);
  return vector;
}

test("every vector file passes the schema, coverage and arithmetic checks", () => {
  assert.deepEqual(checkSuite(suite), []);
});

test("the suite has more than 60 vectors", () => {
  assert.ok(allVectors(suite.files).length > 60);
});

test("every rule is either covered or explicitly excluded", () => {
  const excluded = excludedRules(suite.readme);
  assert.ok(excluded.length > 0);
  assert.equal(ruleIds(suite.rulesText).length, 59);
});

test("a duplicate id is reported", () => {
  const errors = mutated(({ files }) => files[0].content.vectors.push(structuredClone(files[0].content.vectors[0])));
  assert.ok(errors.some((error) => error.endsWith("duplicate id")), errors.join("\n"));
});

test("a rounding vector with a wrong result is reported", () => {
  const errors = mutated(({ files }) => {
    findVector(files, "round-22-to-25").expect.points = 20;
  });
  assert.ok(errors.some((error) => error.includes("round-22-to-25: expect")), errors.join("\n"));
});

test("a board vector with a wrong score is reported", () => {
  const errors = mutated(({ files }) => {
    findVector(files, "board-example1-step3").expect.score = 0;
  });
  assert.ok(errors.some((error) => error.includes("board-example1-step3: expect")), errors.join("\n"));
});

test("a broken board chain is reported", () => {
  const errors = mutated(({ files }) => {
    findVector(files, "board-example1-step3").given.board.line = ["3-2", "3-3"];
  });
  assert.ok(errors.some((error) => error.includes("board line breaks")), errors.join("\n"));
});

test("an unknown rule reference is reported", () => {
  const errors = mutated(({ files }) => files[0].content.vectors[0].rules.push("R-99"));
  assert.ok(errors.some((error) => error.endsWith("unknown rule R-99")), errors.join("\n"));
});

test("a rule with fewer than three vectors is reported", () => {
  const errors = mutated(({ files }) => {
    for (const file of files) file.content.vectors = file.content.vectors.filter((vector) => !vector.rules.includes("R-47"));
  });
  assert.ok(errors.includes("R-47: covered by 0 vector(s), needs 3"), errors.join("\n"));
});

test("a vector missing a required field fails the schema", () => {
  const errors = mutated(({ files }) => {
    delete files[0].content.vectors[0].description;
  });
  assert.ok(errors.some((error) => error.includes("missing description")), errors.join("\n"));
});

test("the schema checker rejects keywords it does not implement", () => {
  assert.throws(() => schemaErrors({ oneOf: [] }, 1), /unsupported schema keyword/);
});
