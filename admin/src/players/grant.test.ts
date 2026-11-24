import assert from "node:assert/strict";
import { test } from "node:test";
import { emptyGrant, grantChangeSummary, grantReady, grantRequest, type GrantDraft } from "./grant.ts";

const draft: GrantDraft = { ...emptyGrant, sourceRef: " event.spring_cup ", bundleId: " bundle.spring_cup ", coins: 300, cosmetics: "avatar.comet_fox, emoji.victory_dance avatar.comet_fox" };

test("a grant sends only what it grants under the chosen source", () => {
  assert.deepEqual(grantRequest(draft), {
    bundle: { id: "bundle.spring_cup", revision: 1, coins: 300, cosmetics: ["avatar.comet_fox", "emoji.victory_dance"] },
    source: { type: "event", ref: "event.spring_cup" },
  });
  assert.deepEqual(grantRequest({ ...draft, coins: 0, cosmetics: "", medals: "medal.first_win", passXp: 250 }).bundle, { id: "bundle.spring_cup", revision: 1, passXp: 250, medals: ["medal.first_win"] });
});

test("a grant needs a source ref, something to grant and no negative count", () => {
  assert.equal(grantReady(draft), true);
  assert.equal(grantReady({ ...draft, sourceRef: " " }), false);
  assert.equal(grantReady({ ...draft, coins: 0, cosmetics: "" }), false);
  assert.equal(grantReady({ ...draft, passXp: -5 }), false);
  assert.equal(grantReady({ ...draft, coins: Number.NaN }), false);
});

test("a bundle proposal reads as bundle, grants and source", () => {
  assert.equal(grantChangeSummary({ bundle: { id: "bundle.launch_week", revision: 1, coins: 2500, cosmetics: ["table.night_market"] }, source: { type: "promo", ref: "campaign.launch_week" }, request_id: "r", proposer_id: "p" }), "bundle.launch_week: 2500 coins + table.night_market; promo:campaign.launch_week");
  assert.equal(grantChangeSummary({ coins: 2500, request_id: "r" }), null);
});
