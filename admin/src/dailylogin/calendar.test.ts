import assert from "node:assert/strict";
import test from "node:test";
import { calendarIssues, changedDays, cycleCoins, sameCalendar, withDayCoins, type Calendar } from "./calendar.ts";

const week: Calendar = { schema: 1, days: [30, 30, 50, 50, 70, 70, 150].map((coins) => ({ coins })) };

test("editing a day changes only that day", () => {
  const edited = withDayCoins(week, 6, 200);
  assert.equal(edited.days[6].coins, 200);
  assert.equal(edited.days[0], week.days[0]);
  assert.equal(week.days[6].coins, 150);
});

test("the cycle total adds every day", () => {
  assert.equal(cycleCoins(week), 450);
  assert.equal(cycleCoins(withDayCoins(week, 6, 200)), 500);
});

test("every day grants whole coins from one", () => {
  assert.deepEqual(calendarIssues(week), []);
  for (const coins of [0, -5, 1.5, Number.NaN]) {
    assert.deepEqual(calendarIssues(withDayCoins(week, 0, coins)), ["daily_login.issue.coins"]);
  }
});

test("the changed days are numbered from one", () => {
  assert.deepEqual(changedDays(week, week), []);
  assert.deepEqual(changedDays(week, withDayCoins(withDayCoins(week, 0, 40), 6, 200)), [1, 7]);
});

test("two calendars match on their days whatever else differs", () => {
  assert.equal(sameCalendar(week, { schema: 2, days: week.days }), true);
  assert.equal(sameCalendar(week, withDayCoins(week, 0, 40)), false);
});
