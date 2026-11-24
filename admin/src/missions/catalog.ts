import type { MessageKey } from "@/i18n/translate";
import type { Bundle } from "@/shop/catalog";
import { entryChanges, fieldChanges, type CatalogChanges } from "../config/changes.ts";

export type MissionType = "daily" | "weekly" | "achievement" | "event";

export type Availability = "live" | "retired";

export type Filters = { queue?: string; mode?: string };

export type Mission = {
  id: string;
  revision: number;
  type: MissionType;
  nameKey: string;
  event: string;
  target: number;
  filters: Filters;
  slot?: number;
  weight?: number;
  bundle?: Bundle;
  next?: string;
  eventId?: string;
  availability: Availability;
};

export type SlotReward = { coins: number; xp: number };

export type MissionCatalog = {
  schema: number;
  dailySlots?: SlotReward[];
  weeklySlots?: SlotReward[];
  missions: Mission[];
  [field: string]: unknown;
};

export type MissionRules = { types: MissionType[]; events: string[]; queues: string[]; modes: string[] };

export const availabilities: readonly Availability[] = ["live", "retired"];

const slugPattern = /^[a-z0-9_]+$/;
const eventIdPattern = /^[a-z0-9_.]+$/;

export function emptyCatalog(): MissionCatalog {
  return { schema: 1, missions: [] };
}

export function slugOf(id: string): string {
  return id.replace(/^mission\./, "");
}

export function validSlug(slug: string): boolean {
  return slugPattern.test(slug);
}

export function periodic(type: MissionType): boolean {
  return type === "daily" || type === "weekly";
}

export function newMission(slug: string, type: MissionType, rules: MissionRules): Mission {
  const id = `mission.${slug}`;
  return withType({ id, revision: 0, type, nameKey: `${id}.name`, event: rules.events[0] ?? "", target: 1, filters: {}, availability: "live" }, type);
}

export function withType(mission: Mission, type: MissionType): Mission {
  const { slot, weight, bundle, next, eventId, ...rest } = mission;
  if (periodic(type)) {
    return { ...rest, type, slot: slot ?? 1, weight: weight ?? 1 };
  }
  const base = bundle ?? { id: `bundle.mission_${slugOf(mission.id)}`, revision: 1, coins: 0, cosmetics: [], medals: [] };
  const rewarded = { ...rest, type, bundle: type === "achievement" ? { ...base, passXp: 0 } : base };
  if (type === "achievement") {
    return next ? { ...rewarded, next } : rewarded;
  }
  return { ...rewarded, eventId: eventId ?? "" };
}

export function withFilter(mission: Mission, key: keyof Filters, value: string): Mission {
  const filters = { ...mission.filters };
  if (value) {
    filters[key] = value;
  } else {
    delete filters[key];
  }
  return { ...mission, filters };
}

export function withNext(mission: Mission, next: string): Mission {
  const { next: _previous, ...rest } = mission;
  return next ? { ...rest, next } : rest;
}

export function idList(text: string): string[] {
  return [...new Set(text.split(/[\s,]+/).filter(Boolean))];
}

export function withBundle(mission: Mission, change: Partial<Bundle>, published: Mission | undefined): Mission {
  if (!mission.bundle) {
    return mission;
  }
  const next = { ...mission.bundle, ...change };
  return { ...mission, bundle: { ...next, revision: bundleRevision(published?.bundle, next) } };
}

function bundleContent(bundle: Bundle): string {
  return JSON.stringify({ ...bundle, revision: 0, cosmetics: bundle.cosmetics ?? [], medals: bundle.medals ?? [], passXp: bundle.passXp ?? 0 });
}

function bundleRevision(published: Bundle | undefined, next: Bundle): number {
  if (!published) {
    return 1;
  }
  return bundleContent(published) === bundleContent(next) ? published.revision : published.revision + 1;
}

export function replaceMission(catalog: MissionCatalog, mission: Mission): MissionCatalog {
  return { ...catalog, missions: catalog.missions.map((candidate) => (candidate.id === mission.id ? mission : candidate)) };
}

export function addMission(catalog: MissionCatalog, mission: Mission): MissionCatalog | null {
  if (catalog.missions.some((candidate) => candidate.id === mission.id)) {
    return null;
  }
  return { ...catalog, missions: [...catalog.missions, mission] };
}

export function chainCandidates(catalog: MissionCatalog, mission: Mission): string[] {
  return catalog.missions.filter((candidate) => candidate.type === "achievement" && candidate.id !== mission.id).map((candidate) => candidate.id);
}

export function slotCount(catalog: MissionCatalog, type: MissionType): number | null {
  const slots = type === "daily" ? catalog.dailySlots : type === "weekly" ? catalog.weeklySlots : undefined;
  return slots ? slots.length : null;
}

export function issues(catalog: MissionCatalog, mission: Mission): MessageKey[] {
  const found: MessageKey[] = [];
  if (!Number.isInteger(mission.target) || mission.target < 1) found.push("missions.issue.target");
  if (periodic(mission.type)) {
    const slots = slotCount(catalog, mission.type);
    if (!Number.isInteger(mission.slot) || (mission.slot ?? 0) < 1 || (slots !== null && (mission.slot ?? 0) > slots)) found.push("missions.issue.slot");
    if (!Number.isInteger(mission.weight) || (mission.weight ?? 0) < 1) found.push("missions.issue.weight");
    return found;
  }
  const bundle = mission.bundle;
  if (!bundle || ((bundle.coins ?? 0) <= 0 && (bundle.passXp ?? 0) <= 0 && !(bundle.cosmetics ?? []).length && !(bundle.medals ?? []).length)) {
    found.push("missions.issue.bundle");
  }
  if (mission.type === "event" && !eventIdPattern.test(mission.eventId ?? "")) found.push("missions.issue.event_id");
  return found;
}

export function missionChanges(before: MissionCatalog | null, after: MissionCatalog): CatalogChanges {
  const { missions, ...settings } = after;
  if (!before) {
    return { settings: [], entries: entryChanges([], missions) };
  }
  const { missions: previous, ...previousSettings } = before;
  return { settings: fieldChanges(previousSettings, settings), entries: entryChanges(previous, missions) };
}
