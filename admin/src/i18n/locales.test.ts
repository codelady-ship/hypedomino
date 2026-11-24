import assert from "node:assert/strict";
import { test } from "node:test";
import { preferredLocale } from "./locales.ts";

test("the account language wins over the browser cookie, then the default", () => {
  assert.equal(preferredLocale("en", "az"), "en");
  assert.equal(preferredLocale(undefined, "en"), "en");
  assert.equal(preferredLocale("", "en"), "en");
  assert.equal(preferredLocale(undefined, "tr"), "az");
});
