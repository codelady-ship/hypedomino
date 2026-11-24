import Link from "next/link";
import type { ReactNode } from "react";
import { adminApi } from "@/admin/api";
import { googleClientId } from "@/admin/google";
import { can, canView } from "@/admin/navigation";
import { currentAdmin } from "@/admin/session";
import { serverTranslator } from "@/i18n/server";
import { matchHref } from "@/matches/replay";
import type { MessageKey, Translate } from "@/i18n/translate";
import { banActive, missionPlace, missionProgress, muteActive, playerApiPath, reportSide, type PlayerRecord } from "@/players/record";
import { boardSeasons } from "@/players/leaderboard";
import { ladders, type PenaltyView } from "@/players/penalties";
import { banCategoryKeys, runningRankedBan, sanctionKindKeys } from "@/players/sanctions";
import type { Season } from "@/ranked/seasons";
import { Notice } from "../Notice";
import { PlayerActions } from "./PlayerActions";

export const dynamic = "force-dynamic";

type Column<Row> = { label: MessageKey; cell: (row: Row) => ReactNode };

export default async function PlayerPage({ params }: { params: Promise<{ id: string }> }) {
  const t = await serverTranslator();
  const when = (instant: string | null) => (instant ? t.when(instant) : "");
  const admin = await currentAdmin();
  if (!admin) {
    return <Notice text={t("auth.required")} />;
  }
  if (!canView(admin.permissions, "players")) {
    return <Notice text={t("auth.forbidden")} />;
  }
  const reply = await adminApi<PlayerRecord>("GET", playerApiPath((await params).id));
  if (reply.status === 404) {
    return <Notice text={t("player.not_found")} />;
  }
  if (reply.status !== 200) {
    return <Notice text={t("player.load_failed")} />;
  }
  const record = reply.body;
  const [penaltyReply, seasonsReply] = await Promise.all([
    adminApi<PenaltyView>("GET", `${playerApiPath(record.account.id)}/penalties`),
    canView(admin.permissions, "ranked_seasons") ? adminApi<{ seasons: Season[] }>("GET", "/admin/ranked/seasons") : null,
  ]);
  const penalties = penaltyReply.status === 200 ? penaltyReply.body : null;
  const seasons = seasonsReply?.status === 200 ? boardSeasons(seasonsReply.body.seasons) : [];
  const { account, profile } = record;
  const rankedBan = runningRankedBan(record.sanctions, new Date());
  return (
    <main className="mx-auto flex max-w-6xl flex-col gap-8 px-6 py-10">
      <Link href="/players" className="self-start text-sm text-text-secondary underline">
        {t("player.back")}
      </Link>
      <h1 className="text-2xl font-semibold break-words">{account.nickname || account.id}</h1>
      <dl className="grid grid-cols-[auto_1fr] gap-x-6 gap-y-2 text-sm">
        <Fact label={t("player.id")} value={<span className="font-mono">{account.id}</span>} />
        <Fact label={t("players.state")} value={account.state} />
        <Fact label={t("players.country")} value={account.country} />
        <Fact label={t("players.created_at")} value={t.when(account.created_at)} />
        <Fact label={t("player.emails")} value={account.emails.join(", ")} />
        {account.identities && <Fact label={t("player.identities")} value={account.identities.map((identity) => `${identity.provider}:${identity.subject}`).join(", ")} />}
        <Fact label={t("player.banned_until")} value={banActive(account, new Date()) ? when(account.banned_until) : t("player.not_banned")} />
        <Fact label={t("player.ranked_banned_until")} value={rankedBan ? `${when(rankedBan.ends_at)}, ${t(banCategoryKeys[rankedBan.category])}` : t("player.not_banned")} />
        <Fact label={t("player.shadowbanned_at")} value={account.shadowbanned_at ? when(account.shadowbanned_at) : t("player.not_banned")} />
        <Fact label={t("player.muted_until")} value={muteActive(account, new Date()) ? when(account.muted_until) : t("player.not_banned")} />
        {account.deletion_requested_at && <Fact label={t("player.deletion_requested_at")} value={when(account.deletion_requested_at)} />}
        {account.deleted_at && <Fact label={t("player.deleted_at")} value={when(account.deleted_at)} />}
        <Fact label={t("player.coins")} value={t.number(record.coins)} />
        <Fact label={t("player.crystals")} value={t.number(record.crystals)} />
        {profile && <Fact label={t("player.level")} value={`${profile.level} (${profile.xp} XP)`} />}
        {profile?.rename_required && <Fact label={t("player.rename_required")} value={t("player.yes")} />}
      </dl>
      {can(admin.permissions, "players", "act_on_player") && account.state !== "deleted" && (
        <PlayerActions id={account.id} clientId={googleClientId()} banned={banActive(account, new Date())} rankedBanned={rankedBan !== null} shadowbanned={account.shadowbanned_at !== null} muted={muteActive(account, new Date())} renameRequired={profile?.rename_required ?? false} penalties={penalties} boardSeasons={seasons} />
      )}
      {penalties && (
        <dl className="grid grid-cols-[auto_1fr] gap-x-6 gap-y-2 text-sm">
          {ladders.map((queue) => (
            <Fact key={queue} label={`${t("player.penalty_ladder")} ${queue}`} value={`${penalties[queue].step}, ${t("player.penalty_level")} ${penalties[queue].level}${penalties[queue].ban_until ? `, ${t("player.banned_until")} ${when(penalties[queue].ban_until)}` : ""}${penalties[queue].under_review ? `, ${t("player.penalty_review")}` : ""}`} />
          ))}
        </dl>
      )}
      <Rows title="player.penalty_offenses" t={t} rows={penalties ? ladders.flatMap((queue) => penalties[queue].offenses.map((offense) => ({ ...offense, queue }))) : []} columns={[
        { label: "player.penalty_ladder", cell: (row) => row.queue },
        { label: "player.match", cell: (row) => <span className="font-mono">{row.match_id}</span> },
        { label: "player.penalty_kind", cell: (row) => row.kind },
        { label: "player.penalty_weight", cell: (row) => row.weight },
        { label: "player.at", cell: (row) => when(row.at) },
        { label: "player.penalty_expires", cell: (row) => when(row.expires_at) },
      ]} />
      <Rows title="player.sanctions" t={t} rows={record.sanctions} columns={[
        { label: "player.at", cell: (row) => when(row.starts_at) },
        { label: "player.sanction_kind", cell: (row) => t(sanctionKindKeys[row.kind]) },
        { label: "player.sanction_category", cell: (row) => t(banCategoryKeys[row.category]) },
        { label: "player.sanction_ends", cell: (row) => when(row.ends_at) },
        { label: "player.lifted_at", cell: (row) => when(row.lifted_at) },
      ]} />
      <Rows title="player.reports" t={t} rows={record.reports.map((report) => ({ ...report, ...reportSide(account.id, report) }))} columns={[
        { label: "player.at", cell: (row) => when(row.updated_at) },
        { label: "player.report_side", cell: (row) => t(row.about ? "player.report_about" : "player.report_by") },
        { label: "player.counterpart", cell: (row) => <Link href={`/players/${encodeURIComponent(row.counterpart)}`} prefetch={false} className="text-primary underline">{row.counterpart}</Link> },
        { label: "reports.reason_code", cell: (row) => <span className="font-mono">{row.reason}</span> },
        { label: "reports.name_text", cell: (row) => row.name_text },
        { label: "reports.match", cell: (row) => (row.match_id ? <Link href={matchHref(row.match_id)} prefetch={false} className="text-primary underline">{row.match_id}</Link> : "") },
        { label: "players.state", cell: (row) => row.state },
        { label: "player.decided_at", cell: (row) => when(row.decided_at) },
      ]} />
      <Rows title="player.missions" t={t} rows={record.missions} columns={[
        { label: "player.assigned_at", cell: (row) => when(row.assigned_at) },
        { label: "player.mission", cell: (row) => <span className="font-mono">{row.mission_id}</span> },
        { label: "player.mission_place", cell: (row) => missionPlace(row) },
        { label: "player.mission_period", cell: (row) => row.period },
        { label: "player.mission_progress", cell: (row) => missionProgress(row) },
        { label: "player.completed_at", cell: (row) => when(row.completed_at) },
        { label: "player.replaced_at", cell: (row) => when(row.replaced_at) },
      ]} />
      <Rows title="player.nickname_history" t={t} rows={record.nickname_history} columns={[
        { label: "players.nickname", cell: (row) => row.nickname },
        { label: "player.replaced_at", cell: (row) => when(row.replaced_at) },
      ]} />
      <Rows title="player.ledger" t={t} rows={record.ledger} columns={[
        { label: "player.at", cell: (row) => when(row.at) },
        { label: "player.currency", cell: (row) => row.currency },
        { label: "player.amount", cell: (row) => t.signed(row.amount) },
        { label: "player.balance_after", cell: (row) => t.number(row.balance_after) },
        { label: "player.reason", cell: (row) => row.reason },
        { label: "player.source", cell: (row) => `${row.source_type} ${row.source_ref}` },
      ]} />
      <Rows title="player.inventory" t={t} rows={record.inventory} columns={[
        { label: "player.item", cell: (row) => row.item_id },
        { label: "player.source", cell: (row) => `${row.source_type} ${row.source_ref}` },
        { label: "player.granted_at", cell: (row) => when(row.granted_at) },
        { label: "player.revoked_at", cell: (row) => when(row.revoked_at) },
      ]} />
      <Rows title="player.grants" t={t} rows={record.grants} columns={[
        { label: "player.at", cell: (row) => when(row.at) },
        { label: "player.bundle", cell: (row) => `${row.bundle_id} r${row.revision}` },
        { label: "player.source", cell: (row) => `${row.source_type} ${row.source_ref}` },
        { label: "player.revoked_at", cell: (row) => when(row.revoked_at) },
      ]} />
      <Rows title="player.purchases" t={t} rows={record.purchases} columns={[
        { label: "player.at", cell: (row) => when(row.purchased_at) },
        { label: "player.product", cell: (row) => `${row.platform} ${row.product_id} ${row.offer_id}` },
        { label: "players.state", cell: (row) => `${row.state} (${row.environment})` },
        { label: "player.refunded_at", cell: (row) => when(row.refunded_at) },
      ]} />
      <Rows title="player.matches" t={t} rows={record.matches} columns={[
        { label: "player.at", cell: (row) => when(row.started_at) },
        { label: "player.mode", cell: (row) => `${row.mode} ${row.kind}` },
        { label: "player.result", cell: (row) => (row.cancelled ? t("player.cancelled") : `${row.result} ${row.score_a}:${row.score_b}`) },
        { label: "player.suspicious", cell: (row) => <span className="font-mono">{row.suspicious.join(", ")}</span> },
        { label: "match.open_replay", cell: (row) => <Link href={matchHref(row.match_id)} prefetch={false} className="text-primary underline">{t("match.open_replay")}</Link> },
      ]} />
      <Rows title="player.pair_signals" t={t} rows={record.pair_signals} columns={[
        { label: "player.at", cell: (row) => when(row.updated_at) },
        { label: "reviews.signal", cell: (row) => <span className="font-mono">{row.kind}</span> },
        { label: "player.counterpart", cell: (row) => <Link href={`/players/${encodeURIComponent(row.counterpart)}`} prefetch={false} className="text-primary underline">{row.counterpart}</Link> },
        { label: "player.pair_matches", cell: (row) => `${row.mode} ${row.wins}/${row.matches} (${row.offenses}), ${row.window_hours} h` },
      ]} />
    </main>
  );
}

function Fact({ label, value }: { label: string; value: ReactNode }) {
  return (
    <>
      <dt className="text-text-muted">{label}</dt>
      <dd className="break-words">{value}</dd>
    </>
  );
}

function Rows<Row>({ title, rows, columns, t }: { title: MessageKey; rows: Row[] | null; columns: Column<Row>[]; t: Translate }) {
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-lg font-semibold">{t(title)}</h2>
      {!rows || rows.length === 0 ? (
        <p className="text-sm text-text-secondary">{t("player.empty")}</p>
      ) : (
        <table className="text-sm">
          <thead className="text-left text-text-muted">
            <tr>
              {columns.map((column) => (
                <th key={column.label} className="py-2 pr-4">
                  {t(column.label)}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((row, index) => (
              <tr key={index} className="border-t border-surface align-top">
                {columns.map((column) => (
                  <td key={column.label} className="py-2 pr-4 break-words">
                    {column.cell(row)}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  );
}
