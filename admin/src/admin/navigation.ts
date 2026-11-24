import type { MessageKey } from "@/i18n/translate";

export type Permissions = Readonly<Record<string, readonly string[] | undefined>>;

export type Screen = { href: string; labelKey: MessageKey; object: string | null };

export const screens: readonly Screen[] = [
  { href: "/cosmetics", labelKey: "nav.cosmetics", object: "cosmetic_items" },
  { href: "/assets", labelKey: "nav.assets", object: "assets" },
  { href: "/rewards", labelKey: "nav.rewards", object: "reward_bundles" },
  { href: "/shop", labelKey: "nav.shop", object: "shop_offers" },
  { href: "/store-prices", labelKey: "nav.store_prices", object: "store_products" },
  { href: "/promo", labelKey: "nav.promo", object: "promo_codes" },
  { href: "/pass", labelKey: "nav.pass", object: "pass_seasons" },
  { href: "/missions", labelKey: "nav.missions", object: "missions" },
  { href: "/daily-login", labelKey: "nav.daily_login", object: "missions" },
  { href: "/medals", labelKey: "nav.medals", object: "medals" },
  { href: "/ranked", labelKey: "nav.ranked", object: "ranked_seasons" },
  { href: "/players", labelKey: "nav.players", object: "players" },
  { href: "/reviews", labelKey: "nav.reviews", object: "queues" },
  { href: "/reports", labelKey: "nav.reports", object: "queues" },
  { href: "/appeals", labelKey: "nav.appeals", object: "queues" },
  { href: "/matches", labelKey: "nav.suspicious_matches", object: "players" },
  { href: "/config/rules", labelKey: "nav.rules", object: "game_config" },
  { href: "/config/matchmaking", labelKey: "nav.matchmaking", object: "game_config" },
  { href: "/config/penalties", labelKey: "nav.penalties", object: "game_config" },
  { href: "/config/liveops", labelKey: "nav.liveops", object: "game_config" },
  { href: "/calendar", labelKey: "nav.calendar", object: "live_events" },
  { href: "/strings", labelKey: "nav.strings", object: "game_strings" },
  { href: "/analytics", labelKey: "nav.analytics", object: "dashboards" },
  { href: "/analytics/kpis", labelKey: "nav.kpis", object: "dashboards" },
  { href: "/quick-messages", labelKey: "nav.quick_messages", object: "game_strings" },
  { href: "/moderation", labelKey: "nav.moderation", object: "moderation_config" },
  { href: "/live-controls", labelKey: "nav.live", object: "live_controls" },
  { href: "/approvals", labelKey: "nav.approvals", object: null },
  { href: "/audit", labelKey: "nav.audit", object: "audit_log" },
  { href: "/accounts", labelKey: "nav.accounts", object: "admin_accounts" },
];

export function can(permissions: Permissions, object: string, action: string): boolean {
  return permissions[object]?.includes(action) ?? false;
}

export function canView(permissions: Permissions, object: string): boolean {
  return can(permissions, object, "view");
}

export function canRollBack(permissions: Permissions, object: string): boolean {
  return can(permissions, object, "publish") && (can(permissions, "snapshots", "rollback") || can(permissions, "snapshots", "rollback_own_domains"));
}

export function visibleScreens(permissions: Permissions, all: readonly Screen[] = screens): Screen[] {
  return all.filter((screen) => screen.object === null || canView(permissions, screen.object));
}
