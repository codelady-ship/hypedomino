import assert from "node:assert/strict";
import { test } from "node:test";
import type { Season } from "../ranked/seasons.ts";
import { boardSeasons } from "./leaderboard.ts";

const season = (id: string, state: Season["state"]): Season => ({ id, name_key: `${id}.name`, start: "2026-09-01T00:00:00Z", end: "2026-12-01T00:00:00Z", state });

test("a player can be removed from the board of a started season, newest first", () => {
  const seasons = [season("season.one", "closed"), season("season.two", "ending"), season("season.three", "live"), season("season.four", "scheduled")];
  assert.deepEqual(boardSeasons(seasons), ["season.three", "season.two", "season.one"]);
  assert.deepEqual(boardSeasons([season("season.four", "scheduled")]), []);
});
