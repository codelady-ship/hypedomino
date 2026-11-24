"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import type { ConfigResult } from "@/config/result";
import { useTranslator } from "@/i18n/client";
import type { MessageKey } from "@/i18n/translate";
import { idList } from "@/missions/catalog";
import {
  changedStructural,
  configIssues,
  countryList,
  effectiveReset,
  isOverride,
  liveNumberKeys,
  modeIssues,
  modes,
  ratingChanges,
  resetRanks,
  rewardIssues,
  snapshotIssues,
  structuralNumberKeys,
  withMode,
  withoutRegion,
  withRankReward,
  withRegion,
  withResetRow,
  type Mode,
  type ModeConfig,
  type NumberKey,
  type RatingConfig,
} from "@/ranked/config";
import { CatalogChangeList } from "../config/CatalogChangeList";
import { publishRating } from "./actions";

const fieldClass = "rounded-md bg-surface px-3 py-1 text-text-primary disabled:opacity-50";
const labelClass = "flex flex-col gap-1 text-sm text-text-secondary";

const numberLabel: Record<NumberKey, MessageKey> = {
  divisionLp: "ranked.key.divisionLp",
  demotionLp: "ranked.key.demotionLp",
  apexRecalcHourUtc: "ranked.key.apexRecalcHourUtc",
  challengerSlots: "ranked.key.challengerSlots",
  challengerMinLp: "ranked.key.challengerMinLp",
  grandmasterSlots: "ranked.key.grandmasterSlots",
  grandmasterMinLp: "ranked.key.grandmasterMinLp",
  placementMatches: "ranked.key.placementMatches",
  initialMmr: "ranked.key.initialMmr",
  mmrScale: "ranked.key.mmrScale",
  mmrK: "ranked.key.mmrK",
  mmrKProvisional: "ranked.key.mmrKProvisional",
  mmrProvisionalMatches: "ranked.key.mmrProvisionalMatches",
  rankMmrBase: "ranked.key.rankMmrBase",
  rankMmrStep: "ranked.key.rankMmrStep",
  lpBase: "ranked.key.lpBase",
  lpGapScale: "ranked.key.lpGapScale",
  lpFactorMin: "ranked.key.lpFactorMin",
  lpFactorMax: "ranked.key.lpFactorMax",
  lpChangeMin: "ranked.key.lpChangeMin",
  lpChangeMax: "ranked.key.lpChangeMax",
  placementMaxDropDivisions: "ranked.key.placementMaxDropDivisions",
  placementMaxClimbDivisions: "ranked.key.placementMaxClimbDivisions",
  seasonLengthMonths: "ranked.key.seasonLengthMonths",
};

type Props = { published: RatingConfig; version: number; openSeason: string; structuralKeys: string[]; tiers: string[]; divisionRanks: string[]; canPublish: boolean };

