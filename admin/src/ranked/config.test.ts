import assert from "node:assert/strict";
import test from "node:test";
import {
  changedStructural,
  configIssues,
  countryList,
  effectiveConfig,
  effectiveReset,
  generatedReset,
  isOverride,
  modeIssues,
  ratingChanges,
  resetIssues,
  rewardIssues,
  snapshotIssues,
  withMode,
  withoutRegion,
  withRankReward,
  withRegion,
  withResetRow,
  type ModeConfig,
  type RatingConfig,
} from "./config.ts";

const tiers = ["Iron", "Bronze", "Silver", "Gold", "Platinum", "Emerald", "Diamond"];
const divisionRanks = tiers.flatMap((tier) => ["IV", "III", "II", "I"].map((division) => `${tier} ${division}`));

const mode: ModeConfig = {
  divisionLp: 100,
  demotionLp: 75,
  apexRecalcHourUtc: 0,
  challengerSlots: 50,
  challengerMinLp: 500,
  grandmasterSlots: 200,
  grandmasterMinLp: 200,
  initialMmr: 1200,
  mmrScale: 400,
  mmrK: 20,
  mmrKProvisional: 40,
  mmrProvisionalMatches: 20,
  rankMmrBase: 800,
  rankMmrStep: 40,
  lpBase: 20,
  lpGapScale: 200,
  lpFactorMin: 0.5,
  lpFactorMax: 1.5,
  lpChangeMin: 5,
  lpChangeMax: 40,
  placementMatches: 5,
  placementMaxDropDivisions: 4,
  placementMaxClimbDivisions: 4,
  newPlayerPlacementCap: "Gold IV",
  seasonLengthMonths: 3,
  resetTierDrop: 1,
  resetFloor: "Bronze IV",
  resetCeiling: "Diamond IV",
};

const defaults: RatingConfig = {
  schema: 1,
  leaderboardTopSize: 200,
  rankRewardCoins: { Iron: 100, Gold: 700 },
  emblemBrackets: [1, 50, 100, 200],
  modes: { "1v1": mode, "2v2": mode },
};

test("the generated reset table matches the defaults of the ranking spec", () => {
  const table = generatedReset(mode, divisionRanks);
  assert.equal(table.length, 29);
  const row = (old: string) => table[[...divisionRanks, "Master"].indexOf(old)];
  assert.equal(row("Iron III"), "Iron III");
  assert.equal(row("Bronze II"), "Bronze IV");
  assert.equal(row("Silver IV"), "Bronze IV");
  assert.equal(row("Silver III"), "Bronze III");
  assert.equal(row("Gold II"), "Silver II");
  assert.equal(row("Diamond I"), "Emerald I");
  assert.equal(row("Master"), "Diamond IV");
});

test("an edited row stays an override until it matches the generated rank again", () => {
  const edited = withResetRow(mode, divisionRanks, "Gold II", "Silver III");
  assert.deepEqual(edited.seasonResetTable, { "Gold II": "Silver III" });
  assert.equal(isOverride(edited, "Gold II"), true);
  assert.equal(effectiveReset(edited, divisionRanks)[14], "Silver III");
  const restored = withResetRow(edited, divisionRanks, "Gold II", "Silver II");
  assert.equal(restored.seasonResetTable, undefined);
  assert.equal(isOverride(restored, "Gold II"), false);
});

test("the reset table refuses what the server refuses", () => {
  assert.deepEqual(resetIssues(mode, divisionRanks), []);
  assert.deepEqual(resetIssues({ ...mode, resetFloor: "Gold IV", resetCeiling: "Silver IV" }, divisionRanks), ["ranked.issue.floor"]);
  assert.deepEqual(resetIssues({ ...mode, resetCeiling: "Master" }, divisionRanks), ["ranked.issue.floor"]);
  assert.ok(resetIssues(withResetRow(mode, divisionRanks, "Iron III", "Gold IV"), divisionRanks).includes("ranked.issue.reset_above"));
  assert.ok(resetIssues(withResetRow(mode, divisionRanks, "Gold II", "Iron IV"), divisionRanks).includes("ranked.issue.reset_order"));
  assert.ok(resetIssues(withResetRow(mode, divisionRanks, "Master", "Master"), divisionRanks).includes("ranked.issue.reset_apex"));
});

