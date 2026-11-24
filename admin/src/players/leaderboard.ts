import type { Season, SeasonMode } from "@/ranked/seasons";

export const boardModes: readonly SeasonMode[] = ["1v1", "2v2"];

export function boardSeasons(seasons: readonly Season[]): string[] {
  return seasons
    .filter((season) => season.state !== "scheduled")
    .map((season) => season.id)
    .reverse();
}
