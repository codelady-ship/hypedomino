import assert from "node:assert/strict";
import { test } from "node:test";
import { running, type CalendarEntry } from "./calendar.ts";

test("an entry runs from its start", () => {
  const entry: CalendarEntry = { kind: "ranked_season", id: "season.one", start: "2026-11-08T12:00:00Z", end: null };
  assert.equal(running(entry, new Date("2026-11-08T11:59:59Z")), false);
  assert.equal(running(entry, new Date("2026-11-08T12:00:00Z")), true);
});