export function RatingEditor({ published, version, openSeason, structuralKeys, tiers, divisionRanks, canPublish }: Props) {
  const t = useTranslator();
  const router = useRouter();
  const [config, setConfig] = useState(published);
  const [mode, setMode] = useState<Mode>("1v1");
  const [reason, setReason] = useState("");
  const [reviewing, setReviewing] = useState(false);
  const [result, setResult] = useState<ConfigResult | null>(null);
  const [pending, startTransition] = useTransition();
  const blocked = configIssues(config, divisionRanks).length > 0;
  const frozen = openSeason !== "";
  const change = (next: RatingConfig) => {
    setConfig(next);
    setReviewing(false);
  };
  const setCurrentMode = (next: ModeConfig) => change({ ...config, modes: { ...config.modes, [mode]: next } });

  return (
    <div className="flex flex-col gap-8">
      <section className="flex flex-col gap-2">
        <h2 className="text-lg font-semibold">{t("ranked.config")}</h2>
        <p className="text-sm text-text-muted">
          {t("ranked.version")}: {version}
        </p>
        {frozen && (
          <p className="text-sm text-text-muted">
            {t("ranked.frozen_hint")}: <span className="font-mono">{openSeason}</span>
          </p>
        )}
      </section>
      <Leaderboard config={config} tiers={tiers} onChange={change} />
      <section className="flex flex-col gap-4">
        <div role="tablist" aria-label={t("ranked.mode")} className="flex gap-2">
          {modes.map((candidate) => (
            <button
              key={candidate}
              type="button"
              role="tab"
              aria-selected={candidate === mode}
              onClick={() => setMode(candidate)}
              className={`rounded-md px-4 py-2 text-sm ${candidate === mode ? "bg-primary text-on-primary" : "bg-surface text-text-secondary"}`}
            >
              {candidate}
            </button>
          ))}
        </div>
        <ModeFields key={mode} mode={config.modes[mode]} published={published.modes[mode]} frozen={frozen} divisionRanks={divisionRanks} tiers={tiers} onChange={setCurrentMode} />
      </section>
      {canPublish && (
        <section className="flex flex-col gap-3">
          <label className={labelClass}>
            {t("ranked.reason")}
            <input value={reason} onChange={(event) => setReason(event.target.value)} className={fieldClass} />
          </label>
          {!reviewing ? (
            <button
              type="button"
              disabled={pending || blocked}
              onClick={() => {
                setReviewing(true);
                setResult(null);
              }}
              className="self-start rounded-md bg-primary px-4 py-2 text-on-primary disabled:opacity-40"
            >
              {t("ranked.review")}
            </button>
          ) : (
            <Review
              before={published}
              after={config}
              frozen={frozen}
              structuralKeys={structuralKeys}
              divisionRanks={divisionRanks}
              pending={pending}
              onCancel={() => setReviewing(false)}
              onConfirm={() =>
                startTransition(async () => {
                  const outcome = await publishRating(config, reason);
                  setResult(outcome);
                  setReviewing(false);
                  if (outcome.ok) {
                    router.refresh();
                  }
                })
              }
            />
          )}
          {blocked && <p className="text-sm text-text-muted">{t("ranked.blocked")}</p>}
          {result && !result.ok && (
            <p role="status" className="text-sm text-text-secondary">
              {t("ranked.refused")}: {result.error} {result.detail}
            </p>
          )}
        </section>
      )}
    </div>
  );
}

function Review({
  before,
  after,
  frozen,
  structuralKeys,
  divisionRanks,
  pending,
  onCancel,
  onConfirm,
}: {
  before: RatingConfig;
  after: RatingConfig;
  frozen: boolean;
  structuralKeys: string[];
  divisionRanks: string[];
  pending: boolean;
  onCancel: () => void;
  onConfirm: () => void;
}) {
  const t = useTranslator();
  const structural = changedStructural(before, after, structuralKeys, divisionRanks);
  return (
    <div role="region" aria-label={t("ranked.review")} className="flex flex-col gap-2 rounded-md border border-surface p-3 text-sm">
      <p className="text-text-secondary">{t("ranked.review_hint")}</p>
      <div className="flex flex-col gap-1">
        <p>{t("ranked.review_changed")}:</p>
        <div className="font-mono break-all">
          <CatalogChangeList changes={{ settings: ratingChanges(before, after), entries: [] }} t={t} />
        </div>
      </div>
      {frozen && structural.length > 0 && (
        <p className="text-danger-text">
          {t("ranked.review_frozen")}: <span className="font-mono break-all">{structural.join(", ")}</span>
        </p>
      )}
      <div className="flex gap-3">
        <button type="button" disabled={pending} onClick={onConfirm} className="rounded-md bg-primary px-4 py-2 text-on-primary disabled:opacity-40">
          {t("ranked.confirm")}
        </button>
        <button type="button" onClick={onCancel} className="rounded-md bg-surface px-4 py-2 text-text-secondary">
          {t("ranked.cancel")}
        </button>
      </div>
    </div>
  );
}

