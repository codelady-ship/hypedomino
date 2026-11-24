import assert from "node:assert/strict";
import test from "node:test";
import {
  addedItems,
  addItem,
  changedItems,
  emojiGroups,
  emptyFilter,
  filteredItems,
  itemIssues,
  manifestChanges,
  missingDefaults,
  moveLayer,
  newItem,
  newlyDisabled,
  removeItem,
  validSlug,
  withAnimated,
  withBulkChange,
  withNewSource,
  withoutSource,
  withSource,
  type CosmeticItem,
  type CosmeticRules,
} from "./catalog.ts";

const rules: CosmeticRules = {
  types: ["avatar", "frame", "table", "tile", "emoji"],
  rarities: ["common", "rare"],
  availability: ["live", "retired", "disabled"],
  source_types: ["default", "shop", "pass"],
  art_sources: ["handmade", "ai_assisted"],
  builtin: { schemaVersion: 1, manifestVersion: 1, items: [] },
};

const ready = (item: CosmeticItem): CosmeticItem => ({ ...item, culturalReview: "approved", sources: [{ type: "shop", ref: "offer.fox" }] });

const fox = ready(newItem("avatar", "fox", rules));

test("a slug takes lowercase letters, digits and underscores", () => {
  assert.equal(validSlug("comet_fox2"), true);
  assert.equal(validSlug("Comet-Fox"), false);
  assert.equal(validSlug(""), false);
});

test("a new item is unreviewed, has a source to fill and leaves the server fields out", () => {
  const item = newItem("avatar", "fox", rules);
  assert.equal(item.id, "avatar.fox");
  assert.equal(item.animated, false);
  assert.deepEqual(item.sources, [{ type: "shop", ref: "" }]);
  assert.equal(item.nameKey, undefined);
  assert.equal(item.staticAddress, undefined);
  assert.deepEqual(itemIssues(item), ["cosmetics.issue.ref", "cosmetics.issue.review"]);
  assert.deepEqual(itemIssues(fox), []);
});

test("a new item is born static and an emoji with a group to fill", () => {
  assert.equal(newItem("avatar", "fox", rules).animated, false);
  assert.deepEqual(itemIssues(ready(newItem("emoji", "wink", rules))), ["cosmetics.issue.group"]);
  assert.deepEqual(itemIssues({ ...ready(newItem("emoji", "wink", rules)), group: "faces" }), []);
});

test("a taken id is refused", () => {
  const items = addItem([], fox);
  assert.deepEqual(items, [fox]);
  assert.equal(addItem([fox], fox), null);
  assert.deepEqual(removeItem([fox], fox.id), []);
});

test("turning animation off drops the type fields the schema forbids on a static item", () => {
  const avatar: CosmeticItem = { ...fox, animated: true, animationAddress: "a" };
  assert.equal(withAnimated(avatar, false).animationAddress, undefined);
});

test("an animated table needs named, unique layers with one animated", () => {
  const table = withAnimated(ready(newItem("table", "reef", rules)), true);
  assert.deepEqual(table.layers, []);
  assert.deepEqual(itemIssues(table), ["cosmetics.issue.layer_animated"]);
  assert.deepEqual(itemIssues({ ...table, layers: [{ name: "back", animated: false }] }), ["cosmetics.issue.layer_animated"]);
  assert.deepEqual(itemIssues({ ...table, layers: [{ name: "Back!", animated: true }] }), ["cosmetics.issue.layer_name"]);
  assert.deepEqual(itemIssues({ ...table, layers: [{ name: "back", animated: true }, { name: "back", animated: false }] }), ["cosmetics.issue.layer_name"]);
  assert.deepEqual(itemIssues({ ...table, layers: [{ name: "back", animated: true }] }), []);
  assert.equal(withAnimated({ ...table, layers: [{ name: "back", animated: true }] }, false).layers, undefined);
});

test("layers keep their order and stop at the ends", () => {
  const layers = [{ name: "a", animated: true }, { name: "b", animated: false }];
  assert.deepEqual(moveLayer(layers, 0, 1).map((layer) => layer.name), ["b", "a"]);
  assert.equal(moveLayer(layers, 0, -1), layers);
  assert.equal(moveLayer(layers, 1, 1), layers);
});

test("sources check their ref, a default has none and repeats are refused", () => {
  const added = withNewSource(fox, rules);
  assert.deepEqual(added.sources[1], { type: "shop", ref: "" });
  assert.deepEqual(itemIssues(added), ["cosmetics.issue.ref"]);
  assert.deepEqual(itemIssues(withSource(added, 1, { type: "shop", ref: "offer.fox" })), ["cosmetics.issue.duplicate_source"]);
  assert.deepEqual(itemIssues(withSource(fox, 0, { type: "default", ref: "left" })).length, 0);
  assert.deepEqual(withSource(fox, 0, { type: "default", ref: "left" }).sources, [{ type: "default" }]);
  assert.deepEqual(itemIssues(withoutSource(fox, 0)), ["cosmetics.issue.no_source"]);
});

