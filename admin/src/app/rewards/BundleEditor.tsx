"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import type { ConfigResult } from "@/config/result";
import { useTranslator } from "@/i18n/client";
import { bundleEditChanged, bundleEditIssues, editFrom, editOf, formOf, republishedNameKeys, useLabel, type NameKeysByDomain, type RewardBundle } from "@/rewards/bundles";
import { editRewardBundle } from "./actions";
import { BundleFields, fieldClass, labelClass } from "./BundleFields";
import { UnnamedKeys, UnnamedKeysAhead } from "@/app/strings/UnnamedKeys";

type Props = { bundle: RewardBundle; domainLabel: (domain: string) => string; nameKeys: NameKeysByDomain };

export function BundleEditor({ bundle, domainLabel, nameKeys }: Props) {
  const t = useTranslator();
  const router = useRouter();
  const [form, setForm] = useState(formOf(editOf(bundle)));
  const [reason, setReason] = useState("");
  const [confirmed, setConfirmed] = useState(false);
  const [result, setResult] = useState<ConfigResult | null>(null);
  const [pending, startTransition] = useTransition();
  const next = editFrom(form);
  const issues = bundleEditIssues(next);
  const ready = bundleEditChanged(bundle, next) && issues.length === 0 && reason.trim() !== "" && confirmed && !pending;
  return (
    <section className="flex flex-col gap-3 rounded-md border border-surface p-3">
      <h2 className="font-mono text-lg font-semibold break-all">{bundle.id}</h2>
      <BundleFields form={form} onChange={setForm} issues={issues} />
      <div role="alert" className="flex flex-col gap-1 text-sm text-danger-text">
        <p>{t("rewards.edit_warning")}</p>
        <ul className="flex flex-col gap-1">
          {bundle.usedBy.map((use) => (
            <li key={`${use.domain}:${useLabel(use)}`}>
              {domainLabel(use.domain)}: <span className="font-mono break-all">{useLabel(use)}</span>
            </li>
          ))}
        </ul>
      </div>
      <label className={labelClass}>
        {t("rewards.reason")}
        <input value={reason} onChange={(event) => setReason(event.target.value)} className={fieldClass} />
      </label>
      <label className="flex items-center gap-2 text-sm text-text-secondary">
        <input type="checkbox" checked={confirmed} onChange={(event) => setConfirmed(event.target.checked)} />
        {t("rewards.edit_confirm")}
      </label>
      <UnnamedKeysAhead keys={republishedNameKeys(bundle, nameKeys)} />
      <button
        type="button"
        disabled={!ready}
        onClick={() =>
          startTransition(async () => {
            const outcome = await editRewardBundle(bundle.id, bundle.revision, next, reason);
            setResult(outcome);
            if (outcome.ok) {
              router.refresh();
            }
          })
        }
        className="self-start rounded-md bg-primary px-4 py-2 text-on-primary disabled:opacity-40"
      >
        {t("rewards.publish")}
      </button>
      {result && !result.ok && (
        <p role="status" className="text-sm text-text-secondary">
          {t("rewards.refused")}: {result.error} {result.detail}
        </p>
      )}
      {result && !result.ok && <UnnamedKeys keys={result.keys} />}
    </section>
  );
}
