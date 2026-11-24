import assert from "node:assert/strict";
import { test } from "node:test";
import { batchLeads, bundleOf, codeGrants, detailPath, draftFromTemplate, templatePath, templateRequest, type PromoTemplate, keepsRedeemers, targetsPath, disablePath, draftReady, emptyDraft, listOf, olderPromoHref, promoApiPath, promoChangeSummary, promoPageSize, readPromoQuery, promoRequest, shownState, type PromoCode, type PromoDraft } from "./codes.ts";

const draft: PromoDraft = {
  ...emptyDraft,
  id: " promo.launch ",
  campaign: "campaign.launch_week",
  bundleId: "bundle.launch",
  coins: 300,
  cosmetics: "avatar.comet_fox, avatar.comet_fox table.night_market",
  startMinute: "2026-11-05T09:00",
  endMinute: "2026-11-12T09:00",
  cap: 50,
  code: " hype 2345 ",
};

test("lists split on commas and whitespace and drop repeats", () => {
  assert.deepEqual(listOf(" a, b\nc;a "), ["a", "b", "c"]);
  assert.deepEqual(listOf("  "), []);
});

test("the bundle carries only what it grants", () => {
  assert.deepEqual(bundleOf(draft), { id: "bundle.launch", revision: 1, coins: 300, cosmetics: ["avatar.comet_fox", "table.night_market"] });
  assert.deepEqual(bundleOf({ ...draft, coins: 0, cosmetics: "", passXp: 250 }), { id: "bundle.launch", revision: 1, passXp: 250 });
});

test("a multi_use request sends its cap and chosen code in UTC", () => {
  const request = promoRequest(draft, false);
  assert.equal(request.id, "promo.launch");
  assert.equal(request.start, "2026-11-05T09:00:00Z");
  assert.equal(request.end, "2026-11-12T09:00:00Z");
  assert.equal(request.cap, 50);
  assert.equal(request.code, "hype 2345");
});

test("single_use, targeted and batch requests leave out what the server sets", () => {
  const single = promoRequest({ ...draft, kind: "single_use" }, false);
  assert.equal(single.cap, undefined);
  assert.equal(single.code, undefined);
  const targeted = promoRequest({ ...draft, kind: "targeted" }, false);
  assert.equal(targeted.cap, undefined);
  assert.equal("accounts" in targeted, false);
  const batch = promoRequest({ ...draft, count: 500 }, true);
  assert.equal(batch.kind, "single_use");
  assert.equal(batch.count, 500);
  assert.equal(batch.cap, undefined);
  assert.equal(batch.code, undefined);
});

test("a premium Hype Pass draft asks for the premium track on every kind", () => {
  assert.equal("passPremium" in promoRequest(draft, false), false);
  assert.equal(promoRequest({ ...draft, passPremium: true }, false).passPremium, true);
  assert.equal(promoRequest({ ...draft, passPremium: true, count: 500 }, true).passPremium, true);
});

test("a draft is ready with a grant, an ordered window and its kind's fields", () => {
  assert.equal(draftReady(draft, false), true);
  assert.equal(draftReady({ ...draft, coins: 0, cosmetics: "" }, false), false);
  assert.equal(draftReady({ ...draft, endMinute: draft.startMinute }, false), false);
  assert.equal(draftReady({ ...draft, cap: 0 }, false), false);
  assert.equal(draftReady({ ...draft, kind: "targeted" }, false), true);
  assert.equal(draftReady({ ...draft, count: 10001 }, true), false);
  assert.equal(draftReady({ ...draft, count: 10000 }, true), true);
});

const code: PromoCode = {
  id: "promo.launch", campaign: "campaign.launch_week", kind: "multi_use", bundle: { id: "bundle.launch", revision: 1, coins: 300 },
  start: "2026-11-05T09:00:00Z", end: "2026-11-12T09:00:00Z", cap: 50, state: "active", creator: "a", redemptions: 3,
};

test("an active code past its end shows as expired", () => {
  assert.equal(shownState(code, new Date("2026-11-06T00:00:00Z")), "active");
  assert.equal(shownState(code, new Date("2026-11-12T09:00:00Z")), "expired");
  assert.equal(shownState({ ...code, state: "disabled" }, new Date("2026-12-01T00:00:00Z")), "disabled");
});

test("a premium Hype Pass code names the premium track beside its bundle", () => {
  assert.equal(codeGrants(code, "premium Hype Pass"), "300 coins");
  assert.equal(codeGrants({ ...code, passPremium: true }, "premium Hype Pass"), "300 coins + premium Hype Pass");
  assert.equal(promoChangeSummary({ count: 1, code: { ...code, passPremium: true } }), "multi_use; campaign.launch_week; 300 coins + premium Hype Pass; 2026-11-05T09:00:00Z to 2026-11-12T09:00:00Z");
});

