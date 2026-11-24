import { adminApi } from "@/admin/api";
import { canView } from "@/admin/navigation";
import { currentAdmin } from "@/admin/session";
import {
  kpisCsv,
  kpisCsvFileName,
  latestRetention,
  matchModes,
  matchRows,
  maxWeeks,
  reportWeeks,
  retentionPercent,
  type Cohort,
  type KpiReport,
  type Retention,
} from "@/analytics/kpis";
import { serverTranslator } from "@/i18n/server";
import type { Translate } from "@/i18n/translate";

export const dynamic = "force-dynamic";

export default async function KpisPage({ searchParams }: { searchParams: Promise<{ weeks?: string }> }) {
  const t = await serverTranslator();
  const admin = await currentAdmin();
  if (!admin) {
    return <Notice text={t("auth.required")} />;
  }
  if (!canView(admin.permissions, "dashboards")) {
    return <Notice text={t("auth.forbidden")} />;
  }
  const weeks = reportWeeks((await searchParams).weeks);
  const response = await adminApi<KpiReport>("GET", `/admin/analytics/kpis?weeks=${weeks}`);
  if (response.status !== 200) {
    return <Notice text={t("kpis.load_failed")} />;
  }
  const report = response.body;
  return (
    <main className="mx-auto flex max-w-5xl flex-col gap-8 px-6 py-10">
      <div className="flex flex-col gap-2">
        <h1 className="text-2xl font-semibold">{t("kpis.title")}</h1>
        <p className="text-sm text-text-secondary">{t("kpis.explain")}</p>
      </div>
      <form method="get" className="flex flex-wrap items-end gap-3">
        <label className="flex flex-col gap-1 text-sm text-text-secondary">
          {t("kpis.weeks")}
          <input name="weeks" type="number" min={1} max={maxWeeks} defaultValue={weeks} className="w-24 rounded-md bg-surface px-3 py-1 text-text-primary" />
        </label>
        <button type="submit" className="rounded-md bg-primary px-4 py-2 text-on-primary">
          {t("kpis.show")}
        </button>
        <span className="text-sm text-text-secondary">
          {report.from} – {report.to} ({t("kpis.to_excluded")})
        </span>
      </form>
      <Cards t={t} report={report} />
      <Cohorts t={t} title={t("kpis.weekly_cohorts")} cohorts={report.weeklyCohorts} />
      <Matches t={t} report={report} />
      <Cohorts t={t} title={t("kpis.daily_cohorts")} cohorts={report.dailyCohorts} />
      <a
        href={`data:text/csv;charset=utf-8,${encodeURIComponent(kpisCsv(report))}`}
        download={kpisCsvFileName(report)}
        className="self-start rounded-md bg-surface px-3 py-1 text-text-secondary"
      >
        {t("analytics.export_csv")}
      </a>
    </main>
  );
}

function Cards({ t, report }: { t: Translate; report: KpiReport }) {
  const lastDay = matchRows(report.matches).at(-1);
  const lastCohort = report.dailyCohorts.at(-1);
  return (
    <section className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <Card
        title={t("kpis.card.matches")}
        value={lastDay ? t.number(lastDay.total) : "–"}
        detail={lastDay ? `${lastDay.day} · ${Object.entries(lastDay.perMode).map(([mode, completed]) => `${mode} ${t.number(completed)}`).join(" · ")}` : ""}
      />
      <Card title={t("kpis.card.installs")} value={lastCohort ? t.number(lastCohort.size) : "–"} detail={lastCohort?.start ?? ""} />
      <RetentionCard t={t} title={t("kpis.card.d1")} retention={latestRetention(report.dailyCohorts, "d1")} />
      <RetentionCard t={t} title={t("kpis.card.d7")} retention={latestRetention(report.dailyCohorts, "d7")} />
    </section>
  );
}

function RetentionCard({ t, title, retention }: { t: Translate; title: string; retention: Retention | null }) {
  if (!retention) {
    return <Card title={title} value="–" detail={t("kpis.pending")} />;
  }
  return <Card title={title} value={share(t, retention.percent)} detail={`${retention.cohort.start} · ${t.number(retention.retained)} / ${t.number(retention.cohort.size)}`} />;
}

function Card({ title, value, detail }: { title: string; value: string; detail: string }) {
  return (
    <div className="flex flex-col gap-1 rounded-md bg-surface px-4 py-3">
      <span className="text-sm text-text-secondary">{title}</span>
      <span className="text-2xl font-semibold">{value}</span>
      <span className="text-xs text-text-muted">{detail}</span>
    </div>
  );
}

function Cohorts({ t, title, cohorts }: { t: Translate; title: string; cohorts: Cohort[] }) {
  return (
    <section className="flex flex-col gap-2">
      <h2 className="text-lg font-semibold">{title}</h2>
      <table className="w-full text-left text-sm">
        <thead className="text-text-muted">
          <tr>
            <th className="py-2">{t("kpis.cohort_start")}</th>
            <th className="py-2">{t("kpis.cohort_size")}</th>
            <th className="py-2">D1</th>
            <th className="py-2">D7</th>
          </tr>
        </thead>
        <tbody>
          {cohorts.map((cohort) => (
            <tr key={cohort.start}>
              <td className="py-1 font-mono">{cohort.start}</td>
              <td className="py-1">{t.number(cohort.size)}</td>
              <td className="py-1">{retained(t, cohort.d1, cohort.size)}</td>
              <td className="py-1">{retained(t, cohort.d7, cohort.size)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}

function Matches({ t, report }: { t: Translate; report: KpiReport }) {
  const modes = matchModes(report.matches);
  return (
    <section className="flex flex-col gap-2">
      <h2 className="text-lg font-semibold">{t("kpis.matches")}</h2>
      <table className="w-full text-left text-sm">
        <thead className="text-text-muted">
          <tr>
            <th className="py-2">{t("analytics.day")}</th>
            {modes.map((mode) => (
              <th key={mode} className="py-2">
                {mode}
              </th>
            ))}
            <th className="py-2">{t("kpis.total")}</th>
          </tr>
        </thead>
        <tbody>
          {matchRows(report.matches).map((row) => (
            <tr key={row.day}>
              <td className="py-1 font-mono">{row.day}</td>
              {modes.map((mode) => (
                <td key={mode} className="py-1">
                  {t.number(row.perMode[mode] ?? 0)}
                </td>
              ))}
              <td className="py-1">{t.number(row.total)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}

function retained(t: Translate, count: number | null, size: number): string {
  if (count === null) {
    return t("kpis.pending");
  }
  return `${share(t, retentionPercent(count, size))} (${t.number(count)})`;
}

function share(t: Translate, percent: number | null): string {
  return percent === null ? "–" : `${t.number(percent)}%`;
}

function Notice({ text }: { text: string }) {
  return (
    <main className="mx-auto max-w-3xl px-6 py-24">
      <p className="text-text-secondary">{text}</p>
    </main>
  );
}
