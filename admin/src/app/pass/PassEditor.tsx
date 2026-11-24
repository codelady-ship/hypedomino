"use client";

import { useState, useTransition } from "react";
import type { ConfigResult } from "@/config/result";
import { useTranslator } from "@/i18n/client";
import type { MessageKey } from "@/i18n/translate";
import {
  addSeason,
  changedSeasons,
  changedSettings,
  cloneSeason,
  droppedSeasons,
  newSeason,
  rankedWindow,
  removeSeason,
  replaceSeason,
  seasonIssues,
  seasonState,
  settingIssues,
  settingKeys,
  trackTotal,
  withCurve,
  withLevel,
  xpTable,
  type Curve,
  type MatchXp,
  type PassConfig,
  type Season,
  type SeasonState,
  type SettingKey,
  type Track,
  type Window,
} from "@/pass/config";
import { idList } from "@/missions/catalog";
import { windowMinute } from "@/shop/catalog";
import { publishPass } from "./actions";

const fieldClass = "rounded-md bg-surface px-3 py-1 text-text-primary disabled:opacity-50";
const labelClass = "flex flex-col gap-1 text-sm text-text-secondary";

const stateLabel: Record<SeasonState, MessageKey> = { planned: "pass.state.planned", running: "pass.state.running", ended: "pass.state.ended" };

const settingLabel: Record<SettingKey, MessageKey> = {
  freeTrackCoins: "pass.setting.freeTrackCoins",
  premiumTrackCrystals: "pass.setting.premiumTrackCrystals",
  premiumPlusLevels: "pass.setting.premiumPlusLevels",
  dailyMatchXpCap: "pass.setting.dailyMatchXpCap",
  botSeatRewardPercent: "pass.setting.botSeatRewardPercent",
  levelCoinPrice: "pass.setting.levelCoinPrice",
  levelCrystalPrice: "pass.setting.levelCrystalPrice",
};

const curveLabel: Record<keyof Curve, MessageKey> = { levels: "pass.curve.levels", xpBase: "pass.curve.xpBase", xpStep: "pass.curve.xpStep", xpMax: "pass.curve.xpMax" };

const curveKeys: readonly (keyof Curve)[] = ["levels", "xpBase", "xpStep", "xpMax"];

const instant = (minute: string) => (minute ? `${minute}:00Z` : "");

type Props = { initialConfig: PassConfig; published: PassConfig | null; version: number; serverTime: string; rankedSeasons: Window[] | null; canPublish: boolean };

