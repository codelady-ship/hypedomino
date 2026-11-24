"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import { needsStepUp } from "@/admin/refusals";
import { StepUpPrompt } from "@/app/auth/StepUpPrompt";
import { useTranslator } from "@/i18n/client";
import { banCategories, banCategoryKeys, type BanCategory } from "@/players/sanctions";
import { decideReview, type ReviewDecision as Choice, type ReviewResult } from "./actions";

export function ReviewDecision({ id, clientId }: { id: number; clientId: string }) {
  const t = useTranslator();
  const router = useRouter();
  const [reason, setReason] = useState("");
  const [category, setCategory] = useState<BanCategory | "">("");
  const [last, setLast] = useState<Choice | null>(null);
  const [result, setResult] = useState<ReviewResult | null>(null);
  const [pending, startTransition] = useTransition();
  const stepUp = result !== null && !result.ok && needsStepUp(result.error);
  const blocked = pending || reason.trim() === "";

  function send(choice: Choice) {
    setLast(choice);
    startTransition(async () => {
      const outcome = await decideReview(id, choice, reason, category);
      setResult(outcome);
      if (outcome.ok) {
        router.refresh();
      }
    });
  }

  return (
    <div className="flex flex-col gap-2">
      <input aria-label={t("reviews.reason")} placeholder={t("reviews.reason")} value={reason} onChange={(event) => setReason(event.target.value)} className="rounded-md bg-surface px-3 py-1 text-text-primary" />
      <select aria-label={t("actions.ban_category")} value={category} onChange={(event) => setCategory(event.target.value as BanCategory | "")} className="rounded-md bg-surface px-3 py-1 text-text-primary">
        <option value="">{t("actions.ban_category")}</option>
        {banCategories.map((option) => (
          <option key={option} value={option}>
            {t(banCategoryKeys[option])}
          </option>
        ))}
      </select>
      <div className="flex gap-2">
        <button type="button" disabled={blocked} onClick={() => send("clear")} className="rounded-md bg-primary px-3 py-1 text-on-primary disabled:opacity-40">
          {t("reviews.clear")}
        </button>
        <button type="button" disabled={blocked} onClick={() => send("confirm")} className="rounded-md bg-surface px-3 py-1 text-text-secondary disabled:opacity-40">
          {t("reviews.confirm")}
        </button>
        <button type="button" disabled={blocked || category === ""} onClick={() => send("permanent-ban")} className="rounded-md bg-surface px-3 py-1 text-text-secondary disabled:opacity-40">
          {t("reviews.permanent_ban")}
        </button>
      </div>
      {stepUp && last && <StepUpPrompt clientId={clientId} onConfirmed={() => send(last)} />}
      {result?.ok && last === "permanent-ban" && (
        <p role="status" className="text-text-secondary">
          {t("reviews.ban_proposed")}
        </p>
      )}
      {result && !result.ok && !stepUp && (
        <p role="status" className="text-text-secondary">
          {t("reviews.refused")}: {result.error} {result.detail}
        </p>
      )}
    </div>
  );
}
