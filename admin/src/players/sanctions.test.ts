import assert from "node:assert/strict";
import { test } from "node:test";
import { en } from "../i18n/messages/en.ts";
import { az } from "../i18n/messages/az.ts";
import { banCategories, banCategoryKeys, runningRankedBan, sanctionKindKeys, sanctionRunning, type Sanction } from "./sanctions.ts";

const now = new Date("2026-11-22T12:00:00Z");

function sanction(change: Partial<Sanction>): Sanction {
  return { id: 1, kind: "ranked_ban", category: "other", starts_at: "2026-11-22T00:00:00Z", ends_at: "2026-11-23T00:00:00Z", lifted_at: null, ...change };
}

test("a sanction runs until it ends or is lifted, a permanent one until it is lifted", () => {
  const cases: [Partial<Sanction>, boolean][] = [
    [{}, true],
    [{ ends_at: "2026-11-22T12:00:00Z" }, false],
    [{ ends_at: "2026-11-21T00:00:00Z" }, false],
    [{ lifted_at: "2026-11-22T06:00:00Z" }, false],
    [{ kind: "permanent_ban", ends_at: null }, true],
    [{ kind: "permanent_ban", ends_at: null, lifted_at: "2026-11-22T06:00:00Z" }, false],
  ];
  for (const [change, running] of cases) {
    assert.equal(sanctionRunning(sanction(change), now), running, JSON.stringify(change));
  }
});

test("the running ranked ban is the newest running one of its kind", () => {
  const lifted = sanction({ id: 3, lifted_at: "2026-11-22T06:00:00Z" });
  const temporary = sanction({ id: 2, kind: "temporary_ban" });
  const ranked = sanction({ id: 1, category: "collusion" });
  assert.equal(runningRankedBan([lifted, temporary, ranked], now)?.id, 1);
  assert.equal(runningRankedBan([lifted, temporary], now), null);
});

test("every ban category and sanction kind has a label in both languages", () => {
  assert.deepEqual([...banCategories], ["cheating", "collusion", "afk_leaving", "offensive_name", "payment_fraud", "other"]);
  for (const key of [...Object.values(banCategoryKeys), ...Object.values(sanctionKindKeys)]) {
    assert.ok(en[key], key);
    assert.ok(az[key], key);
    assert.notEqual(en[key], az[key], key);
  }
});
