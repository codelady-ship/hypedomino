import Link from "next/link";
import { adminApi } from "@/admin/api";
import { googleClientId } from "@/admin/google";
import { can, canView } from "@/admin/navigation";
import { currentAdmin } from "@/admin/session";
import type { SearchParams } from "@/audit/query";
import { serverTranslator } from "@/i18n/server";
import type { Translate } from "@/i18n/translate";
import { batchLeads, bundleSummary, codeGrants, olderPromoHref, promoApiPath, promoStates, readPromoQuery, shownState, templatesPath, type CampaignUsage, type PromoCode, type PromoQuery, type PromoTemplate } from "@/promo/codes";
import { Notice } from "../players/Notice";
import { DisableCode } from "./DisableCode";
import { PromoCreator } from "./PromoCreator";

export const dynamic = "force-dynamic";

function FilterForm({ query, t }: { query: PromoQuery; t: Translate }) {
  return (
    <form method="get" action="/promo" className="flex flex-wrap items-end gap-4">
      {query.campaign && <input type="hidden" name="campaign" value={query.campaign} />}
      <label className="flex flex-col gap-1 text-sm text-text-secondary">
        {t("promo.id_column")}
        <input name="id" defaultValue={query.id} className="rounded-md bg-surface px-3 py-2 font-mono text-text-primary" />
      </label>
      <label className="flex flex-col gap-1 text-sm text-text-secondary">
        {t("promo.state")}
        <select name="state" defaultValue={query.state} className="rounded-md bg-surface px-3 py-2 text-text-primary">
          <option value="">{t("promo.any_state")}</option>
          {promoStates.map((state) => (
            <option key={state} value={state}>
              {state}
            </option>
          ))}
        </select>
      </label>
      <button type="submit" className="rounded-md bg-primary px-4 py-2 font-semibold text-on-primary">
        {t("promo.search")}
      </button>
      <Link href={query.campaign ? `/promo?campaign=${encodeURIComponent(query.campaign)}` : "/promo"} prefetch={false} className="py-2 text-text-secondary">
        {t("promo.clear")}
      </Link>
    </form>
  );
}

