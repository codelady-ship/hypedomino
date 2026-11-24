import type { Store } from "./prices.ts";

export type PushStatus = "change" | "unchanged" | "not_in_store" | "error" | "written" | "failed";

export type StorePrice = { currency: string; amount: string };

export type PushRow = {
  product: string;
  country: string;
  target_cents: number;
  current?: StorePrice;
  next?: StorePrice;
  status: PushStatus;
  detail?: string;
};

export type StoreReport = { store: Store; configured: boolean; rows: PushRow[]; counts: Partial<Record<PushStatus, number>> };

export type PushPlan = { config_version: number; stores: StoreReport[] };

export type PushResult = { ok: true; plan: PushPlan } | { ok: false; error: string; detail: string };

export const pushStatuses: readonly PushStatus[] = ["change", "written", "failed", "error", "not_in_store", "unchanged"];

const actionable: ReadonlySet<PushStatus> = new Set(["change", "written", "failed", "error"]);

export function visibleRows(rows: readonly PushRow[], onlyChanges: boolean): PushRow[] {
  return onlyChanges ? rows.filter((row) => actionable.has(row.status)) : [...rows];
}

export function statusCounts(report: StoreReport): [PushStatus, number][] {
  return pushStatuses.flatMap((status): [PushStatus, number][] => {
    const count = report.counts[status] ?? 0;
    return count > 0 ? [[status, count]] : [];
  });
}

export function pendingChanges(plan: PushPlan): number {
  return plan.stores.reduce((total, report) => total + (report.counts.change ?? 0), 0);
}

export function shownPrice(price: StorePrice | undefined): string {
  return price ? `${price.amount} ${price.currency}` : "";
}
