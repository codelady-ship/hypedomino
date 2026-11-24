"use client";

import { useState, useTransition } from "react";
import type { ConfigResult } from "@/config/result";
import { useTranslator } from "@/i18n/client";
import type { MessageKey } from "@/i18n/translate";
import {
  addMedal,
  medalIssues,
  medalTextKeys,
  newMedal,
  replaceMedal,
  validMedalSlug,
  validShowcaseSlots,
  withNewSource,
  withoutSource,
  withSource,
  type Medal,
  type MedalCatalog,
  type MedalRules,
} from "@/medals/catalog";
import { publishMedals } from "./actions";
import { UnnamedKeys, UnnamedKeysAhead } from "@/app/strings/UnnamedKeys";
import { ItemsWithoutArt } from "@/app/assets/ItemsWithoutArt";

const fieldClass = "rounded-md bg-surface px-3 py-1 text-text-primary";
const labelClass = "flex flex-col gap-1 text-sm text-text-secondary";

const rarityLabel: Record<string, MessageKey> = { common: "rarity.common", rare: "rarity.rare" };

const availabilityLabel: Record<string, MessageKey> = {
  live: "medals.availability.live",
  retired: "medals.availability.retired",
  disabled: "medals.availability.disabled",
};

const sourceLabel: Record<string, MessageKey> = { mission: "medals.source.mission", rank_reward: "medals.source.rank_reward" };

type Props = { initialCatalog: MedalCatalog; version: number; rules: MedalRules; canPublish: boolean };

export function MedalEditor({ initialCatalog, version, rules, canPublish }: Props) {
  const t = useTranslator();
  const [catalog, setCatalog] = useState(initialCatalog);
  const [selected, setSelected] = useState(initialCatalog.medals[0]?.id ?? "");
  const [reason, setReason] = useState("");
  const [result, setResult] = useState<ConfigResult | null>(null);
  const [pending, startTransition] = useTransition();
  const medal = catalog.medals.find((candidate) => candidate.id === selected);
  const label = (map: Record<string, MessageKey>, id: string) => (map[id] ? t(map[id]) : id);
  const blocked = !validShowcaseSlots(catalog.profileMedalSlots) || catalog.medals.some((candidate) => medalIssues(candidate).length > 0);

  return (
    <div className="flex flex-col gap-8">
      <p className="text-sm text-text-muted">
        {t("medals.version")}: {version}
      </p>
      <section className="flex flex-col gap-3">
        <label className={labelClass}>
          {t("medals.showcase_slots")}
          <input
            type="number"
            min={0}
            step={1}
            value={catalog.profileMedalSlots}
            onChange={(event) => setCatalog((current) => ({ ...current, profileMedalSlots: Number(event.target.value) }))}
            className={`${fieldClass} w-28`}
          />
        </label>
        {!validShowcaseSlots(catalog.profileMedalSlots) && (
          <p role="alert" className="text-sm text-danger-text">
            {t("medals.issue.showcase_slots")}
          </p>
        )}
      </section>
      <section className="flex flex-col gap-3">
        <h2 className="text-lg font-semibold">{t("medals.list")}</h2>
        {catalog.medals.length > 0 && (
          <select aria-label={t("medals.list")} value={selected} onChange={(event) => setSelected(event.target.value)} className={fieldClass}>
            {catalog.medals.map((candidate) => (
              <option key={candidate.id} value={candidate.id}>
                {candidate.id} · {label(rarityLabel, candidate.rarityId)} · {label(availabilityLabel, candidate.availability)}
              </option>
            ))}
          </select>
        )}
        <NewMedal
          onAdd={(slug) => {
            const created = newMedal(slug, rules);
            const next = addMedal(catalog, created);
            if (!next) {
              return false;
            }
            setCatalog(next);
            setSelected(created.id);
            return true;
          }}
        />
      </section>
      {medal && <MedalFields medal={medal} rules={rules} label={label} onChange={(next) => setCatalog((current) => replaceMedal(current, next))} />}
      {canPublish && (
        <section className="flex flex-col gap-3">
          <label className={labelClass}>
            {t("medals.reason")}
            <input value={reason} onChange={(event) => setReason(event.target.value)} className={fieldClass} />
          </label>
          <UnnamedKeysAhead keys={medalTextKeys(catalog)} />
          <button
            type="button"
            disabled={pending || blocked || catalog.medals.length === 0}
            onClick={() => startTransition(async () => setResult(await publishMedals(catalog, reason)))}
            className="self-start rounded-md bg-primary px-4 py-2 text-on-primary disabled:opacity-40"
          >
            {t("medals.publish")}
          </button>
          {blocked && <p className="text-sm text-text-muted">{t("medals.blocked")}</p>}
          {result && (
            <p role="status" className="text-sm text-text-secondary">
              {result.ok ? `${t("medals.published")}: ${result.version}` : `${t("medals.refused")}: ${result.error} ${result.detail}`}
            </p>
          )}
          {result && !result.ok && <UnnamedKeys keys={result.keys} />}
          {result && !result.ok && <ItemsWithoutArt error={result.error} detail={result.detail} />}
        </section>
      )}
    </div>
  );
}

