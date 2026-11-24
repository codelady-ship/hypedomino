"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import type { ConfigResult } from "@/config/result";
import { useTranslator } from "@/i18n/client";
import { type ModerationText, textToModeration, wordLists } from "@/moderation/lists";
import { publishModeration } from "./actions";

const fieldClass = "rounded-md bg-surface px-3 py-1 text-text-primary";
const labelClass = "flex flex-col gap-1 text-sm text-text-secondary";
const lists = [...wordLists, "reserved"] as const;

export function ListsEditor({ published, canPublish }: { published: ModerationText; canPublish: boolean }) {
  const t = useTranslator();
  const router = useRouter();
  const [text, setText] = useState(published);
  const [reason, setReason] = useState("");
  const [result, setResult] = useState<ConfigResult | null>(null);
  const [pending, startTransition] = useTransition();
  const changed = lists.some((list) => text[list] !== published[list]);
  const ready = !pending && changed && reason.trim() !== "";
  return (
    <section className="flex flex-col gap-4">
      {lists.map((list) => (
        <label key={list} className={labelClass}>
          {list === "reserved" ? t("moderation.reserved") : `${t("moderation.words")}: ${list}`}
          <textarea
            readOnly={!canPublish}
            rows={6}
            value={text[list]}
            onChange={(event) => setText({ ...text, [list]: event.target.value })}
            className={`font-mono ${fieldClass}`}
          />
        </label>
      ))}
      {canPublish && (
        <div className="flex flex-col gap-3">
          <label className={labelClass}>
            {t("moderation.reason")}
            <input value={reason} onChange={(event) => setReason(event.target.value)} className={fieldClass} />
          </label>
          <button
            type="button"
            disabled={!ready}
            onClick={() =>
              startTransition(async () => {
                const outcome = await publishModeration(textToModeration(text), reason);
                setResult(outcome);
                if (outcome.ok) {
                  setReason("");
                  router.refresh();
                }
              })
            }
            className="self-start rounded-md bg-primary px-4 py-2 text-on-primary disabled:opacity-40"
          >
            {t("moderation.publish")}
          </button>
          {result && !result.ok && (
            <p role="status" className="text-sm text-text-secondary">
              {t("moderation.refused")}: {result.error} {result.detail}
            </p>
          )}
        </div>
      )}
    </section>
  );
}
