import Link from "next/link";
import { adminApi } from "@/admin/api";
import { canView } from "@/admin/navigation";
import { currentAdmin } from "@/admin/session";
import { serverTranslator } from "@/i18n/server";
import type { Translate } from "@/i18n/translate";
import { dealtOrder, handTiles, replayApiPath, seatNames, teamScore, type MatchReplay, type ReplayHand } from "@/matches/replay";
import { Notice } from "../../players/Notice";

export const dynamic = "force-dynamic";

export default async function MatchPage({ params }: { params: Promise<{ id: string }> }) {
  const t = await serverTranslator();
  const admin = await currentAdmin();
  if (!admin) {
    return <Notice text={t("auth.required")} />;
  }
  if (!canView(admin.permissions, "players")) {
    return <Notice text={t("auth.forbidden")} />;
  }
  const reply = await adminApi<MatchReplay>("GET", replayApiPath((await params).id));
  if (reply.status === 404) {
    return <Notice text={t("match.not_found")} />;
  }
  if (reply.status !== 200) {
    return <Notice text={t("match.load_failed")} />;
  }
  const match = reply.body;
  const names = seatNames(match.participants);
  return (
    <main className="mx-auto flex max-w-6xl flex-col gap-8 px-6 py-10">
      <div className="flex flex-col gap-2">
        <h1 className="text-2xl font-semibold">{t("match.title")}</h1>
        <p className="font-mono text-sm text-text-secondary">{match.match_id}</p>
        <p className="text-sm text-text-secondary">{t("match.explain")}</p>
      </div>
      <dl className="grid grid-cols-[auto_1fr] gap-x-6 gap-y-2 text-sm">
        <dt className="text-text-muted">{t("player.mode")}</dt>
        <dd>
          {match.mode} {match.kind}
        </dd>
        <dt className="text-text-muted">{t("match.window")}</dt>
        <dd>
          {t.when(match.started_at)} → {match.ended_at ? t.when(match.ended_at) : t("match.running")}
        </dd>
        <dt className="text-text-muted">{t("player.result")}</dt>
        <dd>{match.cancelled ? t("player.cancelled") : `${teamScore([match.score_a, match.score_b])}, ${t("match.winner")} ${match.winner_team ?? "-"}, ${match.hands_played} ${t("match.hands")}`}</dd>
        <dt className="text-text-muted">{t("match.config_version")}</dt>
        <dd>{match.config_version}</dd>
      </dl>
      <section className="flex flex-col gap-3">
        <h2 className="text-lg font-semibold">{t("match.seats")}</h2>
        <table className="text-sm">
          <thead className="text-left text-text-muted">
            <tr>
              <th className="py-2 pr-4">{t("match.seat")}</th>
              <th className="py-2 pr-4">{t("match.team")}</th>
              <th className="py-2 pr-4">{t("match.player")}</th>
              <th className="py-2 pr-4">{t("player.result")}</th>
            </tr>
          </thead>
          <tbody>
            {match.participants.map((participant) => (
              <tr key={participant.seat} className="border-t border-surface">
                <td className="py-2 pr-4 font-mono">{participant.seat}</td>
                <td className="py-2 pr-4">{participant.team}</td>
                <td className="py-2 pr-4">
                  {participant.account_id ? (
                    <Link href={`/players/${encodeURIComponent(participant.account_id)}`} prefetch={false} className="text-primary underline">
                      {participant.nickname ?? participant.account_id}
                    </Link>
                  ) : (
                    t("match.bot")
                  )}
                </td>
                <td className="py-2 pr-4">{participant.result ?? ""}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
      {!match.timeline ? (
        <p className="text-text-secondary">{t("match.no_replay")}</p>
      ) : (
        <>
          {match.timeline.problem && (
            <p role="alert" className="text-sm text-danger-text">
              {t("match.problem")}: {match.timeline.problem}
            </p>
          )}
          {match.timeline.hands.map((hand) => (
            <HandSection key={hand.number} hand={hand} names={names} seats={dealtOrder(hand, match.participants)} t={t} />
          ))}
        </>
      )}
    </main>
  );
}

function HandSection({ hand, names, seats, t }: { hand: ReplayHand; names: Map<string, string>; seats: string[]; t: Translate }) {
  return (
    <section className="flex flex-col gap-3 rounded-md border border-surface p-4">
      <h2 className="text-lg font-semibold">
        {t("match.hand")} {hand.number}
      </h2>
      <dl className="grid grid-cols-[auto_1fr] gap-x-6 gap-y-1 text-sm">
        {seats.map((seat) => (
          <div key={seat} className="contents">
            <dt className="text-text-muted">{names.get(seat) ?? seat}</dt>
            <dd className="font-mono">{handTiles(hand.dealt[seat])}</dd>
          </div>
        ))}
      </dl>
      <table className="text-sm">
        <thead className="text-left text-text-muted">
          <tr>
            <th className="py-1 pr-4">#</th>
            <th className="py-1 pr-4">{t("match.seat")}</th>
            <th className="py-1 pr-4">{t("match.move")}</th>
            <th className="py-1 pr-4">{t("match.tile")}</th>
            <th className="py-1 pr-4">{t("match.end")}</th>
            <th className="py-1 pr-4">{t("match.points")}</th>
          </tr>
        </thead>
        <tbody>
          {hand.steps.map((step, index) => (
            <tr key={index} className="border-t border-surface">
              <td className="py-1 pr-4 text-text-muted">{index + 1}</td>
              <td className="py-1 pr-4">{names.get(step.seat) ?? step.seat}</td>
              <td className="py-1 pr-4">{step.kind}</td>
              <td className="py-1 pr-4 font-mono">{step.tile ?? ""}</td>
              <td className="py-1 pr-4">{step.end ?? ""}</td>
              <td className="py-1 pr-4">{step.points ? t.signed(step.points) : ""}</td>
            </tr>
          ))}
        </tbody>
      </table>
      {hand.outcome && (
        <p className="text-sm text-text-secondary">
          {t("match.outcome")}: {t("match.winner")} {hand.outcome.winner ?? "-"}; {t("match.pips")} {teamScore(hand.outcome.pips)}; {t("match.awarded")} {teamScore(hand.outcome.awarded)}; {t("match.scores")} {teamScore(hand.outcome.scores)}
          {hand.outcome.carry_over > 0 && `; ${t("match.carry_over")} ${hand.outcome.carry_over}`}
        </p>
      )}
    </section>
  );
}
