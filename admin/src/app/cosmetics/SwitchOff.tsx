"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import { needsStepUp } from "@/admin/refusals";
import { StepUpPrompt } from "@/app/auth/StepUpPrompt";
import type { ConfigResult } from "@/config/result";
import { useTranslator } from "@/i18n/client";
import { switchOffCosmetic } from "./actions";

const fieldClass = "rounded-md bg-surface px-3 py-2 text-text-primary";

export function SwitchOff({ itemIds, clientId }: { itemIds: readonly string[]; clientId: string }) {
  const t = useTranslator();
  const router = useRouter();
  const [id, setId] = useState(itemIds[0] ?? "");
  const [reason, setReason] = useState("");
  const [incident, setIncident] = useState("");
  const [result, setResult] = useState<ConfigResult | null>(null);
  const [pending, startTransition] = useTransition();
  const stepUp = result !== null && !result.ok && needsStepUp(result.error);

  function send() {
    startTransition(async () => {
      const outcome = await switchOffCosmetic(id, reason, incident);
      setResult(outcome);
      if (outcome.ok) {
        router.refresh();
      }
    });
  }

  return (
    <section className="flex flex-col gap-3 rounded-md bg-background-raised p-4">
      <h2 className="text-lg font-semibold">{t("cosmetics.switch_off")}</h2>
      <div className="flex flex-wrap gap-2">
        <select aria-label={t("cosmetics.switch_off_item")} value={id} onChange={(event) => setId(event.target.value)} className={fieldClass}>
          {itemIds.map((itemId) => (
            <option key={itemId} value={itemId}>
              {itemId}
            </option>
          ))}
        </select>
        <input aria-label={t("cosmetics.switch_off_incident")} placeholder={t("cosmetics.switch_off_incident")} value={incident} onChange={(event) => setIncident(event.target.value)} className={fieldClass} />
        <input aria-label={t("cosmetics.reason")} placeholder={t("cosmetics.reason")} value={reason} onChange={(event) => setReason(event.target.value)} className={`${fieldClass} w-64`} />
        <button type="button" disabled={pending || !id || reason.trim() === "" || incident.trim() === ""} onClick={send} className="rounded-md bg-surface px-4 py-2 text-text-secondary disabled:opacity-40">
          {t("cosmetics.switch_off")}
        </button>
      </div>
      {stepUp && <StepUpPrompt clientId={clientId} onConfirmed={send} />}
      {result && !result.ok && !stepUp && (
        <p role="status" className="text-text-secondary">
          {t("cosmetics.refused")}: {result.error} {result.detail}
        </p>
      )}
    </section>
  );
}
