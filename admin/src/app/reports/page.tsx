import Link from "next/link";
import { adminApi } from "@/admin/api";
import { googleClientId } from "@/admin/google";
import { can, canView } from "@/admin/navigation";
import { currentAdmin } from "@/admin/session";
import type { SearchParams } from "@/audit/query";
import { serverTranslator } from "@/i18n/server";
import type { Translate } from "@/i18n/translate";
import { nextQueueHref, queueApiPath, readReportQuery, reportReasons, type ReportTarget } from "@/reviews/reviews";
import { Notice } from "../players/Notice";
import { EarlierDecisions } from "../reviews/EarlierDecisions";
import { QueueDue } from "../reviews/QueueDue";
import { QueueFilter } from "../reviews/QueueFilter";
import { ReportDecision } from "./ReportDecision";

export const dynamic = "force-dynamic";

export default async function ReportsPage({ searchParams }: { searchParams: Promise<SearchParams> }) {
  const t = await serverTranslator();
  const admin = await currentAdmin();
  if (!admin) {
    return <Notice text={t("auth.required")} />;
  }
  if (!canView(admin.permissions, "queues")) {
    return <Notice text={t("auth.forbidden")} />;
  }
  const query = readReportQuery(await searchParams);
  const reply = await adminApi<{ targets: ReportTarget[] }>("GET", queueApiPath("/admin/reports", query));
  if (reply.status !== 200) {
    return <Notice text={t("reports.load_failed")} />;
  }
  const { targets } = reply.body;
  const clientId = can(admin.permissions, "queues", "act_on_queue") ? googleClientId() : null;
  const next = nextQueueHref("/reports", query, targets.map((target) => target.account_id));
  const filtered = Boolean(query.reason || query.account || query.after);
  return (
    <main className="mx-auto flex max-w-6xl flex-col gap-8 px-6 py-10">
      <div className="flex flex-col gap-2">
        <h1 className="text-2xl font-semibold">{t("reports.title")}</h1>
        <p className="text-sm text-text-secondary">{t("reports.explain")}</p>
      </div>
      <QueueFilter screen="/reports" name="reason" label="reports.reason_code" options={reportReasons} selected={query.reason} account={query.account} t={t} />
      {targets.length === 0 ? <p className="text-text-secondary">{t(filtered ? "queues.empty_match" : "reports.none")}</p> : targets.map((target) => <TargetEntry key={target.account_id} target={target} clientId={clientId} t={t} />)}
      {next && (
        <Link href={next} prefetch={false} className="self-start rounded-md bg-surface px-4 py-2 text-text-secondary">
          {t("queues.next")}
        </Link>
      )}
    </main>
  );
}

function TargetEntry({ target, clientId, t }: { target: ReportTarget; clientId: string | null; t: Translate }) {
  return (
    <section className="flex flex-col gap-3 border-t border-surface pt-4 text-sm">
      <div className="flex flex-wrap gap-x-6 gap-y-1">
        <Link href={`/players/${encodeURIComponent(target.account_id)}`} prefetch={false} className="text-primary underline">
          {target.nickname || target.account_id}
        </Link>
        <span>
          {t("reports.count")}: {target.reports.length}
        </span>
        <QueueDue deadline={target} t={t} />
        {target.rename_required && <span>{t("reports.rename_required")}</span>}
      </div>
      <table>
        <thead className="text-left text-text-muted">
          <tr>
            <th className="py-1 pr-4">{t("reports.reason_code")}</th>
            <th className="py-1 pr-4">{t("reports.name_text")}</th>
            <th className="py-1 pr-4">{t("reports.reporter")}</th>
            <th className="py-1 pr-4">{t("reports.match")}</th>
            <th className="py-1 pr-4">{t("reports.at")}</th>
          </tr>
        </thead>
        <tbody>
          {target.reports.map((entry) => (
            <tr key={entry.id} className="border-t border-surface align-top">
              <td className="py-1 pr-4 font-mono">{entry.reason}</td>
              <td className="py-1 pr-4">{entry.name_text}</td>
              <td className="py-1 pr-4">
                <Link href={`/players/${encodeURIComponent(entry.reporter)}`} prefetch={false} className="font-mono text-primary underline">
                  {entry.reporter}
                </Link>
              </td>
              <td className="py-1 pr-4">
                {entry.match_id ? (
                  <Link href={`/matches/${encodeURIComponent(entry.match_id)}`} prefetch={false} className="font-mono text-primary underline">
                    {entry.match_id}
                  </Link>
                ) : (
                  "-"
                )}
              </td>
              <td className="py-1 pr-4 whitespace-nowrap">{t.when(entry.updated_at)}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <EarlierDecisions decisions={target.decisions} t={t} />
      {clientId && <ReportDecision accountId={target.account_id} clientId={clientId} />}
    </section>
  );
}
