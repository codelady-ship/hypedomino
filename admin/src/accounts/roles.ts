import type { MessageKey } from "@/i18n/translate";

export const adminRoles = ["super_admin", "content", "live_ops", "support", "analyst"] as const;

export type AdminRole = (typeof adminRoles)[number];

export const roleLabels: Readonly<Record<AdminRole, MessageKey>> = {
  super_admin: "role.super_admin",
  content: "role.content",
  live_ops: "role.live_ops",
  support: "role.support",
  analyst: "role.analyst",
};

export function roleLabelKey(role: string): MessageKey | null {
  return (adminRoles as readonly string[]).includes(role) ? roleLabels[role as AdminRole] : null;
}

export function roleNames(roles: readonly string[], t: (key: MessageKey) => string): string {
  return roles.map((role) => roleName(role, t)).join(", ");
}

function roleName(role: string, t: (key: MessageKey) => string): string {
  const key = roleLabelKey(role);
  return key ? t(key) : role;
}

export function grantsSuperAdmin(before: readonly string[], after: readonly string[]): boolean {
  return after.includes("super_admin") && !before.includes("super_admin");
}

export function sameRoles(a: readonly string[], b: readonly string[]): boolean {
  return a.length === b.length && a.every((role) => b.includes(role));
}

export function toggledRole(roles: readonly string[], role: string, on: boolean): string[] {
  const rest = roles.filter((held) => held !== role);
  return on ? adminRoles.filter((known) => known === role || rest.includes(known)) : rest;
}
