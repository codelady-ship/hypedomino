"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import type { ConfigResult } from "@/config/result";
import { useTranslator } from "@/i18n/client";
import { chainError, chainFromText } from "@/strings/strings";
import { saveFallbackChain } from "./actions";

const fieldClass = "rounded-md bg-surface px-3 py-1 font-mono text-text-primary";

type Props = {
  languages: string[];
  chains: Record<string, string[]>;
  mayPublish: boolean;
};

export function FallbackChains({ languages, chains, mayPublish }: Props) {
  const t = useTranslator();
  const router = useRouter();
  const [reason, setReason] = useState("");
  const [result, setResult] = useState<ConfigResult | null>(null);
  const [pending, startTransition] = useTransition();

  function save(language: string, chain: string[]) {
    startTransition(async () => {
      const outcome = await saveFallbackChain(language, chain, reason);
      setResult(outcome);
      if (outcome.ok) {
        router.refresh();
      }
    });
  }

  return (
    <section className="flex flex-col gap-4">
      <div className="flex flex-col gap-1">
        <h2 className="text-lg font-semibold">{t("strings.fallback")}</h2>
        <p className="text-sm text-text-secondary">{t("strings.fallback_intro")}</p>
      </div>
      {mayPublish && (
        <label className="flex flex-col gap-1 text-sm text-text-secondary">
          {t("strings.reason")}
          <input value={reason} onChange={(event) => setReason(event.target.value)} className="rounded-md bg-surface px-3 py-1 text-text-primary" />
        </label>
      )}
      {result && (
        <p role="status" className="text-sm text-text-secondary">
          {result.ok ? `${t("strings.done")}: ${result.version}` : `${t("strings.refused")}: ${result.error} ${result.detail}`}
        </p>
      )}
      {languages.map((language) => (
        <ChainRow
          key={`${language}:${(chains[language] ?? []).join(",")}`}
          language={language}
          chain={chains[language] ?? []}
          languages={languages}
          editable={mayPublish}
          ready={!pending && reason.trim() !== ""}
          onSave={(chain) => save(language, chain)}
        />
      ))}
    </section>
  );
}

type RowProps = {
  language: string;
  chain: string[];
  languages: string[];
  editable: boolean;
  ready: boolean;
  onSave: (chain: string[]) => void;
};

function ChainRow({ language, chain, languages, editable, ready, onSave }: RowProps) {
  const t = useTranslator();
  const saved = chain.join(", ");
  const [text, setText] = useState(saved);
  const next = chainFromText(text);
  const error = chainError(language, next, languages);
  const changed = next.join(", ") !== saved;
  if (!editable) {
    return (
      <p className="text-sm">
        <span className="pr-4 font-mono font-semibold">{language}</span>
        <span className="font-mono">{saved}</span>
      </p>
    );
  }
  return (
    <div className="flex flex-wrap items-center gap-4 text-sm">
      <span className="font-mono font-semibold">{language}</span>
      <input value={text} onChange={(event) => setText(event.target.value)} aria-label={`${t("strings.fallback")} ${language}`} className={fieldClass} />
      <button type="button" disabled={!ready || !changed || error !== null} onClick={() => onSave(next)} className="rounded-md bg-primary px-3 py-1 text-on-primary disabled:opacity-40">
        {t("strings.save")}
      </button>
      {error && <span className="text-text-muted">{t(`strings.chain_error.${error}`)}</span>}
    </div>
  );
}
