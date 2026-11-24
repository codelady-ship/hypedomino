import { adminApi } from "@/admin/api";
import { googleClientId } from "@/admin/google";
import { can, canRollBack, canView } from "@/admin/navigation";
import { currentAdmin } from "@/admin/session";
import { manifestChanges, type CosmeticManifest, type CosmeticRules } from "@/cosmetics/catalog";
import { serverTranslator } from "@/i18n/server";
import type { Translate } from "@/i18n/translate";
import { CatalogChangeList } from "../config/CatalogChangeList";
import { Rollback } from "../config/Rollback";
import { rollbackCosmetics } from "./actions";
import { CosmeticsEditor } from "./CosmeticsEditor";
import { SwitchOff } from "./SwitchOff";

export const dynamic = "force-dynamic";

export default async function CosmeticsPage() {
  const t = await serverTranslator();
  const admin = await currentAdmin();
  if (!admin) {
    return <Notice text={t("auth.required")} />;
  }
  if (!canView(admin.permissions, "cosmetic_items")) {
    return <Notice text={t("auth.forbidden")} />;
  }
  const [config, history] = await Promise.all([
    adminApi<{ config_version: number; catalog: CosmeticManifest | null; rules: CosmeticRules }>("GET", "/admin/config/cosmetics"),
    adminApi<{ versions: ConfigVersion[] }>("GET", "/admin/config/cosmetics/history"),
  ]);
  if (config.status !== 200 || history.status !== 200) {
    return <Notice text={t("cosmetics.load_failed")} />;
  }
  const { rules, catalog } = config.body;
  return (
    <main className="mx-auto flex max-w-6xl flex-col gap-8 px-6 py-10">
      <h1 className="text-2xl font-semibold">{t("cosmetics.title")}</h1>
      {catalog === null && <p className="text-text-secondary">{t("cosmetics.empty")}</p>}
      <CosmeticsEditor
        key={config.body.config_version}
        published={(catalog ?? rules.builtin).items}
        manifestVersion={(catalog ?? rules.builtin).manifestVersion}
        version={config.body.config_version}
        rules={rules}
        canPublish={can(admin.permissions, "cosmetic_items", "publish")}
        canReview={can(admin.permissions, "cosmetic_items", "cultural_review")}
        canSeeArt={canView(admin.permissions, "assets")}
      />
      {catalog !== null && can(admin.permissions, "cosmetic_items", "switch_off_for_incident") && (
        <SwitchOff itemIds={catalog.items.filter((item) => item.availability !== "disabled").map((item) => item.id)} clientId={googleClientId()} />
      )}
      <History versions={history.body.versions} builtin={rules.builtin} rollback={canRollBack(admin.permissions, "cosmetic_items") ? googleClientId() : null} t={t} />
    </main>
  );
}

type ConfigVersion = { config_version: number; published_at: string; catalog: CosmeticManifest };

function History({ versions, builtin, rollback, t }: { versions: ConfigVersion[]; builtin: CosmeticManifest; rollback: string | null; t: Translate }) {
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-lg font-semibold">{t("cosmetics.history")}</h2>
      {versions.length === 0 ? (
        <p className="text-text-secondary">{t("cosmetics.history_empty")}</p>
      ) : (
        <table className="text-sm">
          <thead className="text-left text-text-muted">
            <tr>
              <th className="py-2 pr-4">{t("cosmetics.version")}</th>
              <th className="py-2 pr-4">{t("cosmetics.manifest_version")}</th>
              <th className="py-2 pr-4">{t("cosmetics.published_at")}</th>
              <th className="py-2 pr-4">{t("cosmetics.item_count")}</th>
              <th className="py-2 pr-4">{t("config.changes")}</th>
              {rollback && <th className="py-2 pr-4" />}
            </tr>
          </thead>
          <tbody>
            {versions.map((version, index) => (
              <tr key={version.config_version} className="border-t border-surface align-top">
                <td className="py-2 pr-4">{version.config_version}</td>
                <td className="py-2 pr-4">{version.catalog.manifestVersion}</td>
                <td className="py-2 pr-4 whitespace-nowrap">{t.when(version.published_at)}</td>
                <td className="py-2 pr-4">{version.catalog.items.length}</td>
                <td className="py-2 pr-4 font-mono break-all">
                  <CatalogChangeList changes={manifestChanges(versions[index + 1]?.catalog ?? builtin, version.catalog)} t={t} />
                </td>
                {rollback && <td className="py-2 pr-4">{index > 0 && <Rollback version={version.config_version} clientId={rollback} action={rollbackCosmetics} />}</td>}
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