export function PassEditor({ initialConfig, published, version, serverTime, rankedSeasons, canPublish }: Props) {
  const t = useTranslator();
  const [config, setConfig] = useState(initialConfig);
  const [selected, setSelected] = useState(initialConfig.seasons.at(-1)?.id ?? "");
  const [reason, setReason] = useState("");
  const [reviewing, setReviewing] = useState(false);
  const [result, setResult] = useState<ConfigResult | null>(null);
  const [pending, startTransition] = useTransition();
  const season = config.seasons.find((candidate) => candidate.id === selected);
  const publishedSeason = (id: string) => published?.seasons.find((candidate) => candidate.id === id);
  const stateOf = (candidate: Season) => seasonState(publishedSeason(candidate.id) ?? candidate, serverTime);
  const blocked = settingIssues(config).length > 0 || config.seasons.some((candidate) => seasonIssues(config, candidate).length > 0);
  const change = (next: PassConfig) => {
    setConfig(next);
    setReviewing(false);
  };

  return (
    <div className="flex flex-col gap-8">
      <p className="text-sm text-text-muted">
        {t("pass.version")}: {version} · {t("pass.server_time")}: {t.when(serverTime)}
      </p>
      <Settings config={config} onChange={change} />
      <section className="flex flex-col gap-3">
        <h2 className="text-lg font-semibold">{t("pass.seasons")}</h2>
        {config.seasons.length > 0 && (
          <select aria-label={t("pass.seasons")} value={selected} onChange={(event) => setSelected(event.target.value)} className={fieldClass}>
            {config.seasons.map((candidate) => (
              <option key={candidate.id} value={candidate.id}>
                {candidate.id} · {t(stateLabel[stateOf(candidate)])} · {t.when(candidate.start)} → {t.when(candidate.end)}
              </option>
            ))}
          </select>
        )}
        <NewSeason
          config={config}
          rankedSeasons={rankedSeasons}
          serverTime={serverTime}
          onAdd={(created) => {
            change(addSeason(config, created));
            setSelected(created.id);
          }}
        />
      </section>
      {season && (
        <SeasonFields
          key={season.id}
          config={config}
          season={season}
          published={publishedSeason(season.id)}
          state={stateOf(season)}
          onChange={(next) => change(replaceSeason(config, next))}
          onRemove={() => {
            change(removeSeason(config, season.id));
            setSelected(config.seasons.find((candidate) => candidate.id !== season.id)?.id ?? "");
          }}
        />
      )}
      {canPublish && (
        <section className="flex flex-col gap-3">
          <label className={labelClass}>
            {t("pass.reason")}
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
              {t("pass.review")}
            </button>
          ) : (
            <Review
              before={published}
              after={config}
              pending={pending}
              onCancel={() => setReviewing(false)}
              onConfirm={() =>
                startTransition(async () => {
                  setResult(await publishPass(config, reason));
                  setReviewing(false);
                })
              }
            />
          )}
          {blocked && <p className="text-sm text-text-muted">{t("pass.blocked")}</p>}
          {result && (
            <p role="status" className="text-sm text-text-secondary">
              {result.ok ? `${t("pass.published")}: ${result.version}` : `${t("pass.refused")}: ${result.error} ${result.detail}`}
            </p>
          )}
        </section>
      )}
    </div>
  );
}

function Review({ before, after, pending, onCancel, onConfirm }: { before: PassConfig | null; after: PassConfig; pending: boolean; onCancel: () => void; onConfirm: () => void }) {
  const t = useTranslator();
  const changed = changedSeasons(before, after);
  const dropped = droppedSeasons(before, after);
  const settings = changedSettings(before, after);
  return (
    <div role="region" aria-label={t("pass.review")} className="flex flex-col gap-2 rounded-md border border-surface p-3 text-sm">
      <p className="text-text-secondary">{t("pass.review_hint")}</p>
      <p>
        {t("pass.review_seasons")}: <span className="font-mono">{changed.join(", ") || "—"}</span>
      </p>
      {dropped.length > 0 && (
        <p className="text-danger-text">
          {t("pass.review_dropped")}: <span className="font-mono">{dropped.join(", ")}</span>
        </p>
      )}
      <p>
        {t("pass.review_settings")}: <span className="font-mono">{settings.join(", ") || "—"}</span>
      </p>
      {changed.map((id) => {
        const season = after.seasons.find((candidate) => candidate.id === id);
        return (
          season && (
            <p key={id} className="text-text-muted">
              <span className="font-mono">{id}</span>: {t("pass.total_free")} {trackTotal(season, "free")}, {t("pass.total_premium")} {trackTotal(season, "premium")}, {t("pass.curve.levels")} {season.curve.levels}
            </p>
          )
        );
      })}
      <div className="flex gap-3">
        <button type="button" disabled={pending} onClick={onConfirm} className="rounded-md bg-primary px-4 py-2 text-on-primary disabled:opacity-40">
          {t("pass.confirm")}
        </button>
        <button type="button" onClick={onCancel} className="rounded-md bg-surface px-4 py-2 text-text-secondary">
          {t("pass.cancel")}
        </button>
      </div>
    </div>
  );
}

