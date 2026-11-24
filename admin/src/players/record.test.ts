import assert from "node:assert/strict";
import { test } from "node:test";
import { banActive, missionPlace, missionProgress, muteActive, nextPlayersHref, playerApiPath, playerPageSize, playerSearch, playersApiPath, reportSide, searchQuery, validAdjustment, type PlayerAccount } from "./record.ts";

test("the search query is the first trimmed q value", () => {
  assert.equal(searchQuery({ q: "  ayla " }), "ayla");
  assert.equal(searchQuery({ q: ["first", "second"] }), "first");
  assert.equal(searchQuery({}), "");
});

test("query and id are encoded into the admin API paths", () => {
  assert.equal(playersApiPath({ q: "ayla@example.com", state: "", after: "" }), `/admin/players?q=ayla%40example.com&limit=${playerPageSize}`);
  assert.equal(playersApiPath({ q: "a&b=c", state: "banned", after: "" }), `/admin/players?q=a%26b%3Dc&state=banned&limit=${playerPageSize}`);
  assert.equal(playersApiPath({ provider: "apple", subject: "001.a&b" }), "/admin/players?provider=apple&subject=001.a%26b");
  assert.equal(playerApiPath("../admin/me"), "/admin/players/..%2Fadmin%2Fme");
});

const account: PlayerAccount = { id: "p", nickname: "n", state: "active", country: "AZ", created_at: "2026-11-01T00:00:00Z", emails: [], banned_until: null, shadowbanned_at: null, muted_until: null, deletion_requested_at: null, deleted_at: null };

test("a ban counts only until its end", () => {
  const now = new Date("2026-11-04T10:00:00Z");
  assert.equal(banActive(account, now), false);
  assert.equal(banActive({ ...account, banned_until: "2026-11-04T12:00:00Z" }, now), true);
  assert.equal(banActive({ ...account, banned_until: "2026-11-04T09:00:00Z" }, now), false);
});

test("a mute counts only until its end", () => {
  const now = new Date("2026-11-04T10:00:00Z");
  assert.equal(muteActive(account, now), false);
  assert.equal(muteActive({ ...account, muted_until: "2026-11-05T10:00:00Z" }, now), true);
  assert.equal(muteActive({ ...account, muted_until: "2026-11-04T09:00:00Z" }, now), false);
});

test("a coin adjustment is a whole, non-zero number of coins, credit or debit", () => {
  assert.ok(validAdjustment(100));
  assert.ok(validAdjustment(-40));
  assert.ok(!validAdjustment(0));
  assert.ok(!validAdjustment(1.5));
  assert.ok(!validAdjustment(Number.NaN));
  assert.ok(!validAdjustment(2 ** 60));
});

test("a subject searches by auth identity, otherwise q searches the text", () => {
  assert.deepEqual(playerSearch({ provider: "apple", subject: " 001.abc ", q: "ayla" }), { provider: "apple", subject: "001.abc" });
  assert.deepEqual(playerSearch({ provider: "other", subject: "1234" }), { provider: "google", subject: "1234" });
  assert.deepEqual(playerSearch({ q: " ayla ", subject: " " }), { q: "ayla", state: "", after: "" });
  assert.equal(playerSearch({}), null);
});

test("the player list keeps a known state and a well-formed cursor, and pages only when full", () => {
  const id = "5f2c9d3e-6d4a-4f4e-9d11-1a2b3c4d5e6f";
  assert.deepEqual(playerSearch({ state: "banned", after: id }), { q: "", state: "banned", after: id });
  assert.equal(playerSearch({ state: "toString", after: id }), null);
  assert.deepEqual(playerSearch({ q: "ayla", after: "42" }), { q: "ayla", state: "", after: "" });
  const search = { q: "", state: "banned", after: "" } as const;
  const ids = Array.from({ length: playerPageSize }, () => id);
  assert.equal(nextPlayersHref(search, ids), `/players?state=banned&after=${id}`);
  assert.equal(nextPlayersHref(search, ids.slice(1)), null);
  assert.equal(nextPlayersHref({ provider: "google", subject: "1" }, ids), null);
});

test("mission progress shows the published target when the mission is still in the catalog", () => {
  assert.equal(missionProgress({ progress: 2, target: 3 }), "2/3");
  assert.equal(missionProgress({ progress: 30, target: 30 }), "30/30");
  assert.equal(missionProgress({ progress: 4, target: 0 }), "4");
});

test("an achievement has no slot, a daily or weekly mission shows its slot", () => {
  assert.equal(missionPlace({ type: "daily", slot: 2 }), "daily 2");
  assert.equal(missionPlace({ type: "weekly", slot: 1 }), "weekly 1");
  assert.equal(missionPlace({ type: "achievement", slot: 0 }), "achievement");
});

test("a report names the other account and whether it is about the player", () => {
  assert.deepEqual(reportSide("p", { reporter: "q", target: "p" }), { about: true, counterpart: "q" });
  assert.deepEqual(reportSide("p", { reporter: "p", target: "q" }), { about: false, counterpart: "q" });
});