function Leaderboard({ config, tiers, onChange }: { config: RatingConfig; tiers: string[]; onChange: (config: RatingConfig) => void }) {
  const t = useTranslator();
  const [regionId, setRegionId] = useState("");
  const [regionCountries, setRegionCountries] = useState("");
  const regions = Object.entries(config.leaderboardRegions ?? {});
  return (
    <section className="flex flex-col gap-3">
      <h3 className="font-semibold">{t("ranked.leaderboard")}</h3>
      <p className="text-sm text-text-muted">{t("ranked.leaderboard_hint")}</p>
      <NumberField label={t("ranked.key.leaderboardTopSize")} value={config.leaderboardTopSize} onChange={(value) => onChange({ ...config, leaderboardTopSize: value ?? Number.NaN })} />
      <fieldset className="flex flex-col gap-3 rounded-md border border-surface p-3">
        <legend className="text-sm text-text-secondary">{t("ranked.regions")}</legend>
        {regions.map(([id, countries]) => (
          <div key={id} className="flex flex-wrap items-end gap-3">
            <span className="font-mono text-sm">{id}</span>
            <CountriesInput id={id} countries={countries} onChange={(next) => onChange(withRegion(config, id, next))} />
            <button type="button" onClick={() => onChange(withoutRegion(config, id))} className="rounded-md bg-surface px-3 py-1 text-sm text-text-secondary">
              {t("ranked.remove")}
            </button>
          </div>
        ))}
        <div className="flex flex-wrap items-end gap-3">
          <label className={labelClass}>
            {t("ranked.region_id")}
            <input value={regionId} onChange={(event) => setRegionId(event.target.value)} className={`${fieldClass} font-mono`} />
          </label>
          <label className={labelClass}>
            {t("ranked.region_countries")}
            <input value={regionCountries} onChange={(event) => setRegionCountries(event.target.value)} className={`${fieldClass} font-mono`} />
          </label>
          <button
            type="button"
            disabled={!regionId || !regionCountries}
            onClick={() => {
              onChange(withRegion(config, regionId, countryList(regionCountries)));
              setRegionId("");
              setRegionCountries("");
            }}
            className="rounded-md bg-surface px-4 py-2 text-text-secondary disabled:opacity-40"
          >
            {t("ranked.add_region")}
          </button>
        </div>
      </fieldset>
      <fieldset className="flex flex-col gap-3 rounded-md border border-surface p-3">
        <legend className="text-sm text-text-secondary">{t("ranked.coins")}</legend>
        <div className="flex flex-wrap items-end gap-3">
          {tiers.map((tier) => (
            <NumberField
              key={tier}
              label={tier}
              value={config.rankRewardCoins[tier]}
              onChange={(value) => onChange({ ...config, rankRewardCoins: { ...config.rankRewardCoins, [tier]: value ?? Number.NaN } })}
            />
          ))}
        </div>
      </fieldset>
      <Issues keys={snapshotIssues(config)} />
    </section>
  );
}

function CountriesInput({ id, countries, onChange }: { id: string; countries: string[]; onChange: (countries: string[]) => void }) {
  const t = useTranslator();
  const [text, setText] = useState(countries.join(", "));
  return (
    <input
      aria-label={`${t("ranked.region_countries")} ${id}`}
      value={text}
      onChange={(event) => {
        setText(event.target.value);
        onChange(countryList(event.target.value));
      }}
      className={`${fieldClass} w-64 font-mono`}
    />
  );
}

type ModeProps = { mode: ModeConfig; published: ModeConfig; frozen: boolean; divisionRanks: string[]; tiers: string[]; onChange: (mode: ModeConfig) => void };