function Settings({ config, onChange }: { config: PassConfig; onChange: (config: PassConfig) => void }) {
  const t = useTranslator();
  const found = settingIssues(config);
  const set = (key: SettingKey, value: number | undefined) => {
    const next = { ...config };
    if (value === undefined) {
      delete next[key];
    } else {
      next[key] = value;
    }
    onChange(next);
  };
  const setXp = (key: "matchXp1v1" | "matchXp2v2", field: keyof MatchXp, value: number | undefined) => {
    const next: Partial<MatchXp> = { ...config[key] };
    if (value === undefined) {
      delete next[field];
    } else {
      next[field] = value;
    }
    onChange({ ...config, [key]: next });
  };
  return (
    <section className="flex flex-col gap-3">
      <h2 className="text-lg font-semibold">{t("pass.settings")}</h2>
      <p className="text-sm text-text-muted">{t("pass.settings_hint")}</p>
      <div className="flex flex-wrap items-end gap-3">
        {settingKeys.map((key) => (
          <NumberField key={key} label={t(settingLabel[key])} value={config[key]} onChange={(value) => set(key, value)} />
        ))}
        {(["matchXp1v1", "matchXp2v2"] as const).map((key) =>
          (["win", "loss"] as const).map((field) => (
            <NumberField
              key={`${key}-${field}`}
              label={`${t(key === "matchXp1v1" ? "pass.setting.matchXp1v1" : "pass.setting.matchXp2v2")} · ${t(field === "win" ? "pass.win" : "pass.loss")}`}
              value={config[key]?.[field]}
              onChange={(value) => setXp(key, field, value)}
            />
          )),
        )}
      </div>
      <Issues keys={found} />
    </section>
  );
}

type NewSeasonProps = { config: PassConfig; rankedSeasons: Window[] | null; serverTime: string; onAdd: (season: Season) => void };

function NewSeason({ config, rankedSeasons, serverTime, onAdd }: NewSeasonProps) {
  const t = useTranslator();
  const suggested = (from: PassConfig) => (rankedSeasons ? rankedWindow(from, rankedSeasons, serverTime) : null);
  const [source, setSource] = useState(config.seasons.at(-1)?.id ?? "");
  const [start, setStart] = useState(() => windowMinute(suggested(config)?.start));
  const [end, setEnd] = useState(() => windowMinute(suggested(config)?.end));
  const [curve, setCurve] = useState<Partial<Curve>>({});
  const from = config.seasons.find((candidate) => candidate.id === source);
  const curveReady = curveKeys.every((key) => (curve[key] ?? 0) >= 1);
  const ready = Boolean(start && end) && (from !== undefined || curveReady);
  return (
    <fieldset className="flex flex-col gap-3 rounded-md border border-surface p-3">
      <legend className="text-sm text-text-secondary">{t("pass.new_season")}</legend>
      <div className="flex flex-wrap items-end gap-3">
        <label className={labelClass}>
          {t("pass.clone_from")}
          <select value={source} onChange={(event) => setSource(event.target.value)} className={fieldClass}>
            <option value="">{t("pass.clone_none")}</option>
            {config.seasons.map((candidate) => (
              <option key={candidate.id} value={candidate.id}>
                {candidate.id}
              </option>
            ))}
          </select>
        </label>
        <label className={labelClass}>
          {t("pass.start")}
          <input type="datetime-local" value={start} onChange={(event) => setStart(event.target.value)} className={fieldClass} />
        </label>
        <label className={labelClass}>
          {t("pass.end")}
          <input type="datetime-local" value={end} onChange={(event) => setEnd(event.target.value)} className={fieldClass} />
        </label>
        {!from &&
          curveKeys.map((key) => <NumberField key={key} label={t(curveLabel[key])} value={curve[key]} onChange={(value) => setCurve((current) => ({ ...current, [key]: value }))} />)}
        <button
          type="button"
          disabled={!ready}
          onClick={() => {
            const created = from ? cloneSeason(config, from, instant(start), instant(end)) : newSeason(config, instant(start), instant(end), curve as Curve);
            onAdd(created);
            const next = suggested(addSeason(config, created));
            setStart(windowMinute(next?.start));
            setEnd(windowMinute(next?.end));
          }}
          className="rounded-md bg-surface px-4 py-2 text-text-secondary disabled:opacity-40"
        >
          {t(from ? "pass.clone" : "pass.add_empty")}
        </button>
      </div>
      <p className="text-sm text-text-muted">{t(from ? "pass.clone_hint" : "pass.empty_hint")}</p>
      {suggested(config) && <p className="text-sm text-text-muted">{t("pass.ranked_window_hint")}</p>}
    </fieldset>
  );
}

