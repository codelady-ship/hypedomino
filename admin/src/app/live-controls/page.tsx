import { adminApi } from "@/admin/api";
import { googleClientId } from "@/admin/google";
import { can, canView } from "@/admin/navigation";
import { currentAdmin } from "@/admin/session";
import { unpublishedControls, type LiveControlsView } from "@/config/liveControls";
import { serverTranslator } from "@/i18n/server";
import type { Proposal } from "@/approvals/proposals";
import { LiveControlsEditor } from "./LiveControlsEditor";
import { UnconfirmedBreakGlass } from "./UnconfirmedBreakGlass";

export const dynamic = "force-dynamic";

export default async function LiveControlsPage() {
  const t = await serverTranslator();
  const admin = await currentAdmin();
  if (!admin) {
    return <Notice text={t("auth.required")} />;
  }
  if (!canView(admin.permissions, "live_controls")) {
    return <Notice text={t("auth.forbidden")} />;
  }
  const [current, unconfirmed] = await Promise.all([
    adminApi<LiveControlsView>("GET", "/admin/live-controls"),
    adminApi<{ proposals: Proposal[] }>("GET", "/admin/live-controls/break-glass"),
  ]);
  if (current.status !== 200 || unconfirmed.status !== 200) {
    return <Notice text={t("live.load_failed")} />;
  }
  const canSwitch = can(admin.permissions, "live_controls", "switch_live_control");
  return (
    <main className="mx-auto flex max-w-3xl flex-col gap-8 px-6 py-10">
      <div className="flex flex-col gap-2">
        <h1 className="text-2xl font-semibold">{t("live.title")}</h1>
        <p className="text-sm text-text-muted">
          {t("live.version")}: {current.body.config_version}
        </p>
        <p className="text-sm text-text-secondary">{t("live.explain")}</p>
      </div>
      <section className="flex flex-col gap-2">
        <h2 className="font-semibold">{t("live.break_glass_unconfirmed")}</h2>
        <UnconfirmedBreakGlass proposals={unconfirmed.body.proposals} t={t} />
      </section>
      <LiveControlsEditor published={current.body.controls ?? unpublishedControls} clientId={canSwitch ? googleClientId() : null} />
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
