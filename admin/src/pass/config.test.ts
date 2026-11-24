import assert from "node:assert/strict";
import test from "node:test";
import {
  addSeason,
  changedSeasons,
  changedSettings,
  cloneSeason,
  droppedSeasons,
  emptyPassConfig,
  newSeason,
  nextSeasonId,
  passChanges,
  rankedWindow,
  removeSeason,
  replaceSeason,
  seasonIssues,
  seasonState,
  settingIssues,
  trackTotal,
  withCurve,
  withLevel,
  xpTable,
  xpToNext,
  type PassConfig,
  type Season,
} from "./config.ts";

const curve = { levels: 4, xpBase: 300, xpStep: 50, xpMax: 340 };

const season: Season = {
  id: "season_1",
  start: "2026-12-01T00:00:00Z",
  end: "2027-03-01T00:00:00Z",
  curve,
  free: { "1": { id: "bundle.pass_season_1_free_1", revision: 1, coins: 30 }, "3": { id: "bundle.pass_season_1_free_3", revision: 1, cosmetics: ["emoji.s1_wave"] } },
  premium: {
    "1": { id: "bundle.pass_season_1_premium_1", revision: 1, crystals: 6, cosmetics: ["avatar.s1_navigator"] },
    "2": { id: "bundle.pass_season_1_premium_2", revision: 1, crystals: 6 },
    "3": { id: "bundle.pass_season_1_premium_3", revision: 1, crystals: 6 },
    "4": { id: "bundle.pass_season_1_premium_4", revision: 2, crystals: 6 },
  },
};

const config: PassConfig = { schema: 1, freeTrackCoins: 100, premiumTrackCrystals: 30, premiumPlusLevels: 2, levelCoinPrice: 200, levelCrystalPrice: 40, seasons: [season] };

test("the xp table follows the server curve and stops at the last level", () => {
  assert.deepEqual([1, 2, 3, 4].map((level) => xpToNext(curve, level)), [300, 340, 340, 0]);
  assert.deepEqual(xpTable(curve), [
    { level: 1, toNext: 300, total: 0 },
    { level: 2, toNext: 340, total: 300 },
    { level: 3, toNext: 340, total: 640 },
    { level: 4, toNext: 0, total: 980 },
  ]);
});

test("a season is planned before its start, running inside its window and ended after", () => {
  assert.equal(seasonState(season, "2026-11-30T23:59:59Z"), "planned");
  assert.equal(seasonState(season, "2026-12-01T00:00:00Z"), "running");
  assert.equal(seasonState(season, "2027-03-01T00:00:00Z"), "ended");
});

test("a clone takes the next id, the curve and the currencies, renames bundles, drops cosmetics and free levels left empty", () => {
  assert.equal(nextSeasonId(emptyPassConfig()), "season_1");
  const clone = cloneSeason(config, season, "2027-03-01T00:00:00Z", "2027-06-01T00:00:00Z");
  assert.equal(clone.id, "season_2");
  assert.deepEqual(clone.curve, curve);
  assert.notEqual(clone.curve, season.curve);
  assert.deepEqual(clone.premium["1"], { id: "bundle.pass_season_2_premium_1", revision: 1, crystals: 6 });
  assert.deepEqual(clone.free["1"], { id: "bundle.pass_season_2_free_1", revision: 1, coins: 30 });
  assert.equal(clone.free["3"], undefined);
  assert.equal(addSeason(config, clone).seasons.length, 2);
  assert.deepEqual(seasonIssues(addSeason(config, clone), clone), []);
});

test("a new season gets an empty premium bundle per level, and the curve keeps tracks within its levels", () => {
  const fresh = newSeason(config, "2027-03-01T00:00:00Z", "2027-06-01T00:00:00Z", { levels: 3, xpBase: 100, xpStep: 10, xpMax: 200 });
  assert.deepEqual(Object.keys(fresh.premium), ["1", "2", "3"]);
  assert.deepEqual(fresh.premium["2"], { id: "bundle.pass_season_2_premium_2", revision: 1, crystals: 0 });
  assert.deepEqual(seasonIssues(addSeason(config, fresh), fresh), ["pass.issue.premium_empty"]);
  const shorter = withCurve(season, { ...curve, levels: 2 });
  assert.deepEqual(Object.keys(shorter.premium), ["1", "2"]);
  assert.deepEqual(Object.keys(shorter.free), ["1"]);
  assert.deepEqual(Object.keys(withCurve(season, { ...curve, levels: 5 }).premium), ["1", "2", "3", "4", "5"]);
});

