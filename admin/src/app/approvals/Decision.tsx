"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import { needsStepUp } from "@/admin/refusals";
import { StepUpPrompt } from "@/app/auth/StepUpPrompt";
import { useTranslator } from "@/i18n/client";
import { decide, type DecisionResult } from "./actions";

type Choice = "approve" | "reject";

export function Decision({ id, clientId }: { id: number; clientId: string }) {
  const t = useTranslator();
  const router = useRouter();
  const [reason, setReason] = useState("");
  const [last, setLast] = useState<Choice | null>(null);
  const [result, setResult] = useState<DecisionResult | null>(null);
  const [pending, startTransition] = useTransition();
  const stepUp = result !== null && !result.ok && needsStepUp(result.error);

  function send(choice: Choice) {
    setLast(choice);
    startTransition(async () => {
      const outcome = await decide(id, choice, reason);
      setResult(outcome);
      if (outcome.ok) {
        router.refresh();
      }
    });
  }

  return (
    <div className="flex flex-col gap-2">
      <input aria-label={t("approvals.note")} placeholder={t("approvals.note")} value={reason} onChange={(event) => setReason(event.target.value)} className="rounded-md bg-surface px-3 py-1 text-text-primary" />
      <div className="flex gap-2">
        <button type="button" disabled={pending || reason.trim() === ""} onClick={() => send("approve")} className="rounded-md bg-primary px-3 py-1 text-on-primary disabled:opacity-40">
          {t("approvals.approve")}
        </button>
        <button type="button" disabled={pending || reason.trim() === ""} onClick={() => send("reject")} className="rounded-md bg-surface px-3 py-1 text-text-secondary disabled:opacity-40">
          {t("approvals.reject")}
        </button>
      </div>
      {stepUp && last && <StepUpPrompt clientId={clientId} onConfirmed={() => send(last)} />}
      {result && !result.ok && !stepUp && (
        <p role="status" className="text-text-secondary">
          {t("approvals.refused")}: {result.error} {result.detail}
        </p>
      )}
    </div>
  );
}