test("a cleared tier drop is reported and leaves the table readable", () => {
  const cleared = { ...mode, resetTierDrop: Number.NaN };
  assert.equal(effectiveReset(cleared, divisionRanks)[14], "Gold II");
  assert.deepEqual(modeIssues(cleared, divisionRanks), ["ranked.issue.tier_drop"]);
});

test("mode values are checked against the server rules", () => {
  assert.deepEqual(modeIssues(mode, divisionRanks), []);
  const cases: [Partial<ModeConfig>, string][] = [
    [{ demotionLp: 100 }, "ranked.issue.division_lp"],
    [{ apexRecalcHourUtc: 24 }, "ranked.issue.hour"],
    [{ challengerSlots: -1 }, "ranked.issue.apex"],
    [{ mmrK: 0 }, "ranked.issue.at_least_one"],
    [{ mmrProvisionalMatches: -1 }, "ranked.issue.not_negative"],
    [{ lpFactorMin: 2 }, "ranked.issue.lp_factor"],
    [{ lpChangeMin: 50 }, "ranked.issue.lp_change"],
    [{ lpChangeMax: 100 }, "ranked.issue.lp_division"],
    [{ newPlayerPlacementCap: "Master" }, "ranked.issue.placement_cap"],
    [{ resetTierDrop: -1 }, "ranked.issue.tier_drop"],
    [{ mmrK: undefined }, "ranked.issue.at_least_one"],
  ];
  for (const [change, issue] of cases) {
    assert.ok(modeIssues({ ...mode, ...change }, divisionRanks).includes(issue as never), issue);
  }
});

test("a rank reward raises its revision once against the published bundle and leaves when emptied", () => {
  const published: ModeConfig = { ...mode, rankRewards: { Gold: { id: "bundle.rank_gold", revision: 3, cosmetics: ["frame.a"] } } };
  const once = withRankReward(published, "Gold", { cosmetics: ["frame.b"] }, published);
  assert.equal(once.rankRewards?.Gold.revision, 4);
  const twice = withRankReward(once, "Gold", { cosmetics: ["frame.b", "frame.c"] }, published);
  assert.equal(twice.rankRewards?.Gold.revision, 4);
  const back = withRankReward(twice, "Gold", { cosmetics: ["frame.a"] }, published);
  assert.equal(back.rankRewards?.Gold.revision, 3);
  const emptied = withRankReward(published, "Gold", { cosmetics: [] }, published);
  assert.equal(emptied.rankRewards, undefined);
});

test("a new rank reward takes the tier bundle id and revision 1", () => {
  const added = withRankReward(mode, "Gold", { medals: ["medal.gold"] }, undefined);
  assert.deepEqual(added.rankRewards?.Gold, { id: "bundle.rank_gold", revision: 1, medals: ["medal.gold"] });
  assert.deepEqual(rewardIssues(added), []);
  assert.deepEqual(rewardIssues(withRankReward(added, "Gold", { id: "rank gold" }, undefined)), ["ranked.issue.bundle_id"]);
});

test("a partial catalog is read over the server defaults, as the server parses it", () => {
  const catalog = { schema: 1, rankRewardCoins: { Gold: 900 }, modes: { "1v1": { mmrK: 30 } } } as unknown as RatingConfig;
  const merged = effectiveConfig(catalog, defaults);
  assert.equal(merged.leaderboardTopSize, 200);
  assert.deepEqual(merged.rankRewardCoins, { Iron: 100, Gold: 900 });
  assert.equal(merged.modes["1v1"].mmrK, 30);
  assert.equal(merged.modes["1v1"].divisionLp, 100);
  assert.equal(merged.modes["2v2"].mmrK, 20);
  assert.deepEqual(merged.emblemBrackets, [1, 50, 100, 200]);
  assert.equal(effectiveConfig(null, defaults), defaults);
});

test("regions take upper case countries and a country belongs to one region", () => {
  assert.deepEqual(countryList("az, ge  az"), ["AZ", "GE"]);
  const caucasus = withRegion(defaults, "caucasus", ["AZ", "GE"]);
  assert.deepEqual(snapshotIssues(caucasus), []);
  assert.deepEqual(snapshotIssues(withRegion(caucasus, "west", ["GE"])), ["ranked.issue.country_twice"]);
  assert.deepEqual(snapshotIssues(withRegion(defaults, "EU", ["DE"])), ["ranked.issue.region_id"]);
  assert.deepEqual(snapshotIssues(withRegion(defaults, "eu", ["DEU"])), ["ranked.issue.country"]);
  assert.equal(withoutRegion(caucasus, "caucasus").leaderboardRegions, undefined);
});