export default async function PromoPage({ searchParams }: { searchParams: Promise<SearchParams> }) {
  const t = await serverTranslator();
  const admin = await currentAdmin();
  if (!admin) {
    return <Notice text={t("auth.required")} />;
  }
  if (!canView(admin.permissions, "promo_codes")) {
    return <Notice text={t("auth.forbidden")} />;
  }
  const query = readPromoQuery(await searchParams);
  const { campaign } = query;
  const [reply, templateReply] = await Promise.all([
    adminApi<{ campaigns: CampaignUsage[]; codes: PromoCode[] }>("GET", promoApiPath(query)),
    adminApi<{ templates: PromoTemplate[] }>("GET", templatesPath),
  ]);
  if (reply.status !== 200 || templateReply.status !== 200) {
    return <Notice text={reply.body.error === "not_configured" ? t("promo.not_configured") : t("promo.load_failed")} />;
  }
  const { campaigns, codes } = reply.body;
  const { templates } = templateReply.body;
  const mayDisable = can(admin.permissions, "promo_codes", "disable_promo");
  const now = new Date();
  const leads = batchLeads(codes);
  const older = olderPromoHref(query, codes);
  const filtered = Boolean(query.state || query.id || query.before);
  return (
    <main className="mx-auto flex max-w-7xl flex-col gap-8 px-6 py-10">
      <div className="flex flex-col gap-2">
        <h1 className="text-2xl font-semibold">{t("promo.title")}</h1>
        <p className="text-sm text-text-secondary">{t("promo.explain")}</p>
      </div>
      {can(admin.permissions, "promo_codes", "create") && <PromoCreator templates={templates} mayEditTemplates={can(admin.permissions, "promo_codes", "edit")} />}
      <section className="flex flex-col gap-3">
        <h2 className="text-lg font-semibold">{t("promo.templates")}</h2>
        <p className="text-sm text-text-secondary">{t("promo.templates_explain")}</p>
        {templates.length === 0 ? (
          <p className="text-text-secondary">{t("promo.templates_none")}</p>
        ) : (
          <table className="text-sm">
            <thead className="text-left text-text-muted">
              <tr>
                <th className="py-2 pr-4">{t("promo.id_column")}</th>
                <th className="py-2 pr-4">{t("promo.campaign_column")}</th>
                <th className="py-2 pr-4">{t("promo.kind")}</th>
                <th className="py-2 pr-4">{t("promo.grants")}</th>
                <th className="py-2 pr-4">{t("promo.template_creator")}</th>
                <th className="py-2 pr-4">{t("promo.updated")}</th>
              </tr>
            </thead>
            <tbody>
              {templates.map((template) => (
                <tr key={template.id} className="border-t border-surface align-top">
                  <td className="py-2 pr-4 font-mono break-all">{template.id}</td>
                  <td className="py-2 pr-4 font-mono">{template.campaign}</td>
                  <td className="py-2 pr-4">{template.kind}</td>
                  <td className="py-2 pr-4">{template.bundle ? bundleSummary(template.bundle) : t("promo.template_no_bundle")}</td>
                  <td className="py-2 pr-4 break-all">{template.creator}</td>
                  <td className="py-2 pr-4 whitespace-nowrap">{t.when(template.updated)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
      <section className="flex flex-col gap-3">
        <h2 className="text-lg font-semibold">{t("promo.usage")}</h2>
        {campaigns.length === 0 ? (
          <p className="text-text-secondary">{t("promo.none")}</p>
        ) : (
          <table className="text-sm">
            <thead className="text-left text-text-muted">
              <tr>
                <th className="py-2 pr-4">{t("promo.campaign_column")}</th>
                <th className="py-2 pr-4">{t("promo.codes")}</th>
                <th className="py-2 pr-4">{t("promo.active")}</th>
                <th className="py-2 pr-4">{t("promo.pending")}</th>
                <th className="py-2 pr-4">{t("promo.disabled")}</th>
                <th className="py-2 pr-4">{t("promo.redemptions")}</th>
                <th className="py-2 pr-4">{t("promo.window")}</th>
              </tr>
            </thead>
            <tbody>
              {campaigns.map((usage) => (
                <tr key={usage.campaign} className="border-t border-surface">
                  <td className="py-2 pr-4 font-mono">
                    <Link href={`/promo?campaign=${encodeURIComponent(usage.campaign)}`} prefetch={false} className="text-primary underline">
                      {usage.campaign}
                    </Link>
                  </td>
                  <td className="py-2 pr-4">{usage.codes}</td>
                  <td className="py-2 pr-4">{usage.active}</td>
                  <td className="py-2 pr-4">{usage.pending}</td>
                  <td className="py-2 pr-4">{usage.disabled}</td>
                  <td className="py-2 pr-4">
                    {usage.redemptions} / {usage.cap}
                  </td>
                  <td className="py-2 pr-4 whitespace-nowrap">
                    {t.when(usage.start)} → {t.when(usage.end)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
      <section className="flex flex-col gap-3">
        <div className="flex items-baseline gap-4">
          <h2 className="text-lg font-semibold">{campaign ? `${t("promo.codes_of")} ${campaign}` : t("promo.newest")}</h2>
          {campaign && (
            <Link href="/promo" prefetch={false} className="text-sm text-primary underline">
              {t("promo.all_campaigns")}
            </Link>
          )}
        </div>
        <FilterForm query={query} t={t} />
        {codes.length === 0 && filtered && <p className="text-text-secondary">{t("promo.empty_match")}</p>}
        {codes.length > 0 && (
          <table className="text-sm">
            <thead className="text-left text-text-muted">
              <tr>
                <th className="py-2 pr-4">{t("promo.id_column")}</th>
                <th className="py-2 pr-4">{t("promo.kind")}</th>
                <th className="py-2 pr-4">{t("promo.state")}</th>
                <th className="py-2 pr-4">{t("promo.grants")}</th>
                <th className="py-2 pr-4">{t("promo.window")}</th>
                <th className="py-2 pr-4">{t("promo.redemptions")}</th>
                <th className="py-2 pr-4">{t("promo.creator")}</th>
                {mayDisable && <th className="py-2 pr-4" />}
              </tr>
            </thead>
            <tbody>
              {codes.map((code) => (
                <tr key={code.id} className="border-t border-surface align-top">
                  <td className="py-2 pr-4 font-mono break-all">
                    <Link href={`/promo/${encodeURIComponent(code.id)}`} prefetch={false} className="text-primary underline">
                      {code.id}
                    </Link>
                  </td>
                  <td className="py-2 pr-4">{code.kind}</td>
                  <td className="py-2 pr-4">{shownState(code, now)}</td>
                  <td className="py-2 pr-4">{codeGrants(code, t("promo.pass_premium"))}</td>
                  <td className="py-2 pr-4 whitespace-nowrap">
                    {t.when(code.start)} → {t.when(code.end)}
                  </td>
                  <td className="py-2 pr-4">
                    {code.redemptions} / {code.cap}
                  </td>
                  <td className="py-2 pr-4 break-all">
                    {code.creator}
                    {code.approver && ` / ${code.approver}`}
                  </td>
                  {mayDisable && (
                    <td className="flex flex-col gap-2 py-2 pr-4">
                      {code.state !== "disabled" && <DisableCode id={code.id} batch={false} clientId={googleClientId()} />}
                      {code.batch && leads.has(code.id) && <DisableCode id={code.batch} batch clientId={googleClientId()} />}
                    </td>
                  )}
                </tr>
              ))}
            </tbody>
          </table>
        )}
        {older && (
          <Link href={older} prefetch={false} className="self-start rounded-md bg-surface px-4 py-2 text-text-secondary">
            {t("promo.older")}
          </Link>
        )}
      </section>
    </main>
  );
}
