import assert from "node:assert/strict";
import { test } from "node:test";
import { grantsSuperAdmin, roleLabelKey, roleNames, sameRoles, toggledRole } from "./roles.ts";

test("every known role has a label and unknown roles have none", () => {
  assert.equal(roleLabelKey("live_ops"), "role.live_ops");
  assert.equal(roleLabelKey("owner"), null);
});

test("role names are translated and unknown roles stay as sent", () => {
  assert.equal(roleNames(["support", "owner"], (key) => key.toUpperCase()), "ROLE.SUPPORT, owner");
});

test("only adding super_admin counts as a grant", () => {
  assert.equal(grantsSuperAdmin(["support"], ["support", "super_admin"]), true);
  assert.equal(grantsSuperAdmin(["super_admin"], ["super_admin", "content"]), false);
  assert.equal(grantsSuperAdmin(["super_admin"], ["content"]), false);
});

test("role lists compare without order", () => {
  assert.equal(sameRoles(["content", "support"], ["support", "content"]), true);
  assert.equal(sameRoles(["content"], ["content", "support"]), false);
});

test("toggling keeps the matrix order and drops unchecked roles", () => {
  assert.deepEqual(toggledRole(["support"], "content", true), ["content", "support"]);
  assert.deepEqual(toggledRole(["content", "support"], "content", false), ["support"]);
});
