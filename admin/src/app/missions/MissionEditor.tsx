"use client";

import { useState, useTransition } from "react";
import type { ConfigResult } from "@/config/result";
import { useTranslator } from "@/i18n/client";
import type { MessageKey } from "@/i18n/translate";
import {
  addMission,
  availabilities,
  chainCandidates,
  idList,
  issues,
  newMission,
  periodic,
  replaceMission,
  slotCount,
  validSlug,
  withBundle,
  withFilter,
  withNext,
  withType,
  type Mission,
  type MissionCatalog,
  type MissionRules,
  type MissionType,
} from "@/missions/catalog";
import { publishMissions } from "./actions";
import { UnnamedKeys, UnnamedKeysAhead } from "@/app/strings/UnnamedKeys";

const fieldClass = "rounded-md bg-surface px-3 py-1 text-text-primary";
const labelClass = "flex flex-col gap-1 text-sm text-text-secondary";

const typeLabel: Record<MissionType, MessageKey> = {
  daily: "missions.type.daily",
  weekly: "missions.type.weekly",
  achievement: "missions.type.achievement",
  event: "missions.type.event",
};

const eventLabel: Record<string, MessageKey> = {
  match_completed: "missions.event.match_completed",
  match_won: "missions.event.match_won",
  hand_won: "missions.event.hand_won",
  domino: "missions.event.domino",
  board_points: "missions.event.board_points",
  double_chain: "missions.event.double_chain",
};

const queueLabel: Record<string, MessageKey> = {
  ranked: "missions.queue.ranked",
  casual: "missions.queue.casual",
};

type Props = { initialCatalog: MissionCatalog; published: MissionCatalog | null; version: number; rules: MissionRules; canPublish: boolean };

export function MissionEditor({ initialCatalog, published, version, rules, canPublish }: Props) {
  const t = useTranslator();
  const [catalog, setCatalog] = useState(initialCatalog);
  const [selected, setSelected] = useState(initialCatalog.missions[0]?.id ?? "");
  const [reason, setReason] = useState("");
  const [result, setResult] = useState<ConfigResult | null>(null);
  const [pending, startTransition] = useTransition();
  const mission = catalog.missions.find((candidate) => candidate.id === selected);
  const blocked = catalog.missions.some((candidate) => issues(catalog, candidate).length > 0);

  return (
    <div className="flex flex-col gap-8">
      <p className="text-sm text-text-muted">
        {t("missions.version")}: {version}
      </p>
      <section className="flex flex-col gap-3">
        <h2 className="text-lg font-semibold">{t("missions.list")}</h2>
        {catalog.missions.length > 0 && (
          <select aria-label={t("missions.list")} value={selected} onChange={(event) => setSelected(event.target.value)} className={fieldClass}>
            {catalog.missions.map((candidate) => (
              <option key={candidate.id} value={candidate.id}>
                {candidate.id} · {t(typeLabel[candidate.type])} · {t(candidate.availability === "live" ? "missions.availability.live" : "missions.availability.retired")}
              </option>
            ))}
          </select>
        )}
        <NewMission
          rules={rules}
          onAdd={(created) => {
            const next = addMission(catalog, created);
            if (!next) {
              return false;
            }
            setCatalog(next);
            setSelected(created.id);
            return true;
          }}
        />
      </section>
      {mission && (
        <MissionFields
          key={mission.id}
          catalog={catalog}
          mission={mission}
          published={published?.missions.find((candidate) => candidate.id === mission.id)}
          rules={rules}
          onChange={(next) => setCatalog((current) => replaceMission(current, next))}
        />
      )}
      {canPublish && (
        <section className="flex flex-col gap-3">
          <label className={labelClass}>
            {t("missions.reason")}
            <input value={reason} onChange={(event) => setReason(event.target.value)} className={fieldClass} />
          </label>
          <UnnamedKeysAhead keys={catalog.missions.map((item) => item.nameKey)} />
          <button
            type="button"
            disabled={pending || blocked || catalog.missions.length === 0}
            onClick={() => startTransition(async () => setResult(await publishMissions(catalog, reason)))}
            className="self-start rounded-md bg-primary px-4 py-2 text-on-primary disabled:opacity-40"
          >
            {t("missions.publish")}
          </button>
          {blocked && <p className="text-sm text-text-muted">{t("missions.blocked")}</p>}
          {result && (
            <p role="status" className="text-sm text-text-secondary">
              {result.ok ? `${t("missions.published")}: ${result.version}` : `${t("missions.refused")}: ${result.error} ${result.detail}`}
            </p>
          )}
          {result && !result.ok && <UnnamedKeys keys={result.keys} />}
        </section>
      )}
    </div>
  );
}