function ModeFields({ mode, published, frozen, divisionRanks, tiers, onChange }: ModeProps) {
  const t = useTranslator();
  const setNumber = (key: NumberKey, value: number | undefined) => onChange({ ...mode, [key]: value ?? Number.NaN });
  const numberValue = (key: NumberKey) => {
    const value = mode[key];
    return typeof value === "number" && !Number.isNaN(value) ? value : undefined;
  };
  return (
    <div className="flex flex-col gap-4">
      <fieldset className="flex flex-col gap-3 rounded-md border border-surface p-3">
        <legend className="text-sm text-text-secondary">{t("ranked.structural")}</legend>
        <p className="text-sm text-text-muted">{t("ranked.structural_hint")}</p>
        <div className="flex flex-wrap items-end gap-3">
          {structuralNumberKeys.map((key) => (
            <NumberField key={key} label={t(numberLabel[key])} value={numberValue(key)} disabled={frozen} onChange={(value) => setNumber(key, value)} />
          ))}
        </div>
      </fieldset>
      <fieldset className="flex flex-col gap-3 rounded-md border border-surface p-3">
        <legend className="text-sm text-text-secondary">{t("ranked.live_values")}</legend>
        <p className="text-sm text-text-muted">{t("ranked.live_values_hint")}</p>
        <div className="flex flex-wrap items-end gap-3">
          {liveNumberKeys.map((key) => (
            <NumberField key={key} label={t(numberLabel[key])} value={numberValue(key)} step={key === "lpFactorMin" || key === "lpFactorMax" ? 0.05 : 1} onChange={(value) => setNumber(key, value)} />
          ))}
          <RankSelect label={t("ranked.key.newPlayerPlacementCap")} value={mode.newPlayerPlacementCap} ranks={divisionRanks} onChange={(value) => onChange({ ...mode, newPlayerPlacementCap: value })} />
        </div>
      </fieldset>
      <ResetTable mode={mode} frozen={frozen} divisionRanks={divisionRanks} onChange={onChange} />
      <RankRewards mode={mode} published={published} tiers={tiers} onChange={onChange} />
      <Issues keys={[...modeIssues(mode, divisionRanks), ...rewardIssues(mode)]} />
    </div>
  );
}

