"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import { useTranslator } from "@/i18n/client";
import type { MessageKey } from "@/i18n/translate";
import {
  canRemove,
  createIssues,
  endAfterMonths,
  extendIssues,
  rescheduleIssues,
  seasonIdFromSlug,
  seasonNameKey,
  seasonProgress,
  shiftedBy,
  startLocked,
  type Season,
  type SeasonProgress,
  type SeasonResult,
  type SeasonState,
} from "@/ranked/seasons";
import { windowMinute } from "@/shop/catalog";
import { keyHref } from "@/strings/strings";
import { createSeason, deleteSeason, extendSeason, rescheduleSeason } from "./actions";
import { UnnamedKeys, UnnamedKeysAhead } from "@/app/strings/UnnamedKeys";

const fieldClass = "rounded-md bg-surface px-3 py-1 text-text-primary disabled:opacity-50";
const labelClass = "flex flex-col gap-1 text-sm text-text-secondary";
const actionClass = "self-start rounded-md bg-primary px-4 py-2 text-on-primary disabled:opacity-40";

const stateLabel: Record<SeasonState, MessageKey> = {
  scheduled: "ranked.state.scheduled",
  live: "ranked.state.live",
  ending: "ranked.state.ending",
  closed: "ranked.state.closed",
};

const progressLabel: Record<SeasonProgress, MessageKey> = {
  open: "ranked.progress.open",
  closed: "ranked.progress.closed",
  rewarded: "ranked.progress.rewarded",
};

const instant = (minute: string) => (minute ? `${minute}:00Z` : "");

type Props = { seasons: Season[]; nextStart: string | null; serverTime: string; lengthMonths: number; canCreate: boolean; canEdit: boolean; canDelete: boolean };

