import { adminApi } from "@/admin/api";
import { googleClientId } from "@/admin/google";
import { can, canRollBack, canView } from "@/admin/navigation";
import { currentAdmin } from "@/admin/session";
import { serverTranslator } from "@/i18n/server";
import type { Translate } from "@/i18n/translate";
import { emptyPassConfig, passChanges, type PassConfig, type Window } from "@/pass/config";
import { CatalogChangeList } from "../config/CatalogChangeList";
import { Rollback } from "../config/Rollback";
import { rollbackPass } from "./actions";
import { PassEditor } from "./PassEditor";

export const dynamic = "force-dynamic";

export default async function PassPage() {
  const t = await serverTranslator();
  const admin = await currentAdmin();
  if (!admin) {
    return <Notice text={t("auth.required")} />;
  }
  if (!canView(admin.permissions, "pass_seasons")) {
    return <Notice text={t("auth.forbidden")} />;
  }
  const [config, history, ranked] = await Promise.all([
    adminApi<{ config_version: number; catalog: PassConfig | null; rules: { server_time: string } }>("GET", "/admin/config/pass"),
    adminApi<{ versions: ConfigVersion[] }>("GET", "/admin/config/pass/history"),
    canView(admin.permissions, "ranked_seasons") ? adminApi<{ seasons: Window[] }>("GET", "/admin/ranked/seasons") : null,
  ]);
  if (config.status !== 200 || history.status !== 200) {
    return <Notice text={t("pass.load_failed")} />;
  }
  return (
    <main className="mx-auto flex max-w-6xl flex-col gap-8 px-6 py-10">
      <h1 className="text-2xl font-semibold">{t("pass.title")}</h1>
      {config.body.catalog === null && <p className="text-text-secondary">{t("pass.empty")}</p>}
      <PassEditor
        initialConfig={config.body.catalog ?? emptyPassConfig()}
        published={config.body.catalog}
        version={config.body.config_version}
        serverTime={config.body.rules.server_time}
        rankedSeasons={ranked?.status === 200 ? ranked.body.seasons : null}
        canPublish={can(admin.permissions, "pass_seasons", "publish")}
      />
      <History versions={history.body.versions} rollback={canRollBack(admin.permissions, "pass_seasons") ? googleClientId() : null} t={t} />
    </main>
  );
}

type ConfigVersion = { config_version: number; published_at: string; catalog: PassConfig };

function History({ versions, rollback, t }: { versions: ConfigVersion[]; rollback: string | null; t: Translate }) {
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-lg font-semibold">{t("pass.history")}</h2>
      {versions.length === 0 ? (
        <p className="text-text-secondary">{t("pass.history_empty")}</p>
      ) : (
        <table className="text-sm">
          <thead className="text-left text-text-muted">
            <tr>
              <th className="py-2 pr-4">{t("pass.version")}</th>
              <th className="py-2 pr-4">{t("pass.published_at")}</th>
              <th className="py-2 pr-4">{t("pass.season_count")}</th>
              <th className="py-2 pr-4">{t("config.changes")}</th>
              {rollback && <th className="py-2 pr-4" />}
            </tr>
          </thead>
          <tbody>
            {versions.map((version, index) => (
              <tr key={version.config_version} className="border-t border-surface align-top">
                <td className="py-2 pr-4">{version.config_version}</td>
                <td className="py-2 pr-4 whitespace-nowrap">{t.when(version.published_at)}</td>
                <td className="py-2 pr-4">{version.catalog.seasons.length}</td>
                <td className="py-2 pr-4 font-mono break-all">
                  <CatalogChangeList changes={passChanges(versions[index + 1]?.catalog ?? null, version.catalog)} t={t} />
                </td>
                {rollback && <td className="py-2 pr-4">{index > 0 && <Rollback version={version.config_version} clientId={rollback} action={rollbackPass} />}</td>}
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  );
}

function Notice({ text }: { text: string }) {
  return (
    <main className="mx-auto max-w-3xl px-6 py-24">
      <p className="text-text-secondary">{text}</p>
    </main>
  );
}
