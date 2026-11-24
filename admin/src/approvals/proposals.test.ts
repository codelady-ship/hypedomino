import assert from "node:assert/strict";
import { test } from "node:test";
import { changeSummary, objectHref, ownProposal, type Proposal } from "./proposals.ts";

const grant: Proposal = {
  id: 7,
  kind: "player_grant",
  object_type: "players",
  object_id: "11111111-2222-3333-4444-555555555555",
  change: { coins: 5000, request_id: "r-1", proposer_id: "admin-1" },
  reason: "event compensation",
  proposer_id: "admin-1",
  proposer_roles: ["support"],
  proposed_at: "2026-11-04T09:00:00Z",
  expires_at: "2026-11-05T09:00:00Z",
  state: "pending",
};

test("a proposal is the admin's own only when they proposed it", () => {
  assert.equal(ownProposal(grant, "admin-1"), true);
  assert.equal(ownProposal(grant, "admin-2"), false);
});

test("player proposals link to the player record and others link nowhere", () => {
  assert.equal(objectHref(grant), "/players/11111111-2222-3333-4444-555555555555");
  assert.equal(objectHref({ ...grant, object_type: "admin_accounts" }), "/accounts");
  assert.equal(objectHref({ ...grant, object_type: "live_controls" }), null);
});

test("the change summary shows what changes and hides bookkeeping ids", () => {
  assert.equal(changeSummary(grant.change), "coins: 5000");
  assert.equal(changeSummary({ roles: ["super_admin", "content"] }), "roles: super_admin, content");
  assert.equal(changeSummary("raw"), '"raw"');
  assert.equal(
    changeSummary({ incident: { severity: "SEV1", id: "INC-12" }, config_version: 4, controls: { purchases_disabled: true } }),
    'SEV1 INC-12; config_version: 4; controls: {"purchases_disabled":true}',
  );
});

test("a promo proposal links to the promo screen and reads as its code", () => {
  const promo: Proposal = {
    ...grant,
    kind: "promo_code",
    object_type: "promo_codes",
    object_id: "promo.gift",
    change: { batch: true, count: 3, code: { id: "promo.gift", campaign: "campaign.launch_week", kind: "single_use", bundle: { id: "bundle.gift", revision: 1, coins: 2500 }, start: "s", end: "e", cap: 1, state: "pending_approval", creator: "a", redemptions: 0 } },
  };
  assert.equal(objectHref(promo), "/promo");
  assert.equal(changeSummary(promo.change), "3 single_use codes; campaign.launch_week; 2500 coins; s to e");
});
