"use client";

import { useState, useTransition } from "react";
import { needsStepUp } from "@/admin/refusals";
import { StepUpPrompt } from "@/app/auth/StepUpPrompt";
import type { AuditFilter } from "@/audit/query";
import { useTranslator } from "@/i18n/client";
import { exportAudit, type ExportResult } from "./actions";

function download(json: string) {
  const link = document.createElement("a");
  link.href = URL.createObjectURL(new Blob([json], { type: "application/json" }));
  link.download = "audit.json";
  link.click();
  URL.revokeObjectURL(link.href);
}

export function ExportAudit({ filter, clientId }: { filter: AuditFilter; clientId: string }) {
  const t = useTranslator();
  const [reason, setReason] = useState("");
  const [result, setResult] = useState<ExportResult | null>(null);
  const [pending, startTransition] = useTransition();
  const stepUp = result !== null && !result.ok && needsStepUp(result.error);

  function send() {
    startTransition(async () => {
      const outcome = await exportAudit(filter, reason);
      setResult(outcome);
      if (outcome.ok) {
        download(outcome.json);
      }
    });
  }

  return (
    <div className="flex flex-col gap-2 self-start">
      <div className="flex gap-2">
        <input aria-label={t("audit.export_reason")} placeholder={t("audit.export_reason")} value={reason} onChange={(event) => setReason(event.target.value)} className="w-64 rounded-md bg-surface px-3 py-2 text-text-primary" />
        <button type="button" disabled={pending || reason.trim() === ""} onClick={send} className="rounded-md bg-surface px-4 py-2 whitespace-nowrap text-text-secondary disabled:opacity-40">
          {t("audit.export")}
        </button>
      </div>
      {stepUp && <StepUpPrompt clientId={clientId} onConfirmed={send} />}
      {result && !result.ok && !stepUp && (
        <p role="status" className="text-text-secondary">
          {t("audit.export_refused")}: {result.error} {result.detail}
        </p>
      )}
    </div>
  );
}
