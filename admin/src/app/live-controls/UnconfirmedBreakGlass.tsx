import Link from "next/link";
import type { Proposal } from "@/approvals/proposals";
import { breakGlassOf } from "@/config/liveControls";
import type { Translate } from "@/i18n/translate";

export function UnconfirmedBreakGlass({ proposals, t }: { proposals: Proposal[]; t: Translate }) {
  if (proposals.length === 0) {
    return <p className="text-sm text-text-secondary">{t("live.break_glass_none")}</p>;
  }
  return (
    <ul className="flex flex-col gap-2 text-sm">
      {proposals.map((proposal) => {
        const glass = breakGlassOf(proposal.change);
        return (
          <li key={proposal.id} role="alert" className="rounded-md bg-surface p-3">
            <span className="font-mono">{glass ? `${glass.incident.severity} ${glass.incident.id}` : proposal.id}</span>
            {glass && ` · ${t("live.version")} ${glass.config_version}`} · {proposal.reason} · {proposal.proposer_id} · {t.when(proposal.proposed_at)} ·{" "}
            <Link href="/approvals" prefetch={false} className="text-primary underline">
              {t("live.break_glass_confirm")}
            </Link>
          </li>
        );
      })}
    </ul>
  );
}
