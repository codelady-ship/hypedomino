"use client";

import { useState, useTransition } from "react";
import { needsStepUp } from "@/admin/refusals";
import { StepUpPrompt } from "@/app/auth/StepUpPrompt";
import { useTranslator } from "@/i18n/client";
import { formatUsd } from "@/storeprices/prices";
import { pendingChanges, shownPrice, statusCounts, visibleRows, type PushPlan, type PushResult, type StoreReport } from "@/storeprices/push";
import { applyStorePush, planStorePush } from "./actions";

export function StorePush({ applyClientId }: { applyClientId: string | null }) {
  const t = useTranslator();
  const [plan, setPlan] = useState<PushPlan | null>(null);
  const [applied, setApplied] = useState(false);
  const [refusal, setRefusal] = useState<PushResult | null>(null);
  const [pending, startTransition] = useTransition();

  function run(request: () => Promise<PushResult>, applying: boolean) {
    startTransition(async () => {
      const outcome = await request();
      setRefusal(outcome.ok ? null : outcome);
      if (outcome.ok) {
        setPlan(outcome.plan);
        setApplied(applying);
      }
    });
  }

  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-lg font-semibold">{t("store_prices.push.title")}</h2>
      <p className="text-sm text-text-muted">{t("store_prices.push.hint")}</p>
      <button type="button" disabled={pending} onClick={() => run(planStorePush, false)} className="self-start rounded-md bg-surface px-3 py-1 text-text-secondary disabled:opacity-40">
        {pending ? t("store_prices.push.working") : t("store_prices.push.plan")}
      </button>
      {refusal && !refusal.ok && !needsStepUp(refusal.error) && (
        <p role="status" className="text-danger-text">
          {t("store_prices.refused")}: {refusal.error} {refusal.detail}
        </p>
      )}
      {plan && (
        <>
          <p className="text-sm text-text-secondary">
            {t(applied ? "store_prices.push.applied" : "store_prices.push.planned")} {plan.config_version}
          </p>
          {plan.stores.map((report) => (
            <StoreTable key={report.store} report={report} />
          ))}
        </>
      )}
      {plan && !applied && applyClientId && pendingChanges(plan) > 0 && (
        <ApplyPush clientId={applyClientId} pending={pending} stepUp={refusal !== null && !refusal.ok && needsStepUp(refusal.error)} onApply={(reason) => run(() => applyStorePush(reason), true)} />
      )}
    </section>
  );
}

function ApplyPush({ clientId, pending, stepUp, onApply }: { clientId: string; pending: boolean; stepUp: boolean; onApply: (reason: string) => void }) {
  const t = useTranslator();
  const [reason, setReason] = useState("");
  const [confirmed, setConfirmed] = useState(false);
  return (
    <div className="flex flex-col gap-2 rounded-md border border-surface p-3">
      <label className="flex flex-col gap-1 text-text-secondary">
        {t("store_prices.reason")}
        <input value={reason} onChange={(event) => setReason(event.target.value)} className="rounded-md bg-surface px-3 py-1 text-text-primary" />
      </label>
      <label className="flex items-center gap-2 text-sm text-text-secondary">
        <input type="checkbox" checked={confirmed} onChange={(event) => setConfirmed(event.target.checked)} />
        {t("store_prices.push.confirm")}
      </label>
      <button
        type="button"
        disabled={pending || !confirmed || reason.trim() === ""}
        onClick={() => onApply(reason)}
        className="self-start rounded-md bg-primary px-3 py-1 text-on-primary disabled:opacity-40"
      >
        {t("store_prices.push.apply")}
      </button>
      {stepUp && <StepUpPrompt clientId={clientId} onConfirmed={() => onApply(reason)} />}
    </div>
  );
}

function StoreTable({ report }: { report: StoreReport }) {
  const t = useTranslator();
  const [onlyChanges, setOnlyChanges] = useState(true);
  const rows = visibleRows(report.rows, onlyChanges);
  return (
    <div className="flex flex-col gap-2">
      <h3 className="font-semibold">{t(`store_prices.store.${report.store}`)}</h3>
      {!report.configured ? (
        <p className="text-sm text-text-muted">{t("store_prices.push.not_configured")}</p>
      ) : (
        <>
          <p className="text-sm text-text-secondary">
            {statusCounts(report)
              .map(([status, count]) => `${t(`store_prices.push.status.${status}`)}: ${count}`)
              .join(" · ")}
          </p>
          <label className="flex items-center gap-2 text-sm text-text-secondary">
            <input type="checkbox" checked={onlyChanges} onChange={(event) => setOnlyChanges(event.target.checked)} />
            {t("store_prices.push.only_changes")}
          </label>
          {rows.length > 0 && (
            <table className="text-sm">
              <thead className="text-left text-text-muted">
                <tr>
                  <th className="py-1 pr-4">{t("store_prices.product")}</th>
                  <th className="py-1 pr-4">{t("store_prices.country")}</th>
                  <th className="py-1 pr-4">{t("store_prices.target_usd")}</th>
                  <th className="py-1 pr-4">{t("store_prices.push.current")}</th>
                  <th className="py-1 pr-4">{t("store_prices.push.next")}</th>
                  <th className="py-1 pr-4">{t("store_prices.push.status")}</th>
                </tr>
              </thead>
              <tbody>
                {rows.map((row) => (
                  <tr key={`${row.product}/${row.country}`} className="border-t border-surface align-top">
                    <td className="py-1 pr-4">{row.product}</td>
                    <td className="py-1 pr-4">{row.country}</td>
                    <td className="py-1 pr-4">{formatUsd(row.target_cents)}</td>
                    <td className="py-1 pr-4 whitespace-nowrap">{shownPrice(row.current)}</td>
                    <td className="py-1 pr-4 whitespace-nowrap">{shownPrice(row.next)}</td>
                    <td className="py-1 pr-4">
                      {t(`store_prices.push.status.${row.status}`)}
                      {row.detail && <span className="block text-text-muted">{row.detail}</span>}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </>
      )}
    </div>
  );
}