function NewMission({ rules, onAdd }: { rules: MissionRules; onAdd: (mission: Mission) => boolean }) {
  const t = useTranslator();
  const [slug, setSlug] = useState("");
  const [type, setType] = useState<MissionType>(rules.types[0] ?? "daily");
  const [taken, setTaken] = useState(false);
  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-end gap-3">
        <label className={labelClass}>
          {t("missions.new_slug")}
          <span className="flex items-center gap-1 font-mono">
            mission.
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
        <TypeSelect value={type} types={rules.types} onChange={setType} />
        <button
          type="button"
          disabled={!validSlug(slug)}
          onClick={() => {
            if (onAdd(newMission(slug, type, rules))) {
              setSlug("");
              return;
            }
            setTaken(true);
          }}
          className="rounded-md bg-surface px-4 py-2 text-text-secondary disabled:opacity-40"
        >
          {t("missions.add")}
        </button>
      </div>
      {slug && !validSlug(slug) && <p className="text-sm text-text-muted">{t("missions.slug_rule")}</p>}
      {taken && (
        <p role="alert" className="text-sm text-danger-text">
          {t("missions.slug_taken")}
        </p>
      )}
    </div>
  );
}

function TypeSelect({ value, types, onChange }: { value: MissionType; types: MissionType[]; onChange: (type: MissionType) => void }) {
  const t = useTranslator();
  return (
    <label className={labelClass}>
      {t("missions.type")}
      <select value={value} onChange={(event) => onChange(event.target.value as MissionType)} className={fieldClass}>
        {types.map((candidate) => (
          <option key={candidate} value={candidate}>
            {t(typeLabel[candidate])}
          </option>
        ))}
      </select>
    </label>
  );
}

type FieldsProps = { catalog: MissionCatalog; mission: Mission; published: Mission | undefined; rules: MissionRules; onChange: (mission: Mission) => void };

