import type { SearchParams } from "../audit/query.ts";
import type { MessageKey, Translate } from "../i18n/translate.ts";

export type AssetStatus = "awaiting_file" | "passed" | "rejected";

export type Asset = {
  id: string;
  item_id: string;
  slot: string;
  address: string;
  status: AssetStatus;
  format: string | null;
  width: number | null;
  height: number | null;
  bytes: number | null;
  content_hash: string | null;
  checks: Record<string, boolean> | null;
  texture_bytes: number | null;
  texture_limit: number | null;
  created_at: string;
  view_url?: string;
};

export type AssetRefusal = { ok: false; error: string; detail: string };
export type AssetUploadStart = { ok: true; id: string; uploadUrl: string } | AssetRefusal;
export type AssetCheckResult = { ok: true } | AssetRefusal;

const itemIdPattern = /^(avatar|frame|table|tile|emoji|medal)\.[a-z0-9_]+$/;

const checkLabels: Record<string, MessageKey> = {
  format: "assets.check.format",
  alpha: "assets.check.alpha",
};

const statusLabels: Record<AssetStatus, MessageKey> = {
  awaiting_file: "assets.status.awaiting_file",
  passed: "assets.status.passed",
  rejected: "assets.status.rejected",
};

export const assetStatuses = Object.keys(statusLabels) as AssetStatus[];

export function itemIdValid(id: string): boolean {
  return itemIdPattern.test(id);
}

export function statusLabel(status: AssetStatus, t: Translate): string {
  return t(statusLabels[status]);
}

export function failedCheckLabels(asset: Asset, t: Translate): string[] {
  const failed = Object.entries(asset.checks ?? {}).filter(([, passed]) => !passed);
  return failed.map(([name]) => (name === "size" ? sizeRefusal(asset, t) : name in checkLabels ? t(checkLabels[name]) : name));
}

function sizeRefusal(asset: Asset, t: Translate): string {
  return `${t("assets.check.size")}: ${t.number(asset.texture_bytes ?? 0)} B, ${t("assets.check.size_limit")} ${t.number(asset.texture_limit ?? 0)} B`;
}

export function sizeLabel(bytes: number): string {
  if (bytes < 1024) {
    return `${bytes} B`;
  }
  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)} KB`;
  }
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
}

export const assetPageSize = 50;

export type AssetQuery = { status: AssetStatus | ""; item: string; before: string };

const uuidPattern = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

function single(value: string | string[] | undefined): string {
  return (Array.isArray(value) ? value[0] : value)?.trim() ?? "";
}

export function readAssetQuery(params: SearchParams): AssetQuery {
  const status = single(params.status);
  const before = single(params.before);
  return {
    status: Object.hasOwn(statusLabels, status) ? (status as AssetStatus) : "",
    item: single(params.item),
    before: uuidPattern.test(before) ? before : "",
  };
}

function searchOf(query: AssetQuery): URLSearchParams {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(query)) {
    if (value) {
      search.set(key, value);
    }
  }
  return search;
}

export function assetsApiPath(query: AssetQuery): string {
  const search = searchOf(query);
  search.set("limit", String(assetPageSize));
  return `/admin/assets?${search}`;
}

const artMissingCodes = new Set(["cosmetic_art_missing", "medal_art_missing"]);

export function itemsWithoutArt(error: string, detail: string): string[] {
  const at = detail.lastIndexOf(": ");
  if (!artMissingCodes.has(error) || at < 0) {
    return [];
  }
  return detail
    .slice(at + 2)
    .split(",")
    .map((id) => id.trim())
    .filter(itemIdValid);
}

export function itemAssetsHref(id: string): string {
  return `/assets?${searchOf({ status: "", item: id, before: "" })}`;
}

export type StaticArt = { state: "passed"; url: string; width: number | null; height: number | null; uploadedAt: string } | { state: "missing" } | { state: "unavailable" };

export function staticArtApiPath(itemId: string): string {
  return `/admin/assets?${new URLSearchParams({ item: itemId, exact: "true", slot: "static", status: "passed", limit: "1" })}`;
}

export function staticArtOf(status: number, body: { store_configured?: boolean; assets?: Asset[] }): StaticArt {
  if (status !== 200 || !body.store_configured) {
    return { state: "unavailable" };
  }
  const asset = body.assets?.[0];
  if (!asset?.view_url) {
    return { state: "missing" };
  }
  return { state: "passed", url: asset.view_url, width: asset.width, height: asset.height, uploadedAt: asset.created_at };
}

export function olderAssetsHref(query: AssetQuery, assets: readonly Asset[]): string | null {
  if (assets.length < assetPageSize) {
    return null;
  }
  return `/assets?${searchOf({ ...query, before: assets[assets.length - 1].id })}`;
}
