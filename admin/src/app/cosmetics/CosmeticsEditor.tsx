"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";
import type { ConfigResult } from "@/config/result";
import {
  addedItems,
  addItem,
  changedItems,
  emojiGroups,
  emptyFilter,
  filteredItems,
  isPublished,
  itemIssues,
  missingDefaults,
  moveLayer,
  newItem,
  newlyDisabled,
  removeItem,
  replaceItem,
  validSlug,
  withAnimated,
  withBulkChange,
  withLayers,
  withNewSource,
  withoutSource,
  withSource,
  type BulkChange,
  type CosmeticItem,
  type CosmeticRules,
  type ItemFilter,
} from "@/cosmetics/catalog";
import { useTranslator } from "@/i18n/client";
import type { MessageKey } from "@/i18n/translate";
import { publishCosmetics } from "./actions";
import { StaticPreview } from "./StaticPreview";
import { ItemsWithoutArt } from "@/app/assets/ItemsWithoutArt";
import { UnnamedKeys } from "@/app/strings/UnnamedKeys";

const fieldClass = "rounded-md bg-surface px-3 py-1 text-text-primary disabled:opacity-50";
const labelClass = "flex flex-col gap-1 text-sm text-text-secondary";
const quietButton = "rounded-md bg-surface px-3 py-1 text-sm text-text-secondary disabled:opacity-40";

const typeLabel: Record<string, MessageKey> = {
  avatar: "cosmetics.type.avatar",
  frame: "cosmetics.type.frame",
  table: "cosmetics.type.table",
  tile: "cosmetics.type.tile",
  emoji: "cosmetics.type.emoji",
};

const availabilityLabel: Record<string, MessageKey> = {
  live: "cosmetics.availability.live",
  retired: "cosmetics.availability.retired",
  disabled: "cosmetics.availability.disabled",
};

const sourceLabel: Record<string, MessageKey> = {
  default: "cosmetics.source.default",
  shop: "cosmetics.source.shop",
  pass: "cosmetics.source.pass",
  mission: "cosmetics.source.mission",
  promo: "cosmetics.source.promo",
  rank_reward: "cosmetics.source.rank_reward",
  first_purchase: "cosmetics.source.first_purchase",
  event: "cosmetics.source.event",
  chest: "cosmetics.source.chest",
};

const artLabel: Record<string, MessageKey> = { handmade: "cosmetics.art.handmade", ai_assisted: "cosmetics.art.ai_assisted" };

const rarityLabel: Record<string, MessageKey> = {
  common: "rarity.common",
  rare: "rarity.rare",
  epic: "rarity.epic",
  legendary: "rarity.legendary",
  mythic: "rarity.mythic",
};

function useLabel() {
  const t = useTranslator();
  return (map: Record<string, MessageKey>, id: string) => (map[id] ? t(map[id]) : id);
}

type Props = { published: CosmeticItem[]; manifestVersion: number; version: number; rules: CosmeticRules; canPublish: boolean; canReview: boolean; canSeeArt: boolean };

