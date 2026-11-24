import assert from "node:assert/strict";
import { test } from "node:test";
import { entryCount, moderationToText, textToModeration } from "./lists.ts";

test("a leading = marks a whole word, other lines are substrings, blanks are dropped", () => {
  const moderation = textToModeration({ any: " badword \n\n= scum\n=\n", az: "", en: "", tr: "", ru: "плохо", reserved: "admin\n moderator " });
  assert.deepEqual(moderation, {
    words: { any: [{ entry: "badword", match: "substring" }, { entry: "scum", match: "word" }], ru: [{ entry: "плохо", match: "substring" }] },
    reserved: ["admin", "moderator"],
  });
  assert.equal(entryCount(moderation), 5);
});

test("text and lists round-trip", () => {
  const moderation = { words: { en: [{ entry: "scum", match: "word" as const }], tr: [{ entry: "kötü", match: "substring" as const }] }, reserved: ["admin"] };
  assert.deepEqual(textToModeration(moderationToText(moderation)), moderation);
  assert.deepEqual(moderationToText(null), { any: "", az: "", en: "", tr: "", ru: "", reserved: "" });
});
