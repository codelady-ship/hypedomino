"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import { itemIdValid, type AssetCheckResult } from "@/assets/assets";
import { useTranslator } from "@/i18n/client";
import { checkAsset, startAssetUpload } from "./actions";

const fieldClass = "rounded-md bg-surface px-3 py-1 text-text-primary";
const labelClass = "flex flex-col gap-1 text-sm text-text-secondary";

async function sent(uploadUrl: string, file: File): Promise<boolean> {
  try {
    const response = await fetch(uploadUrl, { method: "PUT", body: file });
    return response.ok;
  } catch {
    return false;
  }
}

export function AssetUpload({ initialItem }: { initialItem: string }) {
  const t = useTranslator();
  const router = useRouter();
  const [itemId, setItemId] = useState(initialItem);
  const [file, setFile] = useState<File | null>(null);
  const [refusal, setRefusal] = useState("");
  const [pending, startTransition] = useTransition();
  const showItemIssue = itemId !== "" && !itemIdValid(itemId);

  const upload = (chosen: File) =>
    startTransition(async () => {
      setRefusal("");
      const started = await startAssetUpload(itemId);
      if (!started.ok) {
        setRefusal(`${t("assets.refused")}: ${started.error} ${started.detail}`);
        return;
      }
      if (!(await sent(started.uploadUrl, chosen))) {
        setRefusal(t("assets.upload_failed"));
        router.refresh();
        return;
      }
      const checked = await checkAsset(started.id);
      if (!checked.ok) {
        setRefusal(`${t("assets.refused")}: ${checked.error} ${checked.detail}`);
      }
      router.refresh();
    });

  return (
    <section className="flex flex-col gap-3 rounded-md border border-surface p-3">
      <h2 className="text-lg font-semibold">{t("assets.upload_title")}</h2>
      <div className="flex flex-wrap gap-3">
        <label className={labelClass}>
          {t("assets.item")}
          <input value={itemId} onChange={(event) => setItemId(event.target.value.trim())} className={`${fieldClass} font-mono`} />
        </label>
        <label className={labelClass}>
          {t("assets.file")}
          <input type="file" accept="image/png" onChange={(event) => setFile(event.target.files?.[0] ?? null)} className={fieldClass} />
        </label>
      </div>
      {showItemIssue && (
        <p role="alert" className="text-sm text-danger-text">
          {t("assets.issue.item")}
        </p>
      )}
      <button type="button" disabled={!itemIdValid(itemId) || file === null || pending} onClick={() => file && upload(file)} className="self-start rounded-md bg-primary px-4 py-2 text-on-primary disabled:opacity-40">
        {t("assets.upload")}
      </button>
      {refusal !== "" && (
        <p role="status" className="text-sm text-danger-text">
          {refusal}
        </p>
      )}
    </section>
  );
}

export function RecheckAsset({ id }: { id: string }) {
  const t = useTranslator();
  const router = useRouter();
  const [result, setResult] = useState<AssetCheckResult | null>(null);
  const [pending, startTransition] = useTransition();
  return (
    <div className="flex flex-col gap-1">
      <button
        type="button"
        disabled={pending}
        onClick={() =>
          startTransition(async () => {
            const outcome = await checkAsset(id);
            setResult(outcome);
            if (outcome.ok) {
              router.refresh();
            }
          })
        }
        className="self-start rounded-md bg-surface px-3 py-1 text-text-primary disabled:opacity-40"
      >
        {t("assets.recheck")}
      </button>
      {result && !result.ok && (
        <p role="status" className="text-sm text-danger-text">
          {t("assets.refused")}: {result.error} {result.detail}
        </p>
      )}
    </div>
  );
}
