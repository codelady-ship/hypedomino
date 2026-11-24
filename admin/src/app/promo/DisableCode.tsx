"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import { needsStepUp } from "@/admin/refusals";
import { StepUpPrompt } from "@/app/auth/StepUpPrompt";
import { useTranslator } from "@/i18n/client";
import { disablePromo, type DisableResult } from "./actions";

export function DisableCode({ id, batch, clientId }: { id: string; batch: boolean; clientId: string }) {
  const t = useTranslator();
  const router = useRouter();
  const [reason, setReason] = useState("");
  const [result, setResult] = useState<DisableResult | null>(null);
  const [pending, startTransition] = useTransition();
  const stepUp = result !== null && !result.ok && needsStepUp(result.error);

  function send() {
    startTransition(async () => {
      const outcome = await disablePromo(id, batch, reason);
      setResult(outcome);
      if (outcome.ok) {
        router.refresh();
      }
    });
  }

  return (
    <div className="flex flex-col gap-1">
      <div className="flex gap-2">
        <input aria-label={t("promo.reason")} placeholder={t("promo.reason")} value={reason} onChange={(event) => setReason(event.target.value)} className="w-36 rounded-md bg-surface px-2 py-1 text-text-primary" />
        <button type="button" disabled={pending || reason.trim() === ""} onClick={send} className="rounded-md bg-surface px-3 py-1 whitespace-nowrap text-text-secondary disabled:opacity-40">
          {batch ? t("promo.disable_batch") : t("promo.disable")}
        </button>
      </div>
      {stepUp && <StepUpPrompt clientId={clientId} onConfirmed={send} />}
      {result && !result.ok && !stepUp && (
        <p role="status" className="text-text-secondary">
          {t("promo.refused")}: {result.error} {result.detail}
        </p>
      )}
    </div>
  );
}
