import assert from "node:assert/strict";
import { test } from "node:test";
import { breakGlassOf, buildsFromText, emptyEnforcedAllowlist, maintenanceEnd, maintenanceInput, sameControls, unpublishedControls, validIncidentId, validVersion } from "./liveControls.ts";

test("an incident id is non-empty, unpadded, short and printable like the API asks", () => {
  assert.equal(validIncidentId("INC-12"), true);
  assert.equal(validIncidentId("a".repeat(128)), true);
  assert.equal(validIncidentId(""), false);
  assert.equal(validIncidentId(" INC-12"), false);
  assert.equal(validIncidentId("INC-12\n"), false);
  assert.equal(validIncidentId("a".repeat(129)), false);
});

test("a break-glass change reads its incident and version and anything else reads as none", () => {
  const change = { incident: { severity: "SEV1", id: "INC-12" }, config_version: 4, controls: {} };
  assert.deepEqual(breakGlassOf(change), { incident: { severity: "SEV1", id: "INC-12" }, config_version: 4 });
  assert.equal(breakGlassOf({ coins: 5 }), null);
  assert.equal(breakGlassOf(null), null);
});

test("a build list drops blanks, spaces and repeats", () => {
  assert.deepEqual(buildsFromText(" abc \n\nabc\ndef\n"), ["abc", "def"]);
  assert.deepEqual(buildsFromText("  \n"), []);
});

test("only major.minor.patch is a version", () => {
  assert.ok(validVersion("1.4.0"));
  assert.ok(!validVersion(""));
  assert.ok(!validVersion("1.4"));
  assert.ok(!validVersion("latest"));
});

test("an enforced allowlist cannot be empty", () => {
  assert.ok(emptyEnforcedAllowlist({ ...unpublishedControls, build_allowlist_enforced: true }));
  assert.ok(!emptyEnforcedAllowlist({ ...unpublishedControls, build_allowlist_enforced: true, build_allowlist: ["abc"] }));
  assert.ok(!emptyEnforcedAllowlist(unpublishedControls));
});

test("a change in any control counts as a change", () => {
  assert.ok(sameControls(unpublishedControls, { ...unpublishedControls }));
  assert.ok(!sameControls(unpublishedControls, { ...unpublishedControls, purchases_disabled: true }));
  assert.ok(!sameControls(unpublishedControls, { ...unpublishedControls, ranked_disabled: true }));
  assert.ok(!sameControls(unpublishedControls, { ...unpublishedControls, build_allowlist: ["abc"] }));
  assert.ok(!sameControls(unpublishedControls, { ...unpublishedControls, maintenance_ends_at: "2026-11-09T06:00:00Z" }));
  assert.ok(sameControls({ ...unpublishedControls, maintenance_ends_at: undefined as unknown as null }, unpublishedControls));
});

test("the maintenance end is a UTC minute and an empty field turns it off", () => {
  assert.equal(maintenanceInput("2026-11-09T06:30:00Z"), "2026-11-09T06:30");
  assert.equal(maintenanceInput(null), "");
  assert.equal(maintenanceInput(undefined), "");
  assert.equal(maintenanceEnd("2026-11-09T06:30"), "2026-11-09T06:30:00Z");
  assert.equal(maintenanceEnd(""), null);
});
