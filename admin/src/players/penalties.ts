export type OffenseView = { match_id: string; kind: string; weight: number; at: string; expires_at: string };

export type LadderView = {
  level: number;
  step: string;
  next_step_level: number;
  queue_delay_seconds: number;
  delay_matches_left: number;
  ban_until: string | null;
  under_review: boolean;
  clean_streak: number;
  clean_matches_to_forgive: number;
  offenses: OffenseView[];
};

export type PenaltyView = { ranked: LadderView; casual: LadderView };

export const ladders = ["ranked", "casual"] as const;

export function penaltyRunning(view: PenaltyView): boolean {
  return ladders.some((queue) => view[queue].ban_until !== null || view[queue].under_review || view[queue].delay_matches_left > 0);
}