export function SeasonsPanel({ seasons, nextStart, serverTime, lengthMonths, canCreate, canEdit, canDelete }: Props) {
  const t = useTranslator();
  const [selected, setSelected] = useState("");
  const season = seasons.find((candidate) => candidate.id === selected);
  return (
    <section className="flex flex-col gap-4">
      <h2 className="text-lg font-semibold">{t("ranked.seasons")}</h2>
      {seasons.length === 0 ? (
        <p className="text-text-secondary">{t("ranked.seasons_empty")}</p>
      ) : (
        <table className="text-sm">
          <thead className="text-left text-text-muted">
            <tr>
              <th className="py-2 pr-4">{t("ranked.season")}</th>
              <th className="py-2 pr-4">{t("ranked.state")}</th>
              <th className="py-2 pr-4">{t("ranked.start")}</th>
              <th className="py-2 pr-4">{t("ranked.end")}</th>
              <th className="py-2 pr-4">{t("ranked.rewards")}</th>
              <th className="py-2 pr-4">{t("ranked.end_run")}</th>
              {(canEdit || canDelete) && <th className="py-2 pr-4" />}
            </tr>
          </thead>
          <tbody>
            {seasons.map((candidate) => (
              <tr key={candidate.id} className="border-t border-surface align-top">
                <td className="py-2 pr-4 font-mono">{candidate.id}</td>
                <td className="py-2 pr-4">{t(stateLabel[candidate.state])}</td>
                <td className="py-2 pr-4 whitespace-nowrap">{t.when(candidate.start)}</td>
                <td className="py-2 pr-4 whitespace-nowrap">{t.when(candidate.end)}</td>
                <td className="py-2 pr-4">{t(candidate.no_rewards ? "ranked.rewards_off" : "ranked.rewards_on")}</td>
                <td className="py-2 pr-4">
                  1v1: {t(progressLabel[seasonProgress(candidate, "1v1")])} · 2v2: {t(progressLabel[seasonProgress(candidate, "2v2")])}
                </td>
                {(canEdit || canDelete) && (
                  <td className="py-2 pr-4">
                    {(candidate.state === "scheduled" || candidate.state === "live") && (
                      <button type="button" onClick={() => setSelected(candidate.id)} className="rounded-md bg-surface px-3 py-1 text-text-secondary">
                        {t("ranked.change")}
                      </button>
                    )}
                  </td>
                )}
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {season?.state === "scheduled" && <ScheduledSeason key={season.id} seasons={seasons} season={season} serverTime={serverTime} canEdit={canEdit} canDelete={canDelete} onDone={() => setSelected("")} />}
      {season?.state === "live" && canEdit && <LiveSeason key={season.id} seasons={seasons} season={season} onDone={() => setSelected("")} />}
      {canCreate && <NewSeason key={nextStart ?? "first"} nextStart={nextStart} serverTime={serverTime} lengthMonths={lengthMonths} />}
    </section>
  );
}

function useSeasonChange(onDone?: () => void) {
  const router = useRouter();
  const [result, setResult] = useState<SeasonResult | null>(null);
  const [pending, startTransition] = useTransition();
  const run = (change: () => Promise<SeasonResult>) =>
    startTransition(async () => {
      const outcome = await change();
      setResult(outcome);
      if (outcome.ok) {
        onDone?.();
        router.refresh();
      }
    });
  return { result, pending, run };
}

function NewSeason({ nextStart, serverTime, lengthMonths }: { nextStart: string | null; serverTime: string; lengthMonths: number }) {
  const t = useTranslator();
  const [slug, setSlug] = useState("");
  const [startMinute, setStartMinute] = useState(windowMinute(nextStart ?? undefined));
  const [endMinute, setEndMinute] = useState(nextStart ? windowMinute(endAfterMonths(nextStart, lengthMonths)) : "");
  const [noRewards, setNoRewards] = useState(false);
  const [reason, setReason] = useState("");
  const { result, pending, run } = useSeasonChange();
  const start = nextStart ?? instant(startMinute);
  const end = instant(endMinute);
  const found = slug ? createIssues(slug, start, end, nextStart, serverTime) : [];
  return (
    <fieldset className="flex flex-col gap-3 rounded-md border border-surface p-3">
      <legend className="text-sm text-text-secondary">{t("ranked.new_season")}</legend>
      <div className="flex flex-wrap items-end gap-3">
        <label className={labelClass}>
          {t("ranked.slug")}
          <input value={slug} onChange={(event) => setSlug(event.target.value)} className={`${fieldClass} font-mono`} />
        </label>
        <label className={labelClass}>
          {t("ranked.start")}
          <input
            type="datetime-local"
            disabled={nextStart !== null}
            value={startMinute}
            onChange={(event) => {
              setStartMinute(event.target.value);
              if (event.target.value) {
                setEndMinute(windowMinute(endAfterMonths(instant(event.target.value), lengthMonths)));
              }
            }}
            className={fieldClass}
          />
        </label>
        <label className={labelClass}>
          {t("ranked.end")}
          <input type="datetime-local" value={endMinute} onChange={(event) => setEndMinute(event.target.value)} className={fieldClass} />
        </label>
        <label className="flex items-center gap-2 text-sm text-text-secondary">
          <input type="checkbox" checked={noRewards} onChange={(event) => setNoRewards(event.target.checked)} />
          {t("ranked.no_rewards")}
        </label>
      </div>
      <p className="text-sm text-text-muted">{t(nextStart ? "ranked.new_hint_next" : "ranked.new_hint_first")}</p>
      {slug && <p className="text-sm text-text-muted">{t("ranked.new_id")}: <span className="font-mono">{seasonIdFromSlug(slug)}</span></p>}
      {slug && <NameKey nameKey={seasonNameKey(seasonIdFromSlug(slug))} />}
      <Reason value={reason} onChange={setReason} />
      <UnnamedKeysAhead keys={slug ? [seasonNameKey(seasonIdFromSlug(slug))] : []} />
      <button
        type="button"
        disabled={pending || !slug || found.length > 0 || reason.trim() === ""}
        onClick={() => run(() => createSeason(seasonIdFromSlug(slug), start, end, noRewards, reason))}
        className={actionClass}
      >
        {t("ranked.create")}
      </button>
      <Issues keys={found} />
      <Outcome result={result} />
    </fieldset>
  );
}

function ScheduledSeason({ seasons, season, serverTime, canEdit, canDelete, onDone }: { seasons: Season[]; season: Season; serverTime: string; canEdit: boolean; canDelete: boolean; onDone: () => void }) {
  const t = useTranslator();
  const [startMinute, setStartMinute] = useState(windowMinute(season.start));
  const [endMinute, setEndMinute] = useState(windowMinute(season.end));
  const [noRewards, setNoRewards] = useState(season.no_rewards ?? false);
  const [reason, setReason] = useState("");
  const { result, pending, run } = useSeasonChange(onDone);
  const locked = startLocked(seasons, season);
  const start = locked ? season.start : instant(startMinute);
  const end = instant(endMinute);
  const found = rescheduleIssues(season, start, end, serverTime);
  const shifted = shiftedBy(seasons, season, end);
  const removable = canDelete && canRemove(seasons, season);
  return (
    <fieldset className="flex flex-col gap-3 rounded-md border border-surface p-3">
      <legend className="font-mono text-sm text-text-secondary">{season.id}</legend>
      <div className="flex flex-wrap items-end gap-3">
        <label className={labelClass}>
          {t("ranked.start")}
          <input type="datetime-local" disabled={locked || !canEdit} value={startMinute} onChange={(event) => setStartMinute(event.target.value)} className={fieldClass} />
        </label>
        <label className={labelClass}>
          {t("ranked.end")}
          <input type="datetime-local" disabled={!canEdit} value={endMinute} onChange={(event) => setEndMinute(event.target.value)} className={fieldClass} />
        </label>
        <label className="flex items-center gap-2 text-sm text-text-secondary">
          <input type="checkbox" disabled={!canEdit} checked={noRewards} onChange={(event) => setNoRewards(event.target.checked)} />
          {t("ranked.no_rewards")}
        </label>
      </div>
      {locked && <p className="text-sm text-text-muted">{t("ranked.start_locked")}</p>}
      <Shifted ids={shifted} />
      <Reason value={reason} onChange={setReason} />
      <div className="flex gap-3">
        {canEdit && (
          <button type="button" disabled={pending || found.length > 0 || reason.trim() === ""} onClick={() => run(() => rescheduleSeason(season.id, start, end, noRewards, reason))} className={actionClass}>
            {t("ranked.save")}
          </button>
        )}
        {removable && (
          <button type="button" disabled={pending || reason.trim() === ""} onClick={() => run(() => deleteSeason(season.id, reason))} className="self-start rounded-md bg-surface px-4 py-2 text-danger-text disabled:opacity-40">
            {t("ranked.remove")}
          </button>
        )}
        <button type="button" onClick={onDone} className="self-start rounded-md bg-surface px-4 py-2 text-text-secondary">
          {t("ranked.cancel")}
        </button>
      </div>
      <Issues keys={found} />
      <Outcome result={result} />
    </fieldset>
  );
}

function LiveSeason({ seasons, season, onDone }: { seasons: Season[]; season: Season; onDone: () => void }) {
  const t = useTranslator();
  const [endMinute, setEndMinute] = useState(windowMinute(season.end));
  const [reason, setReason] = useState("");
  const { result, pending, run } = useSeasonChange(onDone);
  const end = instant(endMinute);
  const found = extendIssues(season, end);
  return (
    <fieldset className="flex flex-col gap-3 rounded-md border border-surface p-3">
      <legend className="font-mono text-sm text-text-secondary">{season.id}</legend>
      <p className="text-sm text-text-muted">{t("ranked.live_hint")}</p>
      <label className={labelClass}>
        {t("ranked.end")}
        <input type="datetime-local" value={endMinute} onChange={(event) => setEndMinute(event.target.value)} className={`${fieldClass} self-start`} />
      </label>
      <Shifted ids={shiftedBy(seasons, season, end)} />
      <Reason value={reason} onChange={setReason} />
      <div className="flex gap-3">
        <button type="button" disabled={pending || found.length > 0 || reason.trim() === ""} onClick={() => run(() => extendSeason(season.id, end, reason))} className={actionClass}>
          {t("ranked.extend")}
        </button>
        <button type="button" onClick={onDone} className="self-start rounded-md bg-surface px-4 py-2 text-text-secondary">
          {t("ranked.cancel")}
        </button>
      </div>
      <Issues keys={found} />
      <Outcome result={result} />
    </fieldset>
  );
}

function NameKey({ nameKey }: { nameKey: string }) {
  const t = useTranslator();
  return (
    <p className="text-sm text-text-muted">
      {t("ranked.name_key")}:{" "}
      <Link href={keyHref(nameKey)} prefetch={false} className="font-mono text-primary underline">
        {nameKey}
      </Link>
      . {t("ranked.name_key_hint")}
    </p>
  );
}

function Shifted({ ids }: { ids: string[] }) {
  const t = useTranslator();
  if (ids.length === 0) {
    return null;
  }
  return (
    <p className="text-sm text-text-muted">
      {t("ranked.shifted")}: <span className="font-mono">{ids.join(", ")}</span>
    </p>
  );
}

function Reason({ value, onChange }: { value: string; onChange: (value: string) => void }) {
  const t = useTranslator();
  return (
    <label className={labelClass}>
      {t("ranked.reason")}
      <input value={value} onChange={(event) => onChange(event.target.value)} className={fieldClass} />
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
      {keys.map((key) => (
        <li key={key}>{t(key)}</li>
      ))}
    </ul>
  );
}

function Outcome({ result }: { result: SeasonResult | null }) {
  const t = useTranslator();
  if (!result || result.ok) {
    return null;
  }
  return (
    <>
      <p role="status" className="text-sm text-text-secondary">
        {t("ranked.refused")}: {result.error} {result.detail}
      </p>
      <UnnamedKeys keys={result.keys} />
    </>
  );
}
