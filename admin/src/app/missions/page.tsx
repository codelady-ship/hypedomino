import { adminApi } from "@/admin/api";
import { googleClientId } from "@/admin/google";
import { can, canRollBack, canView } from "@/admin/navigation";
import { currentAdmin } from "@/admin/session";
import { serverTranslator } from "@/i18n/server";
import type { Translate } from "@/i18n/translate";
import { emptyCatalog, missionChanges, type MissionCatalog, type MissionRules } from "@/missions/catalog";
import { CatalogChangeList } from "../config/CatalogChangeList";
import { Rollback } from "../config/Rollback";
import { rollbackMissions } from "./actions";
import { MissionEditor } from "./MissionEditor";

export const dynamic = "force-dynamic";

export default async function MissionsPage() {
  const t = await serverTranslator();
  const admin = await currentAdmin();
  if (!admin) {
    return <Notice text={t("auth.required")} />;
  }
  if (!canView(admin.permissions, "missions")) {
    return <Notice text={t("auth.forbidden")} />;
  }
  const [missions, history] = await Promise.all([
    adminApi<{ config_version: number; catalog: MissionCatalog | null; rules: MissionRules }>("GET", "/admin/config/missions"),
    adminApi<{ versions: CatalogVersion[] }>("GET", "/admin/config/missions/history"),
  ]);
  if (missions.status !== 200 || history.status !== 200) {
    return <Notice text={t("missions.load_failed")} />;
  }
  return (
    <main className="mx-auto flex max-w-5xl flex-col gap-8 px-6 py-10">
      <h1 className="text-2xl font-semibold">{t("missions.title")}</h1>
      {missions.body.catalog === null && <p className="text-text-secondary">{t("missions.empty")}</p>}
      <MissionEditor
        initialCatalog={missions.body.catalog ?? emptyCatalog()}
        published={missions.body.catalog}
        version={missions.body.config_version}
        rules={missions.body.rules}
        canPublish={can(admin.permissions, "missions", "publish")}
      />
      <History versions={history.body.versions} rollback={canRollBack(admin.permissions, "missions") ? googleClientId() : null} t={t} />
    </main>
  );
}

type CatalogVersion = { config_version: number; published_at: string; catalog: MissionCatalog };

function History({ versions, rollback, t }: { versions: CatalogVersion[]; rollback: string | null; t: Translate }) {
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-lg font-semibold">{t("missions.history")}</h2>
      {versions.length === 0 ? (
        <p className="text-text-secondary">{t("missions.history_empty")}</p>
      ) : (
        <table className="text-sm">
          <thead className="text-left text-text-muted">
            <tr>
              <th className="py-2 pr-4">{t("missions.version")}</th>
              <th className="py-2 pr-4">{t("missions.published_at")}</th>
              <th className="py-2 pr-4">{t("missions.mission_count")}</th>
              <th className="py-2 pr-4">{t("config.changes")}</th>
              {rollback && <th className="py-2 pr-4" />}
            </tr>
          </thead>
          <tbody>
            {versions.map((version, index) => (
              <tr key={version.config_version} className="border-t border-surface align-top">
                <td className="py-2 pr-4">{version.config_version}</td>
                <td className="py-2 pr-4 whitespace-nowrap">{t.when(version.published_at)}</td>
                <td className="py-2 pr-4">{version.catalog.missions.length}</td>
                <td className="py-2 pr-4 font-mono break-all">
                  <CatalogChangeList changes={missionChanges(versions[index + 1]?.catalog ?? null, version.catalog)} t={t} />
                </td>
                {rollback && <td className="py-2 pr-4">{index > 0 && <Rollback version={version.config_version} clientId={rollback} action={rollbackMissions} />}</td>}
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
