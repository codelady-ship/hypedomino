import Link from "next/link";
import { adminApi } from "@/admin/api";
import { canView } from "@/admin/navigation";
import { currentAdmin } from "@/admin/session";
import { serverTranslator } from "@/i18n/server";
import { accountStates, identityProviders, nextPlayersHref, playerSearch, playersApiPath, type PlayerSummary, type SearchParams } from "@/players/record";
import { Notice } from "./Notice";

export const dynamic = "force-dynamic";

export default async function PlayersPage({ searchParams }: { searchParams: Promise<SearchParams> }) {
  const t = await serverTranslator();
  const admin = await currentAdmin();
  if (!admin) {
    return <Notice text={t("auth.required")} />;
  }
  if (!canView(admin.permissions, "players")) {
    return <Notice text={t("auth.forbidden")} />;
  }
  const search = playerSearch(await searchParams);
  const mayFindIdentity = admin.account.roles.includes("super_admin");
  const reply = search ? await adminApi<{ players: PlayerSummary[] | null }>("GET", playersApiPath(search)) : null;
  const players = reply?.status === 200 ? (reply.body.players ?? []) : null;
  const next = search && players ? nextPlayersHref(search, players.map((player) => player.id)) : null;
  return (
    <main className="mx-auto flex max-w-5xl flex-col gap-8 px-6 py-10">
      <h1 className="text-2xl font-semibold">{t("players.title")}</h1>
      <form method="get" className="flex flex-wrap items-end gap-3">
        <label className="flex min-w-80 flex-col gap-1 text-sm text-text-secondary">
          {t("players.query")}
          <input name="q" defaultValue={search && "q" in search ? search.q : ""} className="rounded-md bg-surface px-3 py-1 text-text-primary" />
        </label>
        <label className="flex flex-col gap-1 text-sm text-text-secondary">
          {t("players.state")}
          <select name="state" defaultValue={search && "state" in search ? search.state : ""} className="rounded-md bg-surface px-3 py-1 text-text-primary">
            <option value="">{t("queues.any")}</option>
            {accountStates.map((state) => (
              <option key={state} value={state}>
                {state}
              </option>
            ))}
          </select>
        </label>
        <button type="submit" className="rounded-md bg-primary px-4 py-2 text-on-primary">
          {t("players.search")}
        </button>
      </form>
      {mayFindIdentity && (
        <form method="get" className="flex flex-wrap items-end gap-3">
          <label className="flex flex-col gap-1 text-sm text-text-secondary">
            {t("players.provider")}
            <select name="provider" defaultValue={search && "provider" in search ? search.provider : "google"} className="rounded-md bg-surface px-3 py-1 text-text-primary">
              {identityProviders.map((provider) => (
                <option key={provider} value={provider}>
                  {provider}
                </option>
              ))}
            </select>
          </label>
          <label className="flex min-w-80 flex-col gap-1 text-sm text-text-secondary">
            {t("players.subject")}
            <input name="subject" defaultValue={search && "subject" in search ? search.subject : ""} className="rounded-md bg-surface px-3 py-1 text-text-primary" />
          </label>
          <button type="submit" className="rounded-md bg-primary px-4 py-2 text-on-primary">
            {t("players.find_identity")}
          </button>
          <p className="basis-full text-sm text-text-muted">{t("players.identity_hint")}</p>
        </form>
      )}
      {reply && reply.status !== 200 && (
        <p role="alert" className="text-danger-text">
          {t("players.search_failed")}
        </p>
      )}
      {players?.length === 0 && <p className="text-text-secondary">{t("players.none")}</p>}
      {players && players.length > 0 && (
        <table className="text-sm">
          <thead className="text-left text-text-muted">
            <tr>
              <th className="py-2 pr-4">{t("players.nickname")}</th>
              <th className="py-2 pr-4">{t("players.state")}</th>
              <th className="py-2 pr-4">{t("players.country")}</th>
              <th className="py-2 pr-4">{t("players.created_at")}</th>
            </tr>
          </thead>
          <tbody>
            {players.map((player) => (
              <tr key={player.id} className="border-t border-surface">
                <td className="py-2 pr-4">
                  <Link href={`/players/${encodeURIComponent(player.id)}`} prefetch={false} className="text-primary underline">
                    {player.nickname || player.id}
                  </Link>
                </td>
                <td className="py-2 pr-4">{player.state}</td>
                <td className="py-2 pr-4">{player.country}</td>
                <td className="py-2 pr-4 whitespace-nowrap">{t.when(player.created_at)}</td>
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
