import { adminApi } from "@/admin/api";
import { googleClientId } from "@/admin/google";
import { can, canRollBack, canView } from "@/admin/navigation";
import { currentAdmin } from "@/admin/session";
import { serverTranslator } from "@/i18n/server";
import type { Translate } from "@/i18n/translate";
import { effectiveConfig, ratingChanges, type RatingConfig } from "@/ranked/config";
import type { Season } from "@/ranked/seasons";
import { CatalogChangeList } from "../config/CatalogChangeList";
import { Rollback } from "../config/Rollback";
import { rollbackRating } from "./actions";
import { RatingEditor } from "./RatingEditor";
import { SeasonsPanel } from "./SeasonsPanel";

export const dynamic = "force-dynamic";

type RatingRules = { server_time: string; open_season: string; structural_keys: string[]; tiers: string[]; division_ranks: string[]; defaults: RatingConfig };

export default async function RankedPage() {
  const t = await serverTranslator();
  const admin = await currentAdmin();
  if (!admin) {
    return <Notice text={t("auth.required")} />;
  }
  if (!canView(admin.permissions, "ranked_seasons")) {
    return <Notice text={t("auth.forbidden")} />;
  }
  const [config, history, seasons] = await Promise.all([
    adminApi<{ config_version: number; catalog: RatingConfig | null; rules: RatingRules }>("GET", "/admin/config/rating"),
    adminApi<{ versions: ConfigVersion[] }>("GET", "/admin/config/rating/history"),
    adminApi<{ server_time: string; next_start: string | null; seasons: Season[] }>("GET", "/admin/ranked/seasons"),
  ]);
  if (config.status !== 200 || history.status !== 200 || seasons.status !== 200) {
    return <Notice text={t("ranked.load_failed")} />;
  }
  const { rules } = config.body;
  const published = effectiveConfig(config.body.catalog, rules.defaults);
  const permissions = admin.permissions;
  return (
    <main className="mx-auto flex max-w-6xl flex-col gap-8 px-6 py-10">
      <h1 className="text-2xl font-semibold">{t("ranked.title")}</h1>
      <p className="text-sm text-text-muted">
        {t("ranked.server_time")}: {t.when(rules.server_time)}
      </p>
      <SeasonsPanel
        seasons={seasons.body.seasons}
        nextStart={seasons.body.next_start}
        serverTime={seasons.body.server_time}
        lengthMonths={Number(published.modes["1v1"].seasonLengthMonths)}
        canCreate={can(permissions, "ranked_seasons", "create")}
        canEdit={can(permissions, "ranked_seasons", "edit")}
        canDelete={can(permissions, "ranked_seasons", "delete")}
      />
      {config.body.catalog === null && <p className="text-text-secondary">{t("ranked.empty")}</p>}
      <RatingEditor
        key={config.body.config_version}
        published={published}
        version={config.body.config_version}
        openSeason={rules.open_season}
        structuralKeys={rules.structural_keys}
        tiers={rules.tiers}
        divisionRanks={rules.division_ranks}
        canPublish={can(permissions, "ranked_seasons", "publish")}
      />
      <History versions={history.body.versions} defaults={rules.defaults} rollback={canRollBack(permissions, "ranked_seasons") ? googleClientId() : null} t={t} />
    </main>
  );
}

type ConfigVersion = { config_version: number; published_at: string; catalog: RatingConfig };

function History({ versions, defaults, rollback, t }: { versions: ConfigVersion[]; defaults: RatingConfig; rollback: string | null; t: Translate }) {
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-lg font-semibold">{t("ranked.history")}</h2>
      {versions.length === 0 ? (
        <p className="text-text-secondary">{t("ranked.history_empty")}</p>
      ) : (
        <table className="text-sm">
          <thead className="text-left text-text-muted">
            <tr>
              <th className="py-2 pr-4">{t("ranked.version")}</th>
              <th className="py-2 pr-4">{t("ranked.published_at")}</th>
              <th className="py-2 pr-4">{t("config.changes")}</th>
              {rollback && <th className="py-2 pr-4" />}
            </tr>
          </thead>
          <tbody>
            {versions.map((version, index) => (
              <tr key={version.config_version} className="border-t border-surface align-top">
                <td className="py-2 pr-4">{version.config_version}</td>
                <td className="py-2 pr-4 whitespace-nowrap">{t.when(version.published_at)}</td>
                <td className="py-2 pr-4 font-mono break-all">
                  <CatalogChangeList changes={{ settings: ratingChanges(effectiveConfig(versions[index + 1]?.catalog ?? null, defaults), effectiveConfig(version.catalog, defaults)), entries: [] }} t={t} />
                </td>
                {rollback && <td className="py-2 pr-4">{index > 0 && <Rollback version={version.config_version} clientId={rollback} action={rollbackRating} />}</td>}
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
