import assert from "node:assert/strict";
import test from "node:test";
import {
  canRemove,
  createIssues,
  endAfterMonths,
  extendIssues,
  rescheduleIssues,
  seasonIdFromSlug,
  seasonNameKey,
  seasonProgress,
  shiftedBy,
  startLocked,
  type Season,
} from "./seasons.ts";

const live: Season = { id: "season.one", name_key: "season.one.name", start: "2026-09-01T00:00:00Z", end: "2026-12-01T00:00:00Z", state: "live" };
const next: Season = { id: "season.two", name_key: "season.two.name", start: "2026-12-01T00:00:00Z", end: "2027-03-01T00:00:00Z", state: "scheduled" };
const last: Season = { id: "season.three", name_key: "season.three.name", start: "2027-03-01T00:00:00Z", end: "2027-06-01T00:00:00Z", state: "scheduled" };
const seasons = [live, next, last];
const now = "2026-11-06T12:00:00Z";

test("a season id is built from its slug", () => {
  assert.equal(seasonIdFromSlug("winter_2027"), "season.winter_2027");
});

test("a season name key is the one the server gives the new season", () => {
  assert.equal(seasonNameKey(seasonIdFromSlug("winter_2027")), "season.winter_2027.name");
  assert.equal(seasonNameKey(live.id), live.name_key);
});

test("the default end adds calendar months the way the server does", () => {
  assert.equal(endAfterMonths("2027-06-01T00:00:00Z", 3), "2027-09-01T00:00:00Z");
  assert.equal(endAfterMonths("2027-01-31T00:00:00Z", 1), "2027-03-03T00:00:00Z");
});

test("only the last season can be removed, and only before it starts", () => {
  assert.equal(canRemove(seasons, last), true);
  assert.equal(canRemove(seasons, next), false);
  assert.equal(canRemove([live], live), false);
});

test("the start stays locked while another season ends at it", () => {
  assert.equal(startLocked(seasons, next), true);
  assert.equal(startLocked([next], next), false);
});

test("moving an end names the seasons that move with it", () => {
  assert.deepEqual(shiftedBy(seasons, next, "2027-04-01T00:00:00Z"), ["season.three"]);
  assert.deepEqual(shiftedBy(seasons, live, "2026-12-15T00:00:00Z"), ["season.two", "season.three"]);
  assert.deepEqual(shiftedBy(seasons, last, "2027-07-01T00:00:00Z"), []);
  assert.deepEqual(shiftedBy(seasons, next, next.end), []);
});

test("a new season needs a valid id, a future start equal to the next start and a later end", () => {
  assert.deepEqual(createIssues("four", "2027-06-01T00:00:00Z", "2027-09-01T00:00:00Z", "2027-06-01T00:00:00Z", now), []);
  assert.deepEqual(createIssues("Four!", "2027-06-01T00:00:00Z", "2027-09-01T00:00:00Z", "2027-06-01T00:00:00Z", now), ["ranked.issue.season_id"]);
  assert.deepEqual(createIssues("four", "2027-07-01T00:00:00Z", "2027-09-01T00:00:00Z", "2027-06-01T00:00:00Z", now), ["ranked.issue.start_next"]);
  assert.deepEqual(createIssues("four", "2026-11-01T00:00:00Z", "2027-09-01T00:00:00Z", null, now), ["ranked.issue.start_future"]);
  assert.deepEqual(createIssues("four", "2027-06-01T00:00:00Z", "2027-06-01T00:00:00Z", null, now), ["ranked.issue.window"]);
  assert.deepEqual(createIssues("four", "", "", null, now), ["ranked.issue.window"]);
});

test("a scheduled season keeps a window that ends after it starts", () => {
  assert.deepEqual(rescheduleIssues(next, next.start, "2027-04-01T00:00:00Z", now), []);
  assert.deepEqual(rescheduleIssues(next, next.start, next.start, now), ["ranked.issue.window"]);
  assert.deepEqual(rescheduleIssues(next, "2026-10-01T00:00:00Z", next.end, now), ["ranked.issue.start_future"]);
});

test("the live season may only move later", () => {
  assert.deepEqual(extendIssues(live, "2026-12-15T00:00:00Z"), []);
  assert.deepEqual(extendIssues(live, live.end), ["ranked.issue.extend_later"]);
  assert.deepEqual(extendIssues(live, "2026-11-15T00:00:00Z"), ["ranked.issue.extend_later"]);
});

test("the end of a season shows its progress per mode", () => {
  const ending: Season = { ...live, state: "ending", closed_1v1: "2026-12-01T00:01:00Z", rewarded_1v1: "2026-12-01T00:05:00Z", closed_2v2: "2026-12-01T00:01:00Z" };
  assert.equal(seasonProgress(ending, "1v1"), "rewarded");
  assert.equal(seasonProgress(ending, "2v2"), "closed");
  assert.equal(seasonProgress(live, "1v1"), "open");
});
