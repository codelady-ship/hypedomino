import type { MessageKey } from "@/i18n/translate";

export type CalendarKind = "ranked_season" | "pass_season" | "shop_offer" | "shop_rotation" | "promo_campaign";

export type CalendarEntry = { kind: CalendarKind; id: string; start: string; end: string | null };

export type CalendarView = { server_time: string; entries: CalendarEntry[] };

export const calendarKinds: Readonly<Record<CalendarKind, { labelKey: MessageKey; href: string }>> = {
  ranked_season: { labelKey: "calendar.ranked_season", href: "/ranked" },
  pass_season: { labelKey: "calendar.pass_season", href: "/pass" },
  shop_offer: { labelKey: "calendar.shop_offer", href: "/shop" },
  shop_rotation: { labelKey: "calendar.shop_rotation", href: "/shop" },
  promo_campaign: { labelKey: "calendar.promo_campaign", href: "/promo" },
};

export function running(entry: CalendarEntry, now: Date): boolean {
  return new Date(entry.start) <= now;
}