export function CosmeticsEditor({ published, manifestVersion, version, rules, canPublish, canReview, canSeeArt }: Props) {
  const t = useTranslator();
  const router = useRouter();
  const [items, setItems] = useState(published);
  const [selected, setSelected] = useState("");
  const [filter, setFilter] = useState<ItemFilter>(emptyFilter);
  const [checked, setChecked] = useState<ReadonlySet<string>>(new Set());
  const [reason, setReason] = useState("");
  const [reviewing, setReviewing] = useState(false);
  const [result, setResult] = useState<ConfigResult | null>(null);
  const [pending, startTransition] = useTransition();
  const item = items.find((candidate) => candidate.id === selected);
  const visible = filteredItems(items, filter);
  const uncovered = missingDefaults(items, rules.types);
  const blocked = uncovered.length > 0 || items.some((candidate) => itemIssues(candidate).length > 0);
  const change = (next: CosmeticItem[]) => {
    setItems(next);
    setReviewing(false);
  };

  return (
    <div className="flex flex-col gap-8">
      <p className="text-sm text-text-muted">
        {t("cosmetics.version")}: {version} · {t("cosmetics.manifest_version")}: {manifestVersion} · {t("cosmetics.item_count")}: {items.length}
      </p>
      <NewItem
        rules={rules}
        items={items}
        onAdd={(next, id) => {
          change(next);
          setSelected(id);
        }}
      />
      <section className="flex flex-col gap-3">
        <h2 className="text-lg font-semibold">{t("cosmetics.list")}</h2>
        <Filters rules={rules} filter={filter} onChange={setFilter} />
        <BulkEdit
          rules={rules}
          count={checked.size}
          onApply={(bulk) => {
            change(withBulkChange(items, checked, bulk));
            setChecked(new Set());
          }}
        />
        <ItemTable items={visible} checked={checked} selected={selected} onCheck={setChecked} onSelect={setSelected} />
        {visible.length === 0 && <p className="text-sm text-text-secondary">{t("cosmetics.no_match")}</p>}
      </section>
      {item && (
        <ItemForm
          key={item.id}
          item={item}
          groups={emojiGroups(items, item.id)}
          rules={rules}
          canReview={canReview}
          canSeeArt={canSeeArt}
          removable={!isPublished(published, item.id)}
          onChange={(next) => change(replaceItem(items, next))}
          onRemove={() => {
            change(removeItem(items, item.id));
            setSelected("");
          }}
        />
      )}
      {uncovered.length > 0 && (
        <p role="alert" className="text-sm text-danger-text">
          {t("cosmetics.issue.missing_default")}: <span className="font-mono">{uncovered.join(", ")}</span>
        </p>
      )}
      {canPublish && (
        <section className="flex flex-col gap-3">
          <label className={labelClass}>
            {t("cosmetics.reason")}
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
              {t("cosmetics.review")}
            </button>
          ) : (
            <Review
              before={published}
              after={items}
              pending={pending}
              onCancel={() => setReviewing(false)}
              onConfirm={() =>
                startTransition(async () => {
                  const outcome = await publishCosmetics(items, reason);
                  setResult(outcome);
                  setReviewing(false);
                  if (outcome.ok) {
                    router.refresh();
                  }
                })
              }
            />
          )}
          {blocked && <p className="text-sm text-text-muted">{t("cosmetics.blocked")}</p>}
          {result && !result.ok && (
            <p role="status" className="text-sm text-text-secondary">
              {t("cosmetics.refused")}: {result.error} {result.detail}
            </p>
          )}
          {result && !result.ok && <UnnamedKeys keys={result.keys} />}
          {result && !result.ok && <ItemsWithoutArt error={result.error} detail={result.detail} />}
        </section>
      )}
    </div>
  );
}

function NewItem({ rules, items, onAdd }: { rules: CosmeticRules; items: CosmeticItem[]; onAdd: (items: CosmeticItem[], id: string) => void }) {
  const t = useTranslator();
  const label = useLabel();
  const [type, setType] = useState(rules.types[0] ?? "");
  const [slug, setSlug] = useState("");
  const created = validSlug(slug) ? addItem(items, newItem(type, slug, rules)) : null;
  return (
    <fieldset className="flex flex-col gap-3 rounded-md border border-surface p-3">
      <legend className="text-sm text-text-secondary">{t("cosmetics.new_item")}</legend>
      <div className="flex flex-wrap items-end gap-3">
        <SelectField label={t("cosmetics.type")} value={type} options={rules.types} display={(id) => label(typeLabel, id)} onChange={setType} />
        <label className={labelClass}>
          {t("cosmetics.slug")}
          <input value={slug} onChange={(event) => setSlug(event.target.value)} className={`${fieldClass} font-mono`} />
        </label>
        <button
          type="button"
          disabled={!created}
          onClick={() => {
            if (created) {
              onAdd(created, `${type}.${slug}`);
              setSlug("");
            }
          }}
          className="rounded-md bg-surface px-4 py-2 text-text-secondary disabled:opacity-40"
        >
          {t("cosmetics.add")}
        </button>
      </div>
      <p className="text-sm text-text-muted">{t("cosmetics.new_hint")}</p>
      {slug && !created && (
        <p role="alert" className="text-sm text-danger-text">
          {t(validSlug(slug) ? "cosmetics.issue.id_taken" : "cosmetics.issue.slug")}
        </p>
      )}
    </fieldset>
  );
}

