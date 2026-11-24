"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import type { ConfigResult } from "@/config/result";
import { useTranslator } from "@/i18n/client";
import type { Bundle } from "@/strings/strings";
import { Rollback } from "../config/Rollback";
import { activateBundle, publishBundle } from "./actions";

type Props = {
  language: string;
  languageHrefs: Record<string, string>;
  bundles: Bundle[];
  older: string | null;
  active: Record<string, number>;
  configVersion: number;
  mayPublish: boolean;
  rollbackClientId: string | null;
};

export function Bundles({ language, languageHrefs, bundles, older, active, configVersion, mayPublish, rollbackClientId }: Props) {
  const t = useTranslator();
  const router = useRouter();
  const [reason, setReason] = useState("");
  const [result, setResult] = useState<ConfigResult | null>(null);
  const [pending, startTransition] = useTransition();
  const ready = !pending && reason.trim() !== "";

  function run(action: () => Promise<ConfigResult>) {
    startTransition(async () => {
      const outcome = await action();
      setResult(outcome);
      if (outcome.ok) {
        router.refresh();
      }
    });
  }

  return (
    <section className="flex flex-col gap-4">
      <div className="flex flex-col gap-1">
        <h2 className="text-lg font-semibold">{t("strings.bundles")}</h2>
        <p className="text-sm text-text-muted">
          {t("strings.config_version")}: {configVersion}
        </p>
        <p className="text-sm text-text-secondary">{t("strings.bundles_intro")}</p>
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
      <nav className="flex flex-wrap gap-2 text-sm">
        {Object.entries(languageHrefs).map(([code, href]) => (
          <Link key={code} href={href} prefetch={false} aria-current={code === language ? "page" : undefined} className={`rounded-md px-3 py-1 font-mono ${code === language ? "bg-primary text-on-primary" : "bg-surface text-text-secondary"}`}>
            {code}
          </Link>
        ))}
      </nav>
      {language && (
        <div className="flex flex-col gap-2">
          <div className="flex flex-wrap items-center gap-4">
            <h3 className="font-mono font-semibold">{language}</h3>
            <span className="text-sm text-text-muted">
              {t("strings.active")}: {active[language] ?? t("strings.builtin")}
            </span>
            {mayPublish && (
              <button type="button" disabled={!ready} onClick={() => run(() => publishBundle(language, reason))} className="rounded-md bg-primary px-3 py-1 text-sm text-on-primary disabled:opacity-40">
                {t("strings.publish_bundle")}
              </button>
            )}
          </div>
          <table className="text-sm">
            <tbody>
              {bundles.map((bundle) => (
                <tr key={bundle.version} className="border-t border-surface align-top">
                  <td className="py-2 pr-4">{bundle.version}</td>
                  <td className="py-2 pr-4 font-mono">{bundle.hash.slice(0, 12)}</td>
                  <td className="py-2 pr-4">{bundle.by}</td>
                  <td className="whitespace-nowrap py-2 pr-4">{t.when(bundle.at)}</td>
                  <td className="py-2 pr-4">
                    <BundleAction
                      version={bundle.version}
                      active={active[language] ?? 0}
                      canActivate={mayPublish && ready}
                      onActivate={() => run(() => activateBundle(language, bundle.version, reason))}
                      rollback={rollbackClientId === null ? null : { clientId: rollbackClientId, action: (version, note) => activateBundle(language, version, note) }}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {older && (
            <Link href={older} prefetch={false} className="self-start rounded-md bg-surface px-4 py-2 text-sm text-text-secondary">
              {t("queues.next")}
            </Link>
          )}
        </div>
      )}
    </section>
  );
}

type RollbackProps = { clientId: string; action: (version: number, reason: string) => Promise<ConfigResult> };

function BundleAction({ version, active, canActivate, onActivate, rollback }: { version: number; active: number; canActivate: boolean; onActivate: () => void; rollback: RollbackProps | null }) {
  const t = useTranslator();
  if (version === active) {
    return <span className="text-text-muted">{t("strings.active")}</span>;
  }
  if (version > active) {
    return (
      <button type="button" disabled={!canActivate} onClick={onActivate} className="rounded-md bg-surface px-3 py-1 text-text-secondary disabled:opacity-40">
        {t("strings.activate")}
      </button>
    );
  }
  return rollback && <Rollback version={version} clientId={rollback.clientId} action={rollback.action} />;
}
