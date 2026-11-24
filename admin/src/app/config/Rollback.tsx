"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import { needsStepUp } from "@/admin/refusals";
import { StepUpPrompt } from "@/app/auth/StepUpPrompt";
import { useTranslator } from "@/i18n/client";
import type { ConfigResult } from "@/config/result";
import { UnnamedKeys } from "@/app/strings/UnnamedKeys";
import { ItemsWithoutArt } from "@/app/assets/ItemsWithoutArt";

export function Rollback({ version, clientId, action }: { version: number; clientId: string; action: (version: number, reason: string) => Promise<ConfigResult> }) {
  const t = useTranslator();
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState("");
  const [result, setResult] = useState<ConfigResult | null>(null);
  const [pending, startTransition] = useTransition();
  const stepUp = result !== null && !result.ok && needsStepUp(result.error);

  function rollBack() {
    startTransition(async () => {
      const outcome = await action(version, reason);
      setResult(outcome);
      if (outcome.ok) {
        router.refresh();
      }
    });
  }

  if (!open) {
    return (
      <button type="button" onClick={() => setOpen(true)} className="rounded-md bg-surface px-3 py-1 text-text-secondary">
        {t("rollback.open")}
      </button>
    );
  }
  return (
    <div className="flex flex-col gap-2">
      <label className="flex flex-col gap-1 text-text-secondary">
        {t("rollback.reason")}
        <input value={reason} onChange={(event) => setReason(event.target.value)} className="rounded-md bg-surface px-3 py-1 text-text-primary" />
      </label>
      <button
        type="button"
        disabled={pending || reason.trim() === ""}
        onClick={rollBack}
        className="self-start rounded-md bg-primary px-3 py-1 text-on-primary disabled:opacity-40"
      >
        {t("rollback.confirm")}
      </button>
      {stepUp && <StepUpPrompt clientId={clientId} onConfirmed={rollBack} />}
      {result && !stepUp && (
        <p role="status" className="text-text-secondary">
          {result.ok ? `${t("rollback.done")}: ${result.version}` : `${t("rollback.refused")}: ${result.error} ${result.detail}`}
        </p>
      )}
      {result && !result.ok && !stepUp && <UnnamedKeys keys={result.keys} />}
      {result && !result.ok && !stepUp && <ItemsWithoutArt error={result.error} detail={result.detail} />}
    </div>
  );
}
