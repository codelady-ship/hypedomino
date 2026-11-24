import Link from "next/link";
import { adminApi } from "@/admin/api";
import { googleClientId } from "@/admin/google";
import { can, canView } from "@/admin/navigation";
import { currentAdmin } from "@/admin/session";
import type { SearchParams } from "@/audit/query";
import { serverTranslator } from "@/i18n/server";
import type { Translate } from "@/i18n/translate";
import { banCategoryKeys, sanctionKindKeys, sanctionRunning } from "@/players/sanctions";
import { appealKinds, nextQueueHref, queueApiPath, readAppealQuery, type Appeal } from "@/reviews/reviews";
import { Notice } from "../players/Notice";
import { EarlierDecisions } from "../reviews/EarlierDecisions";
import { QueueDue } from "../reviews/QueueDue";
import { QueueFilter } from "../reviews/QueueFilter";
import { AppealDecision } from "./AppealDecision";

export const dynamic = "force-dynamic";

export default async function AppealsPage({ searchParams }: { searchParams: Promise<SearchParams> }) {
  const t = await serverTranslator();
  const admin = await currentAdmin();
  if (!admin) {
    return <Notice text={t("auth.required")} />;
  }
  if (!canView(admin.permissions, "queues")) {
    return <Notice text={t("auth.forbidden")} />;
  }
  const query = readAppealQuery(await searchParams);
  const reply = await adminApi<{ appeals: Appeal[] }>("GET", queueApiPath("/admin/appeals", query));
  if (reply.status !== 200) {
    return <Notice text={t("appeals.load_failed")} />;
  }
  const { appeals } = reply.body;
  const clientId = can(admin.permissions, "queues", "act_on_queue") ? googleClientId() : null;
  const next = nextQueueHref("/appeals", query, appeals.map((entry) => String(entry.id)));
  const filtered = Boolean(query.kind || query.account || query.after);
  const now = new Date();
  return (
    <main className="mx-auto flex max-w-6xl flex-col gap-8 px-6 py-10">
      <div className="flex flex-col gap-2">
        <h1 className="text-2xl font-semibold">{t("appeals.title")}</h1>
        <p className="text-sm text-text-secondary">{t("appeals.explain")}</p>
      </div>
      <QueueFilter screen="/appeals" name="kind" label="appeals.kind" options={appealKinds} selected={query.kind} account={query.account} t={t} />
      {appeals.length === 0 ? <p className="text-text-secondary">{t(filtered ? "queues.empty_match" : "appeals.none")}</p> : appeals.map((entry) => <AppealEntry key={entry.id} entry={entry} now={now} clientId={clientId} t={t} />)}
      {next && (
        <Link href={next} prefetch={false} className="self-start rounded-md bg-surface px-4 py-2 text-text-secondary">
          {t("queues.next")}
        </Link>
      )}
    </main>
  );
}

function AppealEntry({ entry, now, clientId, t }: { entry: Appeal; now: Date; clientId: string | null; t: Translate }) {
  const { sanction } = entry;
  return (
    <section className="flex flex-col gap-3 border-t border-surface pt-4 text-sm">
      <div className="flex flex-wrap gap-x-6 gap-y-1">
        <Link href={`/players/${encodeURIComponent(entry.account_id)}`} prefetch={false} className="text-primary underline">
          {entry.nickname || entry.account_id}
        </Link>
        <span>
          {t("appeals.kind")}: {t(sanctionKindKeys[sanction.kind])}
        </span>
        <span>
          {t("appeals.category")}: {t(banCategoryKeys[sanction.category])}
        </span>
        <span>
          {t("appeals.sanction_started")}: {t.when(sanction.starts_at)}
        </span>
        <span>
          {t("appeals.sanction_ends")}: {sanction.ends_at ? t.when(sanction.ends_at) : t("appeals.no_end")}
        </span>
        {!sanctionRunning(sanction, now) && <span>{t("appeals.sanction_over")}</span>}
        <span>
          {t("appeals.filed_at")}: {t.when(entry.created_at)}
        </span>
        <QueueDue deadline={entry} t={t} />
      </div>
      <div className="flex flex-col gap-1">
        <span className="font-medium">{t("appeals.text")}</span>
        <p className="whitespace-pre-wrap break-words rounded-md bg-surface px-3 py-2">{entry.text}</p>
      </div>
      <EarlierDecisions decisions={entry.decisions} t={t} />
      {clientId && <AppealDecision appealId={entry.id} clientId={clientId} />}
    </section>
  );
}
