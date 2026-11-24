import { adminApi } from "@/admin/api";
import { googleClientId } from "@/admin/google";
import { can, canRollBack, canView } from "@/admin/navigation";
import { currentAdmin } from "@/admin/session";
import { serverTranslator } from "@/i18n/server";
import { entryCount, type Moderation, moderationToText } from "@/moderation/lists";
import { Rollback } from "../config/Rollback";
import { rollbackModeration } from "./actions";
import { ListsEditor } from "./ListsEditor";

export const dynamic = "force-dynamic";

type ModerationVersion = { config_version: number; published_at: string; catalog: Moderation };

export default async function ModerationPage() {
  const t = await serverTranslator();
  const admin = await currentAdmin();
  if (!admin) {
    return <Notice text={t("auth.required")} />;
  }
  if (!canView(admin.permissions, "moderation_config")) {
    return <Notice text={t("auth.forbidden")} />;
  }
  const [config, history] = await Promise.all([
    adminApi<{ config_version: number; catalog: Moderation | null }>("GET", "/admin/config/moderation"),
    adminApi<{ versions: ModerationVersion[] }>("GET", "/admin/config/moderation/history"),
  ]);
  if (config.status !== 200 || history.status !== 200) {
    return <Notice text={t("moderation.load_failed")} />;
  }
  const rollback = canRollBack(admin.permissions, "moderation_config") ? googleClientId() : null;
  return (
    <main className="mx-auto flex max-w-3xl flex-col gap-8 px-6 py-10">
      <div className="flex flex-col gap-2">
        <h1 className="text-2xl font-semibold">{t("moderation.title")}</h1>
        <p className="text-sm text-text-muted">
          {t("moderation.version")}: {config.body.config_version}
        </p>
        <p className="text-sm text-text-secondary">{t("moderation.intro")}</p>
      </div>
      <ListsEditor
        key={config.body.config_version}
        published={moderationToText(config.body.catalog)}
        canPublish={can(admin.permissions, "moderation_config", "publish")}
      />
      <section className="flex flex-col gap-3">
        <h2 className="text-lg font-semibold">{t("moderation.history")}</h2>
        <table className="text-sm">
          <tbody>
            {history.body.versions.map((version, index) => (
              <tr key={version.config_version} className="border-t border-surface align-top">
                <td className="py-2 pr-4">{version.config_version}</td>
                <td className="whitespace-nowrap py-2 pr-4">{t.when(version.published_at)}</td>
                <td className="py-2 pr-4">
                  {entryCount(version.catalog)} {t("moderation.entries")}
                </td>
                {rollback && <td className="py-2 pr-4">{index > 0 && <Rollback version={version.config_version} clientId={rollback} action={rollbackModeration} />}</td>}
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </main>
  );
}

function Notice({ text }: { text: string }) {
  return (
    <main className="mx-auto max-w-3xl px-6 py-24">
      <p className="text-text-secondary">{text}</p>
    </main>
  );
}
