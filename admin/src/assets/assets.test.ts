import assert from "node:assert/strict";
import test from "node:test";
import type { Translate } from "../i18n/translate.ts";
import { assetPageSize, assetsApiPath, failedCheckLabels, itemAssetsHref, itemIdValid, itemsWithoutArt, olderAssetsHref, readAssetQuery, sizeLabel, staticArtApiPath, staticArtOf, statusLabel, type Asset } from "./assets.ts";

const t = Object.assign((key: string) => `<${key}>`, { number: (value: number) => value.toLocaleString("en-US") }) as unknown as Translate;

const asset: Asset = {
  id: "a-1",
  item_id: "avatar.fox",
  slot: "static",
  address: "cosmetic/avatar/fox/static",
  status: "rejected",
  format: null,
  width: null,
  height: null,
  bytes: 19,
  content_hash: "abc",
  checks: { format: false, alpha: false },
  texture_bytes: null,
  texture_limit: null,
  created_at: "2026-11-06T10:00:00Z",
};

test("an item id names a known type and a slug", () => {
  for (const id of ["avatar.fox_2", "frame.gold", "table.night", "tile.storm", "emoji.wow", "medal.wins_50", "medal.points_1m"]) {
    assert.equal(itemIdValid(id), true, id);
  }
  for (const id of ["", "fox", "pet.fox", "character.ayla", "action.dance", "avatar.", "avatar.Fox", "../avatar.fox", "avatar.fox/static", "avatar.fox\n", "medal.", "medal.Wins", "medals.wins_50", "medal.wins.50", "emblem.global_top10"]) {
    assert.equal(itemIdValid(id), false, id);
  }
});

test("only failed checks are named, and an unknown check still shows", () => {
  assert.deepEqual(failedCheckLabels(asset, t), ["<assets.check.format>", "<assets.check.alpha>"]);
  assert.deepEqual(failedCheckLabels({ ...asset, checks: { format: true, alpha: false } }, t), ["<assets.check.alpha>"]);
  assert.deepEqual(failedCheckLabels({ ...asset, checks: { atlas: false } }, t), ["atlas"]);
  assert.deepEqual(failedCheckLabels({ ...asset, checks: { format: true, alpha: true } }, t), []);
  assert.deepEqual(failedCheckLabels({ ...asset, checks: null }, t), []);
  const wide = { ...asset, item_id: "frame.gold", checks: { format: true, alpha: true, size: false }, texture_bytes: 129_600, texture_limit: 125_000 };
  assert.deepEqual(failedCheckLabels(wide, t), ["<assets.check.size>: 129,600 B, <assets.check.size_limit> 125,000 B"]);
});

test("every status has its own label", () => {
  const labels = (["awaiting_file", "passed", "rejected"] as const).map((status) => statusLabel(status, t));
  assert.equal(new Set(labels).size, 3);
});

test("the list query keeps a known status, the item search and a well-formed cursor", () => {
  const before = "5f2c9d3e-6d4a-4f4e-9d11-1a2b3c4d5e6f";
  assert.deepEqual(readAssetQuery({ status: "passed", item: " avatar. ", before }), { status: "passed", item: "avatar.", before });
  assert.deepEqual(readAssetQuery({ status: "toString", before: "1 or 1=1" }), { status: "", item: "", before: "" });
  assert.deepEqual(readAssetQuery({ status: ["rejected", "passed"] }), { status: "rejected", item: "", before: "" });
});

test("the API path always asks for one page and the older link follows the last row", () => {
  const query = { status: "rejected", item: "avatar.", before: "" } as const;
  assert.equal(assetsApiPath(query), `/admin/assets?status=rejected&item=avatar.&limit=${assetPageSize}`);
  const page = Array.from({ length: assetPageSize }, (_, at) => ({ ...asset, id: `a-${at}` }));
  assert.equal(olderAssetsHref(query, page), `/assets?status=rejected&item=avatar.&before=a-${assetPageSize - 1}`);
  assert.equal(olderAssetsHref(query, page.slice(1)), null);
});

test("sizes read in the nearest unit", () => {
  assert.equal(sizeLabel(0), "0 B");
  assert.equal(sizeLabel(1023), "1023 B");
  assert.equal(sizeLabel(1536), "1.5 KB");
  assert.equal(sizeLabel(1572864), "1.5 MB");
});

test("an art refusal names the items without a passed static, each linking to its uploads", () => {
  const cosmetic = "adminapi: a cosmetic item is added only after its static upload passed its checks: avatar.fox, emoji.s1_wave";
  assert.deepEqual(itemsWithoutArt("cosmetic_art_missing", cosmetic), ["avatar.fox", "emoji.s1_wave"]);
  const medal = "adminapi: a medal is published only after its static upload passed its checks: medal.wins_50";
  assert.deepEqual(itemsWithoutArt("medal_art_missing", medal), ["medal.wins_50"]);
  assert.deepEqual(itemsWithoutArt("invalid", cosmetic), []);
  assert.deepEqual(itemsWithoutArt("cosmetic_art_missing", "avatar.fox"), []);
  assert.equal(itemAssetsHref("medal.wins_50"), "/assets?item=medal.wins_50");
});

test("the static art of an item asks for its newest passed static only", () => {
  assert.equal(staticArtApiPath("avatar.fox"), "/admin/assets?item=avatar.fox&exact=true&slot=static&status=passed&limit=1");
});

test("the static art shows the passed upload, says when there is none and hides without storage or access", () => {
  const passed = { ...asset, status: "passed" as const, width: 512, height: 512, view_url: "https://store.example/get/cosmetic/avatar/fox/static/a-1" };
  assert.deepEqual(staticArtOf(200, { store_configured: true, assets: [passed] }), {
    state: "passed",
    url: passed.view_url,
    width: 512,
    height: 512,
    uploadedAt: passed.created_at,
  });
  assert.deepEqual(staticArtOf(200, { store_configured: true, assets: [] }), { state: "missing" });
  assert.deepEqual(staticArtOf(200, { store_configured: false, assets: [passed] }), { state: "unavailable" });
  assert.deepEqual(staticArtOf(403, {}), { state: "unavailable" });
});
