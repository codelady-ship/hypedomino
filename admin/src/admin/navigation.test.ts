import assert from "node:assert/strict";
import { test } from "node:test";
import { adminApiHeaders, parseReply } from "./forwarding.ts";
import { canRollBack, canView, visibleScreens, type Screen } from "./navigation.ts";
import { needsStepUp, signInRefusalKey, stepUpRefusalKey } from "./refusals.ts";

const shop: Screen = { href: "/shop", labelKey: "nav.shop", object: "shop_offers" };
const audit: Screen = { href: "/audit", labelKey: "nav.shop", object: "audit_log" };

test("an account without permissions sees no screen", () => {
  assert.deepEqual(visibleScreens({}, [shop, audit]), []);
});

test("a screen shows only with view on its object", () => {
  assert.deepEqual(visibleScreens({ shop_offers: ["view", "publish"] }, [shop, audit]), [shop]);
  assert.deepEqual(visibleScreens({ shop_offers: ["view"], audit_log: ["view"] }, [shop, audit]), [shop, audit]);
});

test("other actions without view do not show a screen", () => {
  assert.equal(canView({ shop_offers: ["publish", "edit"] }, "shop_offers"), false);
  assert.equal(canView({ audit_log: ["view"] }, "shop_offers"), false);
});

function incoming(values: Record<string, string>) {
  return { get: (name: string) => values[name] ?? null };
}

test("admin API calls carry the token, the browser address and the user agent", () => {
  const sent = adminApiHeaders(incoming({ "x-forwarded-for": "203.0.113.7", "user-agent": "Browser/1.0" }), "token-1");
  assert.deepEqual(sent, {
    "Content-Type": "application/json",
    Authorization: "Bearer token-1",
    "X-Forwarded-For": "203.0.113.7",
    "User-Agent": "Browser/1.0",
  });
});

test("a sign-in call without a session sends no authorization", () => {
  assert.deepEqual(adminApiHeaders(incoming({}), undefined), { "Content-Type": "application/json" });
});

test("an empty reply body reads as an empty object", () => {
  assert.deepEqual(parseReply(""), {});
  assert.deepEqual(parseReply('{"error":"forbidden"}'), { error: "forbidden" });
});

test("sign-in refusals map to a message and unknown codes to the generic one", () => {
  assert.equal(signInRefusalKey("account_refused"), "signin.account_refused");
  assert.equal(signInRefusalKey("not_company_account"), "signin.not_company_account");
  assert.equal(signInRefusalKey("nonce_rejected"), "signin.retry");
  assert.equal(signInRefusalKey("token_expired"), "signin.retry");
  assert.equal(signInRefusalKey("provider_unavailable"), "signin.unavailable");
  assert.equal(signInRefusalKey("rate_limited"), "signin.rate_limited");
  assert.equal(signInRefusalKey("500"), "signin.failed");
});

test("rollback needs publish on the object and rollback on snapshots, own domains included", () => {
  assert.equal(canRollBack({ game_config: ["view", "publish"], snapshots: ["rollback_own_domains"] }, "game_config"), true);
  assert.equal(canRollBack({ game_config: ["view", "publish"], snapshots: ["rollback"] }, "game_config"), true);
  assert.equal(canRollBack({ game_config: ["view", "publish"], snapshots: ["view"] }, "game_config"), false);
  assert.equal(canRollBack({ game_config: ["view"], snapshots: ["rollback"] }, "game_config"), false);
});

test("a step-up succeeds on an empty error and names a different account apart from sign-in refusals", () => {
  assert.equal(stepUpRefusalKey(""), null);
  assert.equal(stepUpRefusalKey("different_account"), "stepup.different_account");
  assert.equal(stepUpRefusalKey("nonce_rejected"), "signin.retry");
  assert.equal(stepUpRefusalKey("session_rejected"), "signin.failed");
  assert.equal(needsStepUp("step_up_required"), true);
  assert.equal(needsStepUp("reason_required"), false);
  assert.equal(needsStepUp(undefined), false);
});

test("a screen without an object shows to every signed-in admin", () => {
  const approvals: Screen = { href: "/approvals", labelKey: "nav.approvals", object: null };
  assert.deepEqual(visibleScreens({}, [shop, approvals]), [approvals]);
});