type FieldsProps = { config: PassConfig; season: Season; published: Season | undefined; state: SeasonState; onChange: (season: Season) => void; onRemove: () => void };

function SeasonFields({ config, season, published, state, onChange, onRemove }: FieldsProps) {
  const t = useTranslator();
  const frozen = state !== "planned";
  return (
    <section className="flex flex-col gap-4">
      <h2 className="font-mono text-lg font-semibold">
        {season.id} · {t(stateLabel[state])}
      </h2>
      {frozen && <p className="text-sm text-text-muted">{t("pass.frozen_hint")}</p>}
      <div className="flex flex-wrap items-end gap-3">
        <label className={labelClass}>
          {t("pass.start")}
          <input
            type="datetime-local"
            disabled={frozen}
            value={windowMinute(season.start)}
            onChange={(event) => event.target.value && onChange({ ...season, start: instant(event.target.value) })}
            className={fieldClass}
          />
        </label>
        <label className={labelClass}>
          {t("pass.end")}
          <input
            type="datetime-local"
            disabled={state === "ended"}
            value={windowMinute(season.end)}
            onChange={(event) => event.target.value && onChange({ ...season, end: instant(event.target.value) })}
            className={fieldClass}
          />
        </label>
        {!published && (
          <button type="button" onClick={onRemove} className="rounded-md bg-surface px-3 py-1 text-sm text-text-secondary">
            {t("pass.remove_season")}
          </button>
        )}
        {published && state === "planned" && (
          <button type="button" onClick={onRemove} className="rounded-md bg-surface px-3 py-1 text-sm text-text-secondary">
            {t("pass.remove_planned")}
          </button>
        )}
      </div>
      <fieldset className="flex flex-col gap-3 rounded-md border border-surface p-3">
        <legend className="text-sm text-text-secondary">{t("pass.curve")}</legend>
        <div className="flex flex-wrap items-end gap-3">
          {curveKeys.map((key) => (
            <NumberField
              key={key}
              label={t(curveLabel[key])}
              value={season.curve[key]}
              disabled={frozen}
              onChange={(value) => onChange(withCurve(season, { ...season.curve, [key]: value ?? 0 }))}
            />
          ))}
        </div>
        <XpTable curve={season.curve} />
      </fieldset>
      <Levels config={config} season={season} published={published} frozen={frozen} itemsFrozen={state === "ended"} onChange={onChange} />
      <Issues keys={seasonIssues(config, season)} />
    </section>
  );
}

