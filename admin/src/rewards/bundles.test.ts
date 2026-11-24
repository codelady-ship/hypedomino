import assert from "node:assert/strict";
import test from "node:test";
import { bundleEditChanged, bundleEditIssues, bundlePageSize, editFrom, editOf, emptyBundleEdit, formOf, newBundleIssues, nextRewardsHref, parsedIds, readBundleQuery, republishedNameKeys, rewardBundlesApiPath, useLabel, type RewardBundle } from "./bundles.ts";

const daily: RewardBundle = {
  id: "bundle.daily_login_day_1",
  revision: 1,
  coins: 30,
  cosmetics: null,
  medals: null,
  passXp: 0,
  guideValue: 30,
  usedBy: [{ domain: "daily_login", object: "day.1" }],
  differs: false,
  editable: false,
};

const wins: RewardBundle = {
  id: "bundle.wins",
  revision: 2,
  coins: 100,
  cosmetics: ["avatar.fox"],
  medals: ["medal.winner"],
  passXp: 0,
  guideValue: 1600,
  usedBy: [{ domain: "missions", object: "mission.first" }, { domain: "pass", object: "season_1", detail: "free.3" }],
  differs: true,
  editable: false,
};

test("the list query keeps a known domain, the differing flag and the cursor", () => {
  assert.deepEqual(readBundleQuery({ q: " fox ", domain: "pass", differing: "true", after: "bundle.a" }), { q: "fox", domain: "pass", differing: "true", after: "bundle.a" });
  assert.deepEqual(readBundleQuery({ domain: "toString", differing: "yes" }), { q: "", domain: "", differing: "", after: "" });
});

test("the list pages on the server and offers a next page only when full", () => {
  const query = { q: "a&b", domain: "", differing: "true", after: "" } as const;
  assert.equal(rewardBundlesApiPath(query), `/admin/reward-bundles?q=a%26b&differing=true&limit=${bundlePageSize}`);
  const ids = Array.from({ length: bundlePageSize }, (_, at) => `bundle.${at}`);
  assert.equal(nextRewardsHref(query, ids), `/rewards?q=a%26b&differing=true&after=bundle.${bundlePageSize - 1}`);
  assert.equal(nextRewardsHref(query, ids.slice(1)), null);
});
test("a use shows its object and its place inside it", () => {
  assert.equal(useLabel(wins.usedBy[0]), "mission.first");
  assert.equal(useLabel(wins.usedBy[1]), "season_1 free.3");
});

test("an edit warns about the name keys of every catalog it republishes, once per catalog", () => {
  const nameKeys = { missions: ["mission.first.name", "mission.second.name"], shop: ["offer.gold.name"], pass: [] };
  const held: RewardBundle = { ...wins, usedBy: [{ domain: "missions", object: "mission.first" }, { domain: "missions", object: "mission.second" }, { domain: "pass", object: "season_1", detail: "free.3" }] };
  assert.deepEqual(republishedNameKeys(held, nameKeys), ["mission.first.name", "mission.second.name"]);
  assert.deepEqual(republishedNameKeys({ ...wins, usedBy: [{ domain: "shop", object: "offer.gold" }] }, nameKeys), ["offer.gold.name"]);
  assert.deepEqual(republishedNameKeys(daily, nameKeys), []);
});

test("an edit starts from the bundle and fills what the API left out", () => {
  assert.deepEqual(editOf(daily), { coins: 30, crystals: 0, cosmetics: [], medals: [], passXp: 0 });
});

test("ids are split on commas, trimmed and kept once", () => {
  assert.deepEqual(parsedIds(" avatar.fox, ,frame.gold,avatar.fox "), ["avatar.fox", "frame.gold"]);
  assert.deepEqual(parsedIds("  "), []);
});

test("an edit counts as a change only when its content differs, in any order", () => {
  const items: RewardBundle = { ...wins, cosmetics: ["a", "b"] };
  assert.equal(bundleEditChanged(items, editOf(items)), false);
  assert.equal(bundleEditChanged(items, { ...editOf(items), cosmetics: ["b", "a"] }), false);
  assert.equal(bundleEditChanged(items, { ...editOf(items), coins: 1 }), true);
  assert.equal(bundleEditChanged(items, { ...editOf(items), medals: [] }), true);
});

test("an edit needs whole numbers and something to grant", () => {
  assert.deepEqual(bundleEditIssues(editOf(wins)), []);
  assert.deepEqual(bundleEditIssues({ ...editOf(wins), coins: -1 }), ["rewards.issue.number"]);
  assert.deepEqual(bundleEditIssues({ ...editOf(wins), coins: Number.NaN }), ["rewards.issue.number"]);
  assert.deepEqual(bundleEditIssues({ ...editOf(wins), passXp: 1.5 }), ["rewards.issue.number"]);
  assert.deepEqual(bundleEditIssues({ coins: 0, crystals: 0, cosmetics: [], medals: [], passXp: 0 }), ["rewards.issue.empty"]);
});

test("a new bundle needs a bundle id and the checks of an edit", () => {
  const granting = { ...emptyBundleEdit, coins: 50 };
  assert.deepEqual(newBundleIssues("bundle.weekend_pack", granting), []);
  assert.deepEqual(newBundleIssues("weekend_pack", granting), ["rewards.issue.id"]);
  assert.deepEqual(newBundleIssues("bundle.Weekend", granting), ["rewards.issue.id"]);
  assert.deepEqual(newBundleIssues("bundle.", emptyBundleEdit), ["rewards.issue.id", "rewards.issue.empty"]);
});

test("a created bundle can be filtered as its own domain", () => {
  assert.equal(readBundleQuery({ domain: "reward_bundles" }).domain, "reward_bundles");
});

test("a bundle form lists ids as text and reads them back once each", () => {
  const form = formOf(editOf(wins));
  assert.equal(form.cosmetics, "avatar.fox");
  assert.deepEqual(editFrom({ ...form, cosmetics: "avatar.fox, frame.gold, avatar.fox" }).cosmetics, ["avatar.fox", "frame.gold"]);
  assert.deepEqual(editFrom(form), editOf(wins));
});
