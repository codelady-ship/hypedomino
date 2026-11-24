import type { MessageKey } from "@/i18n/translate";
import { entryChanges, fieldChanges, type CatalogChanges } from "../config/changes.ts";

export type MedalSource = { type: string; ref: string };

export type Medal = {
  id: string;
  revision: number;
  nameKey: string;
  descriptionKey: string;
  rarityId: string;
  staticAddress: string;
  sources: MedalSource[];
  availability: string;
};

export type MedalCatalog = {
  schema: number;
  profileMedalSlots: number;
  medals: Medal[];
  [field: string]: unknown;
};

export type MedalRules = { rarities: string[]; source_types: string[]; availability: string[] };

const slugPattern = /^[a-z0-9_]+$/;

export function emptyMedalCatalog(): MedalCatalog {
  return { schema: 1, profileMedalSlots: 0, medals: [] };
}

export function validMedalSlug(slug: string): boolean {
  return slugPattern.test(slug);
}

export function newMedal(slug: string, rules: MedalRules): Medal {
  return {
    id: `medal.${slug}`,
    revision: 0,
    nameKey: `medal.${slug}.name`,
    descriptionKey: `medal.${slug}.description`,
    rarityId: rules.rarities[0] ?? "",
    staticAddress: `medal/${slug}/static`,
    sources: [{ type: rules.source_types[0] ?? "", ref: "" }],
    availability: rules.availability[0] ?? "",
  };
}

export function medalTextKeys(catalog: MedalCatalog): string[] {
  return catalog.medals.flatMap((medal) => [medal.nameKey, medal.descriptionKey]);
}

export function addMedal(catalog: MedalCatalog, medal: Medal): MedalCatalog | null {
  if (catalog.medals.some((candidate) => candidate.id === medal.id)) {
    return null;
  }
  return { ...catalog, medals: [...catalog.medals, medal] };
}

export function replaceMedal(catalog: MedalCatalog, medal: Medal): MedalCatalog {
  return { ...catalog, medals: catalog.medals.map((candidate) => (candidate.id === medal.id ? medal : candidate)) };
}

export function withSource(medal: Medal, index: number, source: MedalSource): Medal {
  return { ...medal, sources: medal.sources.map((candidate, at) => (at === index ? source : candidate)) };
}

export function withoutSource(medal: Medal, index: number): Medal {
  return { ...medal, sources: medal.sources.filter((_, at) => at !== index) };
}

export function withNewSource(medal: Medal, rules: MedalRules): Medal {
  return { ...medal, sources: [...medal.sources, { type: rules.source_types[0] ?? "", ref: "" }] };
}

export function medalIssues(medal: Medal): MessageKey[] {
  const found: MessageKey[] = [];
  if (medal.sources.length === 0) found.push("medals.issue.no_source");
  if (medal.sources.some((source) => !source.ref.trim())) found.push("medals.issue.empty_ref");
  const keys = medal.sources.map((source) => `${source.type}:${source.ref.trim()}`);
  if (new Set(keys).size !== keys.length) found.push("medals.issue.duplicate_source");
  return found;
}

export function validShowcaseSlots(slots: number): boolean {
  return Number.isInteger(slots) && slots >= 0;
}

export function medalChanges(before: MedalCatalog | null, after: MedalCatalog): CatalogChanges {
  const { medals, ...settings } = after;
  if (!before) {
    return { settings: [], entries: entryChanges([], medals) };
  }
  const { medals: previous, ...previousSettings } = before;
  return { settings: fieldChanges(previousSettings, settings), entries: entryChanges(previous, medals) };
}
