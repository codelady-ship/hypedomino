import assert from "node:assert/strict";
import { test } from "node:test";
import { appealKinds, decidedReports, evidenceSummary, nextQueueHref, queueApiPath, queuePageSize, readAppealQuery, readReportQuery, readReviewQuery, readSuspiciousQuery, signalWeight } from "./reviews.ts";

test("the listed signals add up to their weight", () => {
  assert.equal(signalWeight([]), 0);
  const signal = { kind: "refund", evidence: {}, at: "2026-11-05T00:00:00Z" };
  assert.equal(signalWeight([{ ...signal, weight: 3 }, { ...signal, weight: 20 }]), 23);
});

test("evidence reads as its fields", () => {
  assert.equal(evidenceSummary({ reason: "token_stale", count: 5 }), "reason: token_stale; count: 5");
  assert.equal(evidenceSummary({}), "");
  assert.equal(evidenceSummary(null), "null");
  assert.equal(evidenceSummary(["a"]), '["a"]');
});

test("the queue queries keep a known cause or reason, the account search and a well-formed cursor", () => {
  const account = "5f2c9d3e-6d4a-4f4e-9d11-1a2b3c4d5e6f";
  assert.deepEqual(readReviewQuery({ cause: "score", account: " 5f2c ", after: "42" }), { cause: "score", account: "5f2c", after: "42" });
  assert.deepEqual(readReviewQuery({ cause: "toString", after: "0" }), { cause: "", account: "", after: "" });
  assert.deepEqual(readReportQuery({ reason: ["impersonation", "other"], after: account }), { reason: "impersonation", account: "", after: account });
  assert.deepEqual(readReportQuery({ reason: "spam", after: "42" }), { reason: "", account: "", after: "" });
  assert.deepEqual(readSuspiciousQuery({ signal: "feeding", account: "5f2c", after: account }), { signal: "feeding", account: "5f2c", after: account });
  assert.deepEqual(readSuspiciousQuery({ signal: "pair_frequency", after: "42" }), { signal: "", account: "", after: "" });
  assert.equal(queueApiPath("/admin/matches/suspicious", { signal: "win_trade", account: "", after: account }), `/admin/matches/suspicious?signal=win_trade&after=${account}&limit=${queuePageSize}`);
});

test("the API path always asks for one page and the next link follows the last entry", () => {
  const query = { cause: "hard_signal", account: "", after: "" } as const;
  assert.equal(queueApiPath("/admin/reviews", query), `/admin/reviews?cause=hard_signal&limit=${queuePageSize}`);
  const cursors = Array.from({ length: queuePageSize }, (_, at) => String(at + 1));
  assert.equal(nextQueueHref("/reviews", query, cursors), `/reviews?cause=hard_signal&after=${queuePageSize}`);
  assert.equal(nextQueueHref("/reports", { reason: "", account: "ab", after: "" }, cursors.slice(1)), null);
});

test("earlier report decisions add up the reports they upheld and rejected; review decisions are not reports", () => {
  assert.deepEqual(decidedReports([]), { upheld: 0, rejected: 0 });
  const at = "2026-11-05T00:00:00Z";
  assert.deepEqual(
    decidedReports([
      { queue: "reports", outcome: "rejected", reasons: ["impersonation", "offensive_nickname"], reports: 2, decided_at: at },
      { queue: "reviews", outcome: "confirm", reasons: ["hard_signal"], reports: 0, decided_at: at },
      { queue: "reports", outcome: "upheld", reasons: ["other"], reports: 1, decided_at: at },
      { queue: "reports", outcome: "rejected", reasons: ["other"], reports: 3, decided_at: at },
    ]),
    { upheld: 1, rejected: 5 },
  );
});

test("the appeal queue keeps an appealable sanction kind, the account search and a numeric cursor", () => {
  assert.deepEqual(readAppealQuery({ kind: "forced_rename", account: " 5f2c ", after: "17" }), { kind: "forced_rename", account: "5f2c", after: "17" });
  assert.deepEqual(readAppealQuery({ kind: "shadowban", after: "abc" }), { kind: "", account: "", after: "" });
  assert.deepEqual([...appealKinds], ["temporary_ban", "permanent_ban", "ranked_ban", "penalty_ban", "forced_rename"]);
  const query = { kind: "ranked_ban", account: "", after: "" } as const;
  assert.equal(queueApiPath("/admin/appeals", query), `/admin/appeals?kind=ranked_ban&limit=${queuePageSize}`);
  const cursors = Array.from({ length: queuePageSize }, (_, at) => String(at + 1));
  assert.equal(nextQueueHref("/appeals", query, cursors), `/appeals?kind=ranked_ban&after=${queuePageSize}`);
  assert.equal(nextQueueHref("/appeals", query, cursors.slice(1)), null);
});
