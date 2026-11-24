import { adminApi } from "@/admin/api";
import { googleClientId } from "@/admin/google";
import { can, canRollBack, canView } from "@/admin/navigation";
import { currentAdmin } from "@/admin/session";
import { versionChanges, type ConfigKey, type KeyValues, type KeyedConfig, type KeyedVersion } from "@/config/keys";
import type { ConfigResult } from "@/config/result";
import { serverTranslator } from "@/i18n/server";
import type { MessageKey, Translate } from "@/i18n/translate";
import { KeyEditor } from "./KeyEditor";
import { Rollback } from "./Rollback";

type ConfigAction<T> = (value: T, reason: string) => Promise<ConfigResult>;

type PageText = { title: MessageKey; loadFailed: MessageKey; intro: MessageKey; historyEmpty: MessageKey };

export async function KeyedConfigPage({ path, text, publish, rollback }: { path: string; text: PageText; publish: ConfigAction<KeyValues>; rollback: ConfigAction<number> }) {
  const t = await serverTranslator();
  const admin = await currentAdmin();
  if (!admin) {
    return <Notice text={t("auth.required")} />;
  }
  if (!canView(admin.permissions, "game_config")) {
    return <Notice text={t("auth.forbidden")} />;
  }
  const [config, history] = await Promise.all([adminApi<KeyedConfig>("GET", path), adminApi<{ versions: KeyedVersion[] }>("GET", `${path}/history`)]);
  if (config.status !== 200 || history.status !== 200) {
    return <Notice text={t(text.loadFailed)} />;
  }
  return (
    <main className="mx-auto flex max-w-5xl flex-col gap-8 px-6 py-10">
      <div className="flex flex-col gap-2">
        <h1 className="text-2xl font-semibold">{t(text.title)}</h1>
        <p className="text-sm text-text-muted">
          {t("config.version")}: {config.body.config_version}
        </p>
        <p className="text-sm text-text-secondary">{t(text.intro)}</p>
      </div>
      <KeyEditor keys={config.body.keys} canPublish={can(admin.permissions, "game_config", "publish")} publish={publish} />
      <History
        keys={config.body.keys}
        versions={history.body.versions}
        empty={t(text.historyEmpty)}
        clientId={canRollBack(admin.permissions, "game_config") ? googleClientId() : null}
        rollback={rollback}
        t={t}
      />
    </main>
  );
}

type HistoryProps = { keys: ConfigKey[]; versions: KeyedVersion[]; empty: string; clientId: string | null; rollback: ConfigAction<number>; t: Translate };

function History({ keys, versions, empty, clientId, rollback, t }: HistoryProps) {
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-lg font-semibold">{t("config.history")}</h2>
      {versions.length === 0 ? (
        <p className="text-text-secondary">{empty}</p>
      ) : (
        <table className="text-sm">
          <thead className="text-left text-text-muted">
            <tr>
              <th className="py-2 pr-4">{t("config.version")}</th>
              <th className="py-2 pr-4">{t("config.published_at")}</th>
              <th className="py-2 pr-4">{t("config.changes")}</th>
              <th className="py-2 pr-4">{t("config.values")}</th>
              {clientId && <th className="py-2 pr-4" />}
            </tr>
          </thead>
          <tbody>
            {versions.map((version, index) => (
              <tr key={version.config_version} className="border-t border-surface align-top">
                <td className="py-2 pr-4">{version.config_version}</td>
                <td className="py-2 pr-4 whitespace-nowrap">{t.when(version.published_at)}</td>
                <td className="py-2 pr-4 font-mono">{changesText(keys, versions[index + 1]?.values ?? {}, version.values, t)}</td>
                <td className="py-2 pr-4 font-mono">
                  {Object.entries(version.values)
                    .map(([key, value]) => `${key}=${value}`)
                    .join(", ") || t("config.all_defaults")}
                </td>
                {clientId && <td className="py-2 pr-4">{index > 0 && <Rollback version={version.config_version} clientId={clientId} action={rollback} />}</td>}
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </section>
  );
}

function changesText(keys: ConfigKey[], older: KeyValues, newer: KeyValues, t: Translate): string {
  const changes = versionChanges(keys, older, newer);
  if (changes.length === 0) {
    return t("config.no_changes");
  }
  return changes.map((change) => `${change.key} ${change.before} → ${change.after}`).join(", ");
}

function Notice({ text }: { text: string }) {
  return (
    <main className="mx-auto max-w-3xl px-6 py-24">
      <p className="text-text-secondary">{text}</p>
    </main>
  );
}
