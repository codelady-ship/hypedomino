"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import type { ConfigResult } from "@/config/result";
import { useTranslator } from "@/i18n/client";
import { editFrom, emptyBundleEdit, formOf, newBundleIssues } from "@/rewards/bundles";
import { createRewardBundle } from "./actions";
import { BundleFields, fieldClass, labelClass } from "./BundleFields";

export function NewBundle() {
  const t = useTranslator();
  const router = useRouter();
  const [id, setId] = useState("");
  const [form, setForm] = useState(formOf(emptyBundleEdit));
  const [reason, setReason] = useState("");
  const [result, setResult] = useState<ConfigResult | null>(null);
  const [pending, startTransition] = useTransition();
  const bundle = editFrom(form);
  const issues = newBundleIssues(id, bundle);
  const ready = issues.length === 0 && reason.trim() !== "" && !pending;
  const create = () =>
    startTransition(async () => {
      const outcome = await createRewardBundle(id, bundle, reason);
      setResult(outcome);
      if (outcome.ok) {
        setId("");
        setForm(formOf(emptyBundleEdit));
        setReason("");
        router.refresh();
      }
    });
  return (
    <details className="flex flex-col gap-3 rounded-md border border-surface p-3">
      <summary className="cursor-pointer font-semibold">{t("rewards.new_bundle")}</summary>
      <div className="mt-3 flex flex-col gap-3">
        <p className="text-sm text-text-muted">{t("rewards.new_bundle_hint")}</p>
        <label className={labelClass}>
          {t("rewards.new_bundle_id")}
          <input value={id} onChange={(event) => setId(event.target.value.trim())} className={`${fieldClass} font-mono`} />
        </label>
        <BundleFields form={form} onChange={setForm} issues={id === "" ? [] : issues} />
        <label className={labelClass}>
          {t("rewards.reason")}
          <input value={reason} onChange={(event) => setReason(event.target.value)} className={fieldClass} />
        </label>
        <button type="button" disabled={!ready} onClick={create} className="self-start rounded-md bg-primary px-4 py-2 text-on-primary disabled:opacity-40">
          {t("rewards.create")}
        </button>
        {result && (
          <p role="status" className="text-sm text-text-secondary">
            {result.ok ? `${t("rewards.created")}: ${result.version}` : `${t("rewards.refused")}: ${result.error} ${result.detail}`}
          </p>
        )}
      </div>
    </details>
  );
}
