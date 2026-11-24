"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import type { ConfigResult } from "@/config/result";
import { useTranslator } from "@/i18n/client";
import { badKeys, keysFromText, sameKeys } from "@/quickmessages/keys";
import { publishQuickMessages } from "./actions";

const fieldClass = "rounded-md bg-surface px-3 py-1 text-text-primary";
const labelClass = "flex flex-col gap-1 text-sm text-text-secondary";

export function KeysEditor({ published, canPublish }: { published: string[]; canPublish: boolean }) {
  const t = useTranslator();
  const router = useRouter();
  const [text, setText] = useState(published.join("\n"));
  const [reason, setReason] = useState("");
  const [result, setResult] = useState<ConfigResult | null>(null);
  const [pending, startTransition] = useTransition();
  const keys = keysFromText(text);
  const bad = badKeys(keys);
  const ready = !pending && bad.length === 0 && !sameKeys(keys, published) && reason.trim() !== "";
  return (
    <section className="flex flex-col gap-4">
      <label className={labelClass}>
        {t("quick.keys")}
        <textarea readOnly={!canPublish} rows={10} value={text} onChange={(event) => setText(event.target.value)} className={`font-mono ${fieldClass}`} />
      </label>
      {bad.length > 0 && (
        <p role="alert" className="text-sm text-danger-text">
          {t("quick.bad_keys")}: {bad.join(", ")}
        </p>
      )}
      {canPublish && (
        <div className="flex flex-col gap-3">
          <label className={labelClass}>
            {t("quick.reason")}
            <input value={reason} onChange={(event) => setReason(event.target.value)} className={fieldClass} />
          </label>
          <button
            type="button"
            disabled={!ready}
            onClick={() =>
              startTransition(async () => {
                const outcome = await publishQuickMessages({ keys }, reason);
                setResult(outcome);
                if (outcome.ok) {
                  setReason("");
                  router.refresh();
                }
              })
            }
            className="self-start rounded-md bg-primary px-4 py-2 text-on-primary disabled:opacity-40"
          >
            {t("quick.publish")}
          </button>
          {result && !result.ok && (
            <p role="status" className="text-sm text-text-secondary">
              {t("quick.refused")}: {result.error} {result.detail}
            </p>
          )}
        </div>
      )}
    </section>
  );
}
