import { adminApi } from "@/admin/api";
import { googleClientId } from "@/admin/google";
import { can, canRollBack, canView } from "@/admin/navigation";
import { currentAdmin } from "@/admin/session";
import { serverTranslator } from "@/i18n/server";
import type { Translate } from "@/i18n/translate";
import { emptyMedalCatalog, medalChanges, type MedalCatalog, type MedalRules } from "@/medals/catalog";
import { CatalogChangeList } from "../config/CatalogChangeList";
import { Rollback } from "../config/Rollback";
import { rollbackMedals } from "./actions";
import { MedalEditor } from "./MedalEditor";

export const dynamic = "force-dynamic";

export default async function MedalsPage() {
  const t = await serverTranslator();
  const admin = await currentAdmin();
  if (!admin) {
    return <Notice text={t("auth.required")} />;
  }
  if (!canView(admin.permissions, "medals")) {
    return <Notice text={t("auth.forbidden")} />;
  }
  const [medals, history] = await Promise.all([
    adminApi<{ config_version: number; catalog: MedalCatalog | null; rules: MedalRules }>("GET", "/admin/config/medals"),
    adminApi<{ versions: CatalogVersion[] }>("GET", "/admin/config/medals/history"),
  ]);
  if (medals.status !== 200 || history.status !== 200) {
    return <Notice text={t("medals.load_failed")} />;
  }
  return (
    <main className="mx-auto flex max-w-5xl flex-col gap-8 px-6 py-10">
      <h1 className="text-2xl font-semibold">{t("medals.title")}</h1>
      {medals.body.catalog === null && <p className="text-text-secondary">{t("medals.empty")}</p>}
      <MedalEditor
        initialCatalog={medals.body.catalog ?? emptyMedalCatalog()}
        version={medals.body.config_version}
        rules={medals.body.rules}
        canPublish={can(admin.permissions, "medals", "publish")}
      />
      <History versions={history.body.versions} rollback={canRollBack(admin.permissions, "medals") ? googleClientId() : null} t={t} />
    </main>
  );
}

type CatalogVersion = { config_version: number; published_at: string; catalog: MedalCatalog };

function History({ versions, rollback, t }: { versions: CatalogVersion[]; rollback: string | null; t: Translate }) {
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-lg font-semibold">{t("medals.history")}</h2>
      {versions.length === 0 ? (
        <p className="text-text-secondary">{t("medals.history_empty")}</p>
      ) : (
        <table className="text-sm">
          <thead className="text-left text-text-muted">
            <tr>
              <th className="py-2 pr-4">{t("medals.version")}</th>
              <th className="py-2 pr-4">{t("medals.published_at")}</th>
              <th className="py-2 pr-4">{t("medals.medal_count")}</th>
              <th className="py-2 pr-4">{t("config.changes")}</th>
              {rollback && <th className="py-2 pr-4" />}
            </tr>
          </thead>
          <tbody>
            {versions.map((version, index) => (
              <tr key={version.config_version} className="border-t border-surface align-top">
                <td className="py-2 pr-4">{version.config_version}</td>
                <td className="py-2 pr-4 whitespace-nowrap">{t.when(version.published_at)}</td>
                <td className="py-2 pr-4">{version.catalog.medals.length}</td>
                <td className="py-2 pr-4 font-mono break-all">
                  <CatalogChangeList changes={medalChanges(versions[index + 1]?.catalog ?? null, version.catalog)} t={t} />
                </td>
                {rollback && <td className="py-2 pr-4">{index > 0 && <Rollback version={version.config_version} clientId={rollback} action={rollbackMedals} />}</td>}
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
