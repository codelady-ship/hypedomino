import { sameContent } from "../config/sameContent.ts";
import type { MessageKey } from "../i18n/translate.ts";

export type CalendarDay = { coins: number };

export type Calendar = { schema: number; days: CalendarDay[] };

export function withDayCoins(calendar: Calendar, index: number, coins: number): Calendar {
  return { ...calendar, days: calendar.days.map((day, at) => (at === index ? { coins } : day)) };
}

export function cycleCoins(calendar: Calendar): number {
  return calendar.days.reduce((total, day) => total + day.coins, 0);
}

export function calendarIssues(calendar: Calendar): MessageKey[] {
  return calendar.days.every((day) => Number.isInteger(day.coins) && day.coins >= 1) ? [] : ["daily_login.issue.coins"];
}

export function sameCalendar(a: Calendar, b: Calendar): boolean {
  return sameContent(a.days, b.days);
}

export function changedDays(before: Calendar, after: Calendar): number[] {
  return after.days.flatMap((day, at) => (before.days[at]?.coins === day.coins ? [] : [at + 1]));
}