test("a default item stays live and static", () => {
  const starter = withSource(fox, 0, { type: "default" });
  assert.deepEqual(itemIssues({ ...starter, animated: true }), ["cosmetics.issue.default_item"]);
  assert.deepEqual(itemIssues({ ...starter, availability: "retired" }), ["cosmetics.issue.default_item"]);
});

test("every type keeps a live default item", () => {
  const starter = (type: string): CosmeticItem => withSource(ready(newItem(type, "starter", rules)), 0, { type: "default" });
  const all = rules.types.map(starter);
  assert.deepEqual(missingDefaults(all, rules.types), []);
  assert.deepEqual(missingDefaults(all.slice(1), rules.types), ["avatar"]);
  assert.deepEqual(missingDefaults([{ ...all[0], availability: "retired" }, ...all.slice(1)], rules.types), ["avatar"]);
});

test("the list filters by every field and searches ids and source refs", () => {
  const frame: CosmeticItem = { ...ready(newItem("frame", "gold", rules)), rarityId: "rare", availability: "retired", sources: [{ type: "pass", ref: "pass.s1.free.level_3" }] };
  const items = [fox, frame];
  const ids = (change: object) => filteredItems(items, { ...emptyFilter, ...change }).map((item) => item.id);
  assert.deepEqual(ids({}), ["avatar.fox", "frame.gold"]);
  assert.deepEqual(ids({ type: "frame" }), ["frame.gold"]);
  assert.deepEqual(ids({ rarityId: "common" }), ["avatar.fox"]);
  assert.deepEqual(ids({ availability: "retired" }), ["frame.gold"]);
  assert.deepEqual(ids({ sourceType: "shop" }), ["avatar.fox"]);
  assert.deepEqual(ids({ query: " FOX " }), ["avatar.fox"]);
  assert.deepEqual(ids({ query: "level_3" }), ["frame.gold"]);
});

test("a bulk change touches only the chosen items and the chosen fields", () => {
  const frame = ready(newItem("frame", "gold", rules));
  const [first, second] = withBulkChange([fox, frame], new Set([frame.id]), { availability: "retired" });
  assert.equal(first, fox);
  assert.deepEqual(second, { ...frame, availability: "retired" });
});

test("the review sees added, changed and newly disabled items whatever the key order", () => {
  const published: CosmeticItem = { ...fox, revision: 3 };
  const reordered = JSON.parse(JSON.stringify({ revision: 3, ...Object.fromEntries(Object.entries(fox).reverse()) })) as CosmeticItem;
  const frame = ready(newItem("frame", "gold", rules));
  assert.deepEqual(changedItems([published], [reordered]), []);
  assert.deepEqual(addedItems([published], [reordered, frame]), ["frame.gold"]);
  assert.deepEqual(changedItems([published], [{ ...published, rarityId: "rare" }]), ["avatar.fox"]);
  assert.deepEqual(newlyDisabled([published], [{ ...published, availability: "disabled" }]), ["avatar.fox"]);
  assert.deepEqual(newlyDisabled([{ ...published, availability: "disabled" }], [{ ...published, availability: "disabled" }]), []);
});

test("an emoji is offered the groups the other emoji use, once each and sorted", () => {
  const emoji = (slug: string, group: string) => ({ ...ready(newItem("emoji", slug, rules)), group });
  const items = [fox, emoji("wave", "hands"), emoji("smile", "faces"), emoji("clap", "hands"), emoji("draft", "")];
  assert.deepEqual(emojiGroups(items, "emoji.draft"), ["faces", "hands"]);
  assert.deepEqual(emojiGroups(items, "emoji.smile"), ["hands"]);
  assert.deepEqual(emojiGroups([fox], "emoji.wave"), []);
});

test("a version lists each added item and each changed field against the version before, revisions aside", () => {
  const published: CosmeticItem = { ...fox, revision: 3 };
  const frame = ready(newItem("frame", "gold", rules));
  const before = { schemaVersion: 1, manifestVersion: 4, items: [published] };
  const after = { schemaVersion: 1, manifestVersion: 5, items: [{ ...published, revision: 4, rarityId: "rare" }, frame] };
  assert.deepEqual(manifestChanges(before, after), {
    settings: [],
    entries: [
      { id: "avatar.fox", change: "changed", fields: [{ field: "rarityId", before: fox.rarityId, after: "rare" }] },
      { id: "frame.gold", change: "added", fields: [] },
    ],
  });
  assert.deepEqual(manifestChanges(before, { ...before, manifestVersion: 5, items: [{ ...published, revision: 4 }] }), { settings: [], entries: [] });
});
