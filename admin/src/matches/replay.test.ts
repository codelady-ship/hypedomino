import assert from "node:assert/strict";
import { test } from "node:test";
import { dealtOrder, handTiles, matchHref, replayApiPath, seatNames, teamScore, type Participant } from "./replay.ts";

test("paths escape the match id", () => {
  assert.equal(replayApiPath("5f0c/7a"), "/admin/matches/5f0c%2F7a/replay");
  assert.equal(matchHref("5f0c7a52"), "/matches/5f0c7a52");
});

test("a seat reads as its letter and who sat there", () => {
  const participants: Participant[] = [
    { seat: "X", team: "A", account_id: "a-1", nickname: "Kamal", bot: false, result: "win" },
    { seat: "Y", team: "B", account_id: "a-2", nickname: null, bot: false, result: "loss" },
    { seat: "N", team: "A", account_id: null, nickname: null, bot: true, result: null },
  ];
  assert.deepEqual([...seatNames(participants)], [["X", "X Kamal"], ["Y", "Y a-2"], ["N", "N bot"]]);
});

test("scores and hands read in one line", () => {
  assert.equal(teamScore([150, 90]), "A 150 : B 90");
  assert.equal(handTiles(["0-1", "6-6"]), "0-1  6-6");
});

test("hands are listed in seat order, unseated hands last", () => {
  const hand = { number: 1, dealt: { E: ["0-1"], N: ["1-1"], S: ["2-2"], W: ["3-3"] }, steps: [], outcome: null };
  const seats = ["N", "E", "S"].map((seat): Participant => ({ seat, team: "A", account_id: null, nickname: null, bot: true, result: null }));
  assert.deepEqual(dealtOrder(hand, seats), ["N", "E", "S", "W"]);
});