function Filters({ rules, filter, onChange }: { rules: CosmeticRules; filter: ItemFilter; onChange: (filter: ItemFilter) => void }) {
  const t = useTranslator();
  const label = useLabel();
  return (
    <div className="flex flex-wrap items-end gap-3">
      <label className={labelClass}>
        {t("cosmetics.search")}
        <input value={filter.query} onChange={(event) => onChange({ ...filter, query: event.target.value })} className={fieldClass} />
      </label>
      <SelectField label={t("cosmetics.type")} value={filter.type} empty={t("cosmetics.all")} options={rules.types} display={(id) => label(typeLabel, id)} onChange={(type) => onChange({ ...filter, type })} />
      <SelectField label={t("cosmetics.rarity")} value={filter.rarityId} empty={t("cosmetics.all")} options={rules.rarities} display={(id) => label(rarityLabel, id)} onChange={(rarityId) => onChange({ ...filter, rarityId })} />
      <SelectField label={t("cosmetics.availability")} value={filter.availability} empty={t("cosmetics.all")} options={rules.availability} display={(id) => label(availabilityLabel, id)} onChange={(availability) => onChange({ ...filter, availability })} />
      <SelectField label={t("cosmetics.sources")} value={filter.sourceType} empty={t("cosmetics.all")} options={rules.source_types} display={(id) => label(sourceLabel, id)} onChange={(sourceType) => onChange({ ...filter, sourceType })} />
    </div>
  );
}

function BulkEdit({ rules, count, onApply }: { rules: CosmeticRules; count: number; onApply: (change: BulkChange) => void }) {
  const t = useTranslator();
  const label = useLabel();
  const [rarityId, setRarityId] = useState("");
  const [availability, setAvailability] = useState("");
  return (
    <fieldset className="flex flex-wrap items-end gap-3 rounded-md border border-surface p-3">
      <legend className="text-sm text-text-secondary">
        {t("cosmetics.bulk")}: {count}
      </legend>
      <SelectField label={t("cosmetics.rarity")} value={rarityId} empty={t("cosmetics.no_change")} options={rules.rarities} display={(id) => label(rarityLabel, id)} onChange={setRarityId} />
      <SelectField label={t("cosmetics.availability")} value={availability} empty={t("cosmetics.no_change")} options={rules.availability} display={(id) => label(availabilityLabel, id)} onChange={setAvailability} />
      <button
        type="button"
        disabled={count === 0 || (!rarityId && !availability)}
        onClick={() => {
          onApply({ ...(rarityId && { rarityId }), ...(availability && { availability }) });
          setRarityId("");
          setAvailability("");
        }}
        className={quietButton}
      >
        {t("cosmetics.bulk_apply")}
      </button>
    </fieldset>
  );
}

type TableProps = { items: CosmeticItem[]; checked: ReadonlySet<string>; selected: string; onCheck: (checked: ReadonlySet<string>) => void; onSelect: (id: string) => void };

