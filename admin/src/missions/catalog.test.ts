import assert from "node:assert/strict";
import test from "node:test";
import {
  addMission,
  chainCandidates,
  emptyCatalog,
  idList,
  issues,
  missionChanges,
  newMission,
  replaceMission,
  validSlug,
  withBundle,
  withFilter,
  withNext,
  withType,
  type MissionCatalog,
  type MissionRules,
} from "./catalog.ts";

const rules: MissionRules = { types: ["daily", "weekly", "achievement", "event"], events: ["match_completed", "match_won"], queues: ["ranked", "casual"], modes: ["1v1", "2v2"] };

const catalog: MissionCatalog = {
  schema: 1,
  dailySlots: [{ coins: 30, xp: 100 }, { coins: 40, xp: 150 }],
  weeklySlots: [{ coins: 150, xp: 600 }],
  missions: [
    { id: "mission.win_50", revision: 1, type: "achievement", nameKey: "mission.win_50.name", event: "match_won", target: 50, filters: {}, bundle: { id: "bundle.mission_win_50", revision: 1, coins: 200, medals: ["medal.wins_50"] }, availability: "live" },
    { id: "mission.play_3", revision: 1, type: "daily", nameKey: "mission.play_3.name", event: "match_completed", target: 3, filters: {}, slot: 1, weight: 1, availability: "live" },
  ],
};

test("a new mission gets the fields of its type and nothing else", () => {
  const daily = newMission("hands_3", "daily", rules);
  assert.deepEqual(daily, { id: "mission.hands_3", revision: 0, type: "daily", nameKey: "mission.hands_3.name", event: "match_completed", target: 1, filters: {}, availability: "live", slot: 1, weight: 1 });
  const achievement = newMission("first_win", "achievement", rules);
  assert.deepEqual(achievement.bundle, { id: "bundle.mission_first_win", revision: 1, coins: 0, cosmetics: [], medals: [], passXp: 0 });
  assert.equal(achievement.slot, undefined);
  assert.equal(newMission("cup", "event", rules).eventId, "");
});

test("changing the type drops the fields the new type does not take", () => {
  const event = withType({ ...catalog.missions[0], next: "mission.win_250" }, "event");
  assert.equal(event.next, undefined);
  assert.equal(event.eventId, "");
  assert.equal(event.bundle?.coins, 200);
  const daily = withType(event, "daily");
  assert.equal(daily.bundle, undefined);
  assert.equal(daily.eventId, undefined);
  assert.equal(daily.slot, 1);
  assert.equal(withType({ ...event, bundle: { ...event.bundle!, passXp: 300 } }, "achievement").bundle?.passXp, 0);
});

test("filters and the chain link are removed when left empty", () => {
  const filtered = withFilter(withFilter(catalog.missions[1], "queue", "ranked"), "mode", "2v2");
  assert.deepEqual(filtered.filters, { queue: "ranked", mode: "2v2" });
  assert.deepEqual(withFilter(filtered, "queue", "").filters, { mode: "2v2" });
  const linked = withNext(catalog.missions[0], "mission.win_250");
  assert.equal(linked.next, "mission.win_250");
  assert.equal("next" in withNext(linked, ""), false);
});

test("a bundle edit raises its revision once against the published bundle", () => {
  const published = catalog.missions[0];
  const edited = withBundle(published, { coins: 250 }, published);
  assert.equal(edited.bundle?.revision, 2);
  assert.equal(withBundle(edited, { coins: 300 }, published).bundle?.revision, 2);
  assert.equal(withBundle(edited, { coins: 200 }, published).bundle?.revision, 1);
  assert.equal(withBundle(newMission("x", "achievement", rules), { coins: 5 }, undefined).bundle?.revision, 1);
  assert.deepEqual(idList(" medal.a, medal.b medal.a,,"), ["medal.a", "medal.b"]);
});

test("missions are added once and replaced by id", () => {
  assert.equal(addMission(catalog, newMission("win_50", "daily", rules)), null);
  const added = addMission(catalog, newMission("hands_3", "daily", rules));
  assert.equal(added?.missions.length, 3);
  const replaced = replaceMission(catalog, { ...catalog.missions[1], target: 5 });
  assert.equal(replaced.missions[1].target, 5);
  assert.deepEqual(chainCandidates(catalog, catalog.missions[1]), ["mission.win_50"]);
  assert.deepEqual(chainCandidates(catalog, catalog.missions[0]), []);
  assert.ok(validSlug("win_5") && !validSlug("Win-5") && !validSlug(""));
});

test("issues name what the server would refuse", () => {
  assert.deepEqual(issues(catalog, catalog.missions[0]), []);
  assert.deepEqual(issues(catalog, { ...catalog.missions[1], target: 0, slot: 3, weight: 0 }), ["missions.issue.target", "missions.issue.slot", "missions.issue.weight"]);
  assert.deepEqual(issues(emptyCatalog(), { ...catalog.missions[1], slot: 7 }), []);
  const empty = newMission("cup", "event", rules);
  assert.deepEqual(issues(catalog, empty), ["missions.issue.bundle", "missions.issue.event_id"]);
  assert.deepEqual(issues(catalog, { ...withBundle(empty, { coins: 10 }, undefined), eventId: "cup.spring" }), []);
});

test("history lists each field a version changed, per mission and in the slots", () => {
  const raised = { ...catalog.missions[0], target: 60, filters: { queue: "ranked" } };
  const next = { ...replaceMission(addMission(catalog, newMission("hands_3", "daily", rules))!, raised), weeklySlots: [{ coins: 160, xp: 600 }] };
  assert.deepEqual(missionChanges(catalog, next), {
    settings: [{ field: "weeklySlots", before: '[{"coins":150,"xp":600}]', after: '[{"coins":160,"xp":600}]' }],
    entries: [
      { id: "mission.win_50", change: "changed", fields: [{ field: "target", before: "50", after: "60" }, { field: "filters.queue", before: "—", after: "ranked" }] },
      { id: "mission.hands_3", change: "added", fields: [] },
    ],
  });
  assert.deepEqual(missionChanges(catalog, catalog), { settings: [], entries: [] });
});

test("the oldest listed version shows its missions as added", () => {
  assert.deepEqual(missionChanges(null, catalog), {
    settings: [],
    entries: [
      { id: "mission.win_50", change: "added", fields: [] },
      { id: "mission.play_3", change: "added", fields: [] },
    ],
  });
});
