import Link from "next/link";
import type { ReactNode } from "react";
import { roleNames } from "@/accounts/roles";
import { adminApi, type ApiReply } from "@/admin/api";
import { googleClientId } from "@/admin/google";
import { canView } from "@/admin/navigation";
import { currentAdmin } from "@/admin/session";
import { alertLabels, type EconomyReport } from "@/analytics/economy";
import type { Proposal } from "@/approvals/proposals";
import type { LiveControlsView } from "@/config/liveControls";
import { serverTranslator } from "@/i18n/server";
import type { Translate } from "@/i18n/translate";
import { GoogleSignIn } from "./auth/GoogleSignIn";
import { UnconfirmedBreakGlass } from "./live-controls/UnconfirmedBreakGlass";

export const dynamic = "force-dynamic";

type ActiveVersion = { config_version: number; domain?: string; published_at?: string };

type Proposals = { proposals: Proposal[] };

function load<T>(allowed: boolean, path: string): Promise<ApiReply<T> | null> {
  return allowed ? adminApi<T>("GET", path) : Promise.resolve(null);
}

export default async function HomePage() {
  const t = await serverTranslator();
  const admin = await currentAdmin();
  if (!admin) {
    return (
      <main className="mx-auto flex max-w-3xl flex-col justify-center gap-6 px-6 py-24">
        <h1 className="text-3xl font-semibold break-words">{t("app.title")}</h1>
        <p className="text-text-secondary">{t("auth.required")}</p>
        <GoogleSignIn clientId={googleClientId()} />
      </main>
    );
  }
  const dashboards = canView(admin.permissions, "dashboards");
  const live = canView(admin.permissions, "live_controls");
  const [version, approvals, controls, unconfirmed, economy] = await Promise.all([
    load<ActiveVersion>(dashboards, "/admin/config/version"),
    load<Proposals>(true, "/admin/approvals"),
    load<LiveControlsView>(live, "/admin/live-controls"),
    load<Proposals>(live, "/admin/live-controls/break-glass"),
    load<EconomyReport>(dashboards, "/admin/economy/monitors"),
  ]);
  return (
    <main className="mx-auto flex max-w-5xl flex-col gap-8 px-6 py-10">
      <div className="flex flex-col gap-2">
        <h1 className="text-3xl font-semibold break-words">{t("app.title")}</h1>
        <dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1 text-sm">
          <dt className="text-text-muted">{t("home.signed_in_as")}</dt>
          <dd className="break-words">{admin.account.display_name}</dd>
          <dt className="text-text-muted">{t("home.roles")}</dt>
          <dd className="break-words">{roleNames(admin.account.roles, t)}</dd>
        </dl>
      </div>
      {live && (
        <Panel title={t("live.break_glass_unconfirmed")} reply={unconfirmed} t={t}>
          {(body) => <UnconfirmedBreakGlass proposals={body.proposals} t={t} />}
        </Panel>
      )}
      <Panel title={t("home.pending_approvals")} reply={approvals} t={t}>
        {(body) => (
          <p className="text-sm">
            {t.number(body.proposals.length)} ·{" "}
            <Link href="/approvals" prefetch={false} className="text-primary underline">
              {t("nav.approvals")}
            </Link>
          </p>
        )}
      </Panel>
      {dashboards && (
        <Panel title={t("home.config_version")} reply={version} t={t}>
          {(body) => (
            <p className="text-sm">
              <span className="font-mono">{body.config_version}</span>
              {body.domain && body.published_at && ` · ${body.domain} · ${t.when(body.published_at)}`}
            </p>
          )}
        </Panel>
      )}
      {live && (
        <Panel title={t("live.title")} reply={controls} t={t}>
          {(body) => <LiveStates view={body} t={t} />}
        </Panel>
      )}
      {dashboards && (
        <Panel title={t("analytics.title")} reply={economy} t={t}>
          {(body) => <Economy report={body} t={t} />}
        </Panel>
      )}
    </main>
  );
}

function Panel<T>({ title, reply, t, children }: { title: string; reply: ApiReply<T> | null; t: Translate; children: (body: T) => ReactNode }) {
  return (
    <section className="flex flex-col gap-2">
      <h2 className="font-semibold">{title}</h2>
      {reply?.status === 200 ? children(reply.body) : <p className="text-sm text-danger-text">{t("home.load_failed")}</p>}
    </section>
  );
}

function LiveStates({ view, t }: { view: LiveControlsView; t: Translate }) {
  const controls = view.controls;
  if (controls === null) {
    return <p className="text-sm text-text-secondary">{t("home.live_unpublished")}</p>;
  }
  const on = (value: boolean) => (value ? t("home.on") : t("home.off"));
  const rows: [string, string][] = [
    [t("live.purchases_disabled"), on(controls.purchases_disabled)],
    [t("live.ranked_disabled"), on(controls.ranked_disabled)],
    [t("live.build_allowlist_enforced"), on(controls.build_allowlist_enforced)],
    [t("live.minimum_app_version"), controls.minimum_app_version],
    [t("live.maintenance_ends_at"), controls.maintenance_ends_at ? t.when(controls.maintenance_ends_at) : t("home.off")],
  ];
  return (
    <dl className="grid grid-cols-[1fr_auto] gap-x-4 gap-y-1 text-sm">
      {rows.map(([label, value]) => (
        <div key={label} className="contents">
          <dt className="text-text-secondary">{label}</dt>
          <dd className="font-mono">{value}</dd>
        </div>
      ))}
      <dt className="text-text-muted">{t("live.version")}</dt>
      <dd className="font-mono">{view.config_version}</dd>
    </dl>
  );
}

function Economy({ report, t }: { report: EconomyReport; t: Translate }) {
  return (
    <div className="flex flex-col gap-2 text-sm">
      {report.alerts.length === 0 ? (
        <p className="text-text-secondary">{t("analytics.no_alerts")}</p>
      ) : (
        <ul className="flex flex-col gap-1 font-semibold text-danger-text">
          {report.alerts.map((alert) => (
            <li key={alert}>{t(alertLabels[alert])}</li>
          ))}
        </ul>
      )}
      <p>
        {t("analytics.coins_created")} {t.number(report.coinsCreated)} · {t("analytics.coins_spent")} {t.number(report.coinsSpent)} · {t("analytics.sink_ratio")}{" "}
        {report.sinkRatioPercent === null ? "–" : `${t.number(report.sinkRatioPercent)}%`} · {t("analytics.median_balance")} {t.number(report.medianBalance)} ·{" "}
        <Link href="/analytics" prefetch={false} className="text-primary underline">
          {t("nav.analytics")}
        </Link>
      </p>
    </div>
  );
}