function XpTable({ curve }: { curve: Curve }) {
  const t = useTranslator();
  return (
    <details>
      <summary className="cursor-pointer text-sm text-text-secondary">{t("pass.xp_table")}</summary>
      <div className="max-h-64 overflow-y-auto">
        <table className="text-sm">
          <thead className="text-left text-text-muted">
            <tr>
              <th className="py-1 pr-4">{t("pass.level")}</th>
              <th className="py-1 pr-4">{t("pass.xp_to_next")}</th>
              <th className="py-1 pr-4">{t("pass.xp_total")}</th>
            </tr>
          </thead>
          <tbody>
            {xpTable(curve).map((row) => (
              <tr key={row.level} className="border-t border-surface">
                <td className="py-1 pr-4">{row.level}</td>
                <td className="py-1 pr-4">{row.toNext}</td>
                <td className="py-1 pr-4">{row.total}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </details>
  );
}

function Levels({ config, season, published, frozen, itemsFrozen, onChange }: { config: PassConfig; season: Season; published: Season | undefined; frozen: boolean; itemsFrozen: boolean; onChange: (season: Season) => void }) {
  const t = useTranslator();
  const levels = Array.from({ length: Math.max(0, season.curve.levels) }, (_, index) => index + 1);
  const edit = (track: Track, level: number, change: { amount?: number; cosmetics?: string[] }) => onChange(withLevel(season, track, level, change, published));
  return (
    <fieldset className="flex flex-col gap-3 rounded-md border border-surface p-3">
      <legend className="text-sm text-text-secondary">{t("pass.levels")}</legend>
      <p className="text-sm text-text-muted">
        {t("pass.total_free")}: {trackTotal(season, "free")}
        {config.freeTrackCoins !== undefined && ` / ${config.freeTrackCoins}`} · {t("pass.total_premium")}: {trackTotal(season, "premium")}
        {config.premiumTrackCrystals !== undefined && ` / ${config.premiumTrackCrystals}`}
      </p>
      <p className="text-sm text-text-muted">{t("pass.sources_hint")}</p>
      <div className="max-h-[32rem] overflow-y-auto">
        <table className="text-sm">
          <thead className="sticky top-0 bg-background-base text-left text-text-muted">
            <tr>
              <th className="py-1 pr-3">{t("pass.level")}</th>
              <th className="py-1 pr-3">{t("pass.free_coins")}</th>
              <th className="py-1 pr-3">{t("pass.free_cosmetics")}</th>
              <th className="py-1 pr-3">{t("pass.premium_crystals")}</th>
              <th className="py-1 pr-3">{t("pass.premium_cosmetics")}</th>
            </tr>
          </thead>
          <tbody>
            {levels.map((level) => (
              <tr key={level} className="border-t border-surface">
                <td className="py-1 pr-3">{level}</td>
                <td className="py-1 pr-3">
                  <CellNumber value={season.free[level]?.coins} disabled={frozen} onChange={(amount) => edit("free", level, { amount })} label={`${t("pass.free_coins")} ${level}`} />
                </td>
                <td className="py-1 pr-3">
                  <CellIds key={`free-${level}`} ids={season.free[level]?.cosmetics ?? []} disabled={itemsFrozen} onChange={(cosmetics) => edit("free", level, { cosmetics })} label={`${t("pass.free_cosmetics")} ${level}`} />
                </td>
                <td className="py-1 pr-3">
                  <CellNumber value={season.premium[level]?.crystals} disabled={frozen} onChange={(amount) => edit("premium", level, { amount })} label={`${t("pass.premium_crystals")} ${level}`} />
                </td>
                <td className="py-1 pr-3">
                  <CellIds key={`premium-${level}`} ids={season.premium[level]?.cosmetics ?? []} disabled={itemsFrozen} onChange={(cosmetics) => edit("premium", level, { cosmetics })} label={`${t("pass.premium_cosmetics")} ${level}`} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </fieldset>
  );
}

function Issues({ keys }: { keys: MessageKey[] }) {
  const t = useTranslator();
  if (keys.length === 0) {
    return null;
  }
  return (
    <ul role="alert" className="flex flex-col gap-1 text-sm text-danger-text">
      {keys.map((key) => (
        <li key={key}>{t(key)}</li>
      ))}
    </ul>
  );
}

function parsed(text: string): number | undefined {
  return text === "" ? undefined : Number(text);
}

function NumberField({ label, value, disabled, onChange }: { label: string; value: number | undefined; disabled?: boolean; onChange: (value: number | undefined) => void }) {
  return (
    <label className={labelClass}>
      {label}
      <input type="number" min={0} step={1} disabled={disabled} value={value ?? ""} onChange={(event) => onChange(parsed(event.target.value))} className={`${fieldClass} w-28`} />
    </label>
  );
}

function CellNumber({ label, value, disabled, onChange }: { label: string; value: number | undefined; disabled: boolean; onChange: (value: number) => void }) {
  return (
    <input
      aria-label={label}
      type="number"
      min={0}
      step={1}
      disabled={disabled}
      value={value ?? ""}
      onChange={(event) => onChange(parsed(event.target.value) ?? 0)}
      className={`${fieldClass} w-20`}
    />
  );
}

function CellIds({ label, ids, disabled, onChange }: { label: string; ids: string[]; disabled: boolean; onChange: (ids: string[]) => void }) {
  const [text, setText] = useState(ids.join(", "));
  return (
    <input
      aria-label={label}
      disabled={disabled}
      value={text}
      onChange={(event) => {
        setText(event.target.value);
        onChange(idList(event.target.value));
      }}
      className={`${fieldClass} w-56 font-mono`}
    />
  );
}
