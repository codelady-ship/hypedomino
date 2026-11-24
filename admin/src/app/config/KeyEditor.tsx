"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import { changedKeys, invalidKeys, publishedValues, type ConfigKey, type KeyValues } from "@/config/keys";
import type { ConfigResult } from "@/config/result";
import { useTranslator } from "@/i18n/client";
import type { MessageKey } from "@/i18n/translate";

const unitLabels: Readonly<Record<string, MessageKey>> = { seconds: "config.unit.seconds", timeouts: "config.unit.timeouts", events: "config.unit.events", hours: "config.unit.hours" };

const fieldClass = "w-24 rounded-md bg-surface px-3 py-1 text-text-primary";

export function KeyEditor({ keys, canPublish, publish }: { keys: ConfigKey[]; canPublish: boolean; publish: (values: KeyValues, reason: string) => Promise<ConfigResult> }) {
  const t = useTranslator();
  const router = useRouter();
  const [values, setValues] = useState<KeyValues>({});
  const [reason, setReason] = useState("");
  const [result, setResult] = useState<ConfigResult | null>(null);
  const [pending, startTransition] = useTransition();
  const invalid = invalidKeys(keys, values);
  const changed = changedKeys(keys, values);

  function publishChanges() {
    startTransition(async () => {
      const outcome = await publish(publishedValues(keys, values), reason);
      setResult(outcome);
      if (outcome.ok) {
        setValues({});
        router.refresh();
      }
    });
  }

  return (
    <div className="flex flex-col gap-6">
      <table className="text-sm">
        <thead className="text-left text-text-muted">
          <tr>
            <th className="py-2 pr-4">{t("config.key")}</th>
            <th className="py-2 pr-4">{t("config.value")}</th>
            <th className="py-2 pr-4">{t("config.bounds")}</th>
            <th className="py-2 pr-4">{t("config.default")}</th>
          </tr>
        </thead>
        <tbody>
          {keys.map((key) => (
            <tr key={key.key} className="border-t border-surface">
              <td className="py-2 pr-4 font-mono">{key.key}</td>
              <td className="py-2 pr-4">
                <input
                  type="number"
                  min={key.min}
                  max={key.max}
                  step={1}
                  readOnly={!canPublish}
                  aria-label={key.key}
                  aria-invalid={invalid.includes(key.key)}
                  value={Number.isNaN(values[key.key]) ? "" : (values[key.key] ?? key.value)}
                  onChange={(event) => setValues((current) => ({ ...current, [key.key]: event.target.valueAsNumber }))}
                  className={fieldClass}
                />{" "}
                <span className="text-text-muted">{unitLabels[key.unit] ? t(unitLabels[key.unit]) : key.unit}</span>
              </td>
              <td className="py-2 pr-4 text-text-secondary">{key.max === undefined ? `≥ ${key.min}` : `${key.min} – ${key.max}`}</td>
              <td className="py-2 pr-4 text-text-secondary">{key.default}</td>
            </tr>
          ))}
        </tbody>
      </table>
      {invalid.length > 0 && (
        <p role="alert" className="text-sm text-danger-text">
          {t("config.out_of_bounds")}: {invalid.join(", ")}
        </p>
      )}
      {canPublish && (
        <section className="flex flex-col gap-3">
          <label className="flex flex-col gap-1 text-sm text-text-secondary">
            {t("config.reason")}
            <input value={reason} onChange={(event) => setReason(event.target.value)} className="rounded-md bg-surface px-3 py-1 text-text-primary" />
          </label>
          <button
            type="button"
            disabled={pending || changed.length === 0 || invalid.length > 0}
            onClick={publishChanges}
            className="self-start rounded-md bg-primary px-4 py-2 text-on-primary disabled:opacity-40"
          >
            {t("config.publish")}
          </button>
          {result && (
            <p role="status" className="text-sm text-text-secondary">
              {result.ok ? `${t("config.published")}: ${result.version}` : `${t("config.refused")}: ${result.error} ${result.detail}`}
            </p>
          )}
        </section>
      )}
    </div>
  );
}