function MissionFields({ catalog, mission, published, rules, onChange }: FieldsProps) {
  const t = useTranslator();
  const found = issues(catalog, mission);
  return (
    <section className="flex flex-col gap-4">
      <h2 className="font-mono text-lg font-semibold">{mission.id}</h2>
      <p className="text-sm text-text-muted">
        {t("missions.name_key")}: <span className="font-mono">{mission.nameKey}</span> · {t("missions.revision")}: {mission.revision || "—"}
      </p>
      <div className="flex flex-wrap items-end gap-3">
        <TypeSelect value={mission.type} types={rules.types} onChange={(type) => onChange(withType(mission, type))} />
        <label className={labelClass}>
          {t("missions.availability")}
          <select value={mission.availability} onChange={(event) => onChange({ ...mission, availability: event.target.value as Mission["availability"] })} className={fieldClass}>
            {availabilities.map((candidate) => (
              <option key={candidate} value={candidate}>
                {t(candidate === "live" ? "missions.availability.live" : "missions.availability.retired")}
              </option>
            ))}
          </select>
        </label>
      </div>
      <fieldset className="flex flex-col gap-3 rounded-md border border-surface p-3">
        <legend className="text-sm text-text-secondary">{t("missions.condition")}</legend>
        <div className="flex flex-wrap items-end gap-3">
          <label className={labelClass}>
            {t("missions.event")}
            <select value={mission.event} onChange={(event) => onChange({ ...mission, event: event.target.value })} className={fieldClass}>
              {rules.events.map((candidate) => (
                <option key={candidate} value={candidate}>
                  {eventLabel[candidate] ? t(eventLabel[candidate]) : candidate}
                </option>
              ))}
            </select>
          </label>
          <NumberField label={t("missions.target")} value={mission.target} onChange={(target) => onChange({ ...mission, target })} />
          <label className={labelClass}>
            {t("missions.queue")}
            <select value={mission.filters.queue ?? ""} onChange={(event) => onChange(withFilter(mission, "queue", event.target.value))} className={fieldClass}>
              <option value="">{t("missions.any")}</option>
              {rules.queues.map((candidate) => (
                <option key={candidate} value={candidate}>
                  {queueLabel[candidate] ? t(queueLabel[candidate]) : candidate}
                </option>
              ))}
            </select>
          </label>
          <label className={labelClass}>
            {t("missions.mode")}
            <select value={mission.filters.mode ?? ""} onChange={(event) => onChange(withFilter(mission, "mode", event.target.value))} className={fieldClass}>
              <option value="">{t("missions.any")}</option>
              {rules.modes.map((candidate) => (
                <option key={candidate} value={candidate}>
                  {candidate}
                </option>
              ))}
            </select>
          </label>
        </div>
      </fieldset>
      {periodic(mission.type) ? <SlotFields catalog={catalog} mission={mission} onChange={onChange} /> : <RewardFields catalog={catalog} mission={mission} published={published} onChange={onChange} />}
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

function SlotFields({ catalog, mission, onChange }: { catalog: MissionCatalog; mission: Mission; onChange: (mission: Mission) => void }) {
  const t = useTranslator();
  const slots = slotCount(catalog, mission.type);
  return (
    <fieldset className="flex flex-col gap-3 rounded-md border border-surface p-3">
      <legend className="text-sm text-text-secondary">{t("missions.rotation")}</legend>
      <div className="flex flex-wrap items-end gap-3">
        <NumberField label={slots === null ? t("missions.slot") : `${t("missions.slot")} (1–${slots})`} value={mission.slot ?? 0} onChange={(slot) => onChange({ ...mission, slot })} />
        <NumberField label={t("missions.weight")} value={mission.weight ?? 0} onChange={(weight) => onChange({ ...mission, weight })} />
      </div>
      <p className="text-sm text-text-muted">{t("missions.slot_reward")}</p>
    </fieldset>
  );
}

function RewardFields({ catalog, mission, published, onChange }: { catalog: MissionCatalog; mission: Mission; published: Mission | undefined; onChange: (mission: Mission) => void }) {
  const t = useTranslator();
  const bundle = mission.bundle;
  if (!bundle) {
    return null;
  }
  const change = (next: Parameters<typeof withBundle>[1]) => onChange(withBundle(mission, next, published));
  return (
    <fieldset className="flex flex-col gap-3 rounded-md border border-surface p-3">
      <legend className="text-sm text-text-secondary">
        {t("missions.reward")} · <span className="font-mono">{bundle.id}</span> · {t("missions.revision")} {bundle.revision}
      </legend>
      <div className="flex flex-wrap items-end gap-3">
        <NumberField label={t("currency.coin")} value={bundle.coins ?? 0} onChange={(coins) => change({ coins })} />
        {mission.type === "event" && <NumberField label={t("missions.pass_xp")} value={bundle.passXp ?? 0} onChange={(passXp) => change({ passXp })} />}
      </div>
      <IdsField label={t("missions.cosmetics")} ids={bundle.cosmetics ?? []} onChange={(cosmetics) => change({ cosmetics })} />
      <IdsField label={t("missions.medals")} ids={bundle.medals ?? []} onChange={(medals) => change({ medals })} />
      <p className="text-sm text-text-muted">{t("missions.sources_hint")}</p>
      {mission.type === "achievement" && (
        <label className={labelClass}>
          {t("missions.next")}
          <select value={mission.next ?? ""} onChange={(event) => onChange(withNext(mission, event.target.value))} className={fieldClass}>
            <option value="">{t("missions.next_none")}</option>
            {chainCandidates(catalog, mission).map((candidate) => (
              <option key={candidate} value={candidate}>
                {candidate}
              </option>
            ))}
          </select>
        </label>
      )}
      {mission.type === "event" && (
        <label className={labelClass}>
          {t("missions.event_id")}
          <input value={mission.eventId ?? ""} onChange={(event) => onChange({ ...mission, eventId: event.target.value })} className={`${fieldClass} font-mono`} />
        </label>
      )}
    </fieldset>
  );
}

function NumberField({ label, value, onChange }: { label: string; value: number; onChange: (value: number) => void }) {
  return (
    <label className={labelClass}>
      {label}
      <input type="number" min={0} step={1} value={value} onChange={(event) => onChange(Number(event.target.value))} className={`${fieldClass} w-28`} />
    </label>
  );
}

function IdsField({ label, ids, onChange }: { label: string; ids: string[]; onChange: (ids: string[]) => void }) {
  const [text, setText] = useState(ids.join(", "));
  return (
    <label className={labelClass}>
      {label}
      <input
        value={text}
        onChange={(event) => {
          setText(event.target.value);
          onChange(idList(event.target.value));
        }}
        className={`${fieldClass} font-mono`}
      />
    </label>
  );
}
