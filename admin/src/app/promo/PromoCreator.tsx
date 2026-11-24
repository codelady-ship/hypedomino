"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import { useTranslator } from "@/i18n/client";
import type { MessageKey } from "@/i18n/translate";
import { draftFromTemplate, draftReady, emptyDraft, maxBatchCount, promoKinds, promoRequest, templateRequest, type PromoDraft, type PromoTemplate } from "@/promo/codes";
import { createPromo, createPromoBatch, saveTemplate, type CreatedResult, type TemplateResult } from "./actions";

const fieldClass = "rounded-md bg-surface px-3 py-1 text-text-primary";
const labelClass = "flex flex-col gap-1 text-sm text-text-secondary";

function download(csv: string, filename: string) {
  const link = document.createElement("a");
  link.href = URL.createObjectURL(new Blob([csv], { type: "text/csv" }));
  link.download = filename;
  link.click();
  URL.revokeObjectURL(link.href);
}

function stateKey(result: CreatedResult & { ok: true }): MessageKey {
  return result.approvalId === null ? "promo.created" : "promo.created_pending";
}

export function PromoCreator({ templates, mayEditTemplates }: { templates: readonly PromoTemplate[]; mayEditTemplates: boolean }) {
  const t = useTranslator();
  const router = useRouter();
  const [draft, setDraft] = useState<PromoDraft>(emptyDraft);
  const [batch, setBatch] = useState(false);
  const [reason, setReason] = useState("");
  const [result, setResult] = useState<CreatedResult | null>(null);
  const [templateResult, setTemplateResult] = useState<TemplateResult | null>(null);
  const [pending, startTransition] = useTransition();
  const edit = <K extends keyof PromoDraft>(key: K, value: PromoDraft[K]) => setDraft((current) => ({ ...current, [key]: value }));
  const ready = draftReady(draft, batch) && reason.trim() !== "";
  const existingTemplate = templates.some((template) => template.id === draft.id.trim());
  const templateReady = !batch && reason.trim() !== "" && (mayEditTemplates || !existingTemplate);

  function startFrom(id: string) {
    const template = templates.find((candidate) => candidate.id === id);
    setDraft(template ? draftFromTemplate(template) : emptyDraft);
    setResult(null);
    setTemplateResult(null);
  }

  function keepTemplate() {
    startTransition(async () => {
      const outcome = await saveTemplate(templateRequest(draft), existingTemplate, reason);
      setTemplateResult(outcome);
      if (outcome.ok) {
        router.refresh();
      }
    });
  }

  function create() {
    startTransition(async () => {
      const request = promoRequest(draft, batch);
      const outcome = batch ? await createPromoBatch(request, reason) : await createPromo(request, reason);
      setResult(outcome);
      if (!outcome.ok) {
        return;
      }
      if (outcome.csv) {
        download(outcome.csv, outcome.filename);
      }
      setDraft(emptyDraft);
      router.refresh();
    });
  }

  return (
    <section className="flex flex-col gap-4 rounded-md border border-surface p-4">
      <h2 className="text-lg font-semibold">{t("promo.create")}</h2>
      {templates.length > 0 && (
        <label className={`${labelClass} self-start`}>
          {t("promo.from_template")}
          <select defaultValue="" onChange={(event) => startFrom(event.target.value)} className={fieldClass}>
            <option value="">{t("promo.no_template")}</option>
            {templates.map((template) => (
              <option key={template.id} value={template.id}>
                {template.id}
              </option>
            ))}
          </select>
        </label>
      )}
      <div className="flex gap-4 text-sm">
        <label className="flex items-center gap-2">
          <input type="radio" checked={!batch} onChange={() => setBatch(false)} />
          {t("promo.one_code")}
        </label>
        <label className="flex items-center gap-2">
          <input type="radio" checked={batch} onChange={() => setBatch(true)} />
          {t("promo.batch")}
        </label>
      </div>
      <div className="grid gap-3 sm:grid-cols-2">
        <label className={labelClass}>
          {t("promo.id")}
          <input value={draft.id} onChange={(event) => edit("id", event.target.value)} className={fieldClass} />
        </label>
        <label className={labelClass}>
          {t("promo.campaign")}
          <input value={draft.campaign} onChange={(event) => edit("campaign", event.target.value)} className={fieldClass} />
        </label>
        {batch ? (
          <label className={labelClass}>
            {t("promo.count")}
            <input type="number" min={1} max={maxBatchCount} step={1} value={Number.isNaN(draft.count) ? "" : draft.count} onChange={(event) => edit("count", event.target.valueAsNumber)} className={fieldClass} />
          </label>
        ) : (
          <label className={labelClass}>
            {t("promo.kind")}
            <select value={draft.kind} onChange={(event) => edit("kind", event.target.value as PromoDraft["kind"])} className={fieldClass}>
              {promoKinds.map((kind) => (
                <option key={kind} value={kind}>
                  {kind}
                </option>
              ))}
            </select>
          </label>
        )}
        <label className={labelClass}>
          {t("promo.bundle_id")}
          <input value={draft.bundleId} onChange={(event) => edit("bundleId", event.target.value)} className={fieldClass} />
        </label>
        <label className={labelClass}>
          {t("promo.coins")}
          <input type="number" min={0} step={1} value={Number.isNaN(draft.coins) ? "" : draft.coins} onChange={(event) => edit("coins", event.target.valueAsNumber)} className={fieldClass} />
        </label>
        <label className={labelClass}>
          {t("promo.pass_xp")}
          <input type="number" min={0} step={1} value={Number.isNaN(draft.passXp) ? "" : draft.passXp} onChange={(event) => edit("passXp", event.target.valueAsNumber)} className={fieldClass} />
        </label>
        <label className={`${labelClass} sm:col-span-2`}>
          {t("promo.cosmetics")}
          <input value={draft.cosmetics} onChange={(event) => edit("cosmetics", event.target.value)} className={fieldClass} />
        </label>
        <label className="flex items-start gap-2 text-sm text-text-secondary sm:col-span-2">
          <input type="checkbox" checked={draft.passPremium} onChange={(event) => edit("passPremium", event.target.checked)} className="mt-1" />
          <span className="flex flex-col gap-1">
            <span className="text-text-primary">{t("promo.pass_premium")}</span>
            <span className="text-text-muted">{t("promo.pass_premium_hint")}</span>
          </span>
        </label>
        <label className={labelClass}>
          {t("promo.start")}
          <input type="datetime-local" value={draft.startMinute} onChange={(event) => edit("startMinute", event.target.value)} className={fieldClass} />
        </label>
        <label className={labelClass}>
          {t("promo.end")}
          <input type="datetime-local" value={draft.endMinute} onChange={(event) => edit("endMinute", event.target.value)} className={fieldClass} />
        </label>
        {!batch && draft.kind === "multi_use" && (
          <>
            <label className={labelClass}>
              {t("promo.cap")}
              <input type="number" min={1} step={1} value={Number.isNaN(draft.cap) ? "" : draft.cap} onChange={(event) => edit("cap", event.target.valueAsNumber)} className={fieldClass} />
            </label>
            <label className={labelClass}>
              {t("promo.chosen_code")}
              <input value={draft.code} onChange={(event) => edit("code", event.target.value)} className={fieldClass} />
            </label>
          </>
        )}
        {!batch && draft.kind === "targeted" && <p className="text-sm text-text-muted sm:col-span-2">{t("promo.targeted_hint")}</p>}
        <label className={`${labelClass} sm:col-span-2`}>
          {t("promo.reason")}
          <input value={reason} onChange={(event) => setReason(event.target.value)} className={fieldClass} />
        </label>
      </div>
      <p className="text-sm text-text-muted">{t("promo.utc_hint")}</p>
      <div className="flex flex-wrap gap-3">
        <button type="button" disabled={pending || !ready} onClick={create} className="rounded-md bg-primary px-4 py-2 text-on-primary disabled:opacity-40">
          {batch ? t("promo.generate") : t("promo.create")}
        </button>
        {!batch && (
          <button type="button" disabled={pending || !templateReady} onClick={keepTemplate} className="rounded-md bg-surface px-4 py-2 text-text-secondary disabled:opacity-40">
            {existingTemplate ? t("promo.update_template") : t("promo.save_template")}
          </button>
        )}
      </div>
      {templateResult && !templateResult.ok && (
        <p role="alert" className="text-sm text-danger-text">
          {t("promo.refused")}: {templateResult.error} {templateResult.detail}
        </p>
      )}
      {templateResult?.ok && (
        <p role="status" className="text-sm text-text-secondary">
          {t("promo.template_saved")}
        </p>
      )}
      {result && !result.ok && (
        <p role="alert" className="text-sm text-danger-text">
          {t("promo.refused")}: {result.error} {result.detail}
        </p>
      )}
      {result?.ok && (
        <div role="status" className="flex flex-col gap-2 text-sm">
          <p className="text-text-secondary">{t(stateKey(result))}</p>
          {result.plaintext && (
            <p>
              {t("promo.plaintext_once")} <code className="rounded bg-surface px-2 py-1 font-mono text-text-primary select-all">{result.plaintext}</code>
            </p>
          )}
          {result.csv && (
            <button type="button" onClick={() => download(result.csv, result.filename)} className="self-start rounded-md bg-surface px-3 py-1 text-text-secondary">
              {t("promo.download_again")}
            </button>
          )}
        </div>
      )}
    </section>
  );
}