test("top size and rank coins are checked", () => {
  assert.deepEqual(snapshotIssues({ ...defaults, leaderboardTopSize: 0 }), ["ranked.issue.top_size"]);
  assert.deepEqual(snapshotIssues({ ...defaults, rankRewardCoins: { Gold: -1 } }), ["ranked.issue.coins"]);
});

test("every mode counts towards the publish block", () => {
  assert.deepEqual(configIssues(defaults, divisionRanks), []);
  assert.deepEqual(configIssues(withMode(defaults, "2v2", { apexRecalcHourUtc: 30 }), divisionRanks), ["ranked.issue.hour"]);
});

test("the stored key order of a snapshot is not a change", () => {
  const stored = { ...defaults, abuse: { pairMinMatches: 6, repeatPairMax: 3 } } as RatingConfig;
  const reordered = { ...defaults, abuse: { repeatPairMax: 3, pairMinMatches: 6 } } as RatingConfig;
  assert.deepEqual(ratingChanges(stored, reordered), []);
  const published: ModeConfig = { ...mode, rankRewards: { Gold: { id: "bundle.rank_gold", revision: 3, medals: ["medal.a"], cosmetics: ["frame.a"] } } };
  const same = { ...mode, rankRewards: { Gold: { revision: 3, id: "bundle.rank_gold", cosmetics: ["frame.a"], medals: ["medal.a"] } } };
  assert.equal(withRankReward(same, "Gold", { cosmetics: ["frame.a"] }, published).rankRewards?.Gold.revision, 3);
});

test("the review lists changed values and the keys a running season freezes", () => {
  const structural = ["divisionLp", "resetTierDrop", "seasonResetTable"];
  const after = withMode(withMode({ ...defaults, leaderboardTopSize: 300 }, "1v1", { mmrK: 30, divisionLp: 120 }), "2v2", withResetRow(mode, divisionRanks, "Gold II", "Silver III"));
  assert.deepEqual(ratingChanges(defaults, after), [
    { field: "leaderboardTopSize", before: "200", after: "300" },
    { field: "1v1.divisionLp", before: "100", after: "120" },
    { field: "1v1.mmrK", before: "20", after: "30" },
    { field: "2v2.seasonResetTable.Gold II", before: "—", after: "Silver III" },
  ]);
  assert.deepEqual(changedStructural(defaults, after, structural, divisionRanks), ["1v1.divisionLp", "2v2.seasonResetTable"]);
  const dropped = withMode(defaults, "1v1", { resetTierDrop: 2 });
  assert.deepEqual(changedStructural(defaults, dropped, structural, divisionRanks), ["1v1.resetTierDrop", "1v1.seasonResetTable"]);
});

test("a version lists coins, regions and rank rewards it changed by path", () => {
  const rewarded = withMode(defaults, "1v1", withRankReward(mode, "Gold", { cosmetics: ["frame.a"] }, undefined));
  const before = withRegion(rewarded, "caucasus", ["AZ"]);
  const after = withRegion({ ...withMode(before, "1v1", withRankReward(before.modes["1v1"], "Gold", { cosmetics: ["frame.a", "frame.b"] }, before.modes["1v1"])), rankRewardCoins: { Iron: 100, Gold: 800 } }, "caucasus", ["AZ", "GE"]);
  assert.deepEqual(ratingChanges(before, after), [
    { field: "rankRewardCoins.Gold", before: "700", after: "800" },
    { field: "leaderboardRegions.caucasus", before: '["AZ"]', after: '["AZ","GE"]' },
    { field: "1v1.rankRewards.Gold.revision", before: "1", after: "2" },
    { field: "1v1.rankRewards.Gold.cosmetics", before: '["frame.a"]', after: '["frame.a","frame.b"]' },
  ]);
  assert.deepEqual(ratingChanges(after, withoutRegion(after, "caucasus")), [{ field: "leaderboardRegions.caucasus", before: '["AZ","GE"]', after: "—" }]);
});