test("a level edit raises the bundle revision once and an emptied free level is removed", () => {
  const raised = withLevel(season, "premium", 4, { amount: 7 }, season);
  assert.equal(raised.premium["4"].revision, 3);
  assert.equal(withLevel(raised, "premium", 4, { amount: 8 }, season).premium["4"].revision, 3);
  assert.equal(withLevel(raised, "premium", 4, { amount: 6 }, season).premium["4"].revision, 2);
  const added = withLevel(season, "free", 2, { amount: 20 }, season);
  assert.deepEqual(added.free["2"], { id: "bundle.pass_season_1_free_2", revision: 1, coins: 20 });
  assert.equal(withLevel(season, "free", 3, { cosmetics: [] }, season).free["3"], undefined);
  assert.deepEqual(withLevel(season, "premium", 1, { cosmetics: [] }, season).premium["1"], { id: "bundle.pass_season_1_premium_1", revision: 2, crystals: 6 });
  assert.equal(trackTotal(added, "free"), 50);
  assert.equal(trackTotal(season, "premium"), 24);
});

test("issues name what the server would refuse", () => {
  assert.deepEqual(seasonIssues(config, season), []);
  const broken: Season = { ...withLevel(season, "free", 2, { amount: 80 }, season), end: season.start, curve: { levels: 2, xpBase: 0, xpStep: 1, xpMax: 1 } };
  assert.deepEqual(seasonIssues(config, broken), ["pass.issue.window", "pass.issue.curve", "pass.issue.levels", "pass.issue.free_budget"]);
  const overlapping = { ...season, id: "season_2", start: "2027-02-01T00:00:00Z", end: "2027-04-01T00:00:00Z" };
  assert.deepEqual(seasonIssues(addSeason(config, overlapping), overlapping), ["pass.issue.overlap"]);
  assert.deepEqual(seasonIssues(config, withLevel(season, "premium", 2, { amount: 20 }, season)), ["pass.issue.premium_budget"]);
  assert.deepEqual(settingIssues(config), []);
  assert.deepEqual(settingIssues({ ...config, botSeatRewardPercent: 120, levelCoinPrice: 0, dailyMatchXpCap: -1 }), ["pass.issue.settings", "pass.issue.bot_percent", "pass.issue.level_price"]);
});

test("the publish summary lists changed and dropped seasons and changed settings", () => {
  const clone = cloneSeason(config, season, "2027-03-01T00:00:00Z", "2027-06-01T00:00:00Z");
  const next = { ...replaceSeason(addSeason(config, clone), withLevel(season, "free", 2, { amount: 5 }, season)), levelCoinPrice: 250 };
  assert.deepEqual(changedSeasons(config, next), ["season_1", "season_2"]);
  assert.deepEqual(changedSettings(config, next), ["levelCoinPrice"]);
  assert.deepEqual(droppedSeasons(next, removeSeason(next, "season_2")), ["season_2"]);
  assert.deepEqual(changedSeasons(null, config), ["season_1"]);
});

test("the history names each changed setting and season field, and the seasons added or removed", () => {
  const clone = cloneSeason(config, season, "2027-03-01T00:00:00Z", "2027-06-01T00:00:00Z");
  const next = { ...replaceSeason(addSeason(config, clone), withLevel(season, "premium", 4, { amount: 7 }, season)), levelCoinPrice: 250 };
  assert.deepEqual(passChanges(config, next), {
    settings: [{ field: "levelCoinPrice", before: "200", after: "250" }],
    entries: [
      {
        id: "season_1",
        change: "changed",
        fields: [
          { field: "premium.4.revision", before: "2", after: "3" },
          { field: "premium.4.crystals", before: "6", after: "7" },
        ],
      },
      { id: "season_2", change: "added", fields: [] },
    ],
  });
  assert.deepEqual(passChanges(next, removeSeason(next, "season_2")), { settings: [], entries: [{ id: "season_2", change: "removed", fields: [] }] });
  assert.deepEqual(passChanges(config, config), { settings: [], entries: [] });
});

test("the oldest listed version shows its seasons as added", () => {
  assert.deepEqual(passChanges(null, config), { settings: [], entries: [{ id: "season_1", change: "added", fields: [] }] });
});

test("a new season defaults to the first ranked season not ended and free of pass seasons", () => {
  const ranked = [
    { start: "2026-09-01T00:00:00Z", end: "2026-12-01T00:00:00Z" },
    { start: "2026-12-01T00:00:00Z", end: "2027-03-01T00:00:00Z" },
    { start: "2027-03-01T00:00:00Z", end: "2027-06-01T00:00:00Z" },
  ];
  assert.deepEqual(rankedWindow(emptyPassConfig(), ranked, "2026-11-20T00:00:00Z"), ranked[0]);
  assert.deepEqual(rankedWindow(config, ranked, "2026-11-20T00:00:00Z"), ranked[0]);
  assert.deepEqual(rankedWindow(emptyPassConfig(), ranked, "2026-12-01T00:00:00Z"), ranked[1]);
  assert.deepEqual(rankedWindow(config, ranked, "2026-12-01T00:00:00Z"), ranked[2]);
  const next = addSeason(config, newSeason(config, ranked[2].start, ranked[2].end, curve));
  assert.equal(rankedWindow(next, ranked, "2026-12-01T00:00:00Z"), null);
  assert.equal(rankedWindow(emptyPassConfig(), [], "2026-11-20T00:00:00Z"), null);
});
