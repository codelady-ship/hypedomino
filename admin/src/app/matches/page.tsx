import Link from "next/link";
import { adminApi } from "@/admin/api";
import { canView } from "@/admin/navigation";
import { currentAdmin } from "@/admin/session";
import type { SearchParams } from "@/audit/query";
import { serverTranslator } from "@/i18n/server";
import { matchHref } from "@/matches/replay";
import { nextQueueHref, queueApiPath, readSuspiciousQuery, suspiciousSignals } from "@/reviews/reviews";
import { Notice } from "../players/Notice";
import { QueueFilter } from "../reviews/QueueFilter";

export const dynamic = "force-dynamic";

type SuspiciousMatch = { match_id: string; started_at: string; mode: string; kind: string; signals: string[]; accounts: string[] };

export default async function SuspiciousMatchesPage({ searchParams }: { searchParams: Promise<SearchParams> }) {
  const t = await serverTranslator();
  const admin = await currentAdmin();
  if (!admin) {
    return <Notice text={t("auth.required")} />;
  }
  if (!canView(admin.permissions, "players")) {
    return <Notice text={t("auth.forbidden")} />;
  }
  const query = readSuspiciousQuery(await searchParams);
  const reply = await adminApi<{ matches: SuspiciousMatch[] }>("GET", queueApiPath("/admin/matches/suspicious", query));
  if (reply.status !== 200) {
    return <Notice text={t("suspicious.load_failed")} />;
  }
  const matches = reply.body.matches;
  const next = nextQueueHref("/matches", query, matches.map((match) => match.match_id));
  const filtered = Boolean(query.signal || query.account || query.after);
  return (
    <main className="mx-auto flex max-w-6xl flex-col gap-8 px-6 py-10">
      <div className="flex flex-col gap-2">
        <h1 className="text-2xl font-semibold">{t("suspicious.title")}</h1>
        <p className="text-sm text-text-secondary">{t("suspicious.explain")}</p>
      </div>
      <QueueFilter screen="/matches" name="signal" label="player.suspicious" options={suspiciousSignals} selected={query.signal} account={query.account} t={t} />
      {matches.length === 0 ? (
        <p className="text-text-secondary">{t(filtered ? "queues.empty_match" : "suspicious.none")}</p>
      ) : (
        <table className="text-sm">
          <thead className="text-left text-text-muted">
            <tr>
              <th className="py-1 pr-4">{t("player.at")}</th>
              <th className="py-1 pr-4">{t("player.mode")}</th>
              <th className="py-1 pr-4">{t("player.suspicious")}</th>
              <th className="py-1 pr-4">{t("suspicious.accounts")}</th>
              <th className="py-1 pr-4">{t("match.open_replay")}</th>
            </tr>
          </thead>
          <tbody>
            {matches.map((match) => (
              <tr key={`${match.match_id}-${match.started_at}`} className="border-t border-surface align-top">
                <td className="py-1 pr-4 whitespace-nowrap">{t.when(match.started_at)}</td>
                <td className="py-1 pr-4">{`${match.mode} ${match.kind}`}</td>
                <td className="py-1 pr-4 font-mono">{match.signals.join(", ")}</td>
                <td className="py-1 pr-4">
                  <div className="flex flex-col">
                    {match.accounts.map((account) => (
                      <Link key={account} href={`/players/${encodeURIComponent(account)}`} prefetch={false} className="font-mono text-primary underline">
                        {account}
                      </Link>
                    ))}
                  </div>
                </td>
                <td className="py-1 pr-4">
                  <Link href={matchHref(match.match_id)} prefetch={false} className="text-primary underline">
                    {t("match.open_replay")}
                  </Link>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {next && (
        <Link href={next} prefetch={false} className="self-start rounded-md bg-surface px-4 py-2 text-text-secondary">
          {t("queues.next")}
        </Link>
      )}
    </main>
  );
}