function NewMedal({ onAdd }: { onAdd: (slug: string) => boolean }) {
  const t = useTranslator();
  const [slug, setSlug] = useState("");
  const [taken, setTaken] = useState(false);
  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-end gap-3">
        <label className={labelClass}>
          {t("medals.new_slug")}
          <span className="flex items-center gap-1 font-mono">
            medal.
            <input
              value={slug}
              onChange={(event) => {
                setSlug(event.target.value);
                setTaken(false);
              }}
              className={fieldClass}
            />
          </span>
        </label>
        <button
          type="button"
          disabled={!validMedalSlug(slug)}
          onClick={() => {
            if (onAdd(slug)) {
              setSlug("");
              return;
            }
            setTaken(true);
          }}
          className="rounded-md bg-surface px-4 py-2 text-text-secondary disabled:opacity-40"
        >
          {t("medals.add")}
        </button>
      </div>
      {slug && !validMedalSlug(slug) && <p className="text-sm text-text-muted">{t("medals.slug_rule")}</p>}
      {taken && (
        <p role="alert" className="text-sm text-danger-text">
          {t("medals.slug_taken")}
        </p>
      )}
    </div>
  );
}

type FieldsProps = { medal: Medal; rules: MedalRules; label: (map: Record<string, MessageKey>, id: string) => string; onChange: (medal: Medal) => void };

function MedalFields({ medal, rules, label, onChange }: FieldsProps) {
  const t = useTranslator();
  const found = medalIssues(medal);
  return (
    <section className="flex flex-col gap-4">
      <h2 className="font-mono text-lg font-semibold">{medal.id}</h2>
      <p className="text-sm text-text-muted">
        {t("medals.server_fields")}: <span className="font-mono">{medal.nameKey}</span>, <span className="font-mono">{medal.descriptionKey}</span>,{" "}
        <span className="font-mono">{medal.staticAddress}</span> · {t("medals.revision")}: {medal.revision || "—"}
      </p>
      <div className="flex flex-wrap items-end gap-3">
        <label className={labelClass}>
          {t("medals.rarity")}
          <select value={medal.rarityId} onChange={(event) => onChange({ ...medal, rarityId: event.target.value })} className={fieldClass}>
            {rules.rarities.map((candidate) => (
              <option key={candidate} value={candidate}>
                {label(rarityLabel, candidate)}
              </option>
            ))}
          </select>
        </label>
        <label className={labelClass}>
          {t("medals.availability")}
          <select value={medal.availability} onChange={(event) => onChange({ ...medal, availability: event.target.value })} className={fieldClass}>
            {rules.availability.map((candidate) => (
              <option key={candidate} value={candidate}>
                {label(availabilityLabel, candidate)}
              </option>
            ))}
          </select>
        </label>
      </div>
      <fieldset className="flex flex-col gap-3 rounded-md border border-surface p-3">
        <legend className="text-sm text-text-secondary">{t("medals.sources")}</legend>
        {medal.sources.map((source, index) => (
          <div key={index} className="flex flex-wrap items-end gap-3">
            <label className={labelClass}>
              {t("medals.source_type")}
              <select value={source.type} onChange={(event) => onChange(withSource(medal, index, { ...source, type: event.target.value }))} className={fieldClass}>
                {rules.source_types.map((candidate) => (
                  <option key={candidate} value={candidate}>
                    {label(sourceLabel, candidate)}
                  </option>
                ))}
              </select>
            </label>
            <label className={labelClass}>
              {t("medals.source_ref")}
              <input value={source.ref} onChange={(event) => onChange(withSource(medal, index, { ...source, ref: event.target.value }))} className={`${fieldClass} font-mono`} />
            </label>
            <button type="button" onClick={() => onChange(withoutSource(medal, index))} className="rounded-md bg-surface px-3 py-1 text-sm text-text-secondary">
              {t("medals.source_remove")}
            </button>
          </div>
        ))}
        <button type="button" onClick={() => onChange(withNewSource(medal, rules))} className="self-start rounded-md bg-surface px-3 py-1 text-sm text-text-secondary">
          {t("medals.source_add")}
        </button>
        <p className="text-sm text-text-muted">{t("medals.sources_hint")}</p>
      </fieldset>
      {found.length > 0 && (
        <ul role="alert" className="flex flex-col gap-1 text-sm text-danger-text">
          {found.map((key) => (
            <li key={key}>{t(key)}</li>
          ))}
        </ul>
      )}
    </section>
  );
}
