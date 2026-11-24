import { breakGlassOf } from "../config/liveControls.ts";
import { grantChangeSummary } from "../players/grant.ts";
import { promoChangeSummary } from "../promo/codes.ts";

export type Proposal = {
  id: number;
  kind: string;
  object_type: string;
  object_id: string;
  change: unknown;
  reason: string;
  proposer_id: string;
  proposer_roles: string[];
  proposed_at: string;
  expires_at: string;
  state: string;
};

export function ownProposal(proposal: Proposal, accountId: string): boolean {
  return proposal.proposer_id === accountId;
}

export function objectHref(proposal: Proposal): string | null {
  if (proposal.object_type === "promo_codes") {
    return "/promo";
  }
  if (proposal.object_type === "admin_accounts") {
    return "/accounts";
  }
  return proposal.object_type === "players" ? `/players/${encodeURIComponent(proposal.object_id)}` : null;
}

function breakGlassSummary(change: unknown): string | null {
  const glass = breakGlassOf(change);
  if (glass === null) {
    return null;
  }
  return `${glass.incident.severity} ${glass.incident.id}; config_version: ${glass.config_version}; controls: ${JSON.stringify((change as { controls?: unknown }).controls)}`;
}

export function changeSummary(change: unknown): string {
  const special = promoChangeSummary(change) ?? grantChangeSummary(change) ?? breakGlassSummary(change);
  if (special !== null) {
    return special;
  }
  if (change === null || typeof change !== "object" || Array.isArray(change)) {
    return JSON.stringify(change);
  }
  return Object.entries(change)
    .filter(([key]) => key !== "proposer_id" && key !== "request_id")
    .map(([key, value]) => `${key}: ${Array.isArray(value) ? value.join(", ") : JSON.stringify(value)}`)
    .join("; ");
}
