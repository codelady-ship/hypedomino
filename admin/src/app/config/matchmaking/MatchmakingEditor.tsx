"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import type { ConfigResult } from "@/config/result";
import { useTranslator } from "@/i18n/client";
import { badFields, botLevels, configFromFields, fieldsFromConfig, levelFields, numberFields, type MatchmakingConfig } from "@/matchmaking/config";
import { publishMatchmaking } from "./actions";

const fieldClass = "rounded-md bg-surface px-3 py-1 text-text-primary";
const labelClass = "flex flex-col gap-1 text-sm text-text-secondary";

export function MatchmakingEditor({ published, canPublish }: { published: MatchmakingConfig | null; canPublish: boolean }) {
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
        {numberFields.map((name) => (
          <label key={name} className={labelClass}>
            <span className="font-mono">{name}</span>
            <input inputMode="numeric" readOnly={!canPublish} placeholder={t("matchmaking.default")} value={fields[name]} onChange={(event) => set(name, event.target.value.trim())} className={fieldClass} />
          </label>
        ))}
        {levelFields.map((name) => (
          <label key={name} className={labelClass}>
            <span className="font-mono">{name}</span>
            <select disabled={!canPublish} value={fields[name]} onChange={(event) => set(name, event.target.value)} className={fieldClass}>
              <option value="">{t("matchmaking.default")}</option>
              {botLevels.map((level) => (
                <option key={level} value={level}>
                  {level}
                </option>
              ))}
            </select>
          </label>
        ))}
      </div>
      {bad.length > 0 && (
        <p role="alert" className="text-sm text-danger-text">
          {t("matchmaking.bad_fields")}: {bad.join(", ")}
        </p>
      )}
      {canPublish && (
        <div className="flex flex-col gap-3">
          <label className={labelClass}>
            {t("matchmaking.reason")}
            <input value={reason} onChange={(event) => setReason(event.target.value)} className={fieldClass} />
          </label>
          <button
            type="button"
            disabled={!ready}
            onClick={() =>
              startTransition(async () => {
                const outcome = await publishMatchmaking(configFromFields(fields), reason);
                setResult(outcome);
                if (outcome.ok) {
                  setReason("");
                  router.refresh();
                }
              })
            }
            className="self-start rounded-md bg-primary px-4 py-2 text-on-primary disabled:opacity-40"
          >
            {t("matchmaking.publish")}
          </button>
          {result && !result.ok && (
            <p role="status" className="text-sm text-text-secondary">
              {t("matchmaking.refused")}: {result.error} {result.detail}
            </p>
          )}
        </div>
      )}
    </section>
  );
}
