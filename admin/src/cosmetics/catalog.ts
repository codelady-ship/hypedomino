import { entryChanges, fieldChanges, type CatalogChanges } from "../config/changes.ts";
import { sameContent } from "../config/sameContent.ts";
import type { MessageKey } from "../i18n/translate.ts";

export type ItemSource = { type: string; ref?: string };

export type ItemLayer = { name: string; animated: boolean; address?: string };

export type CosmeticItem = {
  id: string;
  type: string;
  revision: number;
  nameKey?: string;
  rarityId: string;
  animated: boolean;
  staticAddress?: string;
  availability: string;
  sources: ItemSource[];
  artSource: string;
  culturalReview: string;
  animationAddress?: string;
  layers?: ItemLayer[];
  group?: string;
};

export type CosmeticManifest = { schemaVersion: number; manifestVersion: number; items: CosmeticItem[] };

export type CosmeticRules = { types: string[]; rarities: string[]; availability: string[]; source_types: string[]; art_sources: string[]; builtin: CosmeticManifest };

export type ItemFilter = { query: string; type: string; rarityId: string; availability: string; sourceType: string };

export type BulkChange = { rarityId?: string; availability?: string };

const defaultSource = "default";
const approved = "approved";
const slugPattern = /^[a-z0-9_]+$/;
const referencePattern = /^[a-z0-9][a-z0-9_.-]*$/;

export const emptyFilter: ItemFilter = { query: "", type: "", rarityId: "", availability: "", sourceType: "" };

export function validSlug(slug: string): boolean {
  return slugPattern.test(slug);
}

export function newItem(type: string, slug: string, rules: CosmeticRules): CosmeticItem {
  const item: CosmeticItem = {
    id: `${type}.${slug}`,
    type,
    revision: 0,
    rarityId: rules.rarities[0] ?? "",
    animated: false,
    availability: rules.availability[0] ?? "",
    sources: [{ type: rules.source_types.find((candidate) => candidate !== defaultSource) ?? "", ref: "" }],
    artSource: rules.art_sources[0] ?? "",
    culturalReview: "",
  };
  return type === "emoji" ? { ...item, group: "" } : item;
}

export function addItem(items: CosmeticItem[], item: CosmeticItem): CosmeticItem[] | null {
  return items.some((candidate) => candidate.id === item.id) ? null : [...items, item];
}

export function replaceItem(items: CosmeticItem[], item: CosmeticItem): CosmeticItem[] {
  return items.map((candidate) => (candidate.id === item.id ? item : candidate));
}

export function removeItem(items: CosmeticItem[], id: string): CosmeticItem[] {
  return items.filter((candidate) => candidate.id !== id);
}

export function withAnimated(item: CosmeticItem, animated: boolean): CosmeticItem {
  const { animationAddress: _animation, layers: _layers, ...rest } = item;
  if (!animated) {
    return { ...rest, animated };
  }
  return item.type === "table" ? { ...rest, animated, layers: item.layers ?? [] } : { ...rest, animated };
}

export function withLayers(item: CosmeticItem, layers: ItemLayer[]): CosmeticItem {
  return { ...item, layers };
}

export function moveLayer(layers: ItemLayer[], index: number, by: number): ItemLayer[] {
  const target = index + by;
  if (target < 0 || target >= layers.length) {
    return layers;
  }
  const moved = [...layers];
  [moved[index], moved[target]] = [moved[target], moved[index]];
  return moved;
}

export function withSource(item: CosmeticItem, index: number, source: ItemSource): CosmeticItem {
  const next = source.type === defaultSource ? { type: defaultSource } : { type: source.type, ref: source.ref ?? "" };
  return { ...item, sources: item.sources.map((candidate, at) => (at === index ? next : candidate)) };
}

export function withoutSource(item: CosmeticItem, index: number): CosmeticItem {
  return { ...item, sources: item.sources.filter((_, at) => at !== index) };
}

export function withNewSource(item: CosmeticItem, rules: CosmeticRules): CosmeticItem {
  return { ...item, sources: [...item.sources, { type: rules.source_types.find((candidate) => candidate !== defaultSource) ?? "", ref: "" }] };
}

