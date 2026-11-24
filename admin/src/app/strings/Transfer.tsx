"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import { useTranslator } from "@/i18n/client";
import { transferFormats } from "@/strings/strings";
import { exportStrings, importStrings, type ExportResult, type ImportResult } from "./actions";

const fieldClass = "rounded-md bg-surface px-3 py-1 text-text-primary";
const labelClass = "flex flex-col gap-1 text-sm text-text-secondary";
const buttonClass = "self-start rounded-md bg-primary px-3 py-1 text-on-primary disabled:opacity-40";

function download(text: string, filename: string) {
  const link = document.createElement("a");
  link.href = URL.createObjectURL(new Blob([text]));
  link.download = filename;
  link.click();
  URL.revokeObjectURL(link.href);
}

export function Transfer({ languages, mayImport }: { languages: string[]; mayImport: boolean }) {
  const t = useTranslator();
  const router = useRouter();
  const [language, setLanguage] = useState(languages[0] ?? "");
  const [format, setFormat] = useState<string>(transferFormats[0]);
  const [file, setFile] = useState<File | null>(null);
  const [result, setResult] = useState<ExportResult | ImportResult | null>(null);
  const [pending, startTransition] = useTransition();

  function exportFile() {
    startTransition(async () => {
      const outcome = await exportStrings(language, format);
      setResult(outcome);
      if (outcome.ok) {
        download(outcome.text, outcome.filename);
      }
    });
  }

  function importFile(chosen: File) {
    startTransition(async () => {
      const outcome = await importStrings(language, format, await chosen.text());
      setResult(outcome);
      if (outcome.ok) {
        router.refresh();
      }
    });
  }

  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-lg font-semibold">{t("strings.transfer")}</h2>
      <p className="text-sm text-text-secondary">{t("strings.transfer_intro")}</p>
      <div className="flex flex-wrap items-end gap-4">
        <label className={labelClass}>
          {t("strings.language")}
          <select value={language} onChange={(event) => setLanguage(event.target.value)} className={fieldClass}>
            {languages.map((option) => (
              <option key={option} value={option}>
                {option}
              </option>
            ))}
          </select>
        </label>
        <label className={labelClass}>
          {t("strings.format")}
          <select value={format} onChange={(event) => setFormat(event.target.value)} className={fieldClass}>
            {transferFormats.map((option) => (
              <option key={option} value={option}>
                {option}
              </option>
            ))}
          </select>
        </label>
        <button type="button" disabled={pending || language === ""} onClick={exportFile} className={buttonClass}>
          {t("strings.export")}
        </button>
      </div>
      {mayImport && (
        <div className="flex flex-wrap items-end gap-4">
          <label className={labelClass}>
            {t("strings.import_file")}
            <input type="file" accept={`.${format}`} onChange={(event) => setFile(event.target.files?.[0] ?? null)} className={fieldClass} />
          </label>
          <button type="button" disabled={pending || file === null || language === ""} onClick={() => file && importFile(file)} className={buttonClass}>
            {t("strings.import")}
          </button>
        </div>
      )}
      {result && !result.ok && (
        <p role="status" className="text-sm text-text-secondary">
          {t("strings.refused")}: {result.error} {result.detail}
        </p>
      )}
      {result && result.ok && "changed" in result && (
        <p role="status" className="text-sm text-text-secondary">
          {t("strings.imported")}: {t.number(result.changed)}
        </p>
      )}
    </section>
  );
}
