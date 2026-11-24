"use client";

import { stepUpRefusalKey } from "@/admin/refusals";
import { useTranslator } from "@/i18n/client";
import { completeStepUp } from "./actions";
import { GoogleButton } from "./GoogleButton";

export function StepUpPrompt({ clientId, onConfirmed }: { clientId: string; onConfirmed: () => void }) {
  const t = useTranslator();
  return (
    <section role="dialog" aria-label={t("stepup.title")} className="flex flex-col gap-3 rounded-md bg-surface p-4">
      <h3 className="font-semibold">{t("stepup.title")}</h3>
      <p className="text-sm text-text-secondary">{t("stepup.explain")}</p>
      <GoogleButton
        clientId={clientId}
        onCredential={async (credential, nonce) => {
          const refusal = stepUpRefusalKey((await completeStepUp(credential, nonce)).error);
          if (!refusal) {
            onConfirmed();
          }
          return refusal;
        }}
      />
    </section>
  );
}
