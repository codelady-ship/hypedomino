import assert from "node:assert/strict";
import { test } from "node:test";
import { bundlesApiPath, chainError, chainFromText, distinctNames, keyHref, nextStringsHref, olderBundlesHref, placeholdersFromText, placeholdersText, problemsText, readStringsQuery, stringsApiPath, stringsPageSize, valueOf, type GameString } from "./strings.ts";

function entry(key: string, fields: Partial<GameString> = {}): GameString {
  return { key, note: "", max_length: 40, placeholders: [], deprecated: false, values: {}, ...fields };
}

const strings = [
  entry("match.result.win", { note: "Shown after a win", values: { en: { text: "You won", status: "approved" } } }),
  entry("shop.offer.title", { values: { az: { text: "Təklif", status: "draft" } } }),
  entry("quick.hello"),
];

test("the strings query keeps known languages and a numeric version, defaulting the bundle language", () => {
  const languages = ["az", "en"];
  assert.deepEqual(readStringsQuery({ q: " win ", missing: "en", after: "match.a", bundles: "en", before: "3" }, languages), { q: "win", missing: "en", after: "match.a", bundles: "en", before: "3" });
  assert.deepEqual(readStringsQuery({ missing: "de", bundles: "toString", before: "0" }, languages), { q: "", missing: "", after: "", bundles: "az", before: "" });
});

test("keys and bundle versions page on the server with their own cursors", () => {
  const query = { q: "a&b", missing: "en", after: "", bundles: "az", before: "" };
  assert.equal(stringsApiPath(query), `/admin/strings?q=a%26b&missing=en&limit=${stringsPageSize}`);
  assert.equal(bundlesApiPath({ ...query, before: "7" }), `/admin/strings/bundles?language=az&before=7&limit=${stringsPageSize}`);
  const keys = Array.from({ length: stringsPageSize }, (_, at) => `key.${at}`);
  assert.equal(nextStringsHref(query, keys), `/strings?q=a%26b&missing=en&after=key.${stringsPageSize - 1}&bundles=az`);
  assert.equal(nextStringsHref(query, keys.slice(1)), null);
  const versions = Array.from({ length: stringsPageSize }, (_, at) => stringsPageSize - at);
  assert.equal(olderBundlesHref(query, versions), "/strings?q=a%26b&missing=en&bundles=az&before=1");
  assert.equal(olderBundlesHref(query, versions.slice(1)), null);
});
test("a language without a value reads as missing", () => {
  assert.deepEqual(valueOf(strings[2], "ru"), { text: "", status: "missing" });
  assert.deepEqual(valueOf(strings[0], "en"), { text: "You won", status: "approved" });
});

test("placeholders round-trip through name:type text", () => {
  const placeholders = placeholdersFromText(" name:string , count : integer,,");
  assert.deepEqual(placeholders, [{ name: "name", type: "string" }, { name: "count", type: "integer" }]);
  assert.equal(placeholdersText(placeholders), "name:string, count:integer");
  assert.deepEqual(placeholdersFromText("bare"), [{ name: "bare", type: "" }]);
});

test("blocked problems read as key: error", () => {
  assert.equal(problemsText([{ key: "a.b", error: "too long" }, { key: "c.d", error: "bad placeholder" }]), "a.b: too long; c.d: bad placeholder");
});

const languages = ["az", "en", "ru", "tr"];

test("a chain reads from comma separated language codes", () => {
  assert.deepEqual(chainFromText(" az, tr ,,en "), ["az", "tr", "en"]);
  assert.deepEqual(chainFromText("  "), []);
});

test("a chain starts with its language, ends with en and repeats no known language", () => {
  assert.equal(chainError("az", ["az", "tr", "en"], languages), null);
  assert.equal(chainError("en", ["en"], languages), null);
  assert.equal(chainError("az", [], languages), "start");
  assert.equal(chainError("az", ["tr", "az", "en"], languages), "start");
  assert.equal(chainError("tr", ["tr", "az"], languages), "end");
  assert.equal(chainError("tr", ["tr", "de", "en"], languages), "unknown");
  assert.equal(chainError("az", ["az", "tr", "tr", "en"], languages), "repeat");
});

test("an unnamed key links to its search", () => {
  assert.equal(keyHref("season.one.name"), "/strings?q=season.one.name");
  assert.equal(keyHref("a&b"), "/strings?q=a%26b");
});

test("the names checked before a publish are trimmed, non-empty and listed once in their order", () => {
  assert.deepEqual(distinctNames([" mission.b.name", "mission.a.name", "", "mission.b.name ", "  "]), ["mission.b.name", "mission.a.name"]);
  assert.deepEqual(distinctNames([]), []);
});