export function itemIssues(item: CosmeticItem): MessageKey[] {
  const found: MessageKey[] = [];
  const isDefault = item.sources.some((source) => source.type === defaultSource);
  if (item.sources.length === 0) found.push("cosmetics.issue.no_source");
  if (item.sources.some((source) => source.type !== defaultSource && !referencePattern.test(source.ref ?? ""))) found.push("cosmetics.issue.ref");
  const keys = item.sources.map((source) => `${source.type}:${source.ref ?? ""}`);
  if (new Set(keys).size !== keys.length) found.push("cosmetics.issue.duplicate_source");
  if (isDefault && (item.animated || item.availability !== "live")) found.push("cosmetics.issue.default_item");
  if (item.culturalReview !== approved) found.push("cosmetics.issue.review");
  if (item.type === "emoji" && !referencePattern.test(item.group ?? "")) found.push("cosmetics.issue.group");
  if (item.type === "table" && item.animated) found.push(...layerIssues(item.layers ?? []));
  return found;
}

function layerIssues(layers: ItemLayer[]): MessageKey[] {
  const found: MessageKey[] = [];
  const names = layers.map((layer) => layer.name);
  if (layers.length === 0 || !layers.some((layer) => layer.animated)) found.push("cosmetics.issue.layer_animated");
  if (names.some((name) => !slugPattern.test(name)) || new Set(names).size !== names.length) found.push("cosmetics.issue.layer_name");
  return found;
}

export function emojiGroups(items: CosmeticItem[], exceptId: string): string[] {
  const groups = items.filter((item) => item.type === "emoji" && item.id !== exceptId && item.group).map((item) => item.group as string);
  return [...new Set(groups)].sort();
}

export function missingDefaults(items: CosmeticItem[], types: readonly string[]): string[] {
  const covered = new Set(items.filter((item) => item.availability === "live" && item.sources.some((source) => source.type === defaultSource)).map((item) => item.type));
  return types.filter((type) => !covered.has(type));
}

export function filteredItems(items: CosmeticItem[], filter: ItemFilter): CosmeticItem[] {
  const query = filter.query.trim().toLowerCase();
  return items.filter(
    (item) =>
      (!filter.type || item.type === filter.type) &&
      (!filter.rarityId || item.rarityId === filter.rarityId) &&
      (!filter.availability || item.availability === filter.availability) &&
      (!filter.sourceType || item.sources.some((source) => source.type === filter.sourceType)) &&
      (!query || item.id.includes(query) || item.sources.some((source) => (source.ref ?? "").toLowerCase().includes(query))),
  );
}

export function withBulkChange(items: CosmeticItem[], ids: ReadonlySet<string>, change: BulkChange): CosmeticItem[] {
  return items.map((item) => (ids.has(item.id) ? { ...item, ...change } : item));
}

export function addedItems(before: CosmeticItem[], after: CosmeticItem[]): string[] {
  const known = new Set(before.map((item) => item.id));
  return after.filter((item) => !known.has(item.id)).map((item) => item.id);
}

export function changedItems(before: CosmeticItem[], after: CosmeticItem[]): string[] {
  const previous = new Map(before.map((item) => [item.id, item]));
  return after
    .filter((item) => {
      const old = previous.get(item.id);
      return old !== undefined && !sameContent({ ...old, revision: 0 }, { ...item, revision: 0 });
    })
    .map((item) => item.id);
}

function unrevised(items: CosmeticItem[]): CosmeticItem[] {
  return items.map((item) => ({ ...item, revision: 0 }));
}

export function manifestChanges(before: CosmeticManifest, after: CosmeticManifest): CatalogChanges {
  return {
    settings: fieldChanges({ schemaVersion: before.schemaVersion }, { schemaVersion: after.schemaVersion }),
    entries: entryChanges(unrevised(before.items), unrevised(after.items)),
  };
}

export function newlyDisabled(before: CosmeticItem[], after: CosmeticItem[]): string[] {
  const previous = new Map(before.map((item) => [item.id, item.availability]));
  return after.filter((item) => item.availability === "disabled" && previous.get(item.id) !== "disabled").map((item) => item.id);
}

export function isPublished(before: CosmeticItem[], id: string): boolean {
  return before.some((item) => item.id === id);
}