function ResetTable({ mode, frozen, divisionRanks, onChange }: { mode: ModeConfig; frozen: boolean; divisionRanks: string[]; onChange: (mode: ModeConfig) => void }) {
  const t = useTranslator();
  const ranks = resetRanks(divisionRanks);
  const table = effectiveReset(mode, divisionRanks);
  return (
    <fieldset className="flex flex-col gap-3 rounded-md border border-surface p-3">
      <legend className="text-sm text-text-secondary">{t("ranked.reset_table")}</legend>
      <p className="text-sm text-text-muted">{t("ranked.reset_hint")}</p>
      <div className="flex flex-wrap items-end gap-3">
        <NumberField label={t("ranked.key.resetTierDrop")} value={Number.isNaN(mode.resetTierDrop) ? undefined : mode.resetTierDrop} disabled={frozen} onChange={(value) => onChange({ ...mode, resetTierDrop: value ?? Number.NaN })} />
        <RankSelect label={t("ranked.key.resetFloor")} value={mode.resetFloor} ranks={divisionRanks} disabled={frozen} onChange={(value) => onChange({ ...mode, resetFloor: value })} />
        <RankSelect label={t("ranked.key.resetCeiling")} value={mode.resetCeiling} ranks={divisionRanks} disabled={frozen} onChange={(value) => onChange({ ...mode, resetCeiling: value })} />
      </div>
      <details>
        <summary className="cursor-pointer text-sm text-text-secondary">{t("ranked.reset_rows")}</summary>
        <div className="max-h-[32rem] overflow-y-auto">
          <table className="text-sm">
            <thead className="sticky top-0 bg-background-base text-left text-text-muted">
              <tr>
                <th className="py-1 pr-4">{t("ranked.final_rank")}</th>
                <th className="py-1 pr-4">{t("ranked.reset_rank")}</th>
                <th className="py-1 pr-4" />
              </tr>
            </thead>
            <tbody>
              {ranks.map((old, index) => (
                <tr key={old} className="border-t border-surface">
                  <td className="py-1 pr-4">{old}</td>
                  <td className="py-1 pr-4">
                    <select aria-label={`${t("ranked.reset_rank")} ${old}`} disabled={frozen} value={table[index]} onChange={(event) => onChange(withResetRow(mode, divisionRanks, old, event.target.value))} className={fieldClass}>
                      {divisionRanks.map((rank) => (
                        <option key={rank} value={rank}>
                          {rank}
                        </option>
                      ))}
                    </select>
                  </td>
                  <td className="py-1 pr-4 text-text-muted">{isOverride(mode, old) && t("ranked.overridden")}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </details>
    </fieldset>
  );
}

function RankRewards({ mode, published, tiers, onChange }: { mode: ModeConfig; published: ModeConfig; tiers: string[]; onChange: (mode: ModeConfig) => void }) {
  const t = useTranslator();
  return (
    <fieldset className="flex flex-col gap-3 rounded-md border border-surface p-3">
      <legend className="text-sm text-text-secondary">{t("ranked.rank_rewards")}</legend>
      <p className="text-sm text-text-muted">{t("ranked.rank_rewards_hint")}</p>
      <table className="text-sm">
        <thead className="text-left text-text-muted">
          <tr>
            <th className="py-1 pr-3">{t("ranked.tier")}</th>
            <th className="py-1 pr-3">{t("ranked.bundle_id")}</th>
            <th className="py-1 pr-3">{t("ranked.cosmetics")}</th>
            <th className="py-1 pr-3">{t("ranked.medals")}</th>
          </tr>
        </thead>
        <tbody>
          {tiers.map((tier) => {
            const bundle = mode.rankRewards?.[tier];
            const edit = (change: Parameters<typeof withRankReward>[2]) => onChange(withRankReward(mode, tier, change, published));
            return (
              <tr key={tier} className="border-t border-surface">
                <td className="py-1 pr-3">{tier}</td>
                <td className="py-1 pr-3">
                  <input aria-label={`${t("ranked.bundle_id")} ${tier}`} disabled={!bundle} value={bundle?.id ?? ""} onChange={(event) => edit({ id: event.target.value })} className={`${fieldClass} w-64 font-mono`} />
                </td>
                <td className="py-1 pr-3">
                  <IdsInput label={`${t("ranked.cosmetics")} ${tier}`} ids={bundle?.cosmetics ?? []} onChange={(cosmetics) => edit({ cosmetics })} />
                </td>
                <td className="py-1 pr-3">
                  <IdsInput label={`${t("ranked.medals")} ${tier}`} ids={bundle?.medals ?? []} onChange={(medals) => edit({ medals })} />
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </fieldset>
  );
}

function IdsInput({ label, ids, onChange }: { label: string; ids: string[]; onChange: (ids: string[]) => void }) {
  const [text, setText] = useState(ids.join(", "));
  return (
    <input
      aria-label={label}
      value={text}
      onChange={(event) => {
        setText(event.target.value);
        onChange(idList(event.target.value));
      }}
      className={`${fieldClass} w-64 font-mono`}
    />
  );
}

function RankSelect({ label, value, ranks, disabled, onChange }: { label: string; value: string; ranks: string[]; disabled?: boolean; onChange: (value: string) => void }) {
  return (
    <label className={labelClass}>
      {label}
      <select disabled={disabled} value={value} onChange={(event) => onChange(event.target.value)} className={fieldClass}>
        {ranks.map((rank) => (
          <option key={rank} value={rank}>
            {rank}
          </option>
        ))}
      </select>
    </label>
  );
}

function NumberField({ label, value, step, disabled, onChange }: { label: string; value: number | undefined; step?: number; disabled?: boolean; onChange: (value: number | undefined) => void }) {
  return (
    <label className={labelClass}>
      {label}
      <input
        type="number"
        step={step ?? 1}
        disabled={disabled}
        value={value === undefined || Number.isNaN(value) ? "" : value}
        onChange={(event) => onChange(event.target.value === "" ? undefined : Number(event.target.value))}
        className={`${fieldClass} w-28`}
      />
    </label>
  );
}

function Issues({ keys }: { keys: MessageKey[] }) {
  const t = useTranslator();
  if (keys.length === 0) {
    return null;
  }
  return (
    <ul role="alert" className="flex flex-col gap-1 text-sm text-danger-text">
      {[...new Set(keys)].map((key) => (
        <li key={key}>{t(key)}</li>
      ))}
    </ul>
  );
}