test("the list query keeps the campaign, a known state, the id search and a well-formed cursor", () => {
  assert.deepEqual(readPromoQuery({ campaign: " campaign.a ", state: "pending_approval", id: " promo.gift_ ", before: "promo.gift_2" }), { campaign: "campaign.a", state: "pending_approval", id: "promo.gift_", before: "promo.gift_2" });
  assert.deepEqual(readPromoQuery({ state: "expired", before: "promo.x' or 1=1" }), { campaign: "", state: "", id: "", before: "" });
  assert.deepEqual(readPromoQuery({ state: ["disabled", "active"] }), { campaign: "", state: "disabled", id: "", before: "" });
});

test("the API path always asks for one page and the older link follows the last code", () => {
  const query = { campaign: "campaign.a&b", state: "active", id: "", before: "" } as const;
  assert.equal(promoApiPath(query), `/admin/promo-codes?campaign=campaign.a%26b&state=active&limit=${promoPageSize}`);
  const page = Array.from({ length: promoPageSize }, (_, at) => ({ ...code, id: `promo.c_${at}` }));
  assert.equal(olderPromoHref(query, page), `/promo?campaign=campaign.a%26b&state=active&before=promo.c_${promoPageSize - 1}`);
  assert.equal(olderPromoHref(query, page.slice(1)), null);
});

test("paths encode the id", () => {
  assert.equal(disablePath("promo.x/../y", false), "/admin/promo-codes/promo.x%2F..%2Fy/disable");
  assert.equal(disablePath("promo.gift", true), "/admin/promo-batches/promo.gift/disable");
});

test("a promo proposal reads as count, campaign, bundle and window", () => {
  assert.equal(promoChangeSummary({ batch: true, count: 3, code }), "3 single_use codes; campaign.launch_week; 300 coins; 2026-11-05T09:00:00Z to 2026-11-12T09:00:00Z");
  assert.equal(promoChangeSummary({ batch: false, count: 1, code }), "multi_use; campaign.launch_week; 300 coins; 2026-11-05T09:00:00Z to 2026-11-12T09:00:00Z");
  assert.equal(promoChangeSummary({ coins: 5 }), null);
});

test("each batch offers its disable once, on its first listed code", () => {
  const listed = [{ ...code, id: "promo.solo" }, { ...code, id: "promo.gift_2", batch: "promo.gift" }, { ...code, id: "promo.gift_1", batch: "promo.gift" }, { ...code, id: "promo.mass_1", batch: "promo.mass" }];
  assert.deepEqual([...batchLeads(listed)], ["promo.gift_2", "promo.mass_1"]);
});

test("detail and account paths encode the id", () => {
  assert.equal(detailPath("promo.vip"), "/admin/promo-codes/promo.vip");
  assert.equal(targetsPath("promo.a/b"), "/admin/promo-codes/promo.a%2Fb/accounts");
});

test("a new account list keeps every account that redeemed", () => {
  const redeemed = [{ account_id: "a1", redeemed_at: "2026-11-04T10:00:00Z" }];
  assert.equal(keepsRedeemers(["a1", "a2"], redeemed), true);
  assert.equal(keepsRedeemers(["a2"], redeemed), false);
  assert.equal(keepsRedeemers([], []), true);
});

test("a template keeps only id, campaign, kind and what it grants", () => {
  assert.deepEqual(templateRequest(draft), { id: "promo.launch", campaign: "campaign.launch_week", kind: "multi_use", bundle: { id: "bundle.launch", revision: 1, coins: 300, cosmetics: ["avatar.comet_fox", "table.night_market"] } });
  assert.equal(templateRequest({ ...draft, coins: 0, cosmetics: "" }).bundle, null);
});

test("cloning a template fills the code form and leaves the window, cap and code to set", () => {
  const template: PromoTemplate = { id: "promo.welcome", campaign: "campaign.launch", kind: "multi_use", bundle: { id: "bundle.welcome", revision: 1, coins: 200, cosmetics: ["avatar.comet_fox"] }, creator: "leyla", updated: "2026-11-04T10:00:00Z" };
  const clone = draftFromTemplate(template);
  assert.deepEqual(clone, { ...emptyDraft, id: "promo.welcome", campaign: "campaign.launch", kind: "multi_use", bundleId: "bundle.welcome", coins: 200, cosmetics: "avatar.comet_fox", passXp: 0 });
  assert.equal(draftReady(clone, false), false);
  assert.deepEqual(draftFromTemplate({ ...template, kind: "single_use", bundle: null }), { ...emptyDraft, id: "promo.welcome", campaign: "campaign.launch", kind: "single_use" });
  assert.equal(templatePath("promo.welcome"), "/admin/promo-templates/promo.welcome");
});
