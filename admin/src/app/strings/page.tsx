import Link from "next/link";
import { adminApi } from "@/admin/api";
import { googleClientId } from "@/admin/google";
import { can, canRollBack, canView } from "@/admin/navigation";
import { currentAdmin } from "@/admin/session";
import type { SearchParams } from "@/audit/query";
import { serverTranslator } from "@/i18n/server";
import { bundlesApiPath, nextStringsHref, olderBundlesHref, readStringsQuery, stringsApiPath, stringsHref, type Bundle, type Coverage, type GameString } from "@/strings/strings";
import { Bundles } from "./Bundles";
import { FallbackChains } from "./FallbackChains";
import { StringsEditor } from "./StringsEditor";
import { Transfer } from "./Transfer";

export const dynamic = "force-dynamic";

type Active = { config_version: number; bundles: Record<string, number>; fallback_chains: Record<string, string[]> };

export default async function StringsPage({ searchParams }: { searchParams: Promise<SearchParams> }) {
  const t = await serverTranslator();
  const admin = await currentAdmin();
  if (!admin) {
    return <Notice text={t("auth.required")} />;
  }
  if (!canView(admin.permissions, "game_strings")) {
    return <Notice text={t("auth.forbidden")} />;
  }
  const [coverageReply, active] = await Promise.all([adminApi<{ coverage: Coverage[] }>("GET", "/admin/strings/coverage"), adminApi<Active>("GET", "/admin/strings/active")]);
  if (coverageReply.status !== 200 || active.status !== 200) {
    return <Notice text={t("strings.load_failed")} />;
  }
  const coverage = coverageReply.body.coverage;
  const languages = coverage.map((language) => language.language);
  const query = readStringsQuery(await searchParams, languages);
  const [strings, bundles] = await Promise.all([adminApi<{ strings: GameString[] }>("GET", stringsApiPath(query)), adminApi<{ bundles: Bundle[] }>("GET", bundlesApiPath(query))]);
  if (strings.status !== 200 || bundles.status !== 200) {
    return <Notice text={t("strings.load_failed")} />;
  }
  const permissions = admin.permissions;
  const languageHrefs = Object.fromEntries(languages.map((language) => [language, stringsHref(query, { bundles: language, before: "" })]));
  return (
    <main className="mx-auto flex max-w-6xl flex-col gap-8 px-6 py-10">
      <div className="flex flex-col gap-2">
        <h1 className="text-2xl font-semibold">{t("strings.title")}</h1>
        <p className="text-sm text-text-secondary">{t("strings.intro")}</p>
      </div>
      <section className="flex flex-col gap-3">
        <h2 className="text-lg font-semibold">{t("strings.coverage")}</h2>
        <table className="text-sm">
          <tbody>
            {coverage.map((report) => (
              <tr key={report.language} className="border-t border-surface align-top">
                <td className="py-2 pr-4 font-mono">{report.language}</td>
                <td className="py-2 pr-4">
                  {t.number(report.approved)} / {t.number(report.keys)}
                </td>
                <td className="py-2 pr-4">{t.number(report.percent)}%</td>
                <td className="py-2 pr-4">
                  {report.keys > report.approved && (
                    <Link href={stringsHref(query, { missing: report.language, after: "" })} prefetch={false} className="text-primary underline">
                      {t("strings.missing")}: {t.number(report.keys - report.approved)}
                    </Link>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
      <StringsEditor
        strings={strings.body.strings}
        languages={languages}
        query={query}
        next={nextStringsHref(query, strings.body.strings.map((entry) => entry.key))}
        mayCreate={can(permissions, "game_strings", "create")}
        mayEdit={can(permissions, "game_strings", "edit")}
        mayApprove={can(permissions, "game_strings", "edit") && can(permissions, "game_strings", "publish")}
      />
      <Transfer languages={languages} mayImport={can(permissions, "game_strings", "edit")} />
      <Bundles
        language={query.bundles}
        languageHrefs={languageHrefs}
        bundles={bundles.body.bundles}
        older={olderBundlesHref(query, bundles.body.bundles.map((bundle) => bundle.version))}
        active={active.body.bundles}
        configVersion={active.body.config_version}
        mayPublish={can(permissions, "game_strings", "publish")}
        rollbackClientId={canRollBack(permissions, "game_strings") ? googleClientId() : null}
      />
      <FallbackChains languages={languages} chains={active.body.fallback_chains} mayPublish={can(permissions, "game_strings", "publish")} />
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
