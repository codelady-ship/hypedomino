"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import { needsStepUp } from "@/admin/refusals";
import { StepUpPrompt } from "@/app/auth/StepUpPrompt";
import { useTranslator } from "@/i18n/client";
import { decideAppeal, type AppealChoice, type AppealResult } from "./actions";

export function AppealDecision({ appealId, clientId }: { appealId: number; clientId: string }) {
  const t = useTranslator();
  const router = useRouter();
  const [reason, setReason] = useState("");
  const [last, setLast] = useState<AppealChoice | null>(null);
  const [result, setResult] = useState<AppealResult | null>(null);
  const [pending, startTransition] = useTransition();
  const stepUp = result !== null && !result.ok && needsStepUp(result.error);
  const blocked = pending || reason.trim() === "";

  function send(choice: AppealChoice) {
    setLast(choice);
    startTransition(async () => {
      const outcome = await decideAppeal(appealId, choice, reason);
      setResult(outcome);
      if (outcome.ok) {
        router.refresh();
      }
    });
  }

  return (
    <div className="flex flex-col gap-2">
      <input aria-label={t("appeals.reason")} placeholder={t("appeals.reason")} value={reason} onChange={(event) => setReason(event.target.value)} className="rounded-md bg-surface px-3 py-1 text-text-primary" />
      <div className="flex gap-2">
        <button type="button" disabled={blocked} onClick={() => send("uphold")} className="rounded-md bg-surface px-3 py-1 text-text-secondary disabled:opacity-40">
          {t("appeals.uphold")}
        </button>
        <button type="button" disabled={blocked} onClick={() => send("lift")} className="rounded-md bg-primary px-3 py-1 text-on-primary disabled:opacity-40">
          {t("appeals.lift")}
        </button>
      </div>
      {stepUp && last && <StepUpPrompt clientId={clientId} onConfirmed={() => send(last)} />}
      {result && !result.ok && !stepUp && (
        <p role="status" className="text-text-secondary">
          {t("appeals.refused")}: {result.error} {result.detail}
        </p>
      )}
    </div>
  );
}
