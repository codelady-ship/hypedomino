import Link from "next/link";
import { adminApi } from "@/admin/api";
import { googleClientId } from "@/admin/google";
import { can, canView } from "@/admin/navigation";
import { currentAdmin } from "@/admin/session";
import { serverTranslator } from "@/i18n/server";
import { codeGrants, detailPath, shownState, type PromoDetail } from "@/promo/codes";
import { Notice } from "../../players/Notice";
import { TargetsEditor } from "./TargetsEditor";

export const dynamic = "force-dynamic";

export default async function PromoCodePage({ params }: { params: Promise<{ id: string }> }) {
  const t = await serverTranslator();
  const admin = await currentAdmin();
  if (!admin) {
    return <Notice text={t("auth.required")} />;
  }
  if (!canView(admin.permissions, "promo_codes")) {
    return <Notice text={t("auth.forbidden")} />;
  }
  const { id } = await params;
  const reply = await adminApi<PromoDetail>("GET", detailPath(id));
  if (reply.status === 404) {
    return <Notice text={t("promo.not_found")} />;
  }
  if (reply.status !== 200) {
    return <Notice text={reply.body.error === "not_configured" ? t("promo.not_configured") : t("promo.load_failed")} />;
  }
  const { code, personal } = reply.body;
  const rows: [string, string][] = [
    [t("promo.campaign_column"), code.campaign],
    [t("promo.kind"), code.kind],
    [t("promo.state"), shownState(code, new Date())],
    [t("promo.grants"), codeGrants(code, t("promo.pass_premium"))],
    [t("promo.window"), `${t.when(code.start)} → ${t.when(code.end)}`],
    [t("promo.redemptions"), `${code.redemptions} / ${code.cap}`],
    [t("promo.creator"), code.approver ? `${code.creator} / ${code.approver}` : code.creator],
  ];
  return (
    <main className="mx-auto flex max-w-5xl flex-col gap-8 px-6 py-10">
      <div className="flex flex-col gap-2">
        <Link href="/promo" prefetch={false} className="text-sm text-primary underline">
          {t("promo.back")}
        </Link>
        <h1 className="font-mono text-2xl font-semibold break-all">{code.id}</h1>
      </div>
      <dl className="grid grid-cols-[max-content_1fr] gap-x-6 gap-y-2 text-sm">
        {rows.map(([label, value]) => (
          <div key={label} className="contents">
            <dt className="text-text-muted">{label}</dt>
            <dd className="break-all">{value}</dd>
          </div>
        ))}
      </dl>
      {!personal ? (
        <p className="text-sm text-text-secondary">{t("promo.personal_hidden")}</p>
      ) : (
        <>
          {code.kind === "targeted" && code.state !== "disabled" && can(admin.permissions, "promo_codes", "edit_targeted_accounts") && (
            <TargetsEditor id={code.id} accounts={personal.accounts} redeemed={personal.redeemed} clientId={googleClientId()} />
          )}
          <section className="flex flex-col gap-3">
            <h2 className="text-lg font-semibold">{t("promo.redeemed_title")}</h2>
            {personal.redeemed.length === 0 ? (
              <p className="text-text-secondary">{t("promo.redeemed_none")}</p>
            ) : (
              <table className="text-sm">
                <thead className="text-left text-text-muted">
                  <tr>
                    <th className="py-2 pr-4">{t("promo.account")}</th>
                    <th className="py-2 pr-4">{t("promo.redeemed_at")}</th>
                  </tr>
                </thead>
                <tbody>
                  {personal.redeemed.map((entry) => (
                    <tr key={entry.account_id} className="border-t border-surface">
                      <td className="py-2 pr-4 font-mono">
                        <Link href={`/players/${encodeURIComponent(entry.account_id)}`} prefetch={false} className="text-primary underline">
                          {entry.account_id}
                        </Link>
                      </td>
                      <td className="py-2 pr-4 whitespace-nowrap">{t.when(entry.redeemed_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </section>
        </>
      )}
    </main>
  );
}
