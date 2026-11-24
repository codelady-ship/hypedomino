"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import { useTranslator } from "@/i18n/client";
import type { MessageKey } from "@/i18n/translate";
import { editableStatuses, placeholdersFromText, placeholdersText, valueOf, type GameString, type Status, type StringsQuery } from "@/strings/strings";
import { saveKey, saveValue, type SavedResult } from "./actions";

const fieldClass = "rounded-md bg-surface px-3 py-1 text-text-primary";
const labelClass = "flex flex-col gap-1 text-sm text-text-secondary";
const buttonClass = "self-start rounded-md bg-primary px-3 py-1 text-on-primary disabled:opacity-40";

const statusLabels: Record<Status, MessageKey> = {
  missing: "strings.status.missing",
  draft: "strings.status.draft",
  approved: "strings.status.approved",
};

type Rights = { mayCreate: boolean; mayEdit: boolean; mayApprove: boolean };

type ListProps = { strings: GameString[]; languages: string[]; query: StringsQuery; next: string | null };

export function StringsEditor({ strings, languages, query, next, ...rights }: ListProps & Rights) {
  const t = useTranslator();
  return (
    <section className="flex flex-col gap-4">
      <h2 className="text-lg font-semibold">{t("strings.keys")}</h2>
      {rights.mayCreate && (
        <details>
          <summary className="text-sm text-text-secondary">{t("strings.new_key")}</summary>
          <KeyForm entry={null} />
        </details>
      )}
      <form method="get" action="/strings" className="flex flex-wrap items-end gap-4">
        <input type="hidden" name="bundles" value={query.bundles} />
        <label className={labelClass}>
          {t("strings.search")}
          <input name="q" defaultValue={query.q} className={fieldClass} />
        </label>
        <label className={labelClass}>
          {t("strings.missing_in")}
          <select name="missing" defaultValue={query.missing} className={fieldClass}>
            <option value="">{t("strings.all_keys")}</option>
            {languages.map((language) => (
              <option key={language} value={language}>
                {language}
              </option>
            ))}
          </select>
        </label>
        <button type="submit" className="rounded-md bg-primary px-4 py-2 font-semibold text-on-primary">
          {t("queues.search")}
        </button>
      </form>
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="text-left text-text-secondary">
              <th className="py-2 pr-4">{t("strings.key")}</th>
              {languages.map((language) => (
                <th key={language} className="py-2 pr-4 font-mono">
                  {language}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {strings.map((entry) => (
              <tr key={entry.key} className="border-t border-surface align-top">
                <td className="py-2 pr-4">
                  <KeyCell entry={entry} mayEdit={rights.mayEdit} />
                </td>
                {languages.map((language) => (
                  <td key={language} className="py-2 pr-4">
                    <ValueCell entry={entry} language={language} mayEdit={rights.mayEdit} mayApprove={rights.mayApprove} />
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {next && (
        <Link href={next} prefetch={false} className="self-start rounded-md bg-surface px-4 py-2 text-text-secondary">
          {t("queues.next")}
        </Link>
      )}
    </section>
  );
}

function KeyCell({ entry, mayEdit }: { entry: GameString; mayEdit: boolean }) {
  const t = useTranslator();
  return (
    <div className="flex flex-col gap-1">
      <span className="font-mono">{entry.key}</span>
      {entry.deprecated && <span className="text-danger-text">{t("strings.deprecated")}</span>}
      {entry.note && <span className="text-text-secondary">{entry.note}</span>}
      <span className="text-text-muted">
        {t("strings.max_length")}: {entry.max_length}
        {entry.placeholders.length > 0 && <span className="font-mono"> · {placeholdersText(entry.placeholders)}</span>}
      </span>
      {mayEdit && (
        <details>
          <summary className="text-text-secondary">{t("strings.edit_key")}</summary>
          <KeyForm entry={entry} />
        </details>
      )}
    </div>
  );
}

function KeyForm({ entry }: { entry: GameString | null }) {
  const t = useTranslator();
  const router = useRouter();
  const [key, setKey] = useState(entry?.key ?? "");
  const [note, setNote] = useState(entry?.note ?? "");
  const [maxLength, setMaxLength] = useState(entry ? String(entry.max_length) : "");
  const [placeholders, setPlaceholders] = useState(entry ? placeholdersText(entry.placeholders) : "");
  const [deprecated, setDeprecated] = useState(entry?.deprecated ?? false);
  const [reason, setReason] = useState("");
  const [result, setResult] = useState<SavedResult | null>(null);
  const [pending, startTransition] = useTransition();
  const ready = !pending && key.trim() !== "" && Number(maxLength) > 0;

  function save() {
    const fields = { note, max_length: Number(maxLength), placeholders: placeholdersFromText(placeholders), deprecated };
    startTransition(async () => {
      const outcome = await saveKey(key.trim(), fields, reason);
      setResult(outcome);
      if (outcome.ok) {
        router.refresh();
      }
    });
  }

  return (
    <div className="flex flex-col gap-2 py-2">
      {entry === null && (
        <label className={labelClass}>
          {t("strings.key")}
          <input value={key} onChange={(event) => setKey(event.target.value)} className={`font-mono ${fieldClass}`} />
        </label>
      )}
      <label className={labelClass}>
        {t("strings.note")}
        <textarea rows={2} value={note} onChange={(event) => setNote(event.target.value)} className={fieldClass} />
      </label>
      <label className={labelClass}>
        {t("strings.max_length")}
        <input type="number" min={1} value={maxLength} onChange={(event) => setMaxLength(event.target.value)} className={fieldClass} />
      </label>
      <label className={labelClass}>
        {t("strings.placeholders")}
        <input value={placeholders} onChange={(event) => setPlaceholders(event.target.value)} className={`font-mono ${fieldClass}`} />
      </label>
      {entry !== null && (
        <label className="flex items-center gap-2 text-sm text-text-secondary">
          <input type="checkbox" checked={deprecated} onChange={(event) => setDeprecated(event.target.checked)} />
          {t("strings.deprecated")}
        </label>
      )}
      <label className={labelClass}>
        {t("strings.reason")}
        <input value={reason} onChange={(event) => setReason(event.target.value)} className={fieldClass} />
      </label>
      <button type="button" disabled={!ready} onClick={save} className={buttonClass}>
        {t("strings.save")}
      </button>
      <Outcome result={result} />
    </div>
  );
}

function ValueCell({ entry, language, mayEdit, mayApprove }: { entry: GameString; language: string; mayEdit: boolean; mayApprove: boolean }) {
  const t = useTranslator();
  const value = valueOf(entry, language);
  const [open, setOpen] = useState(false);
  return (
    <div className="flex min-w-48 flex-col gap-1">
      <span className="whitespace-pre-wrap">{value.text || "—"}</span>
      <span className={value.status === "approved" ? "text-text-muted" : "text-danger-text"}>
        {t(statusLabels[value.status])} · {Array.from(value.text).length}/{entry.max_length}
      </span>
      {mayEdit && !open && (
        <button type="button" onClick={() => setOpen(true)} className="self-start rounded-md bg-surface px-3 py-1 text-text-secondary">
          {t("strings.edit")}
        </button>
      )}
      {open && <ValueForm entry={entry} language={language} mayApprove={mayApprove} />}
    </div>
  );
}

function ValueForm({ entry, language, mayApprove }: { entry: GameString; language: string; mayApprove: boolean }) {
  const t = useTranslator();
  const router = useRouter();
  const current = valueOf(entry, language);
  const [text, setText] = useState(current.text);
  const [status, setStatus] = useState<Status>(current.status === "approved" && mayApprove ? "approved" : "draft");
  const [result, setResult] = useState<SavedResult | null>(null);
  const [pending, startTransition] = useTransition();
  const statuses = mayApprove ? editableStatuses : (["draft"] as const);

  function save() {
    startTransition(async () => {
      const outcome = await saveValue(entry.key, language, text, status);
      setResult(outcome);
      if (outcome.ok) {
        router.refresh();
      }
    });
  }

  return (
    <div className="flex flex-col gap-2">
      <textarea rows={3} value={text} onChange={(event) => setText(event.target.value)} className={fieldClass} />
      <select value={status} onChange={(event) => setStatus(event.target.value as Status)} className={fieldClass}>
        {statuses.map((option) => (
          <option key={option} value={option}>
            {t(statusLabels[option])}
          </option>
        ))}
      </select>
      <button type="button" disabled={pending} onClick={save} className={buttonClass}>
        {t("strings.save")}
      </button>
      <Outcome result={result} />
    </div>
  );
}

function Outcome({ result }: { result: SavedResult | null }) {
  const t = useTranslator();
  if (!result) {
    return null;
  }
  return (
    <p role="status" className="text-sm text-text-secondary">
      {result.ok ? t("strings.saved") : `${t("strings.refused")}: ${result.error} ${result.detail}`}
    </p>
  );
}
