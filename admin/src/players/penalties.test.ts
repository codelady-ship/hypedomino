import assert from "node:assert/strict";
import { test } from "node:test";
import { penaltyRunning, type LadderView } from "./penalties.ts";

const clean: LadderView = { level: 0, step: "none", next_step_level: 1, queue_delay_seconds: 0, delay_matches_left: 0, ban_until: null, under_review: false, clean_streak: 0, clean_matches_to_forgive: 10, offenses: [] };

test("a ban, a review or a queue delay on either ladder can be lifted", () => {
  assert.equal(penaltyRunning({ ranked: clean, casual: clean }), false);
  assert.equal(penaltyRunning({ ranked: { ...clean, ban_until: "2026-11-10T12:00:00Z" }, casual: clean }), true);
  assert.equal(penaltyRunning({ ranked: clean, casual: { ...clean, delay_matches_left: 2 } }), true);
  assert.equal(penaltyRunning({ ranked: { ...clean, under_review: true }, casual: clean }), true);
});
