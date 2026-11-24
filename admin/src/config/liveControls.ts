export type LiveControls = {
  minimum_app_version: string;
  build_allowlist_enforced: boolean;
  build_allowlist: string[];
  purchases_disabled: boolean;
  ranked_disabled: boolean;
  maintenance_ends_at: string | null;
};

export type LiveControlsView = { config_version: number; controls: LiveControls | null };

export const unpublishedControls: LiveControls = {
  minimum_app_version: "",
  build_allowlist_enforced: false,
  build_allowlist: [],
  purchases_disabled: false,
  ranked_disabled: false,
  maintenance_ends_at: null,
};

export function buildsFromText(text: string): string[] {
  const lines = text.split("\n").map((line) => line.trim());
  return [...new Set(lines.filter((line) => line !== ""))];
}

export function validVersion(version: string): boolean {
  return /^\d+\.\d+\.\d+$/.test(version);
}

export function emptyEnforcedAllowlist(controls: LiveControls): boolean {
  return controls.build_allowlist_enforced && controls.build_allowlist.length === 0;
}

export function sameControls(left: LiveControls, right: LiveControls): boolean {
  return (
    left.minimum_app_version === right.minimum_app_version &&
    left.build_allowlist_enforced === right.build_allowlist_enforced &&
    left.purchases_disabled === right.purchases_disabled &&
    left.ranked_disabled === right.ranked_disabled &&
    (left.maintenance_ends_at ?? null) === (right.maintenance_ends_at ?? null) &&
    left.build_allowlist.join("\n") === right.build_allowlist.join("\n")
  );
}

export const breakGlassSeverities = ["SEV1", "SEV2"] as const;

export type Incident = { severity: (typeof breakGlassSeverities)[number]; id: string };

const maxIncidentIdLength = 128;

export function validIncidentId(id: string): boolean {
  return id !== "" && id === id.trim() && id.length <= maxIncidentIdLength && !/\p{Cc}/u.test(id);
}

export type BreakGlass = { incident: Incident; config_version: number };

export function breakGlassOf(change: unknown): BreakGlass | null {
  const candidate = change as Partial<BreakGlass> | null;
  if (typeof candidate?.incident?.id !== "string" || typeof candidate.config_version !== "number") {
    return null;
  }
  return { incident: candidate.incident, config_version: candidate.config_version };
}

export function maintenanceInput(endsAt: string | null | undefined): string {
  return endsAt ? endsAt.slice(0, 16) : "";
}

export function maintenanceEnd(input: string): string | null {
  return input === "" ? null : `${input}:00Z`;
}
