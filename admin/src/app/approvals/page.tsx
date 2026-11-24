import Link from "next/link";
import { adminApi } from "@/admin/api";
import { googleClientId } from "@/admin/google";
import { currentAdmin } from "@/admin/session";
import { changeSummary, objectHref, ownProposal, type Proposal } from "@/approvals/proposals";
import { serverTranslator } from "@/i18n/server";
import { Notice } from "../players/Notice";
import { Decision } from "./Decision";

export const dynamic = "force-dynamic";

export default async function ApprovalsPage() {
  const t = await serverTranslator();
  const admin = await currentAdmin();
  if (!admin) {
    return <Notice text={t("auth.required")} />;
  }
  const reply = await adminApi<{ proposals: Proposal[] }>("GET", "/admin/approvals");
  if (reply.status !== 200) {
    return <Notice text={t("approvals.load_failed")} />;
  }
  const proposals = reply.body.proposals;
  return (
    <main className="mx-auto flex max-w-6xl flex-col gap-8 px-6 py-10">
      <div className="flex flex-col gap-2">
        <h1 className="text-2xl font-semibold">{t("approvals.title")}</h1>
        <p className="text-sm text-text-secondary">{t("approvals.explain")}</p>
      </div>
      {proposals.length === 0 ? (
        <p className="text-text-secondary">{t("approvals.none")}</p>
      ) : (
        <table className="text-sm">
          <thead className="text-left text-text-muted">
            <tr>
              <th className="py-2 pr-4">{t("approvals.kind")}</th>
              <th className="py-2 pr-4">{t("approvals.object")}</th>
              <th className="py-2 pr-4">{t("approvals.change")}</th>
              <th className="py-2 pr-4">{t("approvals.reason")}</th>
              <th className="py-2 pr-4">{t("approvals.proposer")}</th>
              <th className="py-2 pr-4">{t("approvals.expires_at")}</th>
              <th className="py-2 pr-4" />
            </tr>
          </thead>
          <tbody>
            {proposals.map((proposal) => {
              const href = objectHref(proposal);
              return (
                <tr key={proposal.id} className="border-t border-surface align-top">
                  <td className="py-2 pr-4 font-mono">{proposal.kind}</td>
                  <td className="py-2 pr-4 break-all">
                    {href ? (
                      <Link href={href} prefetch={false} className="text-primary underline">
                        {proposal.object_id}
                      </Link>
                    ) : (
                      `${proposal.object_type} ${proposal.object_id}`
                    )}
                  </td>
                  <td className="py-2 pr-4">{changeSummary(proposal.change)}</td>
                  <td className="py-2 pr-4">{proposal.reason}</td>
                  <td className="py-2 pr-4 break-all">
                    {proposal.proposer_id} ({proposal.proposer_roles.join(", ")})
                  </td>
                  <td className="py-2 pr-4 whitespace-nowrap">{proposal.kind === "break_glass" ? t("approvals.no_expiry") : t.when(proposal.expires_at)}</td>
                  <td className="py-2 pr-4">
                    {ownProposal(proposal, admin.account.id) ? (
                      <span className="text-text-muted">{t("approvals.own")}</span>
                    ) : (
                      <Decision id={proposal.id} clientId={googleClientId()} />
                    )}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      )}
    </main>
  );
}