function ItemTable({ items, checked, selected, onCheck, onSelect }: TableProps) {
  const t = useTranslator();
  const label = useLabel();
  const allChecked = items.length > 0 && items.every((item) => checked.has(item.id));
  const toggle = (id: string, on: boolean) => {
    const next = new Set(checked);
    if (on) {
      next.add(id);
    } else {
      next.delete(id);
    }
    onCheck(next);
  };
  return (
    <div className="max-h-[32rem] overflow-y-auto">
      <table className="w-full text-sm">
        <thead className="sticky top-0 bg-background-base text-left text-text-muted">
          <tr>
            <th className="py-1 pr-3">
              <input type="checkbox" aria-label={t("cosmetics.check_all")} checked={allChecked} onChange={(event) => onCheck(event.target.checked ? new Set([...checked, ...items.map((item) => item.id)]) : new Set())} />
            </th>
            <th className="py-1 pr-3">{t("cosmetics.id")}</th>
            <th className="py-1 pr-3">{t("cosmetics.rarity")}</th>
            <th className="py-1 pr-3">{t("cosmetics.availability")}</th>
            <th className="py-1 pr-3">{t("cosmetics.animated")}</th>
            <th className="py-1 pr-3">{t("cosmetics.sources")}</th>
            <th className="py-1 pr-3">{t("cosmetics.revision")}</th>
            <th className="py-1 pr-3" />
          </tr>
        </thead>
        <tbody>
          {items.map((item) => (
            <tr key={item.id} className={`border-t border-surface align-top ${item.id === selected ? "bg-surface" : ""}`}>
              <td className="py-1 pr-3">
                <input type="checkbox" aria-label={`${t("cosmetics.check")} ${item.id}`} checked={checked.has(item.id)} onChange={(event) => toggle(item.id, event.target.checked)} />
              </td>
              <td className="py-1 pr-3 font-mono">
                <button type="button" onClick={() => onSelect(item.id)} className="text-left underline">
                  {item.id}
                </button>
              </td>
              <td className="py-1 pr-3">{label(rarityLabel, item.rarityId)}</td>
              <td className="py-1 pr-3">{label(availabilityLabel, item.availability)}</td>
              <td className="py-1 pr-3">{t(item.animated ? "cosmetics.yes" : "cosmetics.no")}</td>
              <td className="py-1 pr-3">{item.sources.map((source) => label(sourceLabel, source.type)).join(", ")}</td>
              <td className="py-1 pr-3">{item.revision || "—"}</td>
              <td className="py-1 pr-3 text-danger-text">{itemIssues(item).length > 0 && t("cosmetics.has_issues")}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

type FormProps = { item: CosmeticItem; groups: string[]; rules: CosmeticRules; canReview: boolean; canSeeArt: boolean; removable: boolean; onChange: (item: CosmeticItem) => void; onRemove: () => void };

function ItemForm({ item, groups, rules, canReview, canSeeArt, removable, onChange, onRemove }: FormProps) {
  const t = useTranslator();
  const label = useLabel();
  return (
    <section className="flex flex-col gap-4 rounded-md border border-surface p-3">
      <h2 className="font-mono text-lg font-semibold">{item.id}</h2>
      <p className="text-sm text-text-muted">
        {t("cosmetics.server_fields")}: {t("cosmetics.revision")} {item.revision || "—"} · <span className="font-mono">{item.nameKey ?? `cosmetic.${item.id}.name`}</span>
      </p>
      {canSeeArt && <StaticPreview itemId={item.id} />}
      <div className="flex flex-wrap items-end gap-3">
        <SelectField label={t("cosmetics.rarity")} value={item.rarityId} options={rules.rarities} display={(id) => label(rarityLabel, id)} onChange={(rarityId) => onChange({ ...item, rarityId })} />
        <SelectField label={t("cosmetics.availability")} value={item.availability} options={rules.availability} display={(id) => label(availabilityLabel, id)} onChange={(availability) => onChange({ ...item, availability })} />
        <SelectField label={t("cosmetics.art_source")} value={item.artSource} options={rules.art_sources} display={(id) => label(artLabel, id)} onChange={(artSource) => onChange({ ...item, artSource })} />
        <Toggle label={t("cosmetics.animated")} checked={item.animated} onChange={(animated) => onChange(withAnimated(item, animated))} />
        <Toggle label={t("cosmetics.review_approved")} checked={item.culturalReview === "approved"} disabled={!canReview} onChange={(on) => onChange({ ...item, culturalReview: on ? "approved" : "" })} />
      </div>
      <TypeFields item={item} groups={groups} onChange={onChange} />
      <Sources item={item} rules={rules} onChange={onChange} />
      <Issues keys={itemIssues(item)} />
      {removable && (
        <button type="button" onClick={onRemove} className={`${quietButton} self-start`}>
          {t("cosmetics.remove")}
        </button>
      )}
      {!removable && <p className="text-sm text-text-muted">{t("cosmetics.published_hint")}</p>}
    </section>
  );
}

function TypeFields({ item, groups, onChange }: { item: CosmeticItem; groups: string[]; onChange: (item: CosmeticItem) => void }) {
  if (item.type === "emoji") {
    return <EmojiGroup item={item} groups={groups} onChange={onChange} />;
  }
  if (item.type === "table" && item.animated) {
    return <Layers item={item} onChange={onChange} />;
  }
  return null;
}

function EmojiGroup({ item, groups, onChange }: { item: CosmeticItem; groups: string[]; onChange: (item: CosmeticItem) => void }) {
  const t = useTranslator();
  const group = item.group ?? "";
  return (
    <div className="flex flex-col gap-1">
      <label className={labelClass}>
        {t("cosmetics.group")}
        <input list="emoji-groups" value={group} onChange={(event) => onChange({ ...item, group: event.target.value })} className={`${fieldClass} self-start font-mono`} />
      </label>
      <datalist id="emoji-groups">
        {groups.map((known) => (
          <option key={known} value={known} />
        ))}
      </datalist>
      {groups.length > 0 && (
        <p className="text-sm text-text-muted">
          {t("cosmetics.group_known")}: <span className="font-mono">{groups.join(", ")}</span>
        </p>
      )}
      {group !== "" && !groups.includes(group) && <p className="text-sm text-text-muted">{t("cosmetics.group_new")}</p>}
    </div>
  );
}

function Layers({ item, onChange }: { item: CosmeticItem; onChange: (item: CosmeticItem) => void }) {
  const t = useTranslator();
  const layers = item.layers ?? [];
  const set = (index: number, name: string, animated: boolean) => onChange(withLayers(item, layers.map((layer, at) => (at === index ? { ...layer, name, animated } : layer))));
  return (
    <fieldset className="flex flex-col gap-3 rounded-md border border-surface p-3">
      <legend className="text-sm text-text-secondary">{t("cosmetics.layers")}</legend>
      <p className="text-sm text-text-muted">{t("cosmetics.layers_hint")}</p>
      {layers.map((layer, index) => (
        <div key={index} className="flex flex-wrap items-center gap-3">
          <input aria-label={`${t("cosmetics.layer_name")} ${index + 1}`} value={layer.name} onChange={(event) => set(index, event.target.value, layer.animated)} className={`${fieldClass} font-mono`} />
          <Toggle label={t("cosmetics.animated")} checked={layer.animated} onChange={(animated) => set(index, layer.name, animated)} />
          <button type="button" disabled={index === 0} onClick={() => onChange(withLayers(item, moveLayer(layers, index, -1)))} className={quietButton}>
            {t("cosmetics.layer_up")}
          </button>
          <button type="button" disabled={index === layers.length - 1} onClick={() => onChange(withLayers(item, moveLayer(layers, index, 1)))} className={quietButton}>
            {t("cosmetics.layer_down")}
          </button>
          <button type="button" onClick={() => onChange(withLayers(item, layers.filter((_, at) => at !== index)))} className={quietButton}>
            {t("cosmetics.remove")}
          </button>
        </div>
      ))}
      <button type="button" onClick={() => onChange(withLayers(item, [...layers, { name: "", animated: true }]))} className={`${quietButton} self-start`}>
        {t("cosmetics.add_layer")}
      </button>
    </fieldset>
  );
}

function Sources({ item, rules, onChange }: { item: CosmeticItem; rules: CosmeticRules; onChange: (item: CosmeticItem) => void }) {
  const t = useTranslator();
  const label = useLabel();
  return (
    <fieldset className="flex flex-col gap-3 rounded-md border border-surface p-3">
      <legend className="text-sm text-text-secondary">{t("cosmetics.sources")}</legend>
      <p className="text-sm text-text-muted">{t("cosmetics.sources_hint")}</p>
      {item.sources.map((source, index) => (
        <div key={index} className="flex flex-wrap items-end gap-3">
          <SelectField label={`${t("cosmetics.source_type")} ${index + 1}`} value={source.type} options={rules.source_types} display={(id) => label(sourceLabel, id)} onChange={(type) => onChange(withSource(item, index, { type, ref: source.ref }))} />
          <label className={labelClass}>
            {`${t("cosmetics.source_ref")} ${index + 1}`}
            <input disabled={source.type === "default"} value={source.ref ?? ""} onChange={(event) => onChange(withSource(item, index, { type: source.type, ref: event.target.value }))} className={`${fieldClass} w-72 font-mono`} />
          </label>
          <button type="button" onClick={() => onChange(withoutSource(item, index))} className={quietButton}>
            {t("cosmetics.remove")}
          </button>
        </div>
      ))}
      <button type="button" onClick={() => onChange(withNewSource(item, rules))} className={`${quietButton} self-start`}>
        {t("cosmetics.add_source")}
      </button>
    </fieldset>
  );
}

function Review({ before, after, pending, onCancel, onConfirm }: { before: CosmeticItem[]; after: CosmeticItem[]; pending: boolean; onCancel: () => void; onConfirm: () => void }) {
  const t = useTranslator();
  const disabled = newlyDisabled(before, after);
  return (
    <div role="region" aria-label={t("cosmetics.review")} className="flex flex-col gap-2 rounded-md border border-surface p-3 text-sm">
      <p className="text-text-secondary">{t("cosmetics.review_hint")}</p>
      <p>
        {t("cosmetics.review_added")}: <span className="font-mono break-all">{addedItems(before, after).join(", ") || "—"}</span>
      </p>
      <p>
        {t("cosmetics.review_changed")}: <span className="font-mono break-all">{changedItems(before, after).join(", ") || "—"}</span>
      </p>
      {disabled.length > 0 && (
        <p className="text-danger-text">
          {t("cosmetics.review_disabled")}: <span className="font-mono break-all">{disabled.join(", ")}</span>
        </p>
      )}
      <div className="flex gap-3">
        <button type="button" disabled={pending} onClick={onConfirm} className="rounded-md bg-primary px-4 py-2 text-on-primary disabled:opacity-40">
          {t("cosmetics.confirm")}
        </button>
        <button type="button" onClick={onCancel} className="rounded-md bg-surface px-4 py-2 text-text-secondary">
          {t("cosmetics.cancel")}
        </button>
      </div>
    </div>
  );
}

type SelectProps = { label: string; value: string; options: readonly string[]; display: (id: string) => string; empty?: string; onChange: (value: string) => void };

function SelectField({ label, value, options, display, empty, onChange }: SelectProps) {
  return (
    <label className={labelClass}>
      <span>{label}</span>
      <select aria-label={label} value={value} onChange={(event) => onChange(event.target.value)} className={fieldClass}>
        {empty !== undefined && <option value="">{empty}</option>}
        {options.map((option) => (
          <option key={option} value={option}>
            {display(option)}
          </option>
        ))}
      </select>
    </label>
  );
}

function Toggle({ label, checked, disabled, onChange }: { label: string; checked: boolean; disabled?: boolean; onChange: (checked: boolean) => void }) {
  return (
    <label className="flex items-center gap-2 text-sm text-text-secondary">
      <input type="checkbox" disabled={disabled} checked={checked} onChange={(event) => onChange(event.target.checked)} />
      {label}
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
