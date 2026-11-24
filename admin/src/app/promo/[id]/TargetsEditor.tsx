"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import { needsStepUp } from "@/admin/refusals";
import { StepUpPrompt } from "@/app/auth/StepUpPrompt";
import { useTranslator } from "@/i18n/client";
import { keepsRedeemers, listOf, type Redeemed } from "@/promo/codes";
import { setTargets, type TargetsResult } from "../actions";

export function TargetsEditor({ id, accounts, redeemed, clientId }: { id: string; accounts: string[]; redeemed: Redeemed[]; clientId: string }) {
  const t = useTranslator();
  const router = useRouter();
  const [text, setText] = useState(accounts.join("\n"));
  const [reason, setReason] = useState("");
  const [result, setResult] = useState<TargetsResult | null>(null);
  const [pending, startTransition] = useTransition();
  const stepUp = result !== null && !result.ok && needsStepUp(result.error);
  const keeps = keepsRedeemers(listOf(text), redeemed);

  function save() {
    startTransition(async () => {
      const outcome = await setTargets(id, text, reason);
      setResult(outcome);
      if (outcome.ok) {
        router.refresh();
      }
    });
  }

  return (
    <section className="flex flex-col gap-3 rounded-md border border-surface p-4">
      <h2 className="text-lg font-semibold">{t("promo.accounts_title")}</h2>
      <p className="text-sm text-text-muted">{t("promo.accounts_hint")}</p>
      <textarea aria-label={t("promo.accounts_title")} rows={8} value={text} onChange={(event) => setText(event.target.value)} className="rounded-md bg-surface px-3 py-2 font-mono text-sm text-text-primary" />
      <label className="flex flex-col gap-1 text-sm text-text-secondary">
        {t("promo.reason")}
        <input value={reason} onChange={(event) => setReason(event.target.value)} className="rounded-md bg-surface px-3 py-1 text-text-primary" />
      </label>
      {!keeps && (
        <p role="alert" className="text-sm text-danger-text">
          {t("promo.keep_redeemers")}
        </p>
      )}
      <button type="button" disabled={pending || !keeps || reason.trim() === ""} onClick={save} className="self-start rounded-md bg-primary px-4 py-2 text-on-primary disabled:opacity-40">
        {t("promo.save_accounts")}
      </button>
      {stepUp && <StepUpPrompt clientId={clientId} onConfirmed={save} />}
      {result && !stepUp && (
        <p role="status" className="text-sm text-text-secondary">
          {result.ok ? `${t("promo.accounts_saved")}: ${result.accounts} (+${result.added} / -${result.removed})` : `${t("promo.refused")}: ${result.error} ${result.detail}`}
        </p>
      )}
    </section>
  );
}
