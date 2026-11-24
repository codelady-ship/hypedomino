import { adminApi } from "@/admin/api";
import { googleClientId } from "@/admin/google";
import { can, canRollBack, canView } from "@/admin/navigation";
import { currentAdmin } from "@/admin/session";
import { changedDays, cycleCoins, type Calendar } from "@/dailylogin/calendar";
import { serverTranslator } from "@/i18n/server";
import type { Translate } from "@/i18n/translate";
import { Rollback } from "../config/Rollback";
import { rollbackCalendar } from "./actions";
import { CalendarEditor } from "./CalendarEditor";

export const dynamic = "force-dynamic";

type CalendarVersion = { config_version: number; published_at: string; catalog: Calendar };

export default async function DailyLoginPage() {
  const t = await serverTranslator();
  const admin = await currentAdmin();
  if (!admin) {
    return <Notice text={t("auth.required")} />;
  }
  if (!canView(admin.permissions, "missions")) {
    return <Notice text={t("auth.forbidden")} />;
  }
  const [config, history] = await Promise.all([
    adminApi<{ config_version: number; catalog: Calendar | null; rules: { default: Calendar } }>("GET", "/admin/config/daily-login"),
    adminApi<{ versions: CalendarVersion[] }>("GET", "/admin/config/daily-login/history"),
  ]);
  if (config.status !== 200 || history.status !== 200) {
    return <Notice text={t("daily_login.load_failed")} />;
  }
  const { catalog, rules } = config.body;
  return (
    <main className="mx-auto flex max-w-4xl flex-col gap-8 px-6 py-10">
      <h1 className="text-2xl font-semibold">{t("daily_login.title")}</h1>
      <p className="text-text-secondary">{t("daily_login.intro")}</p>
      {catalog === null && (
        <p role="alert" className="text-danger-text">
          {t("daily_login.empty")}
        </p>
      )}
      <CalendarEditor key={config.body.config_version} published={catalog} starting={catalog ?? rules.default} version={config.body.config_version} canPublish={can(admin.permissions, "missions", "publish")} />
      <History versions={history.body.versions} rollback={canRollBack(admin.permissions, "missions") ? googleClientId() : null} t={t} />
    </main>
  );
}

function History({ versions, rollback, t }: { versions: CalendarVersion[]; rollback: string | null; t: Translate }) {
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-lg font-semibold">{t("daily_login.history")}</h2>
      {versions.length === 0 ? (
        <p className="text-text-secondary">{t("daily_login.history_empty")}</p>
      ) : (
        <table className="text-sm">
          <thead className="text-left text-text-muted">
            <tr>
              <th className="py-2 pr-4">{t("daily_login.version")}</th>
              <th className="py-2 pr-4">{t("daily_login.published_at")}</th>
              <th className="py-2 pr-4">{t("daily_login.cycle_coins")}</th>
              <th className="py-2 pr-4">{t("daily_login.changed_days")}</th>
              {rollback && <th className="py-2 pr-4" />}
            </tr>
          </thead>
          <tbody>
            {versions.map((version, index) => {
              const previous = versions[index + 1]?.catalog;
              return (
                <tr key={version.config_version} className="border-t border-surface align-top">
                  <td className="py-2 pr-4">{version.config_version}</td>
                  <td className="py-2 pr-4 whitespace-nowrap">{t.when(version.published_at)}</td>
                  <td className="py-2 pr-4">{cycleCoins(version.catalog)}</td>
                  <td className="py-2 pr-4">{previous ? changedDays(previous, version.catalog).join(", ") : "—"}</td>
                  {rollback && <td className="py-2 pr-4">{index > 0 && <Rollback version={version.config_version} clientId={rollback} action={rollbackCalendar} />}</td>}
                </tr>
              );
            })}
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
