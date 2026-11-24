export type TeamPoints = [number, number];

export type ReplayStep = { seat: string; kind: string; tile?: string; end?: string; points?: number };

export type ReplayHand = {
  number: number;
  dealt: Record<string, string[]>;
  steps: ReplayStep[];
  outcome: { pips: TeamPoints; awarded: TeamPoints; scores: TeamPoints; carry_over: number; winner: string | null } | null;
};

export type Participant = { seat: string; team: string; account_id: string | null; nickname: string | null; bot: boolean; result: string | null };

export type MatchReplay = {
  match_id: string;
  started_at: string;
  ended_at: string | null;
  mode: string;
  kind: string;
  config_version: number;
  hands_played: number;
  score_a: number;
  score_b: number;
  winner_team: string | null;
  cancelled: boolean;
  participants: Participant[];
  timeline: { hands: ReplayHand[]; problem?: string } | null;
};

export function replayApiPath(id: string): string {
  return `/admin/matches/${encodeURIComponent(id)}/replay`;
}

export function matchHref(id: string): string {
  return `/matches/${encodeURIComponent(id)}`;
}

export function seatNames(participants: readonly Participant[]): Map<string, string> {
  return new Map(participants.map((participant) => [participant.seat, `${participant.seat} ${participant.nickname ?? participant.account_id ?? "bot"}`]));
}

export function teamScore(points: TeamPoints): string {
  return `A ${points[0]} : B ${points[1]}`;
}

export function handTiles(tiles: readonly string[]): string {
  return tiles.join("  ");
}

export function dealtOrder(hand: ReplayHand, participants: readonly Participant[]): string[] {
  const seated = participants.map((participant) => participant.seat).filter((seat) => seat in hand.dealt);
  return [...seated, ...Object.keys(hand.dealt).filter((seat) => !seated.includes(seat))];
}
