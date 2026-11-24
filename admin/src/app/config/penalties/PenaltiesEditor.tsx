"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import type { ConfigResult } from "@/config/result";
import { useTranslator } from "@/i18n/client";
import { badFields, configFromFields, decimalFields, fieldsFromConfig, listFields, wholeFields, type PenaltiesConfig } from "@/penalties/config";
import { publishPenalties } from "./actions";

const fieldClass = "rounded-md bg-surface px-3 py-1 text-text-primary";
const labelClass = "flex flex-col gap-1 text-sm text-text-secondary";

export function PenaltiesEditor({ published, canPublish }: { published: PenaltiesConfig | null; canPublish: boolean }) {
  const t = useTranslator();
  const router = useRouter();
  const [fields, setFields] = useState(fieldsFromConfig(published));
  const [reason, setReason] = useState("");
  const [result, setResult] = useState<ConfigResult | null>(null);
  const [pending, startTransition] = useTransition();
  const bad = badFields(fields);
  const changed = JSON.stringify(configFromFields(fields)) !== JSON.stringify(configFromFields(fieldsFromConfig(published)));
  const ready = !pending && bad.length === 0 && changed && reason.trim() !== "";
  const set = (name: keyof typeof fields, value: string) => setFields({ ...fields, [name]: value });
  return (
    <section className="flex flex-col gap-4">
      <div className="grid gap-4 sm:grid-cols-2">
        {[...wholeFields, ...decimalFields, ...listFields].map((name) => (
          <label key={name} className={labelClass}>
            <span className="font-mono">{name}</span>
            <input inputMode={(listFields as readonly string[]).includes(name) ? "text" : "decimal"} readOnly={!canPublish} placeholder={t("penalties.default")} value={fields[name]} onChange={(event) => set(name, event.target.value)} className={fieldClass} />
          </label>
        ))}
      </div>
      {bad.length > 0 && (
        <p role="alert" className="text-sm text-danger-text">
          {t("penalties.bad_fields")}: {bad.join(", ")}
        </p>
      )}
      {canPublish && (
        <div className="flex flex-col gap-3">
          <label className={labelClass}>
            {t("penalties.reason")}
            <input value={reason} onChange={(event) => setReason(event.target.value)} className={fieldClass} />
          </label>
          <button
            type="button"
            disabled={!ready}
            onClick={() =>
              startTransition(async () => {
                const outcome = await publishPenalties(configFromFields(fields), reason);
                setResult(outcome);
                if (outcome.ok) {
                  setReason("");
                  router.refresh();
                }
              })
            }
            className="self-start rounded-md bg-primary px-4 py-2 text-on-primary disabled:opacity-40"
          >
            {t("penalties.publish")}
          </button>
          {result && !result.ok && (
            <p role="status" className="text-sm text-text-secondary">
              {t("penalties.refused")}: {result.error} {result.detail}
            </p>
          )}
        </div>
      )}
    </section>
  );
}
